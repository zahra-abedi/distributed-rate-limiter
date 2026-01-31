# ADR 001: Redis as Storage Backend

**Date:** 2026-01-30

**Status:** Accepted

**Deciders:** Zahra Abedi

## Context

A distributed rate limiter requires shared state across multiple application servers. When a user makes requests that are load-balanced across 3 servers, each server needs to know about requests handled by the others to enforce accurate rate limits.

### Requirements

The storage backend must provide:
1. **Atomic Operations**: Read-modify-write must be atomic to prevent race conditions
2. **Fast Access**: Sub-millisecond latency (rate limiting is on critical request path)
3. **TTL Support**: Automatic expiration of old data for memory management
4. **Shared State**: Accessible from all application servers
5. **High Availability**: Failure must not take down the entire application
6. **Scripting**: Complex operations need atomicity (check + update in single operation)

## Decision

Use **Redis** as the primary storage backend for distributed rate limiting state.

### Implementation Details

- Redis 7.x with Lua scripting for atomic operations
- Each rate limiting algorithm implemented as a Lua script
- Connection pooling for performance
- Circuit breaker pattern for Redis failures
- Support for both Redis Sentinel (HA) and Redis Cluster (scale)

## Consequences

### Positive

- ✅ **Sub-millisecond latency**: ~0.5ms local, ~2ms remote
- ✅ **Lua scripting**: All three algorithms (Fixed Window, Sliding Window, Token Bucket) can be implemented atomically
- ✅ **Built-in TTL**: `EXPIRE` command handles automatic cleanup
- ✅ **Rich data types**: Strings for counters, Hashes for token buckets, can use Sorted Sets if needed
- ✅ **Industry standard**: Same approach used by Stripe, GitHub, AWS, Cloudflare
- ✅ **Mature ecosystem**: Well-documented, extensive tooling (Redis Insight, monitoring)
- ✅ **High availability options**: Redis Sentinel for HA, Redis Cluster for horizontal scaling
- ✅ **Easy development**: Simple to run locally with Docker

### Negative

- ⚠️ **Additional infrastructure**: Requires Redis deployment and management
- ⚠️ **Single point of failure**: Without proper HA setup, Redis downtime affects rate limiting
- ⚠️ **Memory-bound**: All state must fit in RAM (though TTL helps)
- ⚠️ **Cost**: Managed Redis (ElastiCache, Azure Cache) has hosting costs

### Neutral

- Single-threaded nature means careful Lua script optimization needed
- Need to implement circuit breaker for graceful degradation
- Clustering adds complexity but enables horizontal scale

## Alternatives Considered

### Alternative 1: PostgreSQL/MySQL

**Description:** Use relational database with row-level locking

```sql
BEGIN;
SELECT count FROM rate_limits WHERE user_id = '123' FOR UPDATE;
UPDATE rate_limits SET count = count + 1 WHERE user_id = '123';
COMMIT;
```

**Pros:**
- Already in most tech stacks
- Can query historical data
- Strong consistency guarantees
- Persistent storage

**Cons:**
- 5-10ms latency (too slow for critical path)
- Adds load to primary database
- Manual TTL cleanup (cron jobs)
- Row-level locking overhead
- Not designed for high-frequency operations

**Why not chosen:** Latency too high (rate limiting on critical request path), would overload database with high-frequency operations.

### Alternative 2: Memcached

**Description:** Simple distributed cache with atomic INCR operations

**Pros:**
- Very fast (<1ms latency)
- Simple key-value model
- Built-in TTL
- Low memory overhead

**Cons:**
- No Lua scripting (can't implement Token Bucket/Sliding Window atomically)
- CAS (Compare-And-Set) operations are clunky for complex logic
- No built-in replication (need consistent hashing)
- Less rich data types

**Why not chosen:** Lack of scripting means only Fixed Window algorithm can be properly implemented. Token Bucket requires atomic read-modify-write of multiple fields (tokens + last_refill timestamp).

### Alternative 3: etcd

**Description:** Distributed key-value store with strong consistency, used by Kubernetes

**Pros:**
- Strong consistency (Raft consensus)
- Built-in watch mechanism
- Good for configuration management
- Excellent for storing rate limit configurations

**Cons:**
- Higher latency (~5-10ms vs Redis <1ms)
- Not optimized for high-frequency operations
- Overkill for simple rate limiting

**Why not chosen:** Optimized for configuration/coordination, not high-frequency state updates. Latency too high for request path.

**Note:** Could be used in hybrid approach (etcd for rate limit configs, Redis for state).

### Alternative 4: DynamoDB (AWS)

**Description:** Serverless NoSQL database with conditional writes

**Pros:**
- Serverless (auto-scaling)
- Multi-region replication (Global Tables)
- Pay-per-use pricing
- Managed service (no ops)

**Cons:**
- Higher latency (10-20ms)
- Costs scale with request volume
- Conditional writes less flexible than Lua scripts
- AWS vendor lock-in

**Why not chosen:** Latency too high, costs can be significant at scale. Good for AWS Lambda rate limiting but not general-purpose.

### Alternative 5: In-Memory (Local)

**Description:** Simple Go map with mutex, no external dependencies

```go
type LocalLimiter struct {
    mu      sync.RWMutex
    buckets map[string]*Bucket
}
```

**Pros:**
- Zero latency (<0.1ms)
- No external dependencies
- Simple implementation
- Perfect for testing

**Cons:**
- Not distributed (each server has own state)
- Lost on restart (no persistence)
- Can't coordinate across multiple servers

**Why not chosen:** Defeats the purpose of "distributed" rate limiter. Each server would enforce limits independently, allowing 3x the intended rate across 3 servers.

**Note:** Used for unit testing and as fallback when Redis is unavailable.

## Performance Characteristics

**Redis vs Alternatives (single instance):**

| Backend       | Latency (p99) | Throughput (RPS) | Memory/User | TTL Support |
|---------------|---------------|------------------|-------------|-------------|
| Redis         | ~2ms          | 50-100K          | ~150 bytes  | ✅ Built-in  |
| Memcached     | ~1ms          | 80K              | ~100 bytes  | ✅ Built-in  |
| PostgreSQL    | ~50ms         | 5K               | ~200 bytes  | ❌ Manual    |
| DynamoDB      | ~25ms         | 20K              | ~200 bytes  | ✅ Built-in  |
| etcd          | ~10ms         | 10K              | ~150 bytes  | ✅ Leases    |

**Redis Scaling:**
- Single instance: 50K-100K ops/sec
- With Redis Cluster: Millions of ops/sec (horizontal scaling)
- Memory: ~150 bytes per active user (Token Bucket)
- For 1M users: ~150 MB RAM

## Real-World Usage

Companies using Redis for rate limiting:
- **Stripe**: Payment API rate limits
- **GitHub**: API quota enforcement
- **Cloudflare**: DDoS protection and rate limiting
- **Twitter**: API rate limits (historical)
- **AWS API Gateway**: Built-in rate limiting uses similar patterns

## References

- [Redis Documentation](https://redis.io/docs/)
- [Stripe Rate Limiting Blog](https://stripe.com/blog/rate-limiters)
- [Cloudflare Rate Limiting](https://blog.cloudflare.com/counting-things-a-lot-of-different-things/)
- [Redis Lua Scripting](https://redis.io/docs/manual/programmability/eval-intro/)
- [Redis Sentinel HA](https://redis.io/docs/manual/sentinel/)
- [Redis Cluster Scaling](https://redis.io/docs/manual/scaling/)
