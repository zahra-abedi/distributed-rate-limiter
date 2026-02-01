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

// setupMiniredisTokenBucket creates a miniredis instance and returns a Redis client
func setupMiniredisTokenBucket(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return client, mr
}

func TestTokenBucket_Integration_Allow(t *testing.T) {
	client, mr := setupMiniredisTokenBucket(t)
	defer mr.Close()

	config := &Config{
		Algorithm: TokenBucket,
		Limit:     5,
		Window:    time.Minute,
	}

	limiter, err := NewTokenBucket(client, config)
	require.NoError(t, err)
	defer limiter.Close()

	ctx := context.Background()
	key := "user:123"

	// First request should be allowed (bucket starts full)
	result, err := limiter.Allow(ctx, key)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(5), result.Limit)
	assert.Equal(t, int64(4), result.Remaining)

	// Make 4 more requests (total 5, bucket empty)
	for i := 0; i < 4; i++ {
		result, err = limiter.Allow(ctx, key)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
	}

	// 6th request should be denied (bucket empty)
	result, err = limiter.Allow(ctx, key)
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, int64(0), result.Remaining)
	assert.Greater(t, result.RetryAfter, time.Duration(0))
}

func TestTokenBucket_Integration_AllowN(t *testing.T) {
	client, mr := setupMiniredisTokenBucket(t)
	defer mr.Close()

	config := &Config{
		Algorithm: TokenBucket,
		Limit:     10,
		Window:    time.Minute,
	}

	limiter, err := NewTokenBucket(client, config)
	require.NoError(t, err)
	defer limiter.Close()

	ctx := context.Background()
	key := "api:endpoint"

	// Request 3 tokens
	result, err := limiter.AllowN(ctx, key, 3)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(7), result.Remaining)

	// Request 5 more tokens (total 8 consumed, 2 remaining)
	result, err = limiter.AllowN(ctx, key, 5)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(2), result.Remaining)

	// Request 3 more tokens (only 2 available)
	result, err = limiter.AllowN(ctx, key, 3)
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, int64(2), result.Remaining)
}

func TestTokenBucket_Integration_AllowN_InvalidTokens(t *testing.T) {
	client, mr := setupMiniredisTokenBucket(t)
	defer mr.Close()

	config := &Config{
		Algorithm: TokenBucket,
		Limit:     10,
		Window:    time.Minute,
	}

	limiter, err := NewTokenBucket(client, config)
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

func TestTokenBucket_Integration_Refill(t *testing.T) {
	client, mr := setupMiniredisTokenBucket(t)
	defer mr.Close()

	// Configure: 10 tokens per second = very fast refill for testing
	config := &Config{
		Algorithm: TokenBucket,
		Limit:     10,
		Window:    time.Second,
	}

	limiter, err := NewTokenBucket(client, config)
	require.NoError(t, err)
	defer limiter.Close()

	ctx := context.Background()
	key := "user:refill"

	// Use all tokens
	result, err := limiter.AllowN(ctx, key, 10)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(0), result.Remaining)

	// Immediately try again - should be denied
	result, err = limiter.Allow(ctx, key)
	require.NoError(t, err)
	assert.False(t, result.Allowed)

	// Wait for actual time to pass (needed for refill calculation)
	time.Sleep(500 * time.Millisecond)

	// Should be able to consume ~5 tokens now (refilled at 10 tokens/sec for 0.5sec = ~5 tokens)
	result, err = limiter.AllowN(ctx, key, 4)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
}

func TestTokenBucket_Integration_Burst(t *testing.T) {
	client, mr := setupMiniredisTokenBucket(t)
	defer mr.Close()

	config := &Config{
		Algorithm: TokenBucket,
		Limit:     100,
		Window:    time.Minute,
	}

	limiter, err := NewTokenBucket(client, config)
	require.NoError(t, err)
	defer limiter.Close()

	ctx := context.Background()
	key := "user:burst"

	// Token bucket allows burst: consume all 100 tokens at once
	result, err := limiter.AllowN(ctx, key, 100)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(0), result.Remaining)

	// Next request denied
	result, err = limiter.Allow(ctx, key)
	require.NoError(t, err)
	assert.False(t, result.Allowed)
}

func TestTokenBucket_Integration_Reset(t *testing.T) {
	client, mr := setupMiniredisTokenBucket(t)
	defer mr.Close()

	config := &Config{
		Algorithm: TokenBucket,
		Limit:     5,
		Window:    time.Minute,
	}

	limiter, err := NewTokenBucket(client, config)
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

	// Verify bucket is empty
	result, err := limiter.Allow(ctx, key)
	require.NoError(t, err)
	assert.False(t, result.Allowed)

	// Reset the bucket
	err = limiter.Reset(ctx, key)
	require.NoError(t, err)

	// Should be allowed again (bucket refilled to capacity)
	result, err = limiter.Allow(ctx, key)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(4), result.Remaining)
}

func TestTokenBucket_Integration_MultipleKeys(t *testing.T) {
	client, mr := setupMiniredisTokenBucket(t)
	defer mr.Close()

	config := &Config{
		Algorithm: TokenBucket,
		Limit:     5,
		Window:    time.Minute,
	}

	limiter, err := NewTokenBucket(client, config)
	require.NoError(t, err)
	defer limiter.Close()

	ctx := context.Background()

	// Different keys should have independent buckets
	key1 := "user:1"
	key2 := "user:2"

	// Empty bucket for key1
	result, err := limiter.AllowN(ctx, key1, 5)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(0), result.Remaining)

	// key2 should still have full bucket
	result, err = limiter.Allow(ctx, key2)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(4), result.Remaining)

	// key1 should be empty
	result, err = limiter.Allow(ctx, key1)
	require.NoError(t, err)
	assert.False(t, result.Allowed)
}

func TestTokenBucket_Integration_FailOpen(t *testing.T) {
	client, mr := setupMiniredisTokenBucket(t)
	defer mr.Close()

	config := &Config{
		Algorithm: TokenBucket,
		Limit:     5,
		Window:    time.Minute,
		FailOpen:  true,
	}

	limiter, err := NewTokenBucket(client, config)
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
	assert.Equal(t, int64(0), result.Remaining)
}

func TestTokenBucket_Integration_FailClosed(t *testing.T) {
	client, mr := setupMiniredisTokenBucket(t)
	defer mr.Close()

	config := &Config{
		Algorithm: TokenBucket,
		Limit:     5,
		Window:    time.Minute,
		FailOpen:  false,
	}

	limiter, err := NewTokenBucket(client, config)
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

func TestTokenBucket_Integration_RetryAfter(t *testing.T) {
	client, mr := setupMiniredisTokenBucket(t)
	defer mr.Close()

	// Configure: 10 tokens per 10 seconds = 1 token/second
	config := &Config{
		Algorithm: TokenBucket,
		Limit:     10,
		Window:    10 * time.Second,
	}

	limiter, err := NewTokenBucket(client, config)
	require.NoError(t, err)
	defer limiter.Close()

	ctx := context.Background()
	key := "user:retry"

	// Use all tokens
	result, err := limiter.AllowN(ctx, key, 10)
	require.NoError(t, err)
	assert.True(t, result.Allowed)

	// Try to consume 5 more tokens (need to wait for refill)
	result, err = limiter.AllowN(ctx, key, 5)
	require.NoError(t, err)
	assert.False(t, result.Allowed)

	// RetryAfter should be ~5 seconds (5 tokens / 1 token per second)
	assert.Greater(t, result.RetryAfter, 4*time.Second)
	assert.Less(t, result.RetryAfter, 6*time.Second)
}

func TestTokenBucket_Integration_CustomPrefix(t *testing.T) {
	client, mr := setupMiniredisTokenBucket(t)
	defer mr.Close()

	config := &Config{
		Algorithm: TokenBucket,
		Limit:     5,
		Window:    time.Minute,
		Prefix:    "custom",
	}

	limiter, err := NewTokenBucket(client, config)
	require.NoError(t, err)
	defer limiter.Close()

	ctx := context.Background()
	key := "test-key"

	result, err := limiter.Allow(ctx, key)
	require.NoError(t, err)
	assert.True(t, result.Allowed)

	// Verify the key format in Redis
	keys := mr.Keys()
	require.Len(t, keys, 1)
	assert.Contains(t, keys[0], "custom:")
}

func TestTokenBucket_Integration_ContinuousRefill(t *testing.T) {
	client, mr := setupMiniredisTokenBucket(t)
	defer mr.Close()

	// Configure: 20 tokens per second = fast refill for testing
	config := &Config{
		Algorithm: TokenBucket,
		Limit:     20,
		Window:    time.Second,
	}

	limiter, err := NewTokenBucket(client, config)
	require.NoError(t, err)
	defer limiter.Close()

	ctx := context.Background()
	key := "user:continuous"

	// Use 10 tokens
	result, err := limiter.AllowN(ctx, key, 10)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, int64(10), result.Remaining)

	// Wait for actual time to pass (20 tokens/sec * 0.1sec = 2 tokens)
	time.Sleep(100 * time.Millisecond)

	result, err = limiter.Allow(ctx, key)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	// After consuming 1, should have ~11 remaining (10 + 2 refilled - 1 consumed)
	assert.GreaterOrEqual(t, result.Remaining, int64(10))
	assert.LessOrEqual(t, result.Remaining, int64(12))
}

func TestTokenBucket_Integration_MaxCapacity(t *testing.T) {
	client, mr := setupMiniredisTokenBucket(t)
	defer mr.Close()

	config := &Config{
		Algorithm: TokenBucket,
		Limit:     10,
		Window:    time.Second,
	}

	limiter, err := NewTokenBucket(client, config)
	require.NoError(t, err)
	defer limiter.Close()

	ctx := context.Background()
	key := "user:maxcap"

	// Start with full bucket
	result, err := limiter.Allow(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, int64(9), result.Remaining)

	// Wait long enough for bucket to refill beyond capacity
	time.Sleep(2 * time.Second)

	result, err = limiter.Allow(ctx, key)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	// Should be at capacity (10), after consuming 1 = 9 remaining
	assert.Equal(t, int64(9), result.Remaining)
}
