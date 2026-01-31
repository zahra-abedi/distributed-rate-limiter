# ADR 002: Configurable Fail-Open vs Fail-Closed Behavior

**Date:** 2026-01-30

**Status:** Accepted

**Deciders:** Zahra Abedi

## Context

When the rate limiter's storage backend (Redis) becomes unavailable due to network issues, Redis crashes, or other failures, the system must decide how to handle incoming requests.

This decision has significant implications:
- **Security**: Failing open could allow malicious actors to bypass rate limits
- **Availability**: Failing closed could make the entire application unavailable
- **User Experience**: False positives (blocking legitimate users) vs false negatives (allowing abuse)

Different use cases have different priorities:
- **Login throttling** (brute-force prevention): Security is paramount
- **Public API** (user-facing): Availability is critical
- **Internal rate limiting**: Balance of both

There's no one-size-fits-all answer, so the decision must be configurable.

## Decision

Implement **configurable fail-open/fail-closed behavior** via the `Config.FailOpen` boolean field.

### Default Behavior

```go
type Config struct {
    // ...
    FailOpen bool  // default: false (fail-closed)
}
```

**Default is fail-closed** (conservative approach):
- When `FailOpen = false` (default): Deny requests when Redis is unavailable
- When `FailOpen = true` (explicit): Allow requests when Redis is unavailable

### Implementation

```go
result, err := limiter.Allow(ctx, key)
if err != nil {
    // Redis is unavailable
    if config.FailOpen {
        return NewFailOpenResult(), err  // Allow the request
    } else {
        return NewFailClosedResult(), err  // Deny the request
    }
}
```

The error is always returned so callers can log/monitor Redis issues, but the `Result.Allowed` field respects the fail-open/fail-closed setting.

## Consequences

### Positive

- ✅ **Flexibility**: Each use case can choose appropriate behavior
- ✅ **Security by default**: Conservative default protects against accidental fail-open
- ✅ **Explicit configuration**: Developers must consciously choose fail-open
- ✅ **Observable**: Errors are always returned for monitoring
- ✅ **Testable**: Can test both behaviors independently

### Negative

- ⚠️ **Configuration complexity**: Developers must understand the trade-off
- ⚠️ **Potential misconfiguration**: Wrong setting for use case could have serious consequences
- ⚠️ **Documentation burden**: Must clearly explain implications

### Neutral

- Adds one boolean field to configuration
- Requires documentation and examples for proper usage

## Alternatives Considered

### Alternative 1: Always Fail-Closed

**Description:** Always deny requests when Redis is down (no configuration option)

```go
// Always fail-closed
result, err := limiter.Allow(ctx, key)
if err != nil {
    return NewFailClosedResult(), err  // Always deny
}
```

**Pros:**
- Simplest implementation
- Most secure (no bypass possibility)
- No configuration needed
- Predictable behavior

**Cons:**
- Inflexible for availability-critical applications
- Redis outage takes down entire application
- Not suitable for all use cases
- Cascading failures (Redis → App → Users)

**Why not chosen:** Too inflexible. Public APIs need to remain available even if rate limiting fails temporarily.

### Alternative 2: Always Fail-Open

**Description:** Always allow requests when Redis is down (no configuration option)

```go
// Always fail-open
result, err := limiter.Allow(ctx, key)
if err != nil {
    log.Error("rate limiter unavailable, allowing request", err)
    return NewFailOpenResult(), err  // Always allow
}
```

**Pros:**
- Best availability
- User experience prioritized
- Application stays up during Redis outage

**Cons:**
- Security risk (rate limits can be bypassed)
- Could allow abuse during outages
- Dangerous default for security-critical use cases
- Might overload downstream systems

**Why not chosen:** Too dangerous for security-critical applications like login throttling.

### Alternative 3: Automatic Fallback to In-Memory

**Description:** When Redis fails, fall back to local in-memory rate limiting

```go
type RateLimiter struct {
    redis    *RedisStorage
    fallback *InMemoryStorage  // Local fallback
}

func (r *RateLimiter) Allow(ctx context.Context, key string) (*Result, error) {
    result, err := r.redis.Allow(ctx, key)
    if err != nil {
        // Fall back to in-memory (per-server limits)
        return r.fallback.Allow(ctx, key)
    }
    return result, nil
}
```

**Pros:**
- Best of both worlds (availability + some protection)
- Graceful degradation
- No complete bypass of rate limiting

**Cons:**
- Complex implementation
- Limits become per-server instead of distributed
- 3 servers = 3x the actual limit during outage
- Difficult to reason about behavior
- More memory usage
- State inconsistency when Redis comes back

**Why not chosen:** Too complex for initial version. Could be added as "Alternative 4" later with a `FallbackMode` config option.

### Alternative 4: Circuit Breaker with Retry

**Description:** Use circuit breaker pattern with exponential backoff

```go
type CircuitBreaker struct {
    state        State  // Closed, Open, Half-Open
    failureCount int
    lastFailure  time.Time
}

// Deny requests during cooldown, retry after threshold
```

**Pros:**
- Prevents cascading failures
- Automatic recovery when Redis comes back
- Industry standard pattern
- Reduces load on failing Redis

**Cons:**
- Still need to decide: allow or deny during open circuit?
- Complexity in implementation
- Tuning required (thresholds, timeouts)

**Why not chosen:** Circuit breaker is orthogonal to fail-open/fail-closed decision. We should implement both:
- Circuit breaker to detect failures and reduce Redis load
- Fail-open/fail-closed to decide what to do during failure

**Note:** Circuit breaker planned for future implementation (see ROADMAP.md).

## Use Case Examples

### Security-Critical: Fail-Closed

```go
// Login attempt throttling (brute-force prevention)
loginLimiter := ratelimiter.New(&ratelimiter.Config{
    Algorithm: ratelimiter.FixedWindow,
    Limit:     5,
    Window:    15 * time.Minute,
    FailOpen:  false,  // Fail-closed: security first
})

// If Redis is down, better to block all logins than allow brute-force
```

**Rationale:** Security is paramount. Temporary inability to log in is better than allowing unlimited password attempts.

### Availability-Critical: Fail-Open

```go
// Public API rate limiting
apiLimiter := ratelimiter.New(&ratelimiter.Config{
    Algorithm: ratelimiter.TokenBucket,
    Limit:     1000,
    Window:    time.Hour,
    FailOpen:  true,  // Fail-open: availability first
})

// If Redis is down, API stays available (temp loss of rate limiting)
```

**Rationale:** Availability is paramount. Temporary loss of rate limiting is better than API downtime. Monitor Redis closely to detect abuse.

### Balanced: Fail-Closed with Monitoring

```go
// Payment processing rate limit
paymentLimiter := ratelimiter.New(&ratelimiter.Config{
    Algorithm: ratelimiter.SlidingWindow,
    Limit:     100,
    Window:    time.Minute,
    FailOpen:  false,  // Fail-closed: prevent payment abuse
})

// Monitor Redis availability closely
// Have Redis HA (Sentinel) setup
// Alert on Redis failures immediately
```

**Rationale:** Financial operations require security. Use high-availability Redis to minimize downtime, but fail closed if it does happen.

## Monitoring and Alerting

When using fail-closed, monitor Redis availability closely:

```go
result, err := limiter.Allow(ctx, key)
if err != nil {
    metrics.RedisErrors.Inc()

    if !config.FailOpen {
        // We're denying requests due to Redis failure
        log.Error("rate limiter unavailable, denying requests",
            zap.Error(err),
            zap.String("key", key),
        )
        alerts.TriggerCritical("rate_limiter_redis_down")
    } else {
        // We're allowing requests despite Redis failure
        log.Warn("rate limiter unavailable, allowing requests",
            zap.Error(err),
        )
        alerts.TriggerWarning("rate_limiter_degraded")
    }
}
```

## Future Enhancements

1. **Circuit Breaker**: Add circuit breaker pattern to reduce load on failing Redis
2. **Fallback Modes**: Add in-memory fallback as third option
3. **Degraded Mode**: Allow reduced limits during outage (e.g., 10% of normal)
4. **Health Checks**: Expose health endpoint that reflects Redis status

## References

- [Fail-Open vs Fail-Closed in Security](https://en.wikipedia.org/wiki/Fail-safe#Fail-open_and_fail-closed)
- [Netflix Circuit Breaker (Hystrix)](https://github.com/Netflix/Hystrix/wiki/How-it-Works)
- [AWS Well-Architected: Reliability](https://aws.amazon.com/architecture/well-architected/)
- [Google SRE Book: Cascading Failures](https://sre.google/sre-book/addressing-cascading-failures/)
