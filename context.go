package pgxtx

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type contextKey int

const (
	txContextKey contextKey = iota
)

// TxToContext returns a new context with the transaction stored in it.
// This allows repositories to automatically use the transaction
// without explicitly passing it around.
func TxToContext(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txContextKey, tx)
}

// TxFromContext returns the transaction from the context.
// Returns nil if no transaction is present.
// Repositories can use this to check if they should use a transaction
// or a direct database connection.
func TxFromContext(ctx context.Context) pgx.Tx {
	tx, _ := ctx.Value(txContextKey).(pgx.Tx)
	return tx
}

// HasTransaction checks if the context contains a transaction.
func HasTransaction(ctx context.Context) bool {
	return TxFromContext(ctx) != nil
}
