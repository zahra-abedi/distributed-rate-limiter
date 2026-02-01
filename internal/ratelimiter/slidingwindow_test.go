package ratelimiter

import (
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSlidingWindow(t *testing.T) {
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
				Algorithm: SlidingWindow,
				Limit:     10,
				Window:    time.Minute,
			},
			expectError: false,
		},
		{
			name:        "nil client",
			client:      nil,
			config:      &Config{Algorithm: SlidingWindow, Limit: 10, Window: time.Minute},
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
				Algorithm: SlidingWindow,
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
				Algorithm: SlidingWindow,
				Limit:     100,
				Window:    time.Minute,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter, err := NewSlidingWindow(tt.client, tt.config)

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

func TestSlidingWindow_FormatKey(t *testing.T) {
	client := redis.NewClient(&redis.Options{})

	tests := []struct {
		name        string
		config      *Config
		key         string
		windowStart int64
		expected    string
	}{
		{
			name: "with default prefix",
			config: &Config{
				Algorithm: SlidingWindow,
				Limit:     10,
				Window:    time.Minute,
			},
			key:         "user:123",
			windowStart: 1640000000,
			expected:    "ratelimit:user:123:1640000000",
		},
		{
			name: "with custom prefix",
			config: &Config{
				Algorithm: SlidingWindow,
				Limit:     10,
				Window:    time.Minute,
				Prefix:    "custom",
			},
			key:         "api:endpoint",
			windowStart: 1640000060,
			expected:    "custom:api:endpoint:1640000060",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter, err := NewSlidingWindow(client, tt.config)
			require.NoError(t, err)
			defer limiter.Close()

			sw := limiter.(*slidingWindowLimiter)
			result := sw.formatKey(tt.key, tt.windowStart)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSlidingWindow_CalculateResetTime(t *testing.T) {
	client := redis.NewClient(&redis.Options{})
	config := &Config{
		Algorithm: SlidingWindow,
		Limit:     10,
		Window:    time.Minute,
	}

	limiter, err := NewSlidingWindow(client, config)
	require.NoError(t, err)
	defer limiter.Close()

	sw := limiter.(*slidingWindowLimiter)

	tests := []struct {
		name        string
		windowStart int64
		window      time.Duration
		expected    time.Time
	}{
		{
			name:        "1 minute window",
			windowStart: 1640000000,
			window:      time.Minute,
			expected:    time.Unix(1640000060, 0),
		},
		{
			name:        "1 hour window",
			windowStart: 1640000000,
			window:      time.Hour,
			expected:    time.Unix(1640003600, 0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sw.config.Window = tt.window
			result := sw.calculateResetTime(tt.windowStart)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSlidingWindow_CalculateWeightedCount(t *testing.T) {
	client := redis.NewClient(&redis.Options{})
	config := &Config{
		Algorithm: SlidingWindow,
		Limit:     100,
		Window:    time.Minute,
	}

	limiter, err := NewSlidingWindow(client, config)
	require.NoError(t, err)
	defer limiter.Close()

	sw := limiter.(*slidingWindowLimiter)

	tests := []struct {
		name        string
		now         time.Time
		windowStart int64
		prevCount   int64
		currCount   int64
		expected    float64
	}{
		{
			name:        "at start of window (0% progress)",
			now:         time.Unix(1640000000, 0),
			windowStart: 1640000000,
			prevCount:   50,
			currCount:   10,
			expected:    60.0, // 50 * 1.0 + 10 * 0.0 = 50... wait, + currCount = 60
		},
		{
			name:        "halfway through window (50% progress)",
			now:         time.Unix(1640000030, 0),
			windowStart: 1640000000,
			prevCount:   50,
			currCount:   10,
			expected:    35.0, // 50 * 0.5 + 10 = 35
		},
		{
			name:        "at end of window (100% progress)",
			now:         time.Unix(1640000060, 0),
			windowStart: 1640000000,
			prevCount:   50,
			currCount:   10,
			expected:    10.0, // 50 * 0.0 + 10 = 10
		},
		{
			name:        "25% through window",
			now:         time.Unix(1640000015, 0),
			windowStart: 1640000000,
			prevCount:   40,
			currCount:   20,
			expected:    50.0, // 40 * 0.75 + 20 = 50
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sw.calculateWeightedCount(tt.now, tt.windowStart, tt.prevCount, tt.currCount)
			assert.InDelta(t, tt.expected, result, 0.1)
		})
	}
}

func TestSlidingWindow_InterfaceContract(t *testing.T) {
	// Verify that slidingWindowLimiter implements RateLimiter interface
	var _ RateLimiter = (*slidingWindowLimiter)(nil)
}

func TestSlidingWindow_Close(t *testing.T) {
	t.Run("close nil client", func(t *testing.T) {
		limiter := &slidingWindowLimiter{
			client: nil,
			config: &Config{},
		}
		err := limiter.Close()
		assert.NoError(t, err)
	})

	t.Run("close with client", func(t *testing.T) {
		client := redis.NewClient(&redis.Options{})
		config := &Config{
			Algorithm: SlidingWindow,
			Limit:     10,
			Window:    time.Minute,
		}

		limiter, err := NewSlidingWindow(client, config)
		require.NoError(t, err)

		err = limiter.Close()
		assert.NoError(t, err)
	})
}
