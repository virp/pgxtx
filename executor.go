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
	Exec(ctx context.Context, sql string, arguments ...interface{}) (commandTag pgconn.CommandTag, err error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

// Ensure that pgx.Conn implements Executor.
var _ Executor = (*pgx.Conn)(nil)

// Ensure that pgx.Tx implements Executor.
var _ Executor = (pgx.Tx)(nil)

// TxProvider is an interface for obtaining a database connection.
// This is typically implemented by pgxpool.Pool.
type TxProvider interface {
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}
