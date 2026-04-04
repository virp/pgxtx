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
- [`go.opentelemetry.io/otel`](https://github.com/open-telemetry/opentelemetry-go)

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

### Repository Pattern with ExecutorProvider

```go
type UserRepository struct {
    ep pgxtx.ExecutorProvider
}

func NewUserRepository(ep pgxtx.ExecutorProvider) *UserRepository {
    return &UserRepository{ep: ep}
}

func (r *UserRepository) Create(ctx context.Context, name, email string) error {
    // GetExecutor returns the transaction from context if present,
    // or the pool if no transaction exists
    exec := r.ep.GetExecutor(ctx)

    _, err := exec.Exec(ctx,
        "INSERT INTO users (name, email, created_at) VALUES ($1, $2, NOW())",
        name, email,
    )
    return err
}

func (r *UserRepository) GetByID(ctx context.Context, id int) (*User, error) {
    exec := r.ep.GetExecutor(ctx)

    var user User
    err := exec.QueryRow(ctx,
        "SELECT id, name, email FROM users WHERE id = $1", id,
    ).Scan(&user.ID, &user.Name, &user.Email)
    if err != nil {
        return nil, err
    }
    return &user, nil
}

func (r *UserRepository) CreateAuditLog(ctx context.Context, fromUserID, toUserID int, amount decimal.Decimal) error {
    exec := r.ep.GetExecutor(ctx)

    _, err := exec.Exec(ctx,
        "INSERT INTO audit_log (from_user_id, to_user_id, amount, created_at) VALUES ($1, $2, $3, NOW())",
        fromUserID, toUserID, amount,
    )
    return err
}
```

### Service Layer with Multiple Repositories

```go
type AccountRepository struct {
    ep pgxtx.ExecutorProvider
}

func NewAccountRepository(ep pgxtx.ExecutorProvider) *AccountRepository {
    return &AccountRepository{ep: ep}
}

func (r *AccountRepository) Debit(ctx context.Context, userID int, amount decimal.Decimal) error {
    exec := r.ep.GetExecutor(ctx)

    _, err := exec.Exec(ctx,
        "UPDATE accounts SET balance = balance - $1 WHERE user_id = $2",
        amount, userID,
    )
    return err
}

func (r *AccountRepository) Credit(ctx context.Context, userID int, amount decimal.Decimal) error {
    exec := r.ep.GetExecutor(ctx)

    _, err := exec.Exec(ctx,
        "UPDATE accounts SET balance = balance + $1 WHERE user_id = $2",
        amount, userID,
    )
    return err
}

// TransferService coordinates operations across multiple repositories
type TransferService struct {
    tr          pgxtx.TxRunner
    userRepo    *UserRepository
    accountRepo *AccountRepository
}

func NewTransferService(tr *pgxtx.TxRunner, userRepo *UserRepository, accountRepo *AccountRepository) *TransferService {
    return &TransferService{
        tr:          tr,
        userRepo:    userRepo,
        accountRepo: accountRepo,
    }
}

// Transfer executes a money transfer atomically
// Both debit and credit operations participate in the same transaction
func (s *TransferService) Transfer(ctx context.Context, fromUserID, toUserID int, amount decimal.Decimal) error {
    // Single transaction wraps both repository calls
    return s.tr.WithTx(ctx, func(ctx context.Context) error {
        // Debit from sender - uses transaction from context
        if err := s.accountRepo.Debit(ctx, fromUserID, amount); err != nil {
            return err
        }

        // Credit to recipient - uses same transaction from context
        if err := s.accountRepo.Credit(ctx, toUserID, amount); err != nil {
            return err
        }

        // Create audit log entry - uses same transaction from context
        return s.userRepo.CreateAuditLog(ctx, fromUserID, toUserID, amount)
    })
}

// Usage:
// transferService := NewTransferService(tr, userRepo, accountRepo)
// err := transferService.Transfer(ctx, 1, 2, decimal.NewFromInt(100))
// if err != nil {
//     log.Printf("Transfer failed: %v", err) // All operations rolled back
// }
```

### Unit Tests with EXPECT()

```go
import (
    "context"
    "testing"

    "github.com/jackc/pgx/v5/pgconn"
    "github.com/stretchr/testify/require"
    "github.com/virp/pgxtx"
    "github.com/virp/pgxtx/mocks"
)

func TestUserRepositoryCreate(t *testing.T) {
    ep := mocks.NewExecutorProvider(t)
    exec := mocks.NewExecutor(t)
    repo := NewUserRepository(ep)

    ep.EXPECT().
        GetExecutor(mocks.Anything).
        Return(exec).
        Once()

    exec.EXPECT().
        Exec(
            mocks.Anything,
            "INSERT INTO users (name, email, created_at) VALUES ($1, $2, NOW())",
            "John",
            "john@example.com",
        ).
        Return(pgconn.CommandTag{}, nil).
        Once()

    err := repo.Create(context.Background(), "John", "john@example.com")
    require.NoError(t, err)
}

func TestUserRepositoryList(t *testing.T) {
    ep := mocks.NewExecutorProvider(t)
    exec := mocks.NewExecutor(t)
    repo := NewUserRepository(ep)

    rows := mocks.NewRows(t).
        AddRow(1, "John", "john@example.com").
        AddRow(2, "Jane", "jane@example.com")

    ep.EXPECT().
        GetExecutor(mocks.Anything).
        Return(exec).
        Once()

    exec.EXPECT().
        Query(
            mocks.Anything,
            "SELECT id, name, email FROM users ORDER BY id",
        ).
        Return(rows, nil).
        Once()

    users, err := repo.List(context.Background())
    require.NoError(t, err)
    require.Len(t, users, 2)
    require.Equal(t, "John", users[0].Name)
    require.Equal(t, "Jane", users[1].Name)
}

func TestTransferServiceTransfer(t *testing.T) {
    tr := mocks.NewTxRunner(t)
    svc := NewTransferService(tr, userRepo, accountRepo)

    tr.EXPECT().
        WithTx(mocks.Anything, mocks.Anything).
        Run(func(ctx context.Context, fn pgxtx.TxFunc) {
            require.NoError(t, fn(ctx))
        }).
        Return(nil).
        Once()

    require.NoError(t, svc.Transfer(context.Background(), 1, 2, amount))
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
- `ExecutorProvider` - Minimal interface for repositories that only need `GetExecutor`
- `TxRunner` - Minimal interface for services that open transactions
- `Manager` - Combined interface for transaction execution and executor lookup
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
