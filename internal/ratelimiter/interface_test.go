package ratelimiter

import (
	"context"
	"testing"
	"time"
)

// InterfaceTestSuite is a collection of tests that verify the RateLimiter interface contract
// Any implementation of RateLimiter should pass all these tests
type InterfaceTestSuite struct {
	// NewLimiter creates a new rate limiter instance for testing
	// The test suite will call this function to create limiters with different configs
	NewLimiter func(config *Config) (RateLimiter, error)

	// Cleanup is called after each test to clean up resources
	// Optional - can be nil if no cleanup is needed
	Cleanup func()
}

// RunAllTests runs all interface contract tests
func (suite *InterfaceTestSuite) RunAllTests(t *testing.T) {
	t.Run("Allow", suite.TestAllow)
	t.Run("AllowN", suite.TestAllowN)
	t.Run("Reset", suite.TestReset)
	t.Run("InvalidInput", suite.TestInvalidInput)
	t.Run("Concurrency", suite.TestConcurrency)
}

// TestAllow tests the basic Allow functionality
func (suite *InterfaceTestSuite) TestAllow(t *testing.T) {
	config := &Config{
		Algorithm: FixedWindow,
		Limit:     5,
		Window:    time.Minute,
		Prefix:    "test",
	}

	limiter, err := suite.NewLimiter(config)
	if err != nil {
		t.Fatalf("Failed to create limiter: %v", err)
	}
	defer limiter.Close()
	if suite.Cleanup != nil {
		defer suite.Cleanup()
	}

	ctx := context.Background()
	key := "test:user:123"

	// Test: First 5 requests should be allowed
	for i := 1; i <= 5; i++ {
		result, err := limiter.Allow(ctx, key)
		if err != nil {
			t.Fatalf("Request %d: unexpected error: %v", i, err)
		}

		if !result.Allowed {
			t.Errorf("Request %d: expected to be allowed", i)
		}

		if result.Limit != 5 {
			t.Errorf("Request %d: Limit = %d, want 5", i, result.Limit)
		}

		expectedRemaining := int64(5 - i)
		if result.Remaining != expectedRemaining {
			t.Errorf("Request %d: Remaining = %d, want %d", i, result.Remaining, expectedRemaining)
		}
	}

	// Test: 6th request should be denied
	result, err := limiter.Allow(ctx, key)
	if err != nil {
		t.Fatalf("Request 6: unexpected error: %v", err)
	}

	if result.Allowed {
		t.Error("Request 6: expected to be denied")
	}

	if result.Remaining != 0 {
		t.Errorf("Request 6: Remaining = %d, want 0", result.Remaining)
	}

	if result.RetryAfter <= 0 {
		t.Error("Request 6: RetryAfter should be > 0")
	}

	if result.ResetAt.IsZero() {
		t.Error("Request 6: ResetAt should not be zero")
	}
}

// TestAllowN tests the AllowN functionality
func (suite *InterfaceTestSuite) TestAllowN(t *testing.T) {
	config := &Config{
		Algorithm: FixedWindow,
		Limit:     10,
		Window:    time.Minute,
		Prefix:    "test",
	}

	limiter, err := suite.NewLimiter(config)
	if err != nil {
		t.Fatalf("Failed to create limiter: %v", err)
	}
	defer limiter.Close()
	if suite.Cleanup != nil {
		defer suite.Cleanup()
	}

	ctx := context.Background()
	key := "test:batch:user"

	// Test: Allow 3 requests at once
	result, err := limiter.AllowN(ctx, key, 3)
	if err != nil {
		t.Fatalf("AllowN(3): unexpected error: %v", err)
	}

	if !result.Allowed {
		t.Error("AllowN(3): expected to be allowed")
	}

	if result.Remaining != 7 {
		t.Errorf("AllowN(3): Remaining = %d, want 7", result.Remaining)
	}

	// Test: Allow 5 more requests
	result, err = limiter.AllowN(ctx, key, 5)
	if err != nil {
		t.Fatalf("AllowN(5): unexpected error: %v", err)
	}

	if !result.Allowed {
		t.Error("AllowN(5): expected to be allowed")
	}

	if result.Remaining != 2 {
		t.Errorf("AllowN(5): Remaining = %d, want 2", result.Remaining)
	}

	// Test: Try to allow 5 more (should be denied, only 2 remaining)
	result, err = limiter.AllowN(ctx, key, 5)
	if err != nil {
		t.Fatalf("AllowN(5) over limit: unexpected error: %v", err)
	}

	if result.Allowed {
		t.Error("AllowN(5) over limit: expected to be denied")
	}

	// Test: The denial should not consume any quota (atomic)
	// Should still have 2 remaining
	result, err = limiter.AllowN(ctx, key, 2)
	if err != nil {
		t.Fatalf("AllowN(2) after denial: unexpected error: %v", err)
	}

	if !result.Allowed {
		t.Error("AllowN(2) after denial: expected to be allowed (denial should not consume quota)")
	}

	if result.Remaining != 0 {
		t.Errorf("AllowN(2) after denial: Remaining = %d, want 0", result.Remaining)
	}
}

// TestReset tests the Reset functionality
func (suite *InterfaceTestSuite) TestReset(t *testing.T) {
	config := &Config{
		Algorithm: FixedWindow,
		Limit:     3,
		Window:    time.Minute,
		Prefix:    "test",
	}

	limiter, err := suite.NewLimiter(config)
	if err != nil {
		t.Fatalf("Failed to create limiter: %v", err)
	}
	defer limiter.Close()
	if suite.Cleanup != nil {
		defer suite.Cleanup()
	}

	ctx := context.Background()
	key := "test:reset:user"

	// Use up the quota
	for i := 0; i < 3; i++ {
		limiter.Allow(ctx, key)
	}

	// Verify we're at the limit
	result, err := limiter.Allow(ctx, key)
	if err != nil {
		t.Fatalf("Before reset: unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("Before reset: expected to be denied")
	}

	// Reset the limit
	err = limiter.Reset(ctx, key)
	if err != nil {
		t.Fatalf("Reset failed: %v", err)
	}

	// Should be able to make requests again
	result, err = limiter.Allow(ctx, key)
	if err != nil {
		t.Fatalf("After reset: unexpected error: %v", err)
	}

	if !result.Allowed {
		t.Error("After reset: expected to be allowed")
	}

	if result.Remaining != 2 {
		t.Errorf("After reset: Remaining = %d, want 2", result.Remaining)
	}
}

// TestInvalidInput tests error handling for invalid inputs
func (suite *InterfaceTestSuite) TestInvalidInput(t *testing.T) {
	config := &Config{
		Algorithm: FixedWindow,
		Limit:     10,
		Window:    time.Minute,
		Prefix:    "test",
	}

	limiter, err := suite.NewLimiter(config)
	if err != nil {
		t.Fatalf("Failed to create limiter: %v", err)
	}
	defer limiter.Close()
	if suite.Cleanup != nil {
		defer suite.Cleanup()
	}

	ctx := context.Background()

	t.Run("empty key", func(t *testing.T) {
		_, err := limiter.Allow(ctx, "")
		if err == nil {
			t.Error("Expected error for empty key")
		}
	})

	t.Run("AllowN with zero", func(t *testing.T) {
		_, err := limiter.AllowN(ctx, "user:123", 0)
		if err == nil {
			t.Error("Expected error for n=0")
		}
	})

	t.Run("AllowN with negative", func(t *testing.T) {
		_, err := limiter.AllowN(ctx, "user:123", -5)
		if err == nil {
			t.Error("Expected error for negative n")
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		cancelCtx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := limiter.Allow(cancelCtx, "user:123")
		if err == nil {
			t.Error("Expected error for cancelled context")
		}
	})
}

// TestConcurrency tests concurrent access to the rate limiter
func (suite *InterfaceTestSuite) TestConcurrency(t *testing.T) {
	config := &Config{
		Algorithm: FixedWindow,
		Limit:     100,
		Window:    time.Minute,
		Prefix:    "test",
	}

	limiter, err := suite.NewLimiter(config)
	if err != nil {
		t.Fatalf("Failed to create limiter: %v", err)
	}
	defer limiter.Close()
	if suite.Cleanup != nil {
		defer suite.Cleanup()
	}

	ctx := context.Background()
	key := "test:concurrent:user"

	// Run 100 concurrent requests
	concurrency := 100
	results := make(chan bool, concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			result, err := limiter.Allow(ctx, key)
			if err != nil {
				results <- false
				return
			}
			results <- result.Allowed
		}()
	}

	// Collect results
	allowedCount := 0
	deniedCount := 0

	for i := 0; i < concurrency; i++ {
		allowed := <-results
		if allowed {
			allowedCount++
		} else {
			deniedCount++
		}
	}

	// Should allow exactly 100 requests (the limit)
	// even with concurrent access
	if allowedCount != 100 {
		t.Errorf("Concurrent requests: allowed %d, want exactly 100", allowedCount)
	}

	if deniedCount != 0 {
		t.Errorf("Concurrent requests: denied %d, want 0", deniedCount)
	}
}

// TestMultipleKeys tests that different keys are tracked independently
func (suite *InterfaceTestSuite) TestMultipleKeys(t *testing.T) {
	config := &Config{
		Algorithm: FixedWindow,
		Limit:     3,
		Window:    time.Minute,
		Prefix:    "test",
	}

	limiter, err := suite.NewLimiter(config)
	if err != nil {
		t.Fatalf("Failed to create limiter: %v", err)
	}
	defer limiter.Close()
	if suite.Cleanup != nil {
		defer suite.Cleanup()
	}

	ctx := context.Background()
	key1 := "test:user:alice"
	key2 := "test:user:bob"

	// Use up quota for key1
	for i := 0; i < 3; i++ {
		limiter.Allow(ctx, key1)
	}

	// key1 should be at limit
	result, _ := limiter.Allow(ctx, key1)
	if result.Allowed {
		t.Error("key1: expected to be denied")
	}

	// key2 should still have full quota
	result, _ = limiter.Allow(ctx, key2)
	if !result.Allowed {
		t.Error("key2: expected to be allowed")
	}
	if result.Remaining != 2 {
		t.Errorf("key2: Remaining = %d, want 2", result.Remaining)
	}
}
