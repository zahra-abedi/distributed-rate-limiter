# Rate Limiter - Code Examples

## Example 1: Basic API Rate Limiting

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"

    "github.com/zahraabedi/distributed-rate-limiter/pkg/client"
    "github.com/zahraabedi/distributed-rate-limiter/internal/ratelimiter"
)

func main() {
    // Create rate limiter: 100 requests per hour
    limiter, err := client.New(&client.Config{
        RedisAddr: "localhost:6379",
        Algorithm: ratelimiter.TokenBucket,
        Limit:     100,
        Window:    time.Hour,
        Prefix:    "api",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer limiter.Close()

    // HTTP handler
    http.HandleFunc("/api/weather", func(w http.ResponseWriter, r *http.Request) {
        userID := r.Header.Get("X-User-ID")

        // Check rate limit
        result, err := limiter.Allow(r.Context(), userID)
        if err != nil {
            http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
            return
        }

        if !result.Allowed {
            w.Header().Set("X-RateLimit-Limit", fmt.Sprint(result.Limit))
            w.Header().Set("X-RateLimit-Remaining", "0")
            w.Header().Set("X-RateLimit-Reset", fmt.Sprint(result.ResetAt.Unix()))
            w.Header().Set("Retry-After", fmt.Sprint(int(result.RetryAfter.Seconds())))

            http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
            return
        }

        // Add rate limit headers
        w.Header().Set("X-RateLimit-Limit", fmt.Sprint(result.Limit))
        w.Header().Set("X-RateLimit-Remaining", fmt.Sprint(result.Remaining))
        w.Header().Set("X-RateLimit-Reset", fmt.Sprint(result.ResetAt.Unix()))

        // Process request
        w.Write([]byte(`{"temperature": 72, "condition": "sunny"}`))
    })

    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### What happens when user exceeds limit?

```bash
# Request 100
$ curl -H "X-User-ID: user123" http://localhost:8080/api/weather
{"temperature": 72, "condition": "sunny"}
# Headers:
# X-RateLimit-Limit: 100
# X-RateLimit-Remaining: 0
# X-RateLimit-Reset: 1704452400

# Request 101 (exceeded!)
$ curl -i -H "X-User-ID: user123" http://localhost:8080/api/weather
HTTP/1.1 429 Too Many Requests
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1704452400
Retry-After: 3420

Rate limit exceeded
```

## Example 2: Login Attempt Throttling

```go
package auth

import (
    "context"
    "time"

    "github.com/zahraabedi/distributed-rate-limiter/internal/ratelimiter"
)

type AuthService struct {
    loginLimiter ratelimiter.RateLimiter
}

func NewAuthService(redis *redis.Client) *AuthService {
    // Strict limit: 5 attempts per 15 minutes
    // Fail-closed: block all logins if Redis is down (security first)
    limiter := ratelimiter.NewFixedWindow(redis, &ratelimiter.Config{
        Limit:    5,
        Window:   15 * time.Minute,
        FailOpen: false,  // Security-critical: deny if Redis unavailable
        Prefix:   "login",
    })

    return &AuthService{
        loginLimiter: limiter,
    }
}

func (s *AuthService) Login(ctx context.Context, username, password, ipAddr string) error {
    // Rate limit by IP address to prevent brute force
    key := fmt.Sprintf("ip:%s", ipAddr)

    result, err := s.loginLimiter.Allow(ctx, key)
    if err != nil {
        // Redis is down - fail closed for security
        return errors.New("authentication service temporarily unavailable")
    }

    if !result.Allowed {
        return fmt.Errorf("too many login attempts, try again in %v", result.RetryAfter)
    }

    // Verify password
    if !s.verifyPassword(username, password) {
        return errors.New("invalid credentials")
    }

    // Success! Reset the rate limit for this IP
    s.loginLimiter.Reset(ctx, key)

    return nil
}
```

### Attack scenario:

```
Attacker from 203.0.113.50 tries to brute force:

10:00:00 - Login attempt (wrong password) → ALLOW (1/5)
10:00:01 - Login attempt (wrong password) → ALLOW (2/5)
10:00:02 - Login attempt (wrong password) → ALLOW (3/5)
10:00:03 - Login attempt (wrong password) → ALLOW (4/5)
10:00:04 - Login attempt (wrong password) → ALLOW (5/5)
10:00:05 - Login attempt (wrong password) → DENY "try again in 14m55s"
10:00:06 - Login attempt (wrong password) → DENY "try again in 14m54s"
...
10:15:00 - Window resets, attacker can try again

Result: Attacker can only try 5 passwords every 15 minutes
        = 480 attempts per day (vs unlimited)
```

## Example 3: Tiered API Quotas

```go
package api

import (
    "context"
    "time"

    "github.com/zahraabedi/distributed-rate-limiter/internal/ratelimiter"
)

type Tier string

const (
    Free    Tier = "free"
    Starter Tier = "starter"
    Pro     Tier = "pro"
)

type QuotaManager struct {
    limiters map[Tier]ratelimiter.RateLimiter
}

func NewQuotaManager(redis *redis.Client) *QuotaManager {
    return &QuotaManager{
        limiters: map[Tier]ratelimiter.RateLimiter{
            Free: ratelimiter.NewTokenBucket(redis, &ratelimiter.Config{
                Limit:  1000,
                Window: 24 * time.Hour,
                Prefix: "quota:free",
            }),
            Starter: ratelimiter.NewTokenBucket(redis, &ratelimiter.Config{
                Limit:  10000,
                Window: 24 * time.Hour,
                Prefix: "quota:starter",
            }),
            Pro: ratelimiter.NewTokenBucket(redis, &ratelimiter.Config{
                Limit:  100000,
                Window: 24 * time.Hour,
                Prefix: "quota:pro",
            }),
        },
    }
}

func (qm *QuotaManager) CheckQuota(ctx context.Context, userID string, tier Tier) (*ratelimiter.Result, error) {
    limiter := qm.limiters[tier]
    return limiter.Allow(ctx, userID)
}

// Example usage in HTTP middleware
func (qm *QuotaManager) QuotaMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        user := getUserFromContext(r.Context())

        result, err := qm.CheckQuota(r.Context(), user.ID, user.Tier)
        if err != nil {
            http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
            return
        }

        if !result.Allowed {
            // Add upgrade CTA for free users
            if user.Tier == Free {
                w.Header().Set("X-Upgrade-URL", "https://example.com/upgrade")
            }

            w.Header().Set("X-RateLimit-Limit", fmt.Sprint(result.Limit))
            w.Header().Set("X-RateLimit-Remaining", "0")
            http.Error(w, "Quota exceeded", http.StatusTooManyRequests)
            return
        }

        // Add quota info to response
        w.Header().Set("X-RateLimit-Limit", fmt.Sprint(result.Limit))
        w.Header().Set("X-RateLimit-Remaining", fmt.Sprint(result.Remaining))

        next.ServeHTTP(w, r)
    })
}
```

### User experience:

```bash
# Free user (1000/day quota)
$ curl -H "X-User-ID: free_user_123" http://api.example.com/data
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 847

# After 1000 requests
$ curl -i -H "X-User-ID: free_user_123" http://api.example.com/data
HTTP/1.1 429 Too Many Requests
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 0
X-Upgrade-URL: https://example.com/upgrade

Quota exceeded

# Pro user (100,000/day quota)
$ curl -H "X-User-ID: pro_user_456" http://api.example.com/data
X-RateLimit-Limit: 100000
X-RateLimit-Remaining: 99847
```

## Example 4: Batch Operations

```go
package batch

import (
    "context"
    "github.com/zahraabedi/distributed-rate-limiter/internal/ratelimiter"
)

type BatchProcessor struct {
    limiter ratelimiter.RateLimiter
}

func (bp *BatchProcessor) ProcessBatch(ctx context.Context, items []Item, userID string) error {
    // Check if user has quota for all items at once
    result, err := bp.limiter.AllowN(ctx, userID, int64(len(items)))
    if err != nil {
        return err
    }

    if !result.Allowed {
        return fmt.Errorf(
            "batch size %d exceeds quota (remaining: %d)",
            len(items),
            result.Remaining,
        )
    }

    // Process all items
    for _, item := range items {
        if err := bp.processItem(item); err != nil {
            return err
        }
    }

    return nil
}
```

### Batch scenario:

```
User wants to upload 50 files
Quota: 100 files/hour
Current usage: 60 files

Check: AllowN(ctx, "user:123", 50)
  Calculation: 60 + 50 = 110 > 100 limit
  Result: DENIED (remaining: 40)

Response: "You can only upload 40 more files this hour"

User uploads 40 files instead → SUCCESS
```

## Example 5: Different Algorithms for Different Use Cases

```go
package main

import (
    "github.com/zahraabedi/distributed-rate-limiter/internal/ratelimiter"
)

func setupRateLimiters(redis *redis.Client) {
    // 1. Token Bucket: API with burst tolerance
    //    Allows occasional bursts but maintains average rate
    apiLimiter := ratelimiter.NewTokenBucket(redis, &ratelimiter.Config{
        Limit:  100,           // 100 requests
        Window: time.Minute,   // per minute (avg rate: 100/min)
        // But can burst up to 100 at once if bucket is full
    })

    // 2. Sliding Window: Strict quota enforcement
    //    Prevents boundary gaming, more precise
    quotaLimiter := ratelimiter.NewSlidingWindow(redis, &ratelimiter.Config{
        Limit:  10000,
        Window: 24 * time.Hour,
    })

    // 3. Fixed Window: Simple internal rate limiting
    //    Good enough for internal services
    internalLimiter := ratelimiter.NewFixedWindow(redis, &ratelimiter.Config{
        Limit:  1000,
        Window: time.Minute,
    })
}
```

### Algorithm comparison in action:

```
Scenario: Limit is 10 requests/minute

Fixed Window (10:00:00 - 10:01:00):
  10:00:59 → 10 requests (ALLOWED, 10/10 used)
  11:01:00 → Counter resets
  11:01:00 → 10 requests (ALLOWED, 10/10 used)
  Total: 20 requests in 1 second! ⚠️

Sliding Window (better):
  10:00:59 → 10 requests (ALLOWED)
  10:01:00 → Try 10 more
    Calculation: (10 from [10:00:00-10:01:00] * 0.98) + (10 from [10:01:00-10:02:00] * 1.0)
                = 9.8 + 10 = 19.8 > 10 limit
    Result: DENIED (most requests blocked)

Token Bucket (smooth):
  10:00:00 → Bucket has 10 tokens
  10:00:00 → 10 rapid requests (uses all tokens, ALLOWED)
  10:00:00 → Request 11 (DENIED, no tokens)
  10:00:01 → +1 token refilled
  10:00:01 → Request 12 (ALLOWED, uses the 1 token)
  Smooth refill prevents bursts beyond initial capacity
```

## Example 6: Testing Without Redis

```go
package ratelimiter_test

import (
    "testing"
    "time"

    "github.com/zahraabedi/distributed-rate-limiter/internal/ratelimiter"
    "github.com/zahraabedi/distributed-rate-limiter/internal/ratelimiter/mocks"
)

func TestRateLimiter_AllowsUpToLimit(t *testing.T) {
    // Use in-memory mock storage for testing
    storage := mocks.NewMemoryStorage()
    limiter := ratelimiter.NewFixedWindow(storage, &ratelimiter.Config{
        Limit:  5,
        Window: time.Minute,
    })

    ctx := context.Background()

    // Should allow first 5 requests
    for i := 0; i < 5; i++ {
        result, err := limiter.Allow(ctx, "test-key")
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if !result.Allowed {
            t.Fatalf("request %d should be allowed", i+1)
        }
    }

    // 6th request should be denied
    result, err := limiter.Allow(ctx, "test-key")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if result.Allowed {
        t.Fatal("request 6 should be denied")
    }
}
```

## Key Takeaways

1. **Same interface, different use cases**: One rate limiter design serves many purposes
2. **Choose algorithm based on needs**: Token bucket for APIs, fixed window for simple cases, sliding window for strict enforcement
3. **Key design matters**: Use meaningful keys (user ID, IP, API key) to identify who to rate limit
4. **Headers are important**: Always return rate limit info to clients
5. **Security vs availability**: Fail-closed for auth, fail-open for API
6. **Testable**: Can test without Redis using mocks
