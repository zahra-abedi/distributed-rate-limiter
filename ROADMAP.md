# Distributed Rate Limiter - Roadmap

This roadmap outlines the development plan for the Distributed Rate Limiter project. The project is being developed iteratively with a focus on production-quality code and comprehensive testing.

## Project Status: 🚧 In Active Development

**Current Version:** v0.2.0-alpha

---

## ✅ v0.1.0 - Core Interface & Design (Completed)

Foundation work: interface design, documentation, and testing framework.

### Completed Features

- [x] **RateLimiter Interface**
  - `Allow(ctx, key)` - Check single request
  - `AllowN(ctx, key, n)` - Check batch requests (atomic)
  - `Reset(ctx, key)` - Clear rate limit state
  - `Close()` - Resource cleanup

- [x] **Configuration System**
  - Config validation
  - Sensible defaults (prefix, fail-closed)
  - Key formatting and prefixing

- [x] **Test Infrastructure**
  - InterfaceTestSuite (reusable contract tests)
  - Config validation tests (15 test cases)
  - Result helper tests (4 test cases)
  - 100% coverage of interface layer

- [x] **Comprehensive Documentation**
  - Architecture documentation (ARCHITECTURE.md)
  - Algorithm analysis (ALGORITHMS.md)
  - Architecture Decision Records (3 ADRs)
  - Code examples (EXAMPLES.md)

- [x] **CI/CD Pipeline**
  - GitHub Actions workflow
  - Automated testing (unit + integration)
  - golangci-lint integration
  - Go 1.25 support

### Key Decisions Made

See [docs/ADR/](docs/ADR/) for detailed decision records:
- [ADR-001](docs/ADR/001-redis-as-storage-backend.md): Redis as storage backend
- [ADR-002](docs/ADR/002-configurable-fail-open-vs-fail-closed.md): Configurable fail-open/fail-closed
- [ADR-003](docs/ADR/003-decorator-pattern-for-observability.md): Decorator pattern for observability

---

## 🚧 v0.2.0 - Core Algorithms (In Progress)

Algorithm implementations with Redis backend.

**Implementation Note:** Originally planned to implement in-memory storage first, then migrate to Redis in v0.3.0. Changed approach to implement Redis-backed algorithms from the start for better production readiness and to avoid duplicate work.

### Features

- [x] **Fixed Window Counter Algorithm**
  - Redis-backed implementation with Lua scripts
  - Atomic INCR + EXPIRE operations
  - Unit tests for validation and edge cases
  - Integration tests with miniredis
  - Fail-open/fail-closed behavior
  - Benchmark tests (11 scenarios)

- [x] **Sliding Window Counter Algorithm**
  - Redis-backed implementation with Lua scripts
  - Weighted average calculations for smooth rate limiting
  - Unit tests for weighted count logic
  - Integration tests with miniredis
  - Benchmark tests (12 scenarios)
  - Fail-open/fail-closed behavior

- [ ] **Token Bucket Algorithm**
  - Redis-backed implementation
  - Refill rate calculations
  - Float precision handling
  - Unit and integration tests
  - Benchmark tests

### Success Criteria

- All algorithms pass InterfaceTestSuite
- Algorithm-specific edge cases tested
- Benchmarks show expected performance characteristics
- Code coverage > 90%

---

## 📦 v0.3.0 - Advanced Redis Features (Planned)

Enhanced Redis integration and production features.

### Planned Features

- [ ] **Circuit Breaker**
  - Detect Redis failures
  - Automatic recovery
  - Cooldown periods

- [ ] **Connection Pooling**
  - Configurable pool size
  - Timeout configuration
  - Automatic reconnection

- [ ] **Script Optimization**
  - Script preloading
  - Performance tuning

- [ ] **Concurrency Tests**
  - Multi-server scenarios
  - Race condition testing
  - Distributed correctness verification

### Success Criteria

- Circuit breaker prevents cascading failures
- Performance benchmarks meet targets (30K+ RPS)
- Concurrency tests pass consistently

---

## 🌐 v0.4.0 - gRPC Service (Planned)

Language-agnostic gRPC API for the rate limiter.

### Planned Features

- [ ] **Protocol Buffers**
  - `ratelimiter.proto` service definition
  - Request/response message types
  - Code generation setup

- [ ] **gRPC Server**
  - Server implementation
  - Request handling
  - Error mapping
  - Context propagation

- [ ] **Service Features**
  - Health check endpoint
  - Graceful shutdown
  - Connection management
  - Request logging

- [ ] **Go Client Library**
  - `pkg/client` package
  - Connection pooling
  - Retry logic
  - Error handling

### Success Criteria

- gRPC service deployable as standalone binary
- Health checks working
- Client library provides ergonomic API
- Documentation for multi-language usage

---

## 📊 v0.5.0 - Observability (Planned)

Production-grade metrics, logging, and monitoring.

### Planned Features

- [ ] **Metrics Decorator**
  - Prometheus metrics integration
  - Request counters (by algorithm, result, error)
  - Latency histograms
  - Active requests gauge

- [ ] **Logging Decorator**
  - Structured logging with Zap
  - Error logging (Redis failures)
  - Debug logging (denied requests)
  - Configurable log levels

- [ ] **HTTP Middleware**
  - Metrics endpoint (/metrics)
  - Health check endpoint (/health)
  - Rate limit headers (X-RateLimit-*)

- [ ] **Example Dashboards**
  - Grafana dashboard JSON
  - Prometheus alerting rules
  - Common queries

### Success Criteria

- Metrics exposed in Prometheus format
- Logs provide actionable information
- Example dashboards visualize key metrics
- Documentation for monitoring setup

---

## 🚀 v1.0.0 - Production Ready (Planned)

Final polish, deployment tooling, and comprehensive documentation.

### Planned Features

- [ ] **Deployment**
  - Dockerfile (multi-stage build)
  - Docker Compose (app + Redis + Prometheus + Grafana)
  - Kubernetes manifests (optional)
  - Helm chart (optional)

- [ ] **Documentation**
  - Complete README with quick start
  - API documentation (godoc)
  - Deployment guide
  - Troubleshooting guide
  - Performance tuning guide

- [ ] **Examples**
  - HTTP middleware example
  - gRPC interceptor example
  - Multi-algorithm example
  - Kubernetes deployment example

### Success Criteria

- All tests passing in CI
- Docker image available
- Complete documentation
- Production deployment guide
- Performance benchmarks published

---

## 🔮 Future Enhancements (Post v1.0.0)

Features for future releases:

### v1.1.0 - Advanced Storage
- Alternative storage backends (Memcached, DynamoDB)
- In-memory fallback mode
- Multi-region Redis support

### v1.2.0 - Advanced Features
- Dynamic rate limit configuration (from database/etcd)
- Per-key custom limits
- Rate limit rule engine
- WebSocket streaming API

### v1.3.0 - Performance
- Batch request optimization
- Redis pipelining
- Client-side caching
- Connection pooling improvements

### v2.0.0 - Distributed Coordination
- Consensus-based rate limiting (Raft)
- Multi-datacenter coordination
- Conflict resolution strategies

---

## Contributing

This is a personal portfolio project, but suggestions and feedback are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Versioning

This project follows [Semantic Versioning](https://semver.org/):
- **Major** (v1.0.0 → v2.0.0): Breaking API changes
- **Minor** (v1.0.0 → v1.1.0): New features, backwards compatible
- **Patch** (v1.0.0 → v1.0.1): Bug fixes, backwards compatible

---

## License

MIT License - see [LICENSE](LICENSE) for details.

## Author

**Zahra Abedi**
Senior Backend & Software Developer
