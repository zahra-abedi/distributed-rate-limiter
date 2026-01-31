# Architecture Decision Records (ADRs)

This directory contains Architecture Decision Records (ADRs) for the Distributed Rate Limiter project.

## What are ADRs?

An Architecture Decision Record (ADR) captures an important architectural decision made along with its context and consequences.

ADRs help us:
- Remember why we made certain decisions
- Onboard new contributors
- Review past decisions when circumstances change
- Document trade-offs and alternatives considered

## Format

Each ADR follows this structure:
- **Context**: What problem are we solving?
- **Decision**: What did we decide?
- **Consequences**: What are the positive, negative, and neutral outcomes?
- **Alternatives Considered**: What other options did we evaluate?

## Index of ADRs

| ADR | Title | Status | Date |
|-----|-------|--------|------|
| [000](000-template.md) | Template | N/A | Template |
| [001](001-redis-as-storage-backend.md) | Redis as Storage Backend | Accepted | 2026-01-30 |
| [002](002-configurable-fail-open-vs-fail-closed.md) | Configurable Fail-Open vs Fail-Closed | Accepted | 2026-01-30 |
| [003](003-decorator-pattern-for-observability.md) | Decorator Pattern for Observability | Accepted | 2026-01-30 |

## Creating a New ADR

1. Copy `000-template.md` to a new file
2. Use the next sequential number: `004-your-decision-title.md`
3. Fill in all sections
4. Update this README with a link to your ADR
5. Submit a pull request

## ADR Statuses

- **Proposed**: Under discussion, not yet decided
- **Accepted**: Decision has been made and is being implemented
- **Deprecated**: Decision is no longer current but kept for historical reference
- **Superseded**: Replaced by a newer ADR (link to the new one)

## Further Reading

- [Documenting Architecture Decisions by Michael Nygard](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions)
- [ADR GitHub Organization](https://adr.github.io/)
- [When Should I Write an ADR?](https://engineering.atspotify.com/2020/04/when-should-i-write-an-architecture-decision-record/)
