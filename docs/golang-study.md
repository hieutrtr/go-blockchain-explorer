# Go Language Study Guide - Learn from Blockchain Explorer Project

**Based on:** Story 1.1 (Ethereum RPC Client with Retry Logic) + Story 1.2 (PostgreSQL Schema & Migrations)
**Project:** go-blockchain-explorer
**Date:** 2025-10-30 (Updated with Story 1.2 concepts)

This guide extracts Go concepts, patterns, and techniques from real production code in this project.

---

## Table of Contents

1. [Go Language Fundamentals](#go-language-fundamentals)
2. [Package Structure & Organization](#package-structure--organization)
3. [Error Handling Patterns](#error-handling-patterns)
4. [Context & Cancellation](#context--cancellation)
5. [Structured Logging](#structured-logging)
6. [Testing Techniques](#testing-techniques)
7. [Design Patterns](#design-patterns)
8. [Go Idioms & Best Practices](#go-idioms--best-practices)
9. [Standard Library Usage](#standard-library-usage)
10. [Third-Party Libraries](#third-party-libraries)
11. [Database Connection Patterns (Story 1.2)](#database-connection-patterns-story-12)
12. [Configuration Management & Validation](#configuration-management--validation)
13. [Database Migrations Strategy](#database-migrations-strategy)
14. [Connection Pooling & Resource Management](#connection-pooling--resource-management)

---

## Go Language Fundamentals

### 1. Struct Types

**Concept:** Structs are Go's way of creating custom data types with named fields.

**Example from Project:**
```go
// internal/rpc/config.go
type Config struct {
    RPCURL            string        // Exported field (capitalized)
    ConnectionTimeout time.Duration // time.Duration is built-in type
    RequestTimeout    time.Duration
    MaxRetries        int
    RetryBaseDelay    time.Duration
}
```

**Key Concepts:**
- **Exported vs Unexported:** Capitalized fields are exported (public), lowercase are unexported (private)
- **Embedding Types:** Can embed other structs or interfaces
- **Zero Values:** Uninitialized fields get zero values (0 for int, "" for string, nil for pointers)

**Another Example:**
```go
// internal/rpc/client.go
type Client struct {
    ethClient *ethclient.Client  // Pointer to external library type
    config    *Config            // Pointer to our own Config struct
    logger    *slog.Logger       // Standard library logger
}
```

### 2. Methods (Receiver Functions)

**Concept:** Functions attached to types (like methods in OOP).

**Value Receiver Example:**
```go
// internal/db/config.go
func (c *Config) SafeString() string {
    return fmt.Sprintf(
        "postgres://%s:***@%s:%d/%s (maxConns=%d)",
        c.User,
        c.Host,
        c.Port,
        c.Name,
        c.MaxConns,
    )
}
```

**Pointer Receiver Example:**
```go
// internal/rpc/client.go
func (c *Client) GetBlockByNumber(ctx context.Context, height uint64) (*types.Block, error) {
    // Implementation...
}
```

**Key Concepts:**
- **Pointer Receiver (`*Client`):** Can modify the struct, more efficient for large structs
- **Value Receiver (`Config`):** Works on a copy, cannot modify original
- **Rule of Thumb:** Use pointer receivers by default unless you have a good reason not to

### 3. Error Handling with Multiple Return Values

**Concept:** Go functions commonly return (result, error) tuples.

**Example:**
```go
func NewConfig() (*Config, error) {
    rpcURL := os.Getenv("RPC_URL")
    if rpcURL == "" {
        return nil, fmt.Errorf("RPC_URL environment variable not set")
    }

    return &Config{
        RPCURL:            rpcURL,
        ConnectionTimeout: 10 * time.Second,
        RequestTimeout:    30 * time.Second,
        MaxRetries:        5,
        RetryBaseDelay:    1 * time.Second,
    }, nil
}
```

**Key Concepts:**
- **Convention:** Last return value is `error`
- **Nil Error:** `nil` means success
- **Check Immediately:** Always check errors: `if err != nil { return nil, err }`

### 4. Interfaces

**Concept:** Interfaces define behavior (methods) without implementation.

**Implicit Implementation:**
```go
// From net package (stdlib)
type Error interface {
    error  // Embeds error interface
    Timeout() bool   // Was this a timeout?
    Temporary() bool // Was this temporary?
}

// No explicit "implements" keyword needed!
// If a type has these methods, it satisfies the interface
```

**Usage in Project:**
```go
// internal/rpc/errors.go
func classifyError(err error) ErrorType {
    // Type assertion to check if error implements net.Error
    if netErr, ok := err.(net.Error); ok {
        if netErr.Timeout() {
            return ErrorTypeTimeout
        }
        if netErr.Temporary() {
            return ErrorTypeTransient
        }
    }
    // ...
}
```

**Key Concepts:**
- **Duck Typing:** If it walks like a duck and quacks like a duck...
- **Empty Interface:** `interface{}` or `any` accepts any type
- **Interface Embedding:** Interfaces can embed other interfaces

### 5. Type Assertions and Type Switches

**Type Assertion:**
```go
// internal/rpc/errors.go
if netErr, ok := err.(net.Error); ok {
    // netErr is now of type net.Error
    // ok is true if assertion succeeded
}
```

**Type Switch:**
```go
switch v := interface{}(value).(type) {
case string:
    // v is string
case int:
    // v is int
default:
    // v is original type
}
```

### 6. Constants and Enums

**String Constants as Enum:**
```go
// internal/rpc/errors.go
type ErrorType string

const (
    ErrorTypeTransient  ErrorType = "transient"   // Retryable errors
    ErrorTypePermanent  ErrorType = "permanent"   // Don't retry
    ErrorTypeRateLimit  ErrorType = "rate_limit"  // Backoff and retry
    ErrorTypeTimeout    ErrorType = "timeout"
    ErrorTypeCanceled   ErrorType = "canceled"
    ErrorTypeNetwork    ErrorType = "network"
    ErrorTypeNotFound   ErrorType = "not_found"
    ErrorTypeInvalidInput ErrorType = "invalid_input"
)
```

**Key Concepts:**
- **Typed Constants:** `ErrorType` is not just a string, it's a distinct type
- **Const Block:** Parentheses create a constant block
- **Method on Const Type:** Can add methods to make it more useful

```go
func (et ErrorType) String() string {
    return string(et)
}

func (et ErrorType) IsRetryable() bool {
    return et == ErrorTypeTransient ||
           et == ErrorTypeTimeout ||
           et == ErrorTypeNetwork ||
           et == ErrorTypeRateLimit
}
```

---

## Package Structure & Organization

### 1. Internal Package Pattern

**Project Structure:**
```
go-blockchain-explorer/
├── internal/           # Private packages (can't be imported by external projects)
│   ├── rpc/           # RPC client package
│   │   ├── client.go
│   │   ├── config.go
│   │   ├── errors.go
│   │   ├── retry.go
│   │   ├── client_test.go
│   │   ├── errors_test.go
│   │   └── retry_test.go
│   └── db/            # Database package
│       ├── config.go
│       ├── connection.go
│       └── migrations.go
├── cmd/               # Command-line applications
│   └── indexer/
│       └── main.go
└── go.mod             # Module definition
```

**Key Concepts:**
- **`internal/` Directory:** Code in `internal/` can only be imported by code in the same module
- **Package per Concern:** Each major concern gets its own package (`rpc`, `db`, etc.)
- **Flat Package Structure:** Avoid deep nesting (bad: `internal/rpc/client/impl/v1/`)

### 2. Package Declaration and Imports

**Example:**
```go
// internal/rpc/client.go
package rpc  // Package name matches directory name

import (
    "context"     // Standard library imports first
    "fmt"
    "log/slog"
    "math/big"
    "time"

    "github.com/ethereum/go-ethereum/common"     // External imports after
    "github.com/ethereum/go-ethereum/core/types"
    "github.com/ethereum/go-ethereum/ethclient"
)
```

**Import Organization:**
1. Standard library imports
2. Blank line
3. External library imports
4. Blank line (if any)
5. Internal imports from your project

### 3. Exported vs Unexported Naming

**Capitalization Determines Visibility:**
```go
// Exported (public) - can be used outside package
type Client struct { }
func NewClient() *Client { }

// Unexported (private) - only visible within package
func classifyError(err error) ErrorType { }
func calculateBackoff(attempt int) time.Duration { }
```

**Convention:** Use constructors (factory functions) for exported types:
```go
// Good: Constructor function
func NewClient(config *Config, logger *slog.Logger) (*Client, error) {
    // Validation and setup
    return &Client{...}, nil
}

// User creates client like this:
client, err := rpc.NewClient(config, logger)
```

---

## Error Handling Patterns

### 1. Error Wrapping with fmt.Errorf

**Basic Error Creation:**
```go
if rpcURL == "" {
    return nil, fmt.Errorf("RPC_URL environment variable not set")
}
```

**Error Wrapping (Preserves Original Error):**
```go
// internal/rpc/client.go
block, err := c.ethClient.BlockByNumber(ctx, big.NewInt(int64(height)))
if err != nil {
    return nil, fmt.Errorf("failed to fetch block %d: %w", height, err)
    //                                                       ^^
    //                                         %w preserves original error
}
```

**Key Concepts:**
- **`%w` vs `%v`:** `%w` wraps error (can unwrap later), `%v` just formats it
- **Error Chain:** Can unwrap to access original error: `errors.Unwrap(err)`

### 2. Custom Error Types

**Creating Custom Error:**
```go
// internal/rpc/errors.go
type RPCError struct {
    ErrorType ErrorType
    Err       error
}

func (e *RPCError) Error() string {
    return fmt.Sprintf("%s: %v", e.ErrorType, e.Err)
}

func (e *RPCError) Unwrap() error {
    return e.Err
}
```

**Key Concepts:**
- **`Error() string` Method:** Required to satisfy `error` interface
- **`Unwrap() error` Method:** Allows `errors.Unwrap()` and `errors.Is()` to work

### 3. Error Checking Patterns

**Pattern 1: errors.Is (Check for specific error)**
```go
if errors.Is(err, context.DeadlineExceeded) {
    return ErrorTypeTimeout
}
```

**Pattern 2: errors.As (Type assertion with unwrapping)**
```go
var opErr *net.OpError
if errors.As(err, &opErr) {
    // opErr is now *net.OpError
    if opErr.Err == syscall.ECONNREFUSED {
        return ErrorTypeNetwork
    }
}
```

**Pattern 3: Type Assertion**
```go
if netErr, ok := err.(net.Error); ok {
    if netErr.Timeout() {
        return ErrorTypeTimeout
    }
}
```

### 4. Error Classification

**From Project:**
```go
// internal/rpc/errors.go
func classifyError(err error) ErrorType {
    if err == nil {
        return ""
    }

    errStr := err.Error()

    // Check for rate limiting
    if strings.Contains(errStr, "429") || strings.Contains(errStr, "rate limit") {
        return ErrorTypeRateLimit
    }

    // Check for timeout
    if errors.Is(err, context.DeadlineExceeded) {
        return ErrorTypeTimeout
    }

    // Check if error implements net.Error interface
    if netErr, ok := err.(net.Error); ok {
        if netErr.Timeout() {
            return ErrorTypeTimeout
        }
        if netErr.Temporary() {
            return ErrorTypeTransient
        }
    }

    // Default to transient (safer to retry)
    return ErrorTypeTransient
}
```

**Key Pattern:** Use multiple strategies to classify errors:
1. Check for specific sentinel errors (`errors.Is`)
2. Check for specific types (`errors.As`, type assertion)
3. Parse error strings (last resort)
4. Default to safe behavior (transient = retry)

---

## Context & Cancellation

### 1. Context Basics

**What is Context?**
- Carries deadlines, cancellation signals, and request-scoped values
- Passed as first parameter to functions (convention)
- Never stored in structs

**Common Context Types:**
```go
// Background context (never canceled)
ctx := context.Background()

// Context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel() // Always call cancel to free resources

// Context with deadline
deadline := time.Now().Add(1 * time.Minute)
ctx, cancel := context.WithDeadline(context.Background(), deadline)
defer cancel()

// Context with manual cancellation
ctx, cancel := context.WithCancel(context.Background())
defer cancel()
```

### 2. Using Context in Functions

**Example from Project:**
```go
// internal/rpc/client.go
func (c *Client) GetBlockByNumber(ctx context.Context, height uint64) (*types.Block, error) {
    // Validate input
    if height == 0 {
        return nil, fmt.Errorf("invalid block height: %d", height)
    }

    c.logger.Info("fetching block",
        slog.Uint64("block_height", height))

    // Create timeout context
    ctx, cancel := context.WithTimeout(ctx, c.config.RequestTimeout)
    defer cancel()

    // Call with retry
    var block *types.Block
    err := retryWithBackoff(ctx, c.config, c.logger, func() error {
        var fetchErr error
        block, fetchErr = c.ethClient.BlockByNumber(ctx, big.NewInt(int64(height)))
        return fetchErr
    })

    return block, err
}
```

**Key Concepts:**
- **First Parameter:** Context is always first parameter (convention)
- **Pass Context Down:** Pass context to nested function calls
- **Derive New Context:** Use `WithTimeout` to add timeout to existing context
- **Defer Cancel:** Always `defer cancel()` to prevent leaks

### 3. Checking for Cancellation

**Example from Project:**
```go
// internal/rpc/retry.go
func retryWithBackoff(
    ctx context.Context,
    config *Config,
    logger *slog.Logger,
    operation func() error,
) error {
    for attempt := 0; attempt < config.MaxRetries; attempt++ {
        // Try operation
        err := operation()
        if err == nil {
            return nil // Success!
        }

        // Check if context was canceled
        if ctx.Err() != nil {
            return fmt.Errorf("context canceled during retry: %w", ctx.Err())
        }

        // Calculate backoff delay
        delay := calculateBackoff(attempt, config)

        // Wait with cancellation support
        select {
        case <-time.After(delay):
            // Delay completed, continue to next retry
        case <-ctx.Done():
            // Context was canceled during wait
            return fmt.Errorf("context canceled during backoff: %w", ctx.Err())
        }
    }

    return fmt.Errorf("max retries exceeded")
}
```

**Key Concepts:**
- **`ctx.Err()`:** Returns error if context is done (canceled or timed out)
- **`ctx.Done()`:** Returns channel that closes when context is done
- **`select` Statement:** Wait on multiple channels, proceeds when one is ready

### 4. Context Values (Rare, Use Sparingly)

```go
// Add value to context
ctx = context.WithValue(ctx, "requestID", "abc-123")

// Retrieve value from context
if requestID, ok := ctx.Value("requestID").(string); ok {
    // Use requestID
}
```

**Warning:** Don't abuse context for passing function parameters!
**Use for:** Request IDs, authentication tokens, tracing data

---

## Structured Logging

### 1. log/slog (Go 1.21+)

**Initialization:**
```go
// internal/rpc/client.go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))
```

**Key Concepts:**
- **Handler:** Formats and outputs logs (JSONHandler, TextHandler)
- **Level:** DEBUG, INFO, WARN, ERROR
- **Attributes:** Key-value pairs attached to log messages

### 2. Structured Logging Examples

**Basic Logging:**
```go
logger.Info("fetching block",
    slog.Uint64("block_height", height))
```

**With Multiple Attributes:**
```go
logger.Error("failed to fetch block",
    slog.Uint64("block_height", height),
    slog.String("error_type", string(errorType)),
    slog.Int("retry_attempt", attempt),
    slog.Int64("duration_ms", duration.Milliseconds()),
    slog.Any("error", err))
```

**Output (JSON):**
```json
{
  "time": "2025-10-30T12:00:00Z",
  "level": "ERROR",
  "msg": "failed to fetch block",
  "block_height": 5000000,
  "error_type": "network",
  "retry_attempt": 2,
  "duration_ms": 1234,
  "error": "connection refused"
}
```

### 3. Log Levels

```go
logger.Debug("detailed debug info")     // Development only
logger.Info("normal operation")         // General info
logger.Warn("unusual but handled")      // Warnings
logger.Error("operation failed")        // Errors
```

### 4. Logger Attributes Pattern

**Create logger with permanent attributes:**
```go
// Create base logger
baseLogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

// Add permanent attributes
logger := baseLogger.With(
    slog.String("component", "rpc-client"),
    slog.String("version", "1.0.0"),
)

// All logs from this logger include component and version
logger.Info("started")
// Output: {"time":"...","level":"INFO","msg":"started","component":"rpc-client","version":"1.0.0"}
```

---

## Testing Techniques

### 1. Table-Driven Tests

**Example:**
```go
// internal/rpc/errors_test.go
func TestClassifyError(t *testing.T) {
    tests := []struct {
        name     string
        err      error
        expected ErrorType
    }{
        {
            name:     "timeout error",
            err:      context.DeadlineExceeded,
            expected: ErrorTypeTimeout,
        },
        {
            name:     "rate limit in message",
            err:      errors.New("rate limit exceeded"),
            expected: ErrorTypeRateLimit,
        },
        {
            name:     "nil error",
            err:      nil,
            expected: "",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := classifyError(tt.err)
            if result != tt.expected {
                t.Errorf("classifyError() = %v, want %v", result, tt.expected)
            }
        })
    }
}
```

**Key Concepts:**
- **Table Struct:** Slice of anonymous structs with test cases
- **`t.Run()`:** Creates subtests for each case
- **`range`:** Iterate over test cases
- **Benefits:** Easy to add new test cases, clear documentation of behavior

### 2. testify Assertions

**Instead of:**
```go
if result != expected {
    t.Errorf("got %v, want %v", result, expected)
}
```

**Use testify:**
```go
import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestSomething(t *testing.T) {
    result := doSomething()

    // assert: continues test even if fails
    assert.Equal(t, expected, result)
    assert.NotNil(t, result)
    assert.True(t, result > 0)

    // require: stops test immediately if fails
    require.NoError(t, err)  // If err != nil, test stops here
    require.NotNil(t, client)
}
```

### 3. Integration Test Pattern

**Example from Project:**
```go
// internal/db/connection_test.go
func TestNewPool_Integration(t *testing.T) {
    // Skip in short mode (go test -short)
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Requires external service (PostgreSQL)
    config, err := NewConfig()
    if err != nil {
        t.Skipf("skipping test: database configuration not available: %v", err)
    }

    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    ctx := context.Background()

    pool, err := NewPool(ctx, config, logger)
    require.NoError(t, err)
    require.NotNil(t, pool)
    defer pool.Close()

    // Test actual database operation
    var result int
    err = pool.QueryRow(ctx, "SELECT 1").Scan(&result)
    require.NoError(t, err)
    assert.Equal(t, 1, result)
}
```

**Key Patterns:**
- **`testing.Short()`:** Skip expensive tests with `go test -short`
- **`t.Skip()`:** Skip test with reason
- **`t.Skipf()`:** Skip with formatted message
- **`defer cleanup()`:** Always clean up resources
- **Separate Files:** Can use `*_integration_test.go` naming

### 4. Benchmarking

```go
func BenchmarkCalculateBackoff(b *testing.B) {
    config := &RetryConfig{RetryBaseDelay: 1 * time.Second}

    // b.N is automatically adjusted by testing framework
    for i := 0; i < b.N; i++ {
        calculateBackoff(3, config)
    }
}
```

**Run benchmarks:**
```bash
go test -bench=. -benchmem
```

### 5. Test Helpers

**Example:**
```go
// Test helper function
func setupTestDB(t *testing.T) *sql.DB {
    t.Helper() // Marks this as helper function for better error messages

    db, err := sql.Open("postgres", testDBURL)
    require.NoError(t, err)

    // Register cleanup
    t.Cleanup(func() {
        db.Close()
    })

    return db
}

func TestSomething(t *testing.T) {
    db := setupTestDB(t) // If this fails, error points to TestSomething, not setupTestDB
    // Use db...
}
```

---

## Design Patterns

### 1. Constructor Pattern (Factory Function)

**Pattern:**
```go
// Don't allow direct struct creation
type Client struct {
    ethClient *ethclient.Client
    config    *Config
    logger    *slog.Logger
}

// Provide constructor with validation
func NewClient(config *Config, logger *slog.Logger) (*Client, error) {
    if config == nil {
        return nil, fmt.Errorf("config cannot be nil")
    }
    if logger == nil {
        return nil, fmt.Errorf("logger cannot be nil")
    }

    ctx, cancel := context.WithTimeout(context.Background(), config.ConnectionTimeout)
    defer cancel()

    ethClient, err := ethclient.DialContext(ctx, config.RPCURL)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to RPC: %w", err)
    }

    return &Client{
        ethClient: ethClient,
        config:    config,
        logger:    logger,
    }, nil
}
```

**Benefits:**
- Validates parameters
- Performs complex initialization
- Can return error if setup fails
- Enforces proper construction

### 2. Functional Options Pattern

**Not used in Story 1.1, but common in Go:**

```go
type ClientOption func(*Client)

func WithTimeout(timeout time.Duration) ClientOption {
    return func(c *Client) {
        c.config.RequestTimeout = timeout
    }
}

func WithLogger(logger *slog.Logger) ClientOption {
    return func(c *Client) {
        c.logger = logger
    }
}

func NewClient(rpcURL string, opts ...ClientOption) (*Client, error) {
    c := &Client{
        config: DefaultConfig(rpcURL),
        logger: slog.Default(),
    }

    for _, opt := range opts {
        opt(c)
    }

    return c, nil
}

// Usage:
client, err := NewClient(
    "https://...",
    WithTimeout(60*time.Second),
    WithLogger(myLogger),
)
```

### 3. Closure Pattern (for Retry Logic)

**Example from Project:**
```go
// internal/rpc/retry.go
func retryWithBackoff(
    ctx context.Context,
    config *Config,
    logger *slog.Logger,
    operation func() error,  // ← Closure: caller defines the operation
) error {
    for attempt := 0; attempt < config.MaxRetries; attempt++ {
        err := operation()  // Execute the provided operation
        if err == nil {
            return nil
        }
        // Retry logic...
    }
    return fmt.Errorf("max retries exceeded")
}

// Usage in Client:
err := retryWithBackoff(ctx, c.config, c.logger, func() error {
    var fetchErr error
    block, fetchErr = c.ethClient.BlockByNumber(ctx, big.NewInt(int64(height)))
    return fetchErr
})
```

**Benefits:**
- Separates retry logic from business logic
- Reusable across different operations
- Captures variables from surrounding scope (closure)

### 4. Interface Segregation

**Small, focused interfaces:**
```go
// Good: Small interface
type Logger interface {
    Info(msg string, args ...any)
    Error(msg string, args ...any)
}

// Bad: Large interface (forces implementations to provide unused methods)
type Logger interface {
    Info(...)
    Debug(...)
    Warn(...)
    Error(...)
    Fatal(...)
    Panic(...)
    WithFields(...) Logger
    SetLevel(...)
    // ... 10 more methods
}
```

**Project uses this implicitly:**
- `error` interface: Just one method `Error() string`
- `net.Error` interface: Only `Timeout() bool` and `Temporary() bool`

---

## Go Idioms & Best Practices

### 1. Early Return Pattern

**Good (Early Return):**
```go
func DoSomething(input string) error {
    if input == "" {
        return fmt.Errorf("input is empty")
    }

    if len(input) > 100 {
        return fmt.Errorf("input too long")
    }

    // Happy path is not nested
    result := process(input)
    return nil
}
```

**Bad (Nested Ifs):**
```go
func DoSomething(input string) error {
    if input != "" {
        if len(input) <= 100 {
            result := process(input)
            return nil
        } else {
            return fmt.Errorf("input too long")
        }
    } else {
        return fmt.Errorf("input is empty")
    }
}
```

### 2. Check Error Immediately

**Good:**
```go
result, err := doSomething()
if err != nil {
    return nil, fmt.Errorf("operation failed: %w", err)
}
// Use result
```

**Bad:**
```go
result, err := doSomething()
result2, err2 := doSomethingElse()
if err != nil {  // Too late! might have used invalid result
    return nil, err
}
```

### 3. Defer for Cleanup

**Pattern:**
```go
func ProcessFile(filename string) error {
    file, err := os.Open(filename)
    if err != nil {
        return err
    }
    defer file.Close()  // Guaranteed to run when function exits

    // Work with file...
    // Even if panic occurs, file.Close() will be called
}
```

**Multiple Defers (Execute in Reverse Order):**
```go
func DoMultipleThings() {
    defer fmt.Println("Third")  // Runs third
    defer fmt.Println("Second") // Runs second
    defer fmt.Println("First")  // Runs first
    fmt.Println("Function body")
}
// Output:
// Function body
// First
// Second
// Third
```

### 4. Don't Ignore Errors

**Bad:**
```go
_ = file.Close()  // Ignoring potential error
```

**Good:**
```go
if err := file.Close(); err != nil {
    logger.Error("failed to close file", slog.Any("error", err))
}
```

**When to ignore:**
- Truly don't care (rare)
- Already handled in defer with named return value

### 5. Accept Interfaces, Return Structs

**Good:**
```go
// Accept interface (flexible)
func ProcessData(logger Logger) error {
    logger.Info("processing")
    // ...
}

// Return concrete type (clear)
func NewClient() (*Client, error) {
    return &Client{}, nil
}
```

### 6. Avoid Pointer to Interface

**Bad:**
```go
func DoSomething(logger *slog.Logger) error {  // ← *slog.Logger
    // ...
}
```

**Good:**
```go
// Interfaces are already references under the hood
func DoSomething(logger slog.Logger) error {  // ← Just slog.Logger
    // ...
}
```

**Exception:** When you need to modify the interface itself (very rare)

---

## Standard Library Usage

### 1. fmt Package (Formatting)

```go
import "fmt"

// String formatting
fmt.Sprintf("block %d", height)              // "block 5000000"
fmt.Sprintf("error: %v", err)                // "error: connection refused"
fmt.Sprintf("wrapped: %w", err)              // Wraps error (unwrappable)
fmt.Sprintf("%+v", structValue)              // Includes field names
fmt.Sprintf("%#v", structValue)              // Go syntax representation

// Printing
fmt.Println("hello")                         // Print with newline
fmt.Printf("block %d\n", height)             // Formatted print

// Error creation
fmt.Errorf("failed: %w", originalErr)        // Create wrapped error
```

### 2. strings Package

```go
import "strings"

// From project:
if strings.Contains(errStr, "429") {         // Check substring
    return ErrorTypeRateLimit
}

// Other useful functions:
strings.ToLower("HELLO")                     // "hello"
strings.HasPrefix("hello", "he")             // true
strings.HasSuffix("hello", "lo")             // true
strings.Split("a,b,c", ",")                  // ["a", "b", "c"]
strings.Join([]string{"a", "b"}, ",")        // "a,b"
strings.Trim("  hello  ", " ")               // "hello"
```

### 3. time Package

```go
import "time"

// Durations
timeout := 30 * time.Second
delay := 5 * time.Minute

// Sleep
time.Sleep(1 * time.Second)

// Time operations
now := time.Now()
later := now.Add(5 * time.Minute)
duration := later.Sub(now)

// Formatting
now.Format(time.RFC3339)                     // "2025-10-30T12:00:00Z"
now.Format("2006-01-02 15:04:05")            // Custom format

// Timers
timer := time.NewTimer(5 * time.Second)
<-timer.C  // Block until timer fires

// From project (exponential backoff):
delay := time.Duration(math.Pow(2, float64(attempt))) * time.Second
```

### 4. os Package

```go
import "os"

// Environment variables (from project):
rpcURL := os.Getenv("RPC_URL")
if rpcURL == "" {
    return fmt.Errorf("RPC_URL not set")
}

// File operations:
file, err := os.Open("file.txt")
os.Create("file.txt")
os.WriteFile("file.txt", []byte("content"), 0644)
data, err := os.ReadFile("file.txt")

// Standard streams:
os.Stdout  // Standard output
os.Stderr  // Standard error
os.Stdin   // Standard input
```

### 5. errors Package

```go
import "errors"

// Create error
err := errors.New("something went wrong")

// Check for specific error
if errors.Is(err, context.DeadlineExceeded) {
    // err is or wraps context.DeadlineExceeded
}

// Type assertion with unwrapping
var opErr *net.OpError
if errors.As(err, &opErr) {
    // opErr is now *net.OpError
}

// Unwrap error
originalErr := errors.Unwrap(err)
```

---

## Third-Party Libraries

### 1. go-ethereum (Ethereum Client)

**Installation:**
```bash
go get github.com/ethereum/go-ethereum@v1.16.5
```

**Usage in Project:**
```go
import (
    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/core/types"
    "github.com/ethereum/go-ethereum/ethclient"
)

// Connect to Ethereum node
client, err := ethclient.Dial("https://eth-sepolia.g.alchemy.com/v2/API_KEY")

// Fetch block
blockNumber := big.NewInt(5000000)
block, err := client.BlockByNumber(ctx, blockNumber)

// Access block data
height := block.Number().Uint64()
hash := block.Hash()
transactions := block.Transactions()

// Fetch transaction receipt
txHash := common.HexToHash("0x123...")
receipt, err := client.TransactionReceipt(ctx, txHash)
```

### 2. testify (Testing Library)

**Installation:**
```bash
go get github.com/stretchr/testify
```

**Usage:**
```go
import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestSomething(t *testing.T) {
    // assert: continues test on failure
    assert.Equal(t, expected, actual)
    assert.NotNil(t, value)
    assert.True(t, condition)
    assert.NoError(t, err)
    assert.Contains(t, slice, element)

    // require: stops test on failure
    require.NoError(t, err)
    require.NotNil(t, client)
}
```

### 3. pgx (PostgreSQL Driver)

**Installation:**
```bash
go get github.com/jackc/pgx/v5
```

**Usage (from Story 1.2):**
```go
import (
    "github.com/jackc/pgx/v5/pgxpool"
)

// Create connection pool
poolConfig, err := pgxpool.ParseConfig(connectionString)
pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
defer pool.Close()

// Query
var result int
err = pool.QueryRow(ctx, "SELECT 1").Scan(&result)

// Execute
_, err = pool.Exec(ctx, "INSERT INTO blocks (...) VALUES (...)")
```

### 4. golang-migrate (Database Migrations)

**Installation:**
```bash
go get github.com/golang-migrate/migrate/v4
```

**Usage (from Story 1.2):**
```go
import (
    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
)

m, err := migrate.New(
    "file://migrations",
    "postgres://user:pass@localhost/dbname",
)

// Run migrations
err = m.Up()

// Check current version
version, dirty, err := m.Version()
```

---

## Advanced Concepts (Brief Introduction)

### 1. Goroutines (Not in Story 1.1, but essential)

```go
// Start goroutine
go func() {
    // Runs concurrently
    fmt.Println("hello from goroutine")
}()

// Start goroutine with parameters
go processBlock(blockNumber)
```

### 2. Channels (Not in Story 1.1, but essential)

```go
// Create channel
ch := make(chan int)

// Send to channel (blocks until received)
ch <- 42

// Receive from channel (blocks until sent)
value := <-ch

// Buffered channel (doesn't block until full)
ch := make(chan int, 10)
```

### 3. Select Statement (Used for Context)

```go
select {
case <-time.After(5 * time.Second):
    fmt.Println("timeout")
case <-ctx.Done():
    fmt.Println("canceled")
case result := <-resultChan:
    fmt.Println("got result:", result)
}
```

### 4. Sync Package (Not in Story 1.1)

```go
// Wait for goroutines
var wg sync.WaitGroup
wg.Add(1)
go func() {
    defer wg.Done()
    // Do work
}()
wg.Wait()  // Block until all Done() called

// Mutex for shared state
var mu sync.Mutex
mu.Lock()
// Critical section
mu.Unlock()
```

---

## Go Commands Reference

```bash
# Initialize module
go mod init github.com/user/project

# Add dependency
go get github.com/some/package

# Update dependencies
go get -u ./...

# Tidy dependencies (remove unused)
go mod tidy

# Run tests
go test ./...
go test -v ./...              # Verbose
go test -short ./...          # Skip slow tests
go test -run TestName ./...   # Run specific test
go test -cover ./...          # With coverage
go test -bench=. ./...        # Run benchmarks

# Build
go build ./cmd/myapp

# Run
go run ./cmd/myapp

# Format code
go fmt ./...

# Check for issues
go vet ./...

# View documentation
go doc fmt.Sprintf
```

---

---

## Database Connection Patterns (Story 1.2)

### 1. Struct Composition for Resource Wrapping

**Concept:** Wrap external library types in your own structs to add methods and control the interface.

**Example from Story 1.2:**
```go
// internal/db/connection.go
type Pool struct {
    *pgxpool.Pool     // Embed external Pool type (anonymous field)
    config *Config    // Add your own fields
}

// NewPool creates a new database connection pool
func NewPool(ctx context.Context, config *Config) (*Pool, error) {
    // ... setup code ...

    pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to create connection pool: %w", err)
    }

    return &Pool{
        Pool:   pool,
        config: config,
    }, nil
}
```

**Key Concepts:**
- **Anonymous Field Embedding:** `*pgxpool.Pool` is embedded without a field name
- **Method Promotion:** Methods from embedded `*pgxpool.Pool` are automatically available on `Pool`
- **Extended Functionality:** Can add custom methods like `Close()` and `HealthCheck()` that wrap or extend embedded methods
- **Encapsulation:** Hide implementation details while exposing necessary API

**Advantage over Inheritance (which Go doesn't have):**
```go
// You can call methods from pgxpool directly:
pool.Ping(ctx)           // Promoted method from pgxpool.Pool
pool.QueryRow(ctx, sql)  // Promoted method

// But you can also override behavior:
func (p *Pool) Close() {
    if p.Pool != nil {
        util.Info("closing database connection pool")
        p.Pool.Close()  // Call embedded method
    }
}
```

### 2. Configuration Constructor with Validation

**Concept:** Create constructors that read environment variables, validate, and provide defaults.

**Example from Story 1.2:**
```go
// internal/db/config.go
func NewConfig() (*Config, error) {
    // Check required fields
    name := os.Getenv("DB_NAME")
    if name == "" {
        return nil, fmt.Errorf("DB_NAME environment variable not set")
    }

    // Parse with validation
    port := 5432
    if portStr := os.Getenv("DB_PORT"); portStr != "" {
        parsedPort, err := strconv.Atoi(portStr)
        if err != nil {
            return nil, fmt.Errorf("invalid DB_PORT value: %w", err)
        }
        if parsedPort < 1 || parsedPort > 65535 {
            return nil, fmt.Errorf("DB_PORT must be between 1 and 65535, got %d", parsedPort)
        }
        port = parsedPort
    }

    return &Config{...}, nil
}
```

**Key Patterns:**
- **Required vs Optional Fields:** Distinguish between must-have and optional fields with defaults
- **Type Conversion:** Use `strconv` package to convert string environment variables to typed values
- **Validation in Constructor:** Fail early with clear error messages
- **Error Wrapping:** Use `%w` to preserve original errors from parsing

---

## Configuration Management & Validation

### 1. Safe String Methods for Logging

**Problem:** Configuration often contains sensitive data (passwords, API keys) that shouldn't appear in logs.

**Solution from Story 1.2:**
```go
type Config struct {
    User     string
    Password string
    Host     string
    Port     int
    Name     string
    MaxConns int
}

// ConnectionString includes password (for actual use)
func (c *Config) ConnectionString() string {
    return fmt.Sprintf(
        "postgres://%s:%s@%s:%d/%s?sslmode=disable",
        c.User,
        c.Password,  // ⚠️ Contains secret
        c.Host,
        c.Port,
        c.Name,
    )
}

// SafeString masks password (safe for logging)
func (c *Config) SafeString() string {
    return fmt.Sprintf(
        "postgres://%s:***@%s:%d/%s (maxConns=%d)",
        c.User,
        c.Host,       // No password
        c.Port,
        c.Name,
        c.MaxConns,
    )
}

// Usage in code:
util.Info("connecting to database", "config", config.SafeString())  // ✅ Safe
```

**Key Concepts:**
- **Dual Methods:** Provide both secure and unsecured string representations
- **Clear Intent:** Name shows whether it's safe (`SafeString()`) or contains secrets
- **Always Use Safe Version:** Always log with `SafeString()` unless you specifically need sensitive data in logs

### 2. Type Conversion with Error Handling

**Pattern for converting environment variable strings:**
```go
// Pattern: strconv for single value conversion
maxConns := 20  // Default
if maxConnsStr := os.Getenv("DB_MAX_CONNS"); maxConnsStr != "" {
    parsedMaxConns, err := strconv.Atoi(maxConnsStr)
    if err != nil {
        return nil, fmt.Errorf("invalid DB_MAX_CONNS value: %w", err)
    }
    // Validate range
    if parsedMaxConns < 1 {
        return nil, fmt.Errorf("DB_MAX_CONNS must be at least 1, got %d", parsedMaxConns)
    }
    maxConns = parsedMaxConns
}
```

**Common strconv functions:**
```go
strconv.Atoi(str)           // Convert to int, returns error if invalid
strconv.ParseInt(str, 10, 64) // Parse with base and bit size
strconv.ParseFloat(str, 64)  // Parse float64
strconv.ParseBool(str)       // Parse boolean
```

---

## Database Migrations Strategy

### 1. The Defer + Close Pattern for Resources

**Concept:** Resources that need cleanup use `defer` to guarantee cleanup happens.

**Example from Story 1.2:**
```go
// internal/db/migrations.go
func RunMigrations(config *Config, migrationsPath string) error {
    // Create migrate instance
    m, err := migrate.New(
        fmt.Sprintf("file://%s", migrationsPath),
        connString,
    )
    if err != nil {
        return fmt.Errorf("failed to create migrate instance: %w", err)
    }
    defer m.Close()  // ← Guarantees cleanup even if error occurs

    // Run migrations
    if err := m.Up(); err != nil {
        // Error occurs, but defer still runs m.Close()
        return fmt.Errorf("failed to run migrations: %w", err)
    }

    return nil  // No error, defer still runs m.Close()
}
```

**Why defer is better than try-catch:**
- Works with panic (try-catch would not)
- Cleanup happens in reverse order if multiple defers
- Simpler syntax than try-finally in other languages
- Impossible to forget if you defer immediately after acquiring resource

### 2. Checking for No-Change Errors

**Pattern: Some operations have special "no-change" outcomes that aren't failures.**

```go
// internal/db/migrations.go
if err := m.Up(); err != nil {
    // Special case: no migrations to apply is not an error
    if errors.Is(err, migrate.ErrNoChange) {
        util.Info("database schema is up to date, no migrations needed")
        return nil  // Success: nothing needed to be done
    }
    return fmt.Errorf("failed to run migrations: %w", err)
}
```

**Key Pattern:**
- Check for sentinel errors using `errors.Is()`
- "No change" is often a valid outcome, not an error condition
- Log at INFO level (not ERROR) for these non-error states

### 3. Getting State After Operations

**Pattern: Query state after successful operation to confirm and log.**

```go
// After successful migration, get version info
version, dirty, err := m.Version()
if err != nil {
    // Log as WARN (non-critical info)
    util.Warn("failed to get migration version after successful migration",
        "error", err.Error())
} else {
    // Log successful state
    util.Info("migrations completed successfully",
        "version", version,
        "dirty", dirty)
}
```

**Key Concepts:**
- **Dirty Flag:** Indicates incomplete migration (should be false after successful run)
- **Version Number:** Track which migration version is active
- **Non-Critical Errors:** If getting version fails after successful operation, log as WARN not ERROR

---

## Connection Pooling & Resource Management

### 1. Configuring Connection Pool Parameters

**Concept from Story 1.2:**
```go
// Configure connection pool settings
poolConfig.MaxConns = int32(config.MaxConns)              // Max concurrent connections
poolConfig.MaxConnIdleTime = config.IdleTimeout           // 5 minutes
poolConfig.MaxConnLifetime = config.ConnLifetime          // 30 minutes
poolConfig.HealthCheckPeriod = 1 * config.ConnTimeout   // Periodic health checks
```

**Key Parameters Explained:**
- **MaxConns:** Maximum number of connections allowed (limits concurrent queries)
- **MaxConnIdleTime:** How long a connection can sit unused before being closed
- **MaxConnLifetime:** Maximum time a connection can exist (prevents stale connections)
- **HealthCheckPeriod:** How often pool checks if connections are still valid

**Why These Matter:**
- Too few connections → Queries wait for available connection
- Too many connections → Database server overwhelmed, memory usage high
- Idle timeout too short → Frequent reconnection overhead
- No health checks → Stale/broken connections remain in pool

### 2. Context-Based Timeout for Initialization

**Pattern: Use context timeouts for resource acquisition.**

```go
// internal/db/connection.go
// Create connection pool with timeout
ctx, cancel := context.WithTimeout(ctx, config.ConnTimeout)
defer cancel()

pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
if err != nil {
    return nil, fmt.Errorf("failed to create connection pool: %w", err)
}

// Verify connection with ping
if err := pool.Ping(ctx); err != nil {
    pool.Close()  // Clean up on failure
    return nil, fmt.Errorf("failed to ping database: %w", err)
}
```

**Key Pattern:**
- Don't hang forever if database is unreachable
- Close partially-created resources on failure (pool.Close())
- Use same context with timeout for verification (ping)

### 3. Health Check Pattern

**Pattern: Provide way to verify resource is still healthy.**

```go
// HealthCheck performs a health check on the database connection
func (p *Pool) HealthCheck(ctx context.Context) error {
    ctx, cancel := context.WithTimeout(ctx, p.config.ConnTimeout)
    defer cancel()

    if err := p.Ping(ctx); err != nil {
        return fmt.Errorf("database health check failed: %w", err)
    }

    return nil
}

// Usage: Can be called periodically to verify database is still accessible
```

**When Used:**
- Before returning response in HTTP handlers
- In liveness probes (Kubernetes, Docker health checks)
- Periodically in background goroutines

---

## Project-Specific Learning Path

Based on this project's stories, here's a suggested learning order:

1. **Story 1.1 (RPC Client)** ← Foundation
   - ✅ Structs, methods, interfaces
   - ✅ Error handling and wrapping
   - ✅ Context and cancellation
   - ✅ Structured logging
   - ✅ Testing patterns

2. **Story 1.2 (Database)** ← Structure & Resources
   - ✅ Struct composition and embedding
   - ✅ Configuration management
   - ✅ Database migrations with golang-migrate
   - ✅ Connection pooling
   - ✅ Resource lifecycle (defer, Close)
   - ✅ Integration testing with real database

3. **Story 1.3 (Backfill Workers)** ← Concurrency Basics
   - Goroutines and channels
   - Worker pool pattern
   - Synchronization (sync.WaitGroup)
   - Concurrent error handling

4. **Story 1.4 (Live-Tail)**
   - Continuous processing
   - Tickers and timers
   - Graceful shutdown

5. **Story 2.1 (REST API)**
   - HTTP server (net/http)
   - Routing
   - Middleware pattern
   - JSON encoding/decoding

---

## Additional Resources

**Official Documentation:**
- [Go Tour](https://go.dev/tour/) - Interactive introduction
- [Effective Go](https://go.dev/doc/effective_go) - Essential reading
- [Go by Example](https://gobyexample.com/) - Code examples

**Books:**
- "The Go Programming Language" by Donovan & Kernighan
- "Learning Go" by Jon Bodner

**Your Project:**
- Read the actual code in `internal/rpc/` - it's production-quality
- Run the tests: `go test -v ./internal/rpc/...`
- Experiment: Modify code and see what happens

---

**Next Steps:** As you implement more stories (1.3, 1.4, 2.1), add new concepts to this guide!
