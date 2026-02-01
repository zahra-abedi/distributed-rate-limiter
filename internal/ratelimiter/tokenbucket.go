package ratelimiter

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// tokenBucketScript atomically refills tokens based on elapsed time,
	// attempts to consume requested tokens, and returns the result.
	//
	// KEYS[1]: Redis key for token bucket state
	// ARGV[1]: Maximum capacity (limit)
	// ARGV[2]: Tokens to consume (n)
	// ARGV[3]: Refill rate (tokens per second as float)
	// ARGV[4]: Current timestamp (seconds)
	// ARGV[5]: TTL for the key (seconds)
	//
	// Returns: {allowed (0/1), tokens_remaining}
	tokenBucketScript = `
local capacity = tonumber(ARGV[1])
local requested = tonumber(ARGV[2])
local refill_rate = tonumber(ARGV[3])
local now = tonumber(ARGV[4])
local ttl = tonumber(ARGV[5])

-- Get current state or initialize
local state = redis.call('HMGET', KEYS[1], 'tokens', 'last_refill')
local tokens = tonumber(state[1]) or capacity
local last_refill = tonumber(state[2]) or now

-- Calculate tokens to add based on elapsed time
local elapsed = now - last_refill
local tokens_to_add = elapsed * refill_rate
tokens = math.min(capacity, tokens + tokens_to_add)

-- Try to consume tokens
local allowed = 0
if tokens >= requested then
    tokens = tokens - requested
    allowed = 1
end

-- Save new state
redis.call('HMSET', KEYS[1], 'tokens', tostring(tokens), 'last_refill', tostring(now))
redis.call('EXPIRE', KEYS[1], ttl)

return {allowed, math.floor(tokens)}
`
)

// tokenBucketLimiter implements the Token Bucket algorithm.
// Tokens are added to the bucket at a constant rate up to a maximum capacity.
type tokenBucketLimiter struct {
	client *redis.Client
	config *Config
}

// NewTokenBucket creates a new Token Bucket rate limiter.
func NewTokenBucket(client *redis.Client, config *Config) (RateLimiter, error) {
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

	return &tokenBucketLimiter{
		client: client,
		config: cfg,
	}, nil
}

// Allow checks if a single request is allowed for the given key.
func (t *tokenBucketLimiter) Allow(ctx context.Context, key string) (*Result, error) {
	return t.AllowN(ctx, key, 1)
}

// AllowN checks if N requests are allowed for the given key.
// Uses token bucket algorithm with continuous refilling.
func (t *tokenBucketLimiter) AllowN(ctx context.Context, key string, n int64) (*Result, error) {
	if n <= 0 {
		return nil, ErrInvalidN
	}

	redisKey := t.config.FormatKey(key)
	refillRate := t.calculateRefillRate()
	now := float64(time.Now().UnixNano()) / 1e9 // Convert to seconds with fractional part

	allowed, remaining, err := t.tryConsume(ctx, redisKey, n, refillRate, now)
	if err != nil {
		if t.config.FailOpen {
			// Fail open: allow the request
			return &Result{
				Allowed:    true,
				Limit:      t.config.Limit,
				Remaining:  0,
				RetryAfter: 0,
				ResetAt:    t.calculateResetTime(now),
			}, nil
		}
		return nil, fmt.Errorf("failed to check rate limit: %w", err)
	}

	result := &Result{
		Allowed:    allowed,
		Limit:      t.config.Limit,
		Remaining:  remaining,
		RetryAfter: 0,
		ResetAt:    t.calculateResetTime(now),
	}

	if !allowed {
		// Calculate time until enough tokens are available
		tokensNeeded := float64(n - remaining)
		secondsToWait := tokensNeeded / refillRate
		result.RetryAfter = time.Duration(secondsToWait * float64(time.Second))
		if result.RetryAfter < 0 {
			result.RetryAfter = 0
		}
	}

	return result, nil
}

// Reset resets the rate limit counter for the given key.
func (t *tokenBucketLimiter) Reset(ctx context.Context, key string) error {
	redisKey := t.config.FormatKey(key)

	if err := t.client.Del(ctx, redisKey).Err(); err != nil {
		return fmt.Errorf("failed to reset rate limit: %w", err)
	}

	return nil
}

// Close closes the rate limiter and releases resources.
func (t *tokenBucketLimiter) Close() error {
	if t.client != nil {
		return t.client.Close()
	}
	return nil
}

// calculateRefillRate calculates tokens per second based on limit and window.
func (t *tokenBucketLimiter) calculateRefillRate() float64 {
	return float64(t.config.Limit) / t.config.Window.Seconds()
}

// calculateResetTime calculates when the bucket will be full again.
// This is approximate since token bucket refills continuously.
func (t *tokenBucketLimiter) calculateResetTime(now float64) time.Time {
	// Estimate: time to fill entire bucket from empty
	secondsToFull := float64(t.config.Limit) / t.calculateRefillRate()
	return time.Unix(int64(now), int64((now-float64(int64(now)))*1e9)).Add(time.Duration(secondsToFull * float64(time.Second)))
}

// tryConsume attempts to consume tokens from the bucket.
func (t *tokenBucketLimiter) tryConsume(ctx context.Context, key string, n int64, refillRate, now float64) (bool, int64, error) {
	capacity := t.config.Limit
	ttl := int64(t.config.Window.Seconds() * 2) // Keep state for 2 windows

	result, err := t.client.Eval(ctx, tokenBucketScript, []string{key}, capacity, n, refillRate, now, ttl).Result()
	if err != nil {
		return false, 0, err
	}

	resultSlice, ok := result.([]interface{})
	if !ok || len(resultSlice) != 2 {
		return false, 0, fmt.Errorf("unexpected result type from Redis: %T", result)
	}

	allowedInt, ok := resultSlice[0].(int64)
	if !ok {
		return false, 0, fmt.Errorf("unexpected allowed type: %T", resultSlice[0])
	}

	remaining, ok := resultSlice[1].(int64)
	if !ok {
		return false, 0, fmt.Errorf("unexpected remaining type: %T", resultSlice[1])
	}

	return allowedInt == 1, remaining, nil
}
