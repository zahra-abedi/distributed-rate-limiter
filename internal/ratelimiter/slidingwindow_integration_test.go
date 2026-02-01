package ratelimiter

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupMiniredisSlidingWindow creates a miniredis instance and returns a Redis client
func setupMiniredisSlidingWindow(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return client, mr
}

func TestSlidingWindow_Integration_Allow(t *testing.T) {
	client, mr := setupMiniredisSlidingWindow(t)
	defer mr.Close()

	config := &Config{
		Algorithm: SlidingWindow,
		Limit:     5,
		Window:    time.Minute,
	}

	limiter, err := NewSlidingWindow(client, config)
	require.NoError(t, err)
	defer limiter.Close()

	ctx := context.Background()
	key := "user:123"

	// First request should be allowed
	result, err := limiter.Allow(ctx, key)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(5), result.Limit)
	assert.Equal(t, int64(4), result.Remaining)
	assert.Equal(t, time.Duration(0), result.RetryAfter)

	// Make 4 more requests (total 5, at limit)
	for i := 0; i < 4; i++ {
		result, err = limiter.Allow(ctx, key)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
	}

	// 6th request should be denied
	result, err = limiter.Allow(ctx, key)
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, int64(0), result.Remaining)
	assert.Greater(t, result.RetryAfter, time.Duration(0))
}

func TestSlidingWindow_Integration_AllowN(t *testing.T) {
	client, mr := setupMiniredisSlidingWindow(t)
	defer mr.Close()

	config := &Config{
		Algorithm: SlidingWindow,
		Limit:     10,
		Window:    time.Minute,
	}

	limiter, err := NewSlidingWindow(client, config)
	require.NoError(t, err)
	defer limiter.Close()

	ctx := context.Background()
	key := "api:endpoint"

	// Request 3 tokens
	result, err := limiter.AllowN(ctx, key, 3)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(7), result.Remaining)

	// Request 5 more tokens (total 8, under limit)
	result, err = limiter.AllowN(ctx, key, 5)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(2), result.Remaining)

	// Request 3 more tokens (would be 11 total, exceeds limit)
	result, err = limiter.AllowN(ctx, key, 3)
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, int64(0), result.Remaining)
	assert.Greater(t, result.RetryAfter, time.Duration(0))
}

func TestSlidingWindow_Integration_AllowN_InvalidTokens(t *testing.T) {
	client, mr := setupMiniredisSlidingWindow(t)
	defer mr.Close()

	config := &Config{
		Algorithm: SlidingWindow,
		Limit:     10,
		Window:    time.Minute,
	}

	limiter, err := NewSlidingWindow(client, config)
	require.NoError(t, err)
	defer limiter.Close()

	ctx := context.Background()

	tests := []struct {
		name   string
		tokens int64
	}{
		{"zero tokens", 0},
		{"negative tokens", -1},
		{"large negative", -100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := limiter.AllowN(ctx, "test-key", tt.tokens)
			assert.ErrorIs(t, err, ErrInvalidN)
			assert.Nil(t, result)
		})
	}
}

func TestSlidingWindow_Integration_Reset(t *testing.T) {
	client, mr := setupMiniredisSlidingWindow(t)
	defer mr.Close()

	config := &Config{
		Algorithm: SlidingWindow,
		Limit:     5,
		Window:    time.Minute,
	}

	limiter, err := NewSlidingWindow(client, config)
	require.NoError(t, err)
	defer limiter.Close()

	ctx := context.Background()
	key := "user:456"

	// Use all tokens
	for i := 0; i < 5; i++ {
		result, err := limiter.Allow(ctx, key)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
	}

	// Verify limit is reached
	result, err := limiter.Allow(ctx, key)
	require.NoError(t, err)
	assert.False(t, result.Allowed)

	// Reset the limit
	err = limiter.Reset(ctx, key)
	require.NoError(t, err)

	// Should be allowed again
	result, err = limiter.Allow(ctx, key)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(4), result.Remaining)
}

func TestSlidingWindow_Integration_WindowBoundary(t *testing.T) {
	client, mr := setupMiniredisSlidingWindow(t)
	defer mr.Close()

	config := &Config{
		Algorithm: SlidingWindow,
		Limit:     3,
		Window:    2 * time.Second,
	}

	limiter, err := NewSlidingWindow(client, config)
	require.NoError(t, err)
	defer limiter.Close()

	ctx := context.Background()
	key := "user:boundary"

	// Use all tokens in first window
	for i := 0; i < 3; i++ {
		result, err := limiter.Allow(ctx, key)
		require.NoError(t, err)
		assert.True(t, result.Allowed, "request %d should be allowed", i+1)
	}

	// Next request should be denied
	result, err := limiter.Allow(ctx, key)
	require.NoError(t, err)
	assert.False(t, result.Allowed)

	// Fast-forward time by advancing miniredis
	mr.FastForward(3 * time.Second)

	// After window reset, should be allowed again
	result, err = limiter.Allow(ctx, key)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(2), result.Remaining)
}

func TestSlidingWindow_Integration_MultipleKeys(t *testing.T) {
	client, mr := setupMiniredisSlidingWindow(t)
	defer mr.Close()

	config := &Config{
		Algorithm: SlidingWindow,
		Limit:     2,
		Window:    time.Minute,
	}

	limiter, err := NewSlidingWindow(client, config)
	require.NoError(t, err)
	defer limiter.Close()

	ctx := context.Background()

	// Different keys should have independent limits
	key1 := "user:1"
	key2 := "user:2"

	// Use limit for key1
	result, err := limiter.AllowN(ctx, key1, 2)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(0), result.Remaining)

	// key2 should still have full limit
	result, err = limiter.Allow(ctx, key2)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(1), result.Remaining)

	// key1 should be at limit
	result, err = limiter.Allow(ctx, key1)
	require.NoError(t, err)
	assert.False(t, result.Allowed)
}

func TestSlidingWindow_Integration_FailOpen(t *testing.T) {
	client, mr := setupMiniredisSlidingWindow(t)
	defer mr.Close()

	config := &Config{
		Algorithm: SlidingWindow,
		Limit:     5,
		Window:    time.Minute,
		FailOpen:  true,
	}

	limiter, err := NewSlidingWindow(client, config)
	require.NoError(t, err)
	defer limiter.Close()

	ctx := context.Background()
	key := "user:failopen"

	// Close Redis to simulate failure
	mr.Close()

	// Should allow request when Redis is down (fail-open)
	result, err := limiter.Allow(ctx, key)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(0), result.Remaining) // No remaining info when failing open
}

func TestSlidingWindow_Integration_FailClosed(t *testing.T) {
	client, mr := setupMiniredisSlidingWindow(t)
	defer mr.Close()

	config := &Config{
		Algorithm: SlidingWindow,
		Limit:     5,
		Window:    time.Minute,
		FailOpen:  false,
	}

	limiter, err := NewSlidingWindow(client, config)
	require.NoError(t, err)
	defer limiter.Close()

	ctx := context.Background()
	key := "user:failclosed"

	// Close Redis to simulate failure
	mr.Close()

	// Should return error when Redis is down (fail-closed)
	result, err := limiter.Allow(ctx, key)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestSlidingWindow_Integration_ResetAt(t *testing.T) {
	client, mr := setupMiniredisSlidingWindow(t)
	defer mr.Close()

	config := &Config{
		Algorithm: SlidingWindow,
		Limit:     10,
		Window:    time.Minute,
	}

	limiter, err := NewSlidingWindow(client, config)
	require.NoError(t, err)
	defer limiter.Close()

	ctx := context.Background()
	key := "user:reset-time"

	now := time.Now()
	result, err := limiter.Allow(ctx, key)
	require.NoError(t, err)

	// ResetAt should be in the future
	assert.True(t, result.ResetAt.After(now))

	// ResetAt should be at the end of the current window
	windowStart := now.Truncate(config.Window)
	expectedReset := windowStart.Add(config.Window)
	assert.Equal(t, expectedReset, result.ResetAt)
}

func TestSlidingWindow_Integration_CustomPrefix(t *testing.T) {
	client, mr := setupMiniredisSlidingWindow(t)
	defer mr.Close()

	config := &Config{
		Algorithm: SlidingWindow,
		Limit:     5,
		Window:    time.Minute,
		Prefix:    "custom",
	}

	limiter, err := NewSlidingWindow(client, config)
	require.NoError(t, err)
	defer limiter.Close()

	ctx := context.Background()
	key := "test-key"

	result, err := limiter.Allow(ctx, key)
	require.NoError(t, err)
	assert.True(t, result.Allowed)

	// Verify the key format in Redis (should have current and previous window keys)
	keys := mr.Keys()
	assert.GreaterOrEqual(t, len(keys), 1)
	for _, k := range keys {
		assert.Contains(t, k, "custom:")
	}
}

func TestSlidingWindow_Integration_SmoothRateLimit(t *testing.T) {
	client, mr := setupMiniredisSlidingWindow(t)
	defer mr.Close()

	config := &Config{
		Algorithm: SlidingWindow,
		Limit:     10,
		Window:    10 * time.Second,
	}

	limiter, err := NewSlidingWindow(client, config)
	require.NoError(t, err)
	defer limiter.Close()

	ctx := context.Background()
	key := "user:smooth"

	// Use 8 requests in previous window
	// Fast-forward to simulate previous window
	mr.FastForward(-5 * time.Second) // Go back 5 seconds
	for i := 0; i < 8; i++ {
		limiter.Allow(ctx, key)
	}

	// Move forward to middle of next window
	mr.FastForward(10 * time.Second) // Move forward to new window

	// At 50% through new window with 0 current requests:
	// Weighted = 8 * (1 - 0.5) + 0 = 4
	// Should allow more than 4 requests
	result, err := limiter.Allow(ctx, key)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
}
