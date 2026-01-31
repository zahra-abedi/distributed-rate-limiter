# Documentation

Welcome to the Distributed Rate Limiter documentation!

## Getting Started

New to the project? Start here:

1. **[README](../README.md)** - Project overview and quick start
2. **[ROADMAP](../ROADMAP.md)** - Development timeline and current status
3. **[EXAMPLES](EXAMPLES.md)** - Code examples and use cases

## Core Documentation

### Architecture & Design

- **[ARCHITECTURE.md](ARCHITECTURE.md)** - System architecture, components, and data flow
  - High-level architecture
  - Request flow
  - Component details
  - Deployment patterns
  - Performance characteristics

- **[ALGORITHMS.md](ALGORITHMS.md)** - Deep dive into rate limiting algorithms
  - Fixed Window Counter
  - Sliding Window Counter
  - Token Bucket
  - Comparison and selection guide

### Architecture Decision Records (ADRs)

Detailed records of key architectural decisions and trade-offs:

- **[ADR-001: Redis as Storage Backend](ADR/001-redis-as-storage-backend.md)**
  - Why Redis over alternatives (PostgreSQL, Memcached, etcd, DynamoDB)
  - Performance characteristics
  - Real-world usage examples

- **[ADR-002: Configurable Fail-Open vs Fail-Closed](ADR/002-configurable-fail-open-vs-fail-closed.md)**
  - Error handling strategies
  - Security vs availability trade-offs
  - Use case examples

- **[ADR-003: Decorator Pattern for Observability](ADR/003-decorator-pattern-for-observability.md)**
  - Why decorators over built-in observability
  - Separation of concerns
  - Composability benefits

See [ADR/README.md](ADR/README.md) for the full ADR index and how to create new ADRs.

## Code Examples

**[EXAMPLES.md](EXAMPLES.md)** provides practical code examples:
- API rate limiting with HTTP headers
- Login throttling (brute-force prevention)
- Tiered quotas (SaaS pricing)
- Batch operations
- Different algorithms for different use cases
- Testing without Redis

## API Reference

_(Coming soon in v0.4.0)_

- Go API documentation (godoc)
- gRPC API reference
- Protocol buffer definitions

## Deployment & Operations

_(Coming soon in v1.0.0)_

- Deployment guide
- Configuration reference
- Monitoring and alerting
- Troubleshooting guide
- Performance tuning

## Contributing

This is a personal portfolio project, but feedback and suggestions are welcome!

- See [ROADMAP](../ROADMAP.md) for planned features
- Check [ADR/](ADR/) for design decisions
- Review [ARCHITECTURE](ARCHITECTURE.md) for system design

## Navigation

```
docs/
├── README.md                   ← You are here
├── ARCHITECTURE.md             System architecture
├── ALGORITHMS.md               Algorithm deep-dive
├── EXAMPLES.md                 Code examples
├── ADR/                        Architecture Decision Records
│   ├── README.md              ADR index
│   ├── 000-template.md        ADR template
│   ├── 001-redis...md         Storage backend decision
│   ├── 002-fail-open...md     Error handling decision
│   └── 003-decorator...md     Observability decision
└── [Future]
    ├── API.md                 API documentation
    ├── DEPLOYMENT.md          Deployment guide
    └── TROUBLESHOOTING.md     Troubleshooting guide
```

## Questions?

For questions about:
- **Design decisions** → Check [ADR/](ADR/)
- **System architecture** → See [ARCHITECTURE.md](ARCHITECTURE.md)
- **Algorithm choice** → See [ALGORITHMS.md](ALGORITHMS.md)
- **How to use** → See [EXAMPLES.md](EXAMPLES.md)
- **Project status** → See [ROADMAP](../ROADMAP.md)
