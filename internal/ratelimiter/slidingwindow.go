package ratelimiter

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// slidingWindowScript atomically retrieves previous and current window counts,
	// increments current count, and sets appropriate TTLs.
	//
	// KEYS[1]: Current window key
	// KEYS[2]: Previous window key
	// ARGV[1]: Increment amount (n)
	// ARGV[2]: Current window TTL in seconds
	// ARGV[3]: Previous window TTL in seconds
	//
	// Returns: {previous_count, current_count}
	slidingWindowScript = `
local prev = tonumber(redis.call('GET', KEYS[2]) or 0)
local curr = redis.call('INCRBY', KEYS[1], ARGV[1])
if curr == tonumber(ARGV[1]) then
    redis.call('EXPIRE', KEYS[1], ARGV[2])
end
redis.call('EXPIRE', KEYS[2], ARGV[3])
return {prev, curr}
`
)

// slidingWindowLimiter implements the Sliding Window Counter algorithm.
// It uses a weighted count from current and previous windows for smoother rate limiting.
type slidingWindowLimiter struct {
	client *redis.Client
	config *Config
}

// NewSlidingWindow creates a new Sliding Window rate limiter.
func NewSlidingWindow(client *redis.Client, config *Config) (RateLimiter, error) {
	if client == nil {
		return nil, fmt.Errorf("redis client cannot be nil")
	}
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Validate and apply defaults
	cfg := config.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &slidingWindowLimiter{
		client: client,
		config: cfg,
	}, nil
}

// Allow checks if a single request is allowed for the given key.
func (s *slidingWindowLimiter) Allow(ctx context.Context, key string) (*Result, error) {
	return s.AllowN(ctx, key, 1)
}

// AllowN checks if N requests are allowed for the given key.
// Uses sliding window algorithm with weighted count from previous and current windows.
func (s *slidingWindowLimiter) AllowN(ctx context.Context, key string, n int64) (*Result, error) {
	if n <= 0 {
		return nil, ErrInvalidN
	}

	now := time.Now()
	currWindowStart := now.Truncate(s.config.Window).Unix()
	prevWindowStart := currWindowStart - int64(s.config.Window.Seconds())

	// Format Redis keys for current and previous windows
	currKey := s.formatKey(key, currWindowStart)
	prevKey := s.formatKey(key, prevWindowStart)

	// Execute Lua script to get counts atomically
	prevCount, currCount, err := s.getCounts(ctx, currKey, prevKey, n)
	if err != nil {
		if s.config.FailOpen {
			// Fail open: allow the request
			return &Result{
				Allowed:    true,
				Limit:      s.config.Limit,
				Remaining:  0,
				RetryAfter: 0,
				ResetAt:    s.calculateResetTime(currWindowStart),
			}, nil
		}
		return nil, fmt.Errorf("failed to check rate limit: %w", err)
	}

	// Calculate weighted count based on position in current window
	weightedCount := s.calculateWeightedCount(now, currWindowStart, prevCount, currCount)

	allowed := weightedCount <= float64(s.config.Limit)
	remaining := s.config.Limit - int64(weightedCount)
	if remaining < 0 {
		remaining = 0
	}

	result := &Result{
		Allowed:    allowed,
		Limit:      s.config.Limit,
		Remaining:  remaining,
		RetryAfter: 0,
		ResetAt:    s.calculateResetTime(currWindowStart),
	}

	if !allowed {
		result.RetryAfter = time.Until(result.ResetAt)
		if result.RetryAfter < 0 {
			result.RetryAfter = 0
		}
	}

	return result, nil
}

// Reset resets the rate limit counter for the given key.
func (s *slidingWindowLimiter) Reset(ctx context.Context, key string) error {
	now := time.Now()
	currWindowStart := now.Truncate(s.config.Window).Unix()
	prevWindowStart := currWindowStart - int64(s.config.Window.Seconds())

	currKey := s.formatKey(key, currWindowStart)
	prevKey := s.formatKey(key, prevWindowStart)

	// Delete both current and previous window keys
	if err := s.client.Del(ctx, currKey, prevKey).Err(); err != nil {
		return fmt.Errorf("failed to reset rate limit: %w", err)
	}

	return nil
}

// Close closes the rate limiter and releases resources.
func (s *slidingWindowLimiter) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}

// formatKey formats the Redis key with prefix, user key, and window timestamp.
func (s *slidingWindowLimiter) formatKey(key string, windowStart int64) string {
	return fmt.Sprintf("%s:%d", s.config.FormatKey(key), windowStart)
}

// calculateResetTime calculates when the current window will reset.
func (s *slidingWindowLimiter) calculateResetTime(windowStart int64) time.Time {
	return time.Unix(windowStart, 0).Add(s.config.Window)
}

// getCounts retrieves previous and current window counts atomically.
func (s *slidingWindowLimiter) getCounts(ctx context.Context, currKey, prevKey string, n int64) (int64, int64, error) {
	currTTL := int64(s.config.Window.Seconds())
	prevTTL := int64(s.config.Window.Seconds() * 2) // Previous window lives for 2 windows

	result, err := s.client.Eval(ctx, slidingWindowScript, []string{currKey, prevKey}, n, currTTL, prevTTL).Result()
	if err != nil {
		return 0, 0, err
	}

	counts, ok := result.([]interface{})
	if !ok || len(counts) != 2 {
		return 0, 0, fmt.Errorf("unexpected result type from Redis: %T", result)
	}

	prevCount, ok := counts[0].(int64)
	if !ok {
		return 0, 0, fmt.Errorf("unexpected previous count type: %T", counts[0])
	}

	currCount, ok := counts[1].(int64)
	if !ok {
		return 0, 0, fmt.Errorf("unexpected current count type: %T", counts[1])
	}

	return prevCount, currCount, nil
}

// calculateWeightedCount calculates the weighted count using sliding window formula.
// Formula: prev_count * (1 - progress) + curr_count
// where progress = time_elapsed_in_current_window / window_duration
func (s *slidingWindowLimiter) calculateWeightedCount(now time.Time, windowStart int64, prevCount, currCount int64) float64 {
	windowStartTime := time.Unix(windowStart, 0)
	elapsedInWindow := now.Sub(windowStartTime)
	progress := float64(elapsedInWindow) / float64(s.config.Window)

	// Weighted count = previous * (1 - progress) + current
	return float64(prevCount)*(1.0-progress) + float64(currCount)
}
