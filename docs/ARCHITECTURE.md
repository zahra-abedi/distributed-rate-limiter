# System Architecture

## Overview

The Distributed Rate Limiter is a production-grade service for controlling request frequency across distributed systems. It provides a unified interface for multiple rate limiting algorithms backed by Redis for shared state coordination.

## Technology Stack

| Layer | Technology | Purpose |
|-------|------------|---------|
| **Language** | Go 1.21+ | High performance, concurrent, simple deployment |
| **Storage** | Redis 7.x | Sub-millisecond latency, Lua scripting, TTL support |
| **API** | gRPC + Protocol Buffers | Language-agnostic, efficient binary protocol |
| **Observability** | Prometheus + Zap | Metrics and structured logging |
| **Testing** | Go testing + testify + Docker | Unit, integration, and benchmark tests |

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     Client Applications                  │
│  (REST APIs, GraphQL, Microservices, Web Apps, etc.)    │
└────────────┬────────────────────────────────┬───────────┘
             │                                │
             │ Option 1: Go Library           │ Option 2: gRPC
             │ (import pkg/client)            │ (any language)
             ▼                                ▼
    ┌─────────────────┐              ┌──────────────────┐
    │  Go Client Lib  │              │  gRPC Service    │
    │  pkg/client     │              │  cmd/server      │
    └────────┬────────┘              └────────┬─────────┘
             │                                │
             └────────────┬───────────────────┘
                          │
                          ▼
             ┌────────────────────────────┐
             │   Rate Limiter Core        │
             │   internal/ratelimiter     │
             │                            │
             │  ┌──────────────────────┐  │
             │  │ Decorators (Metrics, │  │
             │  │ Logging, Tracing)    │  │
             │  └──────────┬───────────┘  │
             │             │               │
             │  ┌──────────▼───────────┐  │
             │  │   RateLimiter        │  │
             │  │   Interface          │  │
             │  └──────────┬───────────┘  │
             │             │               │
             │  ┌──────────▼───────────┐  │
             │  │  Algorithm Impls     │  │
             │  │  - Token Bucket      │  │
             │  │  - Sliding Window    │  │
             │  │  - Fixed Window      │  │
             │  └──────────┬───────────┘  │
             └─────────────┼───────────────┘
                           │
                           ▼
               ┌───────────────────────┐
               │  Storage Layer        │
               │  internal/storage     │
               │                       │
               │  - Redis Client       │
               │  - Lua Scripts        │
               │  - Circuit Breaker    │
               └───────────┬───────────┘
                           │
                           ▼
                   ┌───────────────┐
                   │     Redis     │
                   │   (Sentinel   │
                   │   or Cluster) │
                   └───────────────┘
```

## Request Flow

### Detailed Flow Diagram

```
1. Client Request
   │
   ├─→ HTTP Handler (user's application)
   │
   ├─→ Extract identifier (user ID, API key, IP, etc.)
   │
   ├─→ Call: limiter.Allow(ctx, "user:12345")
   │
   ├─→ [Optional] Metrics Decorator
   │   └─→ Record: request started, start timer
   │
   ├─→ [Optional] Logging Decorator
   │   └─→ Log: debug info
   │
   ├─→ Algorithm Implementation (e.g., Token Bucket)
   │   ├─→ Format key: "ratelimit:user:12345"
   │   ├─→ Prepare Lua script arguments
   │   └─→ Execute Lua script in Redis (atomic)
   │
   ├─→ Redis (Lua Script Execution)
   │   ├─→ EVAL script with key and args
   │   ├─→ Algorithm logic (check + update state)
   │   ├─→ SET TTL if needed
   │   └─→ Return: allowed, remaining, reset_at
   │
   ├─→ Parse Redis response
   │   └─→ Create Result{Allowed, Limit, Remaining, RetryAfter, ResetAt}
   │
   ├─→ [Optional] Logging Decorator
   │   └─→ Log: result (error if denied)
   │
   ├─→ [Optional] Metrics Decorator
   │   └─→ Record: latency, result (allowed/denied), errors
   │
   └─→ Return Result to application
       │
       ├─→ If Allowed: process request
       │
       └─→ If Denied: return HTTP 429 with headers
           └─→ X-RateLimit-Limit: 100
           └─→ X-RateLimit-Remaining: 0
           └─→ X-RateLimit-Reset: 1704452400
           └─→ Retry-After: 3420
```

### Timing Breakdown

Typical request latency breakdown:

```
Total: ~2-5ms
├─ Go overhead: ~0.1ms
│  └─ Context handling, key formatting
├─ Network RTT: ~1-2ms
│  └─ Application → Redis
├─ Redis Lua execution: ~0.5-1ms
│  └─ Script execution, memory updates
└─ Response parsing: ~0.1ms
   └─ Result construction
```

## Distributed Coordination

### The Challenge

```
User makes 100 requests in 1 second
    ↓
Load balancer distributes:
    ├─→ Server 1: 33 requests
    ├─→ Server 2: 34 requests
    └─→ Server 3: 33 requests

Without coordination:
    Each server: "33 < 100 limit ✓ ALLOW"
    Result: User gets 100 requests through ✓

With Redis coordination:
    Server 1 → Redis: INCR counter (returns 1, 2, 3, ..., 33)
    Server 2 → Redis: INCR counter (returns 34, 35, ..., 67)
    Server 3 → Redis: INCR counter (returns 68, 69, ..., 100)
    All servers see true global count
    Request 101: Redis returns 101 > 100 → DENY ✓
```

### Atomicity via Lua Scripts

Why Lua scripts are critical:

```
❌ Bad: Separate Redis commands (race condition)
1. count = GET "user:123"         ← Server 1 reads: 99
2. count = GET "user:123"         ← Server 2 reads: 99 (race!)
3. if count < 100:                ← Both servers: 99 < 100 ✓
4.   INCR "user:123"              ← Server 1: now 100
5.   INCR "user:123"              ← Server 2: now 101 (over limit!)

✅ Good: Lua script (atomic)
1. EVAL "
     local count = redis.call('INCR', KEYS[1])
     if count == 1 then
       redis.call('EXPIRE', KEYS[1], ARGV[1])
     end
     return count
   " 1 "user:123" 60

Only one server can execute at a time → no race condition
```

### Redis High Availability

**Development:**
```
Single Redis instance
└─→ Good enough for testing
    └─→ redis://localhost:6379
```

**Production (High Availability):**
```
Redis Sentinel
├─→ 3+ Sentinel processes monitor Redis
├─→ Automatic failover if master dies
├─→ Clients connect via Sentinel
└─→ redis://sentinel1,sentinel2,sentinel3/mymaster

Downtime: ~2-30 seconds during failover
```

**Production (Horizontal Scale):**
```
Redis Cluster
├─→ 6+ nodes (3 masters, 3 replicas)
├─→ Data sharded across masters
├─→ Hash slot based key distribution
└─→ redis://node1,node2,node3,node4,node5,node6

Throughput: Millions of ops/sec
```

## Component Details

### 1. RateLimiter Interface

**Location:** `internal/ratelimiter/interface.go`

**Responsibilities:**
- Define contract for all algorithms
- Provide `Result` type for responses
- Define `Config` for configuration

**Key Methods:**
```go
Allow(ctx, key) (*Result, error)    // Check single request
AllowN(ctx, key, n) (*Result, error) // Check N requests (atomic)
Reset(ctx, key) error                // Clear state
Close() error                        // Cleanup
```

### 2. Algorithm Implementations

**Location:** `internal/algorithms/`

**Implementations:**
- `fixed_window.go` - Simple counter, resets every window
- `sliding_window.go` - Weighted average of two windows
- `token_bucket.go` - Refilling bucket with burst capacity

**Shared Behavior:**
- All implement `RateLimiter` interface
- All use storage backend for persistence
- All handle fail-open/fail-closed via config

### 3. Storage Layer

**Location:** `internal/storage/`

**Components:**
- `redis.go` - Redis client wrapper, connection pool
- `lua/*.lua` - Lua scripts for atomic operations
- `circuit_breaker.go` - Detect and handle Redis failures

**Redis Client Features:**
- Connection pooling (default: 10 connections)
- Automatic reconnection
- Timeout handling (default: 100ms)
- Pipelining support for future optimization

### 4. Decorators (Observability)

**Location:** `internal/ratelimiter/decorators/`

**Available Decorators:**
- `metrics.go` - Prometheus metrics collection
- `logging.go` - Structured logging with Zap
- `tracing.go` - OpenTelemetry distributed tracing (planned)

**Composition:**
```go
base := NewTokenBucket(redis, config)
limiter := NewLoggingDecorator(
    NewMetricsDecorator(base, metrics),
    logger,
)
```

### 5. gRPC Service (Planned)

**Location:** `cmd/server/`

**Features:**
- gRPC server on port 8080
- Health check endpoint
- Graceful shutdown
- Multiple rate limiter instances (per-tenant)

**API:**
```protobuf
service RateLimiter {
  rpc Allow(AllowRequest) returns (AllowResponse);
  rpc AllowN(AllowNRequest) returns (AllowResponse);
  rpc Reset(ResetRequest) returns (ResetResponse);
}
```

## Data Flow Examples

### Example 1: Fixed Window

```
User: alice
Limit: 5 requests/minute
Window: 10:00:00 - 10:01:00

10:00:05 → Request 1
    ↓
Redis: EVAL fixed_window.lua "ratelimit:alice:1704448800" 60
    ↓
Lua: count = INCR "ratelimit:alice:1704448800"  → 1
     if count == 1:
       EXPIRE "ratelimit:alice:1704448800" 60
     return {count=1, ttl=60}
    ↓
Result: {Allowed=true, Remaining=4, ResetAt=10:01:00}

10:00:58 → Request 5
    ↓
Redis: count = INCR → 5
Result: {Allowed=true, Remaining=0}

10:00:59 → Request 6
    ↓
Redis: count = INCR → 6
Lua: if count > limit: return {allowed=false, ttl=1}
    ↓
Result: {Allowed=false, Remaining=0, RetryAfter=1s}

10:01:00 → Window resets (Redis TTL expires)
10:01:01 → Request 7
    ↓
Redis: Key doesn't exist, create new
Result: {Allowed=true, Remaining=4}
```

### Example 2: Token Bucket

```
User: bob
Capacity: 10 tokens
Refill: 1 token/second

10:00:00 → Bucket full (10 tokens)
    ↓
10:00:05 → 3 rapid requests
    ↓
Redis: EVAL token_bucket.lua "ratelimit:bob"
    ↓
Lua:
  data = HGETALL "ratelimit:bob"
  tokens = data.tokens || capacity  → 10
  last_refill = data.last_refill || now  → 10:00:00

  elapsed = now - last_refill  → 5 seconds
  refill = elapsed * (capacity / window)  → 5 * (10/10) = 5
  tokens = min(tokens + refill, capacity)  → 10

  For each of 3 requests:
    if tokens >= 1:
      tokens -= 1

  tokens after = 7

  HSET "ratelimit:bob" tokens 7 last_refill 10:00:05
  return {allowed=true, remaining=7}
    ↓
Result: All 3 allowed, 7 tokens remaining
```

## Failure Scenarios

### Scenario 1: Redis Temporarily Down

```
Request arrives
    ↓
limiter.Allow(ctx, "user:123")
    ↓
Redis connection fails (timeout)
    ↓
if config.FailOpen == true:
    Return: {Allowed=true, ...}  (temporary loss of rate limiting)
else:
    Return: {Allowed=false, ...}  (deny during outage)
    ↓
Log error + emit metric
    ↓
Alert: "Redis unavailable"
```

### Scenario 2: Network Partition

```
App Server 1 (can reach Redis) ← ✓
App Server 2 (cannot reach Redis) ← ✗

Server 1: Normal operation, enforces limits
Server 2:
  - Fail-closed → Denies all requests
  - Fail-open → Allows all requests (temp bypass)

When partition heals:
  Server 2 reconnects → resume normal operation
```

### Scenario 3: Redis Overload

```
Redis CPU at 100%
    ↓
Timeouts increase: 1ms → 50ms → 500ms
    ↓
Circuit breaker detects slow responses
    ↓
Circuit opens: stop sending requests to Redis
    ↓
Fail-open/fail-closed behavior activates
    ↓
Circuit half-open after cooldown: test Redis
    ↓
If Redis recovered: circuit closes, resume normal operation
```

## Performance Characteristics

### Throughput

**Single Redis Instance:**
- Fixed Window: ~50,000 requests/sec
- Sliding Window: ~30,000 requests/sec
- Token Bucket: ~35,000 requests/sec

**Redis Cluster (6 nodes):**
- Fixed Window: ~300,000 requests/sec
- Sliding Window: ~180,000 requests/sec
- Token Bucket: ~210,000 requests/sec

### Latency (p99)

| Component | Latency |
|-----------|---------|
| Go overhead | 0.1ms |
| Network RTT (local) | 0.5ms |
| Network RTT (same AZ) | 1-2ms |
| Network RTT (cross AZ) | 3-5ms |
| Redis Lua script | 0.5-1ms |
| **Total (same AZ)** | **2-5ms** |

### Memory Usage

**Per active user (Token Bucket):**
- Redis key: ~50 bytes
- Hash fields: ~100 bytes
- Overhead: ~20 bytes
- **Total: ~170 bytes**

**For 1 million users:**
- ~170 MB RAM in Redis
- Overhead: ~30 MB (Redis data structures)
- **Total: ~200 MB**

### Scalability Limits

**Vertical (Single Redis):**
- Up to 100K ops/sec
- Up to 10M active users (with TTL)
- Limited by single-threaded nature

**Horizontal (Redis Cluster):**
- Millions of ops/sec
- Hundreds of millions of users
- Limited by network bandwidth

## Deployment Architectures

### Development

```
┌─────────────┐
│   Your App  │
│  (localhost)│
└──────┬──────┘
       │
       ▼
┌─────────────┐
│   Redis     │
│  (Docker)   │
│   :6379     │
└─────────────┘
```

### Production (Simple)

```
┌──────────┐  ┌──────────┐  ┌──────────┐
│  App     │  │  App     │  │  App     │
│ Server 1 │  │ Server 2 │  │ Server 3 │
└────┬─────┘  └────┬─────┘  └────┬─────┘
     │             │             │
     └─────────────┼─────────────┘
                   │
                   ▼
            ┌──────────────┐
            │  Redis       │
            │  Sentinel    │
            │  (3 nodes)   │
            └──────────────┘
```

### Production (Scale)

```
                Load Balancer
                      │
        ┌─────────────┼─────────────┐
        │             │             │
   ┌────▼────┐   ┌───▼─────┐  ┌───▼─────┐
   │  App    │   │  App    │  │  App    │
   │ Servers │   │ Servers │  │ Servers │
   │ (N pods)│   │ (N pods)│  │ (N pods)│
   └────┬────┘   └────┬────┘  └────┬────┘
        │             │            │
        └─────────────┼────────────┘
                      │
              ┌───────▼───────┐
              │ Redis Cluster │
              │   6+ nodes    │
              │  (sharded)    │
              └───────────────┘
```

## Security Considerations

1. **Key Isolation**: Use prefixes to isolate tenants
2. **Redis Auth**: Enable `requirepass` in production
3. **Network Security**: Redis on private network only
4. **TLS**: Enable Redis TLS for encryption in transit
5. **Input Validation**: Validate all keys before Redis operations
6. **DoS Protection**: Rate limiter protects itself (meta rate limiting)

## Monitoring and Operations

**Key Metrics:**
- `rate_limiter_requests_total{algorithm, result, error}`
- `rate_limiter_latency_seconds{algorithm}`
- `rate_limiter_redis_errors_total{error_type}`

**Alerts:**
- Redis connection failures
- High error rate (> 1%)
- High latency (p99 > 10ms)

**Dashboards:**
- Request rate by algorithm
- Allow/deny ratio
- Redis connection health
- Latency percentiles

## References

- [Redis Documentation](https://redis.io/docs/)
- [gRPC Documentation](https://grpc.io/docs/)
- [Prometheus Best Practices](https://prometheus.io/docs/practices/)
- [Go Concurrency Patterns](https://go.dev/blog/pipelines)
