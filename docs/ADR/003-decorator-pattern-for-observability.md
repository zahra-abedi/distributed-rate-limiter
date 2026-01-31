# ADR 003: Decorator Pattern for Observability

**Date:** 2026-01-30

**Status:** Accepted

**Deciders:** Zahra Abedi

## Context

The rate limiter needs observability features for production use:
- **Metrics**: Request counts, latencies, error rates (Prometheus)
- **Logging**: Structured logs for debugging (Zap)
- **Tracing**: Distributed tracing for request flow (OpenTelemetry)

The question is: Should these concerns be built into the core `RateLimiter` interface, or added as a separate layer?

### Design Goals

1. **Clean Interface**: Keep the core interface focused on rate limiting logic
2. **Testability**: Allow testing core logic without observability overhead
3. **Flexibility**: Metrics/logging should be optional (local dev doesn't need Prometheus)
4. **Composability**: Ability to add multiple cross-cutting concerns (metrics + logging + tracing)
5. **Performance**: Observability shouldn't significantly impact rate limiting performance

## Decision

Use the **Decorator Pattern** to add observability as optional layers around the core `RateLimiter` interface.

### Core Interface (Clean)

```go
// Core interface - NO observability concerns
type RateLimiter interface {
    Allow(ctx context.Context, key string) (*Result, error)
    AllowN(ctx context.Context, key string, n int64) (*Result, error)
    Reset(ctx context.Context, key string) error
    Close() error
}
```

### Decorator Implementation

```go
// Metrics decorator
type MetricsDecorator struct {
    limiter     RateLimiter
    requests    *prometheus.CounterVec
    latency     prometheus.Histogram
}

func (m *MetricsDecorator) Allow(ctx context.Context, key string) (*Result, error) {
    start := time.Now()

    result, err := m.limiter.Allow(ctx, key)

    // Record metrics
    m.latency.Observe(time.Since(start).Seconds())
    m.requests.WithLabelValues(
        getAlgorithm(m.limiter),
        boolToString(result.Allowed),
        getErrorType(err),
    ).Inc()

    return result, err
}

// Logging decorator
type LoggingDecorator struct {
    limiter RateLimiter
    logger  *zap.Logger
}

func (l *LoggingDecorator) Allow(ctx context.Context, key string) (*Result, error) {
    result, err := l.limiter.Allow(ctx, key)

    if err != nil {
        l.logger.Error("rate limiter error",
            zap.String("key", key),
            zap.Error(err),
        )
    } else if !result.Allowed {
        l.logger.Debug("request denied",
            zap.String("key", key),
            zap.Int64("limit", result.Limit),
        )
    }

    return result, err
}
```

### Usage (Composable)

```go
// Development: No observability
limiter := NewTokenBucket(redis, config)

// Production: Metrics only
limiter = NewMetricsDecorator(
    NewTokenBucket(redis, config),
    prometheusRegistry,
)

// Production: Metrics + Logging
limiter = NewLoggingDecorator(
    NewMetricsDecorator(
        NewTokenBucket(redis, config),
        prometheusRegistry,
    ),
    logger,
)

// Production: Full observability stack
limiter = NewTracingDecorator(
    NewLoggingDecorator(
        NewMetricsDecorator(
            NewTokenBucket(redis, config),
            prometheusRegistry,
        ),
        logger,
    ),
    tracer,
)
```

## Consequences

### Positive

- ✅ **Clean separation of concerns**: Rate limiting logic separate from observability
- ✅ **Single Responsibility Principle**: Each decorator has one job
- ✅ **Testable**: Can unit test core algorithms without mocking Prometheus
- ✅ **Optional**: Observability features are opt-in, not mandatory
- ✅ **Composable**: Can mix and match decorators as needed
- ✅ **Flexible**: Easy to add new decorators (tracing, custom metrics, etc.)
- ✅ **No interface bloat**: Core interface stays small and focused
- ✅ **Performance**: Zero overhead when decorators not used

### Negative

- ⚠️ **More code to write**: Each decorator is a separate implementation
- ⚠️ **Setup verbosity**: Production setup has more layers to configure
- ⚠️ **Indirection**: Stack trace shows decorator calls, not just core logic
- ⚠️ **Learning curve**: Developers need to understand decorator pattern

### Neutral

- Slightly more complex initial setup
- Need helper functions to simplify common decorator stacks
- Documentation required to explain decorator usage

## Alternatives Considered

### Alternative 1: Built-in Observability

**Description:** Add metrics/logging methods to the core interface

```go
type RateLimiter interface {
    Allow(ctx context.Context, key string) (*Result, error)
    AllowN(ctx context.Context, key string, n int64) (*Result, error)
    Reset(ctx context.Context, key string) error
    Close() error

    // Observability built-in
    GetMetrics() *Metrics
    SetLogger(logger *zap.Logger)
}

type Metrics struct {
    TotalRequests   int64
    AllowedRequests int64
    DeniedRequests  int64
    Errors          int64
    AvgLatency      time.Duration
}
```

**Pros:**
- Simpler for users (everything in one place)
- Guaranteed metrics for all implementations
- Less code to write

**Cons:**
- Violates Single Responsibility Principle
- Couples business logic with observability
- Makes testing harder (need to mock metrics)
- Interface becomes larger and harder to implement
- Can't easily add new observability features
- Metrics always present (even in tests)

**Why not chosen:** Coupling observability with core logic makes the interface harder to test, implement, and extend. Not all use cases need metrics (e.g., testing, embedded systems).

### Alternative 2: Callback Functions

**Description:** Pass observability callbacks to the rate limiter

```go
type Callbacks struct {
    OnAllow func(key string, result *Result, latency time.Duration)
    OnDeny  func(key string, result *Result, latency time.Duration)
    OnError func(key string, err error)
}

type Config struct {
    // ...
    Callbacks *Callbacks  // Optional
}

limiter := NewTokenBucket(redis, &Config{
    // ...
    Callbacks: &Callbacks{
        OnAllow: func(key string, result *Result, latency time.Duration) {
            metrics.Allowed.Inc()
            logger.Debug("request allowed", zap.String("key", key))
        },
    },
})
```

**Pros:**
- Simple to understand
- No decorator pattern needed
- Flexible (can do anything in callback)

**Cons:**
- Callback hell for multiple observability concerns
- Error-prone (nil check required)
- Hard to compose (can't easily combine callbacks)
- Performance overhead (function call per request)
- Testing complexity (need to verify callbacks called)

**Why not chosen:** Callbacks are harder to compose and test. Decorator pattern is more idiomatic in Go and provides better composability.

### Alternative 3: Middleware/Interceptor Pattern

**Description:** Use gRPC-style interceptors

```go
type Interceptor func(ctx context.Context, key string, handler Handler) (*Result, error)

type Handler func(ctx context.Context, key string) (*Result, error)

func MetricsInterceptor(metrics *prometheus.Registry) Interceptor {
    return func(ctx context.Context, key string, handler Handler) (*Result, error) {
        start := time.Now()
        result, err := handler(ctx, key)
        metrics.RecordLatency(time.Since(start))
        return result, err
    }
}

// Chain interceptors
limiter := Chain(
    MetricsInterceptor(registry),
    LoggingInterceptor(logger),
)(baseLimiter)
```

**Pros:**
- Familiar pattern (like HTTP middleware, gRPC interceptors)
- Easy to chain
- Flexible

**Cons:**
- More complex than decorators
- Requires wrapping the base limiter
- Harder to reason about execution order
- Type safety issues with generics

**Why not chosen:** Decorator pattern is simpler and more type-safe. Interceptor pattern is overkill for this use case.

### Alternative 4: Aspect-Oriented Programming (AOP)

**Description:** Use code generation or runtime proxies to inject observability

```go
//go:generate aspect-weaver -metrics -logging

type TokenBucket struct { ... }

func (tb *TokenBucket) Allow(ctx context.Context, key string) (*Result, error) {
    // Code generator adds metrics/logging here
    // ...
}
```

**Pros:**
- Clean business logic (observability added automatically)
- DRY (don't repeat metrics code)

**Cons:**
- Requires code generation or reflection
- Magic behavior (hard to debug)
- Not idiomatic Go
- Build complexity

**Why not chosen:** Not idiomatic in Go. Prefer explicit code over magic.

## Implementation Plan

### Phase 1: Core Decorators

1. **MetricsDecorator** - Prometheus metrics
   - Request counter (by algorithm, allowed/denied, error type)
   - Latency histogram
   - Active requests gauge

2. **LoggingDecorator** - Structured logging
   - Error logs for Redis failures
   - Debug logs for denied requests
   - Info logs for resets

### Phase 2: Advanced Decorators

3. **TracingDecorator** - OpenTelemetry tracing
   - Span for each Allow/AllowN call
   - Tags: algorithm, key, result

4. **CachingDecorator** - In-memory cache for hot keys
   - Reduce Redis load for high-traffic keys
   - TTL-based invalidation

### Helper Functions

```go
// Helper to create production-ready limiter
func NewProductionLimiter(config *Config, obs *Observability) RateLimiter {
    base := newLimiterByAlgorithm(config)

    limiter := RateLimiter(base)

    if obs.Metrics != nil {
        limiter = NewMetricsDecorator(limiter, obs.Metrics)
    }

    if obs.Logger != nil {
        limiter = NewLoggingDecorator(limiter, obs.Logger)
    }

    if obs.Tracer != nil {
        limiter = NewTracingDecorator(limiter, obs.Tracer)
    }

    return limiter
}
```

## Testing Strategy

### Core Logic Tests (No Decorators)

```go
func TestTokenBucket_Allow(t *testing.T) {
    limiter := NewTokenBucket(mockStorage, config)
    // Test pure business logic, no metrics noise
}
```

### Decorator Tests

```go
func TestMetricsDecorator(t *testing.T) {
    mockLimiter := &MockRateLimiter{
        AllowFunc: func(ctx, key) (*Result, error) {
            return &Result{Allowed: true}, nil
        },
    }

    registry := prometheus.NewRegistry()
    decorator := NewMetricsDecorator(mockLimiter, registry)

    decorator.Allow(ctx, "test")

    // Verify metrics recorded
    metrics := testutil.CollectAndCount(registry)
    assert.Equal(t, 1, metrics["rate_limiter_requests_total"])
}
```

## Real-World Examples

**Decorator pattern used in:**
- Go's `io.Reader` interface (`TeeReader`, `MultiReader`)
- HTTP middleware (Gorilla, Chi, Echo)
- Database drivers (`sql.Conn` wrappers)
- Logging libraries (structured loggers wrapping writers)

## References

- [Design Patterns: Decorator](https://refactoring.guru/design-patterns/decorator)
- [Go's io package](https://pkg.go.dev/io)
- [Prometheus Go Client](https://github.com/prometheus/client_golang)
- [Uber Zap Logger](https://github.com/uber-go/zap)
- [OpenTelemetry Go](https://opentelemetry.io/docs/instrumentation/go/)
