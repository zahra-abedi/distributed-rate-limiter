package ratelimiter

import "time"

// NewAllowedResult creates a Result for an allowed request
func NewAllowedResult(limit, remaining int64, resetAt time.Time) *Result {
    return &Result{
        Allowed:    true,
        Limit:      limit,
        Remaining:  remaining,
        RetryAfter: 0,
        ResetAt:    resetAt,
    }
}

// NewDeniedResult creates a Result for a denied request
func NewDeniedResult(limit int64, retryAfter time.Duration, resetAt time.Time) *Result {
    return &Result{
        Allowed:    false,
        Limit:      limit,
        Remaining:  0,
        RetryAfter: retryAfter,
        ResetAt:    resetAt,
    }
}

// NewFailOpenResult creates a Result for when Redis is down and FailOpen is true
// This allows the request through despite the error
func NewFailOpenResult() *Result {
    return &Result{
        Allowed:    true,
        Limit:      0,
        Remaining:  0,
        RetryAfter: 0,
        ResetAt:    time.Time{},
    }
}

// NewFailClosedResult creates a Result for when Redis is down and FailOpen is false
// This denies the request due to the error
func NewFailClosedResult() *Result {
    return &Result{
        Allowed:    false,
        Limit:      0,
        Remaining:  0,
        RetryAfter: 0,
        ResetAt:    time.Time{},
    }
}
