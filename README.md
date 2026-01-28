# Distributed Rate Limiter

Production-grade distributed rate limiting service implemented in Go with Redis backend.

## Overview

A general-purpose rate limiter that controls the frequency of actions across distributed systems. Supports multiple algorithms and provides both gRPC API and Go client library.

## Features

- **Multiple Algorithms**
  - Token Bucket (smooth rate limiting with burst tolerance)
  - Sliding Window Counter (precise rate limiting)
  - Fixed Window Counter (simple and efficient)

- **Distributed Coordination**
  - Redis-backed shared state
  - Atomic operations via Lua scripts
  - Works correctly across multiple servers

- **Production Patterns**
  - Graceful shutdown
  - Circuit breaker for Redis failures
  - Comprehensive observability (metrics, logging)
  - Configurable fail-open/fail-closed behavior

- **Flexible API**
  - gRPC service for any language
  - Native Go client library
  - Simple integration

## Use Cases

- API rate limiting (REST, GraphQL, webhooks)
- User action throttling (login attempts, file uploads)
- Resource protection (database queries, expensive operations)
- Abuse prevention (spam, scraping, brute force)
- Fair resource sharing (multi-tenant systems)

## Status

🚧 **Under active development** (Week 1-2, Jan 2026)

## Tech Stack

- **Language**: Go 1.21+
- **Storage**: Redis 7.x
- **API**: gRPC + Protocol Buffers
- **Observability**: Prometheus, Uber Zap
- **Testing**: Go testing, testify, Docker for integration tests

## Quick Start

Coming soon...

## Architecture

Coming soon...

## Documentation

- [Project Specification](docs/SPEC.md) - Coming soon
- [API Documentation](docs/API.md) - Coming soon
- [Deployment Guide](docs/DEPLOYMENT.md) - Coming soon

## Development

```bash
# Build
make build

# Run tests
make test

# Run server
make run

# Generate protobuf
make proto
```

## License

MIT

## Author

Zahra Abedi
