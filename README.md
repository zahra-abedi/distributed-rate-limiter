# Distributed Rate Limiter

[![CI](https://github.com/zahra-abedi/distributed-rate-limiter/actions/workflows/ci.yml/badge.svg)](https://github.com/zahra-abedi/distributed-rate-limiter/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/zahra-abedi/distributed-rate-limiter)](https://goreportcard.com/report/github.com/zahra-abedi/distributed-rate-limiter)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.25-blue.svg)](https://golang.org/)

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

🚧 **Under active development** - [See Roadmap](ROADMAP.md)

**Current:** v0.1.0 (Core Interface) ✅
**Next:** v0.2.0 (Algorithm Implementations) 🚧

## Tech Stack

- **Language**: Go 1.25+
- **Storage**: Redis 7.x
- **API**: gRPC + Protocol Buffers
- **Observability**: Prometheus, Uber Zap
- **Testing**: Go testing, testify, Docker for integration tests

## Quick Start

```go
package main

import (
    "context"
    "time"

    "github.com/zahraabedi/distributed-rate-limiter/internal/ratelimiter"
)

func main() {
    config := &ratelimiter.Config{
        Algorithm: ratelimiter.TokenBucket,
        Limit:     100,
        Window:    time.Minute,
    }

    // Algorithm implementations coming in v0.2.0
    // limiter := ratelimiter.New(redisClient, config)
    // result, err := limiter.Allow(ctx, "user:12345")
}
```

See [EXAMPLES.md](docs/EXAMPLES.md) for detailed code examples.

## Documentation

### Core Documentation
- **[Architecture](docs/ARCHITECTURE.md)** - System design, components, data flow
- **[Algorithms](docs/ALGORITHMS.md)** - Algorithm analysis and selection guide
- **[Examples](docs/EXAMPLES.md)** - Code examples and use cases
- **[Roadmap](ROADMAP.md)** - Development plan and timeline

### Architecture Decision Records
- **[ADR-001](docs/ADR/001-redis-as-storage-backend.md)** - Redis as storage backend
- **[ADR-002](docs/ADR/002-configurable-fail-open-vs-fail-closed.md)** - Fail-open vs fail-closed
- **[ADR-003](docs/ADR/003-decorator-pattern-for-observability.md)** - Decorator pattern for observability

See [docs/](docs/) for complete documentation.

## Development

```bash
# Clone the repository
git clone https://github.com/zahra-abedi/distributed-rate-limiter.git
cd distributed-rate-limiter

# Install dependencies
make deps

# Run tests
make test

# Run tests with coverage report
make test-coverage

# Run benchmarks
make bench

# Run linter
make lint

# Run all checks (fmt-check + vet + lint + test)
make check

# Build the server
make build

# See all available commands
make help
```

## License

MIT

## Author

Zahra Abedi
