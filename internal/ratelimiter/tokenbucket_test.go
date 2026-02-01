package ratelimiter

import (
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTokenBucket(t *testing.T) {
	client := redis.NewClient(&redis.Options{})

	tests := []struct {
		name        string
		client      *redis.Client
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name:   "valid config",
			client: client,
			config: &Config{
				Algorithm: TokenBucket,
				Limit:     10,
				Window:    time.Minute,
			},
			expectError: false,
		},
		{
			name:        "nil client",
			client:      nil,
			config:      &Config{Algorithm: TokenBucket, Limit: 10, Window: time.Minute},
			expectError: true,
			errorMsg:    "redis client cannot be nil",
		},
		{
			name:        "nil config",
			client:      client,
			config:      nil,
			expectError: true,
			errorMsg:    "config cannot be nil",
		},
		{
			name:   "invalid config - zero limit",
			client: client,
			config: &Config{
				Algorithm: TokenBucket,
				Limit:     0,
				Window:    time.Minute,
			},
			expectError: true,
			errorMsg:    "invalid config",
		},
		{
			name:   "config with defaults applied",
			client: client,
			config: &Config{
				Algorithm: TokenBucket,
				Limit:     100,
				Window:    time.Minute,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter, err := NewTokenBucket(tt.client, tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, limiter)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, limiter)
			}
		})
	}
}

func TestTokenBucket_CalculateRefillRate(t *testing.T) {
	client := redis.NewClient(&redis.Options{})

	tests := []struct {
		name     string
		limit    int64
		window   time.Duration
		expected float64
	}{
		{
			name:     "10 per minute",
			limit:    10,
			window:   time.Minute,
			expected: 10.0 / 60.0, // ~0.1667 tokens/second
		},
		{
			name:     "100 per hour",
			limit:    100,
			window:   time.Hour,
			expected: 100.0 / 3600.0, // ~0.0278 tokens/second
		},
		{
			name:     "60 per second",
			limit:    60,
			window:   time.Second,
			expected: 60.0, // 60 tokens/second
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Algorithm: TokenBucket,
				Limit:     tt.limit,
				Window:    tt.window,
			}

			limiter, err := NewTokenBucket(client, config)
			require.NoError(t, err)
			defer limiter.Close()

			tb := limiter.(*tokenBucketLimiter)
			rate := tb.calculateRefillRate()
			assert.InDelta(t, tt.expected, rate, 0.0001)
		})
	}
}

func TestTokenBucket_InterfaceContract(t *testing.T) {
	// Verify that tokenBucketLimiter implements RateLimiter interface
	var _ RateLimiter = (*tokenBucketLimiter)(nil)
}

func TestTokenBucket_Close(t *testing.T) {
	t.Run("close nil client", func(t *testing.T) {
		limiter := &tokenBucketLimiter{
			client: nil,
			config: &Config{},
		}
		err := limiter.Close()
		assert.NoError(t, err)
	})

	t.Run("close with client", func(t *testing.T) {
		client := redis.NewClient(&redis.Options{})
		config := &Config{
			Algorithm: TokenBucket,
			Limit:     10,
			Window:    time.Minute,
		}

		limiter, err := NewTokenBucket(client, config)
		require.NoError(t, err)

		err = limiter.Close()
		assert.NoError(t, err)
	})
}
