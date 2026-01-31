package ratelimiter

import (
    "context"
    "testing"
    "time"

    "github.com/redis/go-redis/v9"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestNewFixedWindow(t *testing.T) {
    client := redis.NewClient(&redis.Options{})

    tests := []struct {
        name        string
        client      *redis.Client
        config      *Config
        expectError bool
        errorMsg    string
    }{
        {
            name: "valid config",
            client: client,
            config: &Config{
                Algorithm: FixedWindow,
                Limit:     10,
                Window:    time.Minute,
            },
            expectError: false,
        },
        {
            name:        "nil client",
            client:      nil,
            config:      &Config{Algorithm: FixedWindow, Limit: 10, Window: time.Minute},
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
                Algorithm: FixedWindow,
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
                Algorithm: FixedWindow,
                Limit:     100,
                Window:    time.Minute,
                // Prefix not set, should use default
            },
            expectError: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            limiter, err := NewFixedWindow(tt.client, tt.config)

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

func TestFixedWindow_AllowN_InvalidTokens(t *testing.T) {
    client := redis.NewClient(&redis.Options{})
    config := &Config{
        Algorithm: FixedWindow,
        Limit:     10,
        Window:    time.Minute,
    }

    limiter, err := NewFixedWindow(client, config)
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

func TestFixedWindow_FormatKey(t *testing.T) {
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
                Algorithm: FixedWindow,
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
                Algorithm: FixedWindow,
                Limit:     10,
                Window:    time.Minute,
                Prefix:    "custom",
            },
            key:         "api:endpoint",
            windowStart: 1640000060,
            expected:    "custom:api:endpoint:1640000060",
        },
        {
            name: "with empty prefix (gets default)",
            config: &Config{
                Algorithm: FixedWindow,
                Limit:     10,
                Window:    time.Minute,
                Prefix:    "",
            },
            key:         "test",
            windowStart: 1640000120,
            expected:    "ratelimit:test:1640000120", // WithDefaults() applies default prefix
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            limiter, err := NewFixedWindow(client, tt.config)
            require.NoError(t, err)
            defer limiter.Close()

            fw := limiter.(*fixedWindowLimiter)
            result := fw.formatKey(tt.key, tt.windowStart)
            assert.Equal(t, tt.expected, result)
        })
    }
}

func TestFixedWindow_CalculateResetTime(t *testing.T) {
    client := redis.NewClient(&redis.Options{})
    config := &Config{
        Algorithm: FixedWindow,
        Limit:     10,
        Window:    time.Minute,
    }

    limiter, err := NewFixedWindow(client, config)
    require.NoError(t, err)
    defer limiter.Close()

    fw := limiter.(*fixedWindowLimiter)

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
            fw.config.Window = tt.window
            result := fw.calculateResetTime(tt.windowStart)
            assert.Equal(t, tt.expected, result)
        })
    }
}

func TestFixedWindow_InterfaceContract(t *testing.T) {
    // Verify that fixedWindowLimiter implements RateLimiter interface
    var _ RateLimiter = (*fixedWindowLimiter)(nil)
}

func TestFixedWindow_Close(t *testing.T) {
    t.Run("close nil client", func(t *testing.T) {
        limiter := &fixedWindowLimiter{
            client: nil,
            config: &Config{},
        }
        err := limiter.Close()
        assert.NoError(t, err)
    })

    t.Run("close with client", func(t *testing.T) {
        client := redis.NewClient(&redis.Options{})
        config := &Config{
            Algorithm: FixedWindow,
            Limit:     10,
            Window:    time.Minute,
        }

        limiter, err := NewFixedWindow(client, config)
        require.NoError(t, err)

        err = limiter.Close()
        assert.NoError(t, err)
    })
}
