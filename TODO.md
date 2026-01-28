# Distributed Rate Limiter - Implementation Checklist

## Day 1-2: Core Algorithms (Jan 28-29)
- [ ] Define RateLimiter interface
- [ ] Implement Token Bucket algorithm
- [ ] Implement Sliding Window Counter algorithm
- [ ] Implement Fixed Window Counter algorithm
- [ ] Unit tests for Token Bucket
- [ ] Unit tests for Sliding Window
- [ ] Unit tests for Fixed Window

## Day 3-4: Redis Integration (Jan 30-31)
- [ ] Redis client setup
- [ ] Lua scripts (token_bucket.lua, sliding_window.lua, fixed_window.lua)
- [ ] Redis storage backend
- [ ] Circuit breaker for Redis
- [ ] Integration tests

## Day 5-6: gRPC Service (Feb 1-2)
- [ ] Protobuf schema
- [ ] Generate Go code
- [ ] gRPC server implementation
- [ ] Health check endpoint
- [ ] Graceful shutdown

## Day 7-8: Observability (Feb 3-4)
- [ ] Prometheus metrics
- [ ] Structured logging (Zap)
- [ ] HTTP middleware
- [ ] gRPC interceptor
- [ ] Example applications

## Day 9-10: Deployment & Docs (Feb 5-6)
- [ ] Dockerfile
- [ ] Docker Compose
- [ ] GitHub Actions CI/CD
- [ ] Comprehensive README
- [ ] Architecture diagrams

---

**Quick Reference:**
- All 3 algorithms must be implemented (Token Bucket, Sliding Window, Fixed Window)
- Each algorithm needs its own Lua script for Redis
- Each algorithm needs unit tests
- Each algorithm should demonstrate different trade-offs
