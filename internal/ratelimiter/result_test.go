package ratelimiter

import (
    "testing"
    "time"
)

func TestNewAllowedResult(t *testing.T) {
    limit := int64(100)
    remaining := int64(75)
    resetAt := time.Now().Add(time.Hour)

    result := NewAllowedResult(limit, remaining, resetAt)

    if !result.Allowed {
        t.Error("Expected Allowed to be true")
    }
    if result.Limit != limit {
        t.Errorf("Limit = %d, want %d", result.Limit, limit)
    }
    if result.Remaining != remaining {
        t.Errorf("Remaining = %d, want %d", result.Remaining, remaining)
    }
    if result.RetryAfter != 0 {
        t.Errorf("RetryAfter = %v, want 0", result.RetryAfter)
    }
    if !result.ResetAt.Equal(resetAt) {
        t.Errorf("ResetAt = %v, want %v", result.ResetAt, resetAt)
    }
}

func TestNewDeniedResult(t *testing.T) {
    limit := int64(100)
    retryAfter := 30 * time.Second
    resetAt := time.Now().Add(time.Minute)

    result := NewDeniedResult(limit, retryAfter, resetAt)

    if result.Allowed {
        t.Error("Expected Allowed to be false")
    }
    if result.Limit != limit {
        t.Errorf("Limit = %d, want %d", result.Limit, limit)
    }
    if result.Remaining != 0 {
        t.Errorf("Remaining = %d, want 0", result.Remaining)
    }
    if result.RetryAfter != retryAfter {
        t.Errorf("RetryAfter = %v, want %v", result.RetryAfter, retryAfter)
    }
    if !result.ResetAt.Equal(resetAt) {
        t.Errorf("ResetAt = %v, want %v", result.ResetAt, resetAt)
    }
}

func TestNewFailOpenResult(t *testing.T) {
    result := NewFailOpenResult()

    if !result.Allowed {
        t.Error("Expected Allowed to be true for fail-open")
    }
    if result.Limit != 0 {
        t.Errorf("Limit = %d, want 0", result.Limit)
    }
    if result.Remaining != 0 {
        t.Errorf("Remaining = %d, want 0", result.Remaining)
    }
    if result.RetryAfter != 0 {
        t.Errorf("RetryAfter = %v, want 0", result.RetryAfter)
    }
    if !result.ResetAt.IsZero() {
        t.Errorf("ResetAt = %v, want zero time", result.ResetAt)
    }
}

func TestNewFailClosedResult(t *testing.T) {
    result := NewFailClosedResult()

    if result.Allowed {
        t.Error("Expected Allowed to be false for fail-closed")
    }
    if result.Limit != 0 {
        t.Errorf("Limit = %d, want 0", result.Limit)
    }
    if result.Remaining != 0 {
        t.Errorf("Remaining = %d, want 0", result.Remaining)
    }
    if result.RetryAfter != 0 {
        t.Errorf("RetryAfter = %v, want 0", result.RetryAfter)
    }
    if !result.ResetAt.IsZero() {
        t.Errorf("ResetAt = %v, want zero time", result.ResetAt)
    }
}
