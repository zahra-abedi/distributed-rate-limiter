package ratelimiter

import (
	"context"
	"time"
)

// Algorithm identifies the rate limiting algorithm to use
type Algorithm string

const (
	// TokenBucket provides smooth rate limiting with burst tolerance
	// Best for: APIs, variable traffic patterns, user-facing features
	TokenBucket Algorithm = "token_bucket"

	// SlidingWindow provides precise rate limiting with weighted windows
	// Best for: Billing, SLA enforcement, preventing boundary gaming
	SlidingWindow Algorithm = "sliding_window"

	// FixedWindow provides simple counter-based rate limiting
	// Best for: Internal services, soft quotas, high-throughput systems
	FixedWindow Algorithm = "fixed_window"
)

// Result contains the outcome of a rate limit check
type Result struct {
	// Allowed indicates whether the request should be allowed
	Allowed bool

	// Limit is the maximum number of requests allowed in the window
	Limit int64

	// Remaining is the number of requests remaining in the current window
	// This value is 0 when Allowed is false
	Remaining int64

	// RetryAfter indicates how long to wait before retrying if denied
	// This value is 0 when Allowed is true
	RetryAfter time.Duration

	// ResetAt indicates when the rate limit window resets
	ResetAt time.Time
}

// Config holds configuration for a rate limiter instance
type Config struct {
	// Algorithm specifies which rate limiting algorithm to use
	// Required: must be one of TokenBucket, SlidingWindow, or FixedWindow
	Algorithm Algorithm

	// Limit is the maximum number of requests allowed within the window
	// Required: must be > 0
	Limit int64

	// Window is the time duration for the rate limit
	// Required: must be > 0
	// Examples: time.Second, time.Minute, time.Hour
	Window time.Duration

	// Prefix is prepended to all Redis keys
	// Optional: defaults to "ratelimit" if not specified
	// Set to empty string "" to disable automatic prefixing
	Prefix string

	// FailOpen determines behavior when Redis is unavailable
	// true:  Allow requests when Redis is down (fail-open, prioritizes availability)
	// false: Deny requests when Redis is down (fail-closed, prioritizes security)
	// Default: false (fail-closed)
	FailOpen bool
}

// RateLimiter is the core interface that all rate limiting algorithms implement
//
// Implementations must be safe for concurrent use by multiple goroutines.
// All methods should respect context cancellation and deadlines.
type RateLimiter interface {
	// Allow checks if a single request should be allowed for the given key
	//
	// The key is typically a user identifier (user ID, API key, IP address, etc.)
	// and is used to track rate limits independently per entity.
	//
	// Returns:
	//   - Result: Details about the rate limit decision
	//   - error: Non-nil if Redis is unavailable or other system errors occur
	//
	// When error is non-nil, the Result.Allowed field indicates the fail-open/fail-closed
	// behavior based on the Config.FailOpen setting.
	//
	// Example:
	//   result, err := limiter.Allow(ctx, "user:12345")
	//   if err != nil {
	//       log.Error("rate limiter error", err)
	//   }
	//   if !result.Allowed {
	//       return fmt.Errorf("rate limit exceeded, retry after %v", result.RetryAfter)
	//   }
	Allow(ctx context.Context, key string) (*Result, error)

	// AllowN checks if N requests should be allowed for the given key
	//
	// This is useful for batch operations where you want to check quota for
	// multiple items at once (e.g., bulk uploads, batch API requests).
	//
	// The behavior is atomic: either all N requests are allowed, or none are.
	// If the quota is insufficient, no requests are consumed.
	//
	// Parameters:
	//   - n: Number of requests to check (must be > 0)
	//
	// Returns same as Allow()
	//
	// Example:
	//   result, err := limiter.AllowN(ctx, "user:12345", 50)
	//   if !result.Allowed {
	//       return fmt.Errorf("insufficient quota: need %d, have %d", 50, result.Remaining)
	//   }
	AllowN(ctx context.Context, key string, n int64) (*Result, error)

	// Reset clears the rate limit state for the given key
	//
	// This is useful for:
	//   - Resetting limits after successful authentication (failed login attempts)
	//   - Admin operations (manual quota reset)
	//   - Testing (clean slate between tests)
	//
	// Returns:
	//   - error: Non-nil if Redis is unavailable
	//
	// Example:
	//   // Reset failed login attempts after successful login
	//   if loginSuccessful {
	//       limiter.Reset(ctx, fmt.Sprintf("login:%s", ipAddr))
	//   }
	Reset(ctx context.Context, key string) error

	// Close releases any resources held by the rate limiter
	//
	// After calling Close, the rate limiter should not be used.
	// This method should be called when shutting down to clean up
	// Redis connections and other resources.
	//
	// Example:
	//   defer limiter.Close()
	Close() error
}
