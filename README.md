# pgxtx

PostgreSQL Transaction Manager for Go with automatic retry, nested transaction support, and OpenTelemetry observability.

## Description

`pgxtx` is a Go package that simplifies PostgreSQL transaction management when using [`pgx/v5`](https://github.com/jackc/pgx/v5). It provides:

- **Automatic transaction lifecycle management** - Begin, commit, and rollback handled automatically
- **Nested transaction support** - Reuse existing transactions when already in a transaction context
- **Automatic retry logic** - Handles serialization failures (40001) and deadlocks (40P01) with exponential backoff
- **OpenTelemetry integration** - Built-in tracing and metrics for observability
- **Clean repository pattern** - Use the `Executor` interface for transaction-aware repositories

## Installation

```bash
go get github.com/virp/pgxtx
```

### Dependencies

- Go 1.26+
- [`github.com/jackc/pgx/v5`](https://github.com/jackc/pgx/v5)
- [`go.opentelemetry.io/otel`](https://github.com/open-telemetry/opentelemetry-go) (optional, for tracing)

## Usage Examples

### Basic Setup

```go
package main

import (
    "context"
    "log"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/virp/pgxtx"
)

func main() {
    ctx := context.Background()
    
    // Create connection pool
    pool, err := pgxpool.New(ctx, "postgres://user:pass@localhost:5432/dbname")
    if err != nil {
        log.Fatal(err)
    }
    defer pool.Close()

    // Create transaction manager
    tm := pgxtx.NewTxManager(pool)
    
    // Or with custom retry config
    tm = pgxtx.NewTxManager(pool, 
        pgxtx.WithRetryConfig(pgxtx.RetryConfig{
            MaxRetries:      5,
            InitialInterval: 50 * time.Millisecond,
            MaxInterval:     2 * time.Second,
        }),
    )
}
```

### Simple Transaction

```go
err := tm.WithTx(ctx, func(ctx context.Context) error {
    // Get executor (transaction-aware)
    exec := tm.GetExecutor(ctx)
    
    // Use executor for database operations
    _, err := exec.Exec(ctx, "INSERT INTO users (name, email) VALUES ($1, $2)", "John", "john@example.com")
    if err != nil {
        return err
    }
    
    _, err = exec.Exec(ctx, "UPDATE accounts SET balance = balance - 100 WHERE user_id = $1", 1)
    if err != nil {
        return err
    }
    
    // Transaction automatically commits on success
    // or rolls back on error
    return nil
})

if err != nil {
    log.Printf("Transaction failed: %v", err)
}
```

### Nested Transactions

```go
// Outer transaction
err := tm.WithTx(ctx, func(ctx context.Context) error {
    // This will use the existing transaction
    err := tm.WithTx(ctx, func(ctx context.Context) error {
        exec := tm.GetExecutor(ctx)
        _, err := exec.Exec(ctx, "INSERT INTO audit_log (action) VALUES ($1)", "nested_operation")
        return err
    })
    if err != nil {
        return err
    }
    
    // More operations in outer transaction...
    return nil
})
```

### Repository Pattern with Executor

```go
type UserRepository struct {
    tm *pgxtx.TxManager
}

func NewUserRepository(tm *pgxtx.TxManager) *UserRepository {
    return &UserRepository{tm: tm}
}

func (r *UserRepository) Create(ctx context.Context, name, email string) error {
    return tm.ExecInTx(ctx, func(ctx context.Context, exec pgxtx.Executor) error {
        _, err := exec.Exec(ctx, 
            "INSERT INTO users (name, email, created_at) VALUES ($1, $2, NOW())",
            name, email,
        )
        return err
    })
}

func (r *UserRepository) GetByID(ctx context.Context, id int) (*User, error) {
    // ExecInTx will use existing transaction if present,
    // or execute without transaction if not
    var user User
    err := tm.ExecInTx(ctx, func(ctx context.Context, exec pgxtx.Executor) error {
        return exec.QueryRow(ctx, 
            "SELECT id, name, email FROM users WHERE id = $1", id,
        ).Scan(&user.ID, &user.Name, &user.Email)
    })
    if err != nil {
        return nil, err
    }
    return &user, nil
}
```

### Context Helpers

```go
// Check if context has an active transaction
if pgxtx.HasTransaction(ctx) {
    // We're inside a transaction
}

// Get transaction from context (returns nil if not present)
tx := pgxtx.TxFromContext(ctx)
if tx != nil {
    // Use existing transaction
}

// Store transaction in context (usually done internally)
ctx = pgxtx.TxToContext(ctx, tx)
```

### Disabling Transactions

```go
// Create manager without transaction management
// Useful for read-only operations or testing
tm := pgxtx.NewTxManager(pool, pgxtx.WithoutTx())

// WithTx will execute the function directly without transaction
err := tm.WithTx(ctx, func(ctx context.Context) error {
    // No transaction overhead
    return nil
})
```

### Custom Tracer

```go
import "go.opentelemetry.io/otel"

tracer := otel.Tracer("my-app")
tm := pgxtx.NewTxManager(pool, pgxtx.WithTracer(tracer))
```

## API Reference

### Types

- `TxManager` - Main transaction manager
- `Executor` - Common interface for database operations (implemented by `pgx.Conn`, `pgx.Tx`, `pgxpool.Pool`)
- `RetryConfig` - Configuration for retry logic
- `TxFunc` - Function type for transactional operations

### TxManager Options

- `WithRetryConfig(cfg RetryConfig)` - Set retry configuration
- `WithTracer(tracer trace.Tracer)` - Set OpenTelemetry tracer
- `WithoutTx()` - Disable transaction management

### TxManager Methods

- `NewTxManager(pool *pgxpool.Pool, opts ...TxManagerOption) *TxManager` - Create new manager
- `WithTx(ctx context.Context, fn TxFunc) error` - Execute function in transaction
- `GetExecutor(ctx context.Context) Executor` - Get transaction-aware executor
- `GetPool() *pgxpool.Pool` - Get underlying pool
- `ExecInTx(ctx context.Context, fn func(ctx context.Context, exec Executor) error) error` - Execute in transaction if exists

### Context Functions

- `TxToContext(ctx context.Context, tx pgx.Tx) context.Context` - Store transaction in context
- `TxFromContext(ctx context.Context) pgx.Tx` - Get transaction from context
- `HasTransaction(ctx context.Context) bool` - Check if transaction exists in context

## License

MIT License - see [LICENSE](LICENSE) for details.
