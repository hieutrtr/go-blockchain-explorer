# Code Structure Overview

This note captures how the Go codebase is organised today and the target layering we want to converge on. The focus is on code-level structure rather than infrastructure or product requirements.

## Current Layout (2025-11-01)

- `cmd/api`, `cmd/worker`: entrypoints producing the two binaries (REST API + WebSocket server, background indexer).
- `internal/api`: HTTP routing, middleware, handlers, and WebSocket hub implementation.
- `internal/index`: indexing coordinators (backfill, live tail, reorg) that orchestrate blockchain ingestion.
- `internal/rpc`: thin client wrapper around go-ethereum JSON-RPC with retry logic.
- `internal/store`: query layer built on top of PostgreSQL via `pgx` (models + SQL helpers).
- `internal/db`: database configuration, connection pool setup, and migration helpers.
- `internal/util`: logging and metrics primitives (planned to split into observability packages).
- `web/`: static frontend assets.
- `migrations/`, `scripts/`, `docs/`: operational artifacts (SQL migrations, shell scripts, documentation).

The structure is already modular, but several concerns are still tangled: handlers directly depend on `store` types, business rules live alongside transport code, and configuration is spread across multiple packages.

## Target Layered Architecture

```
┌──────────────────────────────────────────────┐
│                 Interfaces                   │
│  - cmd/api (HTTP)                            │
│  - cmd/worker (scheduler/CLI)                │
└──────────────────────┬───────────────────────┘
                       │
┌──────────────────────▼───────────────────────┐
│            Application Services             │
│  internal/app                                │
│  - BlockService, TxService, StatsService     │
│  - Coordinate use-cases, enforce business    │
│    invariants, orchestrate repositories      │
└──────────────────────┬───────────────────────┘
                       │
┌──────────────────────▼───────────────────────┐
│             Domain / Models                 │
│  internal/domain                             │
│  - Rich models, validation helpers           │
│  - Shared error types                        │
└──────────────────────┬───────────────────────┘
                       │
┌──────────────────────▼───────────────────────┐
│              Infrastructure                  │
│  internal/store (Postgres repositories)      │
│  internal/rpc (Ethereum client)              │
│  internal/index (pipelines, workers)         │
│  internal/db (connection pool + migrations)  │
└──────────────────────┬───────────────────────┘
                       │
┌──────────────────────▼───────────────────────┐
│          Cross-Cutting Foundations           │
│  internal/observability (logging/metrics)    │
│  internal/config (centralised settings)      │
└──────────────────────────────────────────────┘
```

**Layer rules**

1. `cmd/*` packages own wiring. They depend on services defined in `internal/app` and infrastructure implementations (e.g., `internal/store/postgres`), but higher layers never import `cmd/*`.
2. Transport packages (HTTP, WebSocket) call into service interfaces. They do not construct repositories directly.
3. `internal/app` depends only on domain types and interfaces. Infrastructure implementations satisfy those interfaces.
4. Reusable primitives that should never reach outside the repository (logging adapters, configuration loaders) stay under `internal/`.
5. Anything intended for reuse by other projects would live under `pkg/`, but today the repository is strictly internal.

## Configuration Strategy

- Introduce `internal/config` with typed structs for API, database, indexer, RPC, and observability settings.
- `cmd/api` and `cmd/worker` bootstrap configuration exactly once, then pass concrete slices (e.g., `config.API`, `config.Database`) into service constructors.
- Support override order: defaults → `.env`/file → environment variables → CLI flags.

## Dependency Wiring Example

```
// cmd/api/main.go (simplified)
cfg := config.Load()
logger := observability.NewLogger(cfg.Log)
metrics := observability.NewMetrics(cfg.Metrics)

dbPool := db.Open(cfg.Database, logger)
store := store.NewPostgresRepository(dbPool, logger, metrics)

rpcClient := rpc.NewClient(cfg.RPC, logger, metrics)
blockSvc := app.NewBlockService(store, rpcClient, logger)

server := transport.NewHTTPServer(blockSvc, cfg.API, logger, metrics)
server.Run()
```

Key idea: each dependency is constructed once in `cmd/` and passed down, making it easy to swap implementations for tests.

## Testing Implications

- Unit tests target the smallest layer possible (e.g., service tests mock repositories, handler tests use stub services).
- Integration tests spin up the real HTTP server using in-memory or containerised Postgres.
- Workers’ pipeline components (`internal/index`) expose interfaces so backfill logic can be tested with fake RPC and store implementations.

## Migration Plan

1. Create `internal/config` and `internal/observability` packages; move existing config structs and logging/metrics there.
2. Introduce `internal/app` with service interfaces. Start with blocks and transactions because handlers already expose those operations.
3. Refactor API handlers to depend on the new services, removing direct creation of `store.NewStore`.
4. Apply the same pattern to the worker process so it reuses the application services where possible.
5. Gradually split `internal/index` into orchestration (remains infrastructure) and business intent (lives in services/domain) as more rules emerge.

This document should evolve as major refactors land; keep it updated so contributors understand the architectural direction.
