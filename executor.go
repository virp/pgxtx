package pgxtx

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Executor is a common interface for executing database queries.
// It is implemented by *pgx.Conn, pgx.Tx, and *pgxpool.Pool.
// Repositories should depend on this interface instead of concrete types
// to support transactions seamlessly.
type Executor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (commandTag pgconn.CommandTag, err error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// ExecutorProvider provides an Executor for the given context.
// Repositories should typically depend on this interface instead of *TxManager.
type ExecutorProvider interface {
	GetExecutor(ctx context.Context) Executor
}

// TxRunner executes work in a transaction-aware way.
// Services that coordinate multiple repositories should typically depend on this interface.
type TxRunner interface {
	WithTx(ctx context.Context, fn TxFunc) error
	ExecInTx(ctx context.Context, fn func(ctx context.Context, exec Executor) error) error
}

// Manager combines transaction execution and executor lookup.
type Manager interface {
	ExecutorProvider
	TxRunner
}

// Ensure that pgx.Conn implements Executor.
var _ Executor = (*pgx.Conn)(nil)

// Ensure that pgx.Tx implements Executor.
var _ Executor = pgx.Tx(nil)

// Ensure that TxManager implements the public interfaces.
var _ ExecutorProvider = (*TxManager)(nil)
var _ TxRunner = (*TxManager)(nil)
var _ Manager = (*TxManager)(nil)

// TxProvider is an interface for obtaining a database connection.
// This is typically implemented by pgxpool.Pool.
type TxProvider interface {
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}
