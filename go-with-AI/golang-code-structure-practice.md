# Go Service Structure Best Practices

This note captures a pragmatic layout for Go backend services and highlights how the same ideas map to a Python FastAPI clean-architecture project.

## Recommended Go Layout

```
.
├── cmd/
│   ├── api/
│   │   └── main.go        # process wiring (config, logger, DI)
│   └── worker/
│       └── main.go        # second binary
├── internal/
│   ├── app/               # application services / use-cases
│   ├── domain/            # domain types, validation, errors
│   ├── transport/
│   │   ├── http/          # HTTP handlers, middleware, router setup
│   │   └── ws/            # WebSocket handlers
│   ├── store/             # database repositories (e.g. Postgres)
│   ├── rpc/               # external clients (Ethereum, REST, etc.)
│   ├── index/             # asynchronous pipelines / workers
│   ├── config/            # centralised configuration loader
│   └── observability/     # logging, metrics, tracing adapters
├── pkg/                   # optional: libraries you want to publish
├── migrations/            # SQL migrations
├── scripts/               # operational shell helpers
└── web/                   # static assets / SPA
```

### Key Principles

- **Thin `main`**: `cmd/*/main.go` only parses config, builds dependencies, and calls a `Run()` function. All logic lives in libraries.
- **Dependency Direction**: transport → app → domain → infrastructure. Lower-level code never imports transport packages.
- **Interfaces at Boundaries**: define service interfaces in `internal/app`, repository interfaces in the same package or domain, with concrete Postgres/Ethereum implementations in `internal/store`, `internal/rpc`.
- **Internal Visibility**: keep everything under `internal/` unless you intentionally expose it. Forces encapsulation.
- **Test Strategy**: unit-test services with mocked repositories; handler tests use mock services; integration tests live under `test/` or `_test.go` files and spin up real dependencies.
- **Config & Observability**: initialise once and inject downward. Avoid global state where possible (only logger/metrics with careful use).

## FastAPI Clean Architecture Reference

A typical FastAPI project using clean architecture might look like:

```
.
├── app/
│   ├── api/               # routers, controllers
│   ├── core/              # config, logging, dependency overrides
│   ├── schemas/           # Pydantic models (request/response)
│   ├── services/          # application services/use-cases
│   ├── repositories/      # database access (SQLAlchemy, etc.)
│   ├── models/            # ORM entities / domain models
│   └── main.py            # FastAPI app creation
├── tests/
├── alembic/               # migrations
└── scripts/
```

### Comparison Table

| Concept                     | Go Layout (`internal/*`)            | FastAPI Layout (`app/*`)           | Notes |
|-----------------------------|-------------------------------------|------------------------------------|-------|
| Entrypoint binaries         | `cmd/api/main.go`, `cmd/worker`     | `app/main.py` (ASGI), CLI scripts  | Go allows multiple binaries easily; FastAPI often uses `uvicorn` |
| Application services        | `internal/app`                      | `app/services`                     | Same idea: business logic, orchestrate repos |
| Domain models               | `internal/domain`                   | `app/models` (ORM) or `schemas`    | Go can use structs + validation; FastAPI splits ORM vs Pydantic |
| HTTP transport              | `internal/transport/http`           | `app/api`                          | Handlers/controllers mapping HTTP requests |
| Data access/repositories    | `internal/store/postgres`           | `app/repositories`                 | Both expose interfaces consumed by services |
| External clients            | `internal/rpc`                      | `app/clients` or part of repos     | Keep integrations isolated |
| Background jobs             | `internal/index`, separate `cmd`    | Celery/BackgroundTasks             | Go typically ships worker binaries |
| Configuration & observability | `internal/config`, `internal/observability` | `app/core/config`, logging modules | Both centralise settings and logging |
| Reusable libraries          | `pkg/` (optional)                   | external packages (PyPI)           | Go `internal/` blocks reuse; Pyton uses virtualenv packaging |

### Mental Model

- The layers are equivalent: controllers/handlers at the edge, services/use-cases in the middle, repositories/clients underneath, domain types shared across.
- Go emphasises package boundaries and compile-time visibility (`internal/`), while FastAPI relies on directory layout conventions and dependency injection (`Depends`) to keep layers clean.
- Interfaces in Go are consumer-defined. A service defines the methods it needs from a repository, and the Postgres implementation satisfies it. In Python, protocols or abstract base classes can play the same role, but duck typing often suffices.

## Practice Checklist

1. Keep business rules out of HTTP handlers—each operation should live behind an interface in `internal/app`.
2. Avoid global singletons; prefer passing dependencies from `cmd/` into constructors.
3. Give tests a way to swap implementations (use interfaces in Go, dependency overrides in FastAPI).
4. Keep package imports acyclic; if you hit an import cycle, your separation isn’t strict enough.
5. Document the structure (see `docs/code-structure-overview.md`) and update it as the codebase evolves.
