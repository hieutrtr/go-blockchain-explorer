# API Server vs Worker Entrypoints (Go)

This note compares the two binaries in this project, shows how their `main` functions differ, and highlights structure patterns to keep aligned.

## Purpose & Scope

- **cmd/api/main.go**: boots the HTTP + WebSocket API, exposes REST endpoints, and manages the DB connection pool.
- **cmd/worker/main.go**: currently a lightweight background process that initialises shared observability, starts a metrics endpoint, and waits for shutdown signals (future home for indexer jobs).

## Server Entrypoint (cmd/api/main.go)

- Loads API + database + WebSocket configuration.
- Caps DB pool connections, opens `pgx` pool, logs success/failure.
- Constructs the WebSocket hub and runs it in a cancellable goroutine.
- Builds the HTTP server with chi router, middleware stack, and timeouts.
- Starts the HTTP server in a goroutine, listens for OS signals or server errors.
- Executes graceful shutdown: cancel hub, `Shutdown()` HTTP server with timeout, force `Close()` if needed.

## Worker Entrypoint (cmd/worker/main.go)

- Initialises shared observability via `util.Init()`.
- Starts the metrics server in a background goroutine.
- Listens for `SIGINT` / `SIGTERM`.
- Upon signal, logs shutdown intent, creates a 30s context, waits for completion (placeholder for future cleanup).

## Comparison Summary

| Aspect                    | API Server (`cmd/api`)                       | Worker (`cmd/worker`)                        |
|--------------------------|----------------------------------------------|----------------------------------------------|
| Primary role             | Serve HTTP/WS requests, query DB             | Run background jobs (currently metrics demo) |
| Dependencies             | Config, DB pool, WebSocket hub, HTTP router  | Observability package only (today)           |
| Long-running loop        | `httpServer.ListenAndServe()` + hub goroutine| Metrics server goroutine, idle until signal  |
| Shutdown path            | Cancel hub, graceful HTTP shutdown, then close| Cancel context and exit (more to add later)  |
| Configuration loading    | Dedicated config structs per subsystem       | None yet (rely on defaults/env)              |
| Error handling           | Logs and exits on fatal issues (config/DB)   | Logs metrics init failure and exits          |

## Shared Patterns

- Both binaries use structured logging via `util.Info`/`util.Error`.
- Both register signal handlers (`SIGINT`, `SIGTERM`) for orderly shutdown.
- Both run supporting services (WebSocket hub, metrics server) on background goroutines.

## Alignment Opportunities

1. **Centralised config**: worker should consume the same `internal/config` package once it exists, instead of hard-coded timeouts.
2. **Dependency injection**: as worker gains responsibilities (indexer coordinators, RPC clients), mirror the server’s explicit construction pattern.
3. **Graceful cleanup hooks**: replace the placeholder `<-ctx.Done()` with actual teardown (e.g., stop coordinators, close DB pools) to match server shutdown.
4. **Shared observability wiring**: ensure both binaries initialise metrics/logging consistently (metrics server address, labels, etc.).

Keeping both entrypoints aligned makes it easier to run them side by side in production and simplifies onboarding for new contributors.
