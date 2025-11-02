# Blockchain Explorer API `main.go` Walkthrough

These notes explain the `cmd/api/main.go` file from the project in plain language for Go beginners.

## Startup Overview
- Logs that the API is starting so you can see it in stdout or your log aggregation.
- Loads API configuration (port, CORS, timeouts) with `api.NewConfig()`.
- Loads database configuration via `db.NewConfig()`; exiting on error because the API cannot run without a database connection.
- Caps the database connection pool at 10 so the API cannot overwhelm the shared database.
- Creates a database connection pool with `db.NewPool(ctx, dbConfig)` and defers `pool.Close()` to release connections when the program ends.

## WebSocket Hub
- `websocket.LoadConfig()` grabs WebSocket-specific settings like max client count.
- `websocket.NewHub()` builds the hub that will broadcast live blockchain updates.
- `context.WithCancel` produces `hubCtx` and `hubCancel`; the hub runs in its own goroutine with `go hub.Run(hubCtx)`.
- `defer hubCancel()` ensures the hub stops even if the program exits unexpectedly.

## HTTP Server Setup
- `api.NewServerWithHub(pool, apiConfig, hub)` wires HTTP routes to the database and WebSocket hub.
- Builds `http.Server` with address and timeout values taken from the API configuration. Read/Write/Idle timeouts protect the server from slow clients.
- Launches `httpServer.ListenAndServe()` inside a goroutine and records any error into the `serverErrors` channel. The main goroutine can continue watching for shutdown signals.

## Signal Handling (`sigChan`)
- `sigChan := make(chan os.Signal, 1)` creates a buffered channel that can hold one OS signal.
- `signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)` tells the runtime to push those signals into the channel. SIGINT is `Ctrl+C`; SIGTERM typically comes from process managers.
- A `select` waits for either a signal from `sigChan` or an error from `serverErrors`. When a signal arrives, the program logs which signal it received and moves into shutdown.
- Buffering the channel prevents the notifier from blocking if the program is busy when the signal arrives.

## Graceful Shutdown
- Logs that shutdown is starting and notes the configured timeout.
- Calls `hubCancel()` so the WebSocket hub stops accepting new messages and disconnects clients.
- Creates a timeout context: `shutdownCtx, cancel := context.WithTimeout(context.Background(), apiConfig.ShutdownTimeout)`. The timeout limits how long the HTTP server has to finish ongoing requests.
- Calls `httpServer.Shutdown(shutdownCtx)` to let active requests finish. If it times out or fails, falls back to `httpServer.Close()` which forcefully closes connections.
- Logs completion once everything has stopped.

## Go Concepts Highlighted
- **Goroutines**: `go hub.Run(...)` and the goroutine around `ListenAndServe()` run tasks concurrently.
- **Channels**: `serverErrors` and `sigChan` move information (errors or OS signals) between goroutines.
- **Context**: `context.Background()` is the root; `context.WithCancel/WithTimeout` allow coordinated cancellation and time limits.
- **Defer**: Used for cleanup (`pool.Close()`, `hubCancel()`, and `cancel()`) so resources are freed even if the function exits early.

These patterns (config loading, dependency setup, goroutines for background services, signal-driven graceful shutdown) are considered best practice for Go API entrypoints.
