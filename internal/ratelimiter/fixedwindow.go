package ratelimiter

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// fixedWindowScript is a Lua script that atomically increments a counter
	// and sets its expiration time if this is the first increment.
	// This ensures the counter automatically expires at the end of the window.
	//
	// KEYS[1]: The Redis key for the counter
	// ARGV[1]: The increment amount (n)
	// ARGV[2]: The TTL in seconds (window duration)
	//
	// Returns: The new counter value after incrementing
	fixedWindowScript = `
local current = redis.call('INCRBY', KEYS[1], ARGV[1])
if current == tonumber(ARGV[1]) then
    redis.call('EXPIRE', KEYS[1], ARGV[2])
end
return current
`
)

// fixedWindowLimiter implements the Fixed Window Counter algorithm.
// It uses a simple counter that resets at fixed time intervals.
type fixedWindowLimiter struct {
	client *redis.Client
	config *Config
}

// NewFixedWindow creates a new Fixed Window rate limiter.
func NewFixedWindow(client *redis.Client, config *Config) (RateLimiter, error) {
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

	return &fixedWindowLimiter{
		client: client,
		config: cfg,
	}, nil
}

// Allow checks if a single request is allowed for the given key.
func (f *fixedWindowLimiter) Allow(ctx context.Context, key string) (*Result, error) {
	return f.AllowN(ctx, key, 1)
}

// AllowN checks if N requests are allowed for the given key.
// Uses a Lua script to atomically increment and check the counter.
func (f *fixedWindowLimiter) AllowN(ctx context.Context, key string, n int64) (*Result, error) {
	if n <= 0 {
		return nil, ErrInvalidN
	}

	// Calculate current window start timestamp
	now := time.Now()
	windowStart := now.Truncate(f.config.Window).Unix()

	// Format Redis key with window timestamp
	redisKey := f.formatKey(key, windowStart)

	// Execute Lua script for atomic increment + check
	count, err := f.incrementAndCheck(ctx, redisKey, n)
	if err != nil {
		if f.config.FailOpen {
			// Fail open: allow the request
			return &Result{
				Allowed:    true,
				Limit:      f.config.Limit,
				Remaining:  0,
				RetryAfter: 0,
				ResetAt:    f.calculateResetTime(windowStart),
			}, nil
		}
		return nil, fmt.Errorf("failed to check rate limit: %w", err)
	}

	allowed := count <= f.config.Limit
	remaining := f.config.Limit - count
	if remaining < 0 {
		remaining = 0
	}

	result := &Result{
		Allowed:    allowed,
		Limit:      f.config.Limit,
		Remaining:  remaining,
		RetryAfter: 0,
		ResetAt:    f.calculateResetTime(windowStart),
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
func (f *fixedWindowLimiter) Reset(ctx context.Context, key string) error {
	// Calculate current window to delete the right key
	windowStart := time.Now().Truncate(f.config.Window).Unix()
	redisKey := f.formatKey(key, windowStart)

	if err := f.client.Del(ctx, redisKey).Err(); err != nil {
		return fmt.Errorf("failed to reset rate limit: %w", err)
	}

	return nil
}

// Close closes the rate limiter and releases resources.
func (f *fixedWindowLimiter) Close() error {
	if f.client != nil {
		return f.client.Close()
	}
	return nil
}

// formatKey formats the Redis key with prefix, user key, and window timestamp.
func (f *fixedWindowLimiter) formatKey(key string, windowStart int64) string {
	return fmt.Sprintf("%s:%d", f.config.FormatKey(key), windowStart)
}

// calculateResetTime calculates when the current window will reset.
func (f *fixedWindowLimiter) calculateResetTime(windowStart int64) time.Time {
	return time.Unix(windowStart, 0).Add(f.config.Window)
}

// incrementAndCheck atomically increments the counter and returns the new count.
// Uses a Lua script to ensure atomicity.
func (f *fixedWindowLimiter) incrementAndCheck(ctx context.Context, key string, n int64) (int64, error) {
	ttl := int64(f.config.Window.Seconds())
	result, err := f.client.Eval(ctx, fixedWindowScript, []string{key}, n, ttl).Result()
	if err != nil {
		return 0, err
	}

	count, ok := result.(int64)
	if !ok {
		return 0, fmt.Errorf("unexpected result type from Redis: %T", result)
	}

	return count, nil
}
