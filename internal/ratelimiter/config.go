// Package ratelimiter provides a distributed rate limiting interface and implementations.
package ratelimiter

import (
    "fmt"
    "time"
)

const (
    // DefaultPrefix is the default Redis key prefix
    DefaultPrefix = "ratelimit"
)

// Validate checks if the configuration is valid
// Returns an error describing what is invalid
func (c *Config) Validate() error {
    if c == nil {
        return fmt.Errorf("config cannot be nil")
    }

    // Validate algorithm
    switch c.Algorithm {
    case TokenBucket, SlidingWindow, FixedWindow:
        // Valid algorithm
    case "":
        return fmt.Errorf("algorithm is required")
    default:
        return fmt.Errorf("unknown algorithm: %s (must be one of: token_bucket, sliding_window, fixed_window)", c.Algorithm)
    }

    // Validate limit
    if c.Limit <= 0 {
        return fmt.Errorf("limit must be greater than 0, got: %d", c.Limit)
    }

    // Validate window
    if c.Window <= 0 {
        return fmt.Errorf("window must be greater than 0, got: %v", c.Window)
    }

    // Window should be reasonable (at least 1 millisecond, at most 365 days)
    if c.Window < time.Millisecond {
        return fmt.Errorf("window too small: %v (minimum: 1ms)", c.Window)
    }
    if c.Window > 365*24*time.Hour {
        return fmt.Errorf("window too large: %v (maximum: 365 days)", c.Window)
    }

    return nil
}

// WithDefaults returns a new Config with default values applied
// Does not modify the original config
func (c *Config) WithDefaults() *Config {
    if c == nil {
        return nil
    }

    result := *c // Copy

    // Apply default prefix if not set
    if result.Prefix == "" {
        result.Prefix = DefaultPrefix
    }

    return &result
}

// KeyPrefix returns the full prefix to use for Redis keys
// Handles the case where prefix is explicitly set to empty string
func (c *Config) KeyPrefix() string {
    if c == nil {
        return DefaultPrefix
    }
    // Note: We don't apply defaults here - empty string means "no prefix"
    return c.Prefix
}

// FormatKey formats a key with the configured prefix
// If prefix is empty, returns the key unchanged
func (c *Config) FormatKey(key string) string {
    prefix := c.KeyPrefix()
    if prefix == "" {
        return key
    }
    return prefix + ":" + key
}
