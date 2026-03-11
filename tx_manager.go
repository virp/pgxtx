package pgxtx

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// TxManager is a transaction manager that handles transaction lifecycle
// and nested transaction calls.
type TxManager struct {
	pool     *pgxpool.Pool
	retryCfg RetryConfig
	tracer   trace.Tracer
	enableTx bool
}

// TxManagerOption is a functional option for configuring TxManager.
type TxManagerOption func(*TxManager)

// WithRetryConfig sets the retry configuration for the transaction manager.
func WithRetryConfig(cfg RetryConfig) TxManagerOption {
	return func(tm *TxManager) {
		tm.retryCfg = cfg
	}
}

// WithTracer sets the OpenTelemetry tracer for the transaction manager.
func WithTracer(tracer trace.Tracer) TxManagerOption {
	return func(tm *TxManager) {
		tm.tracer = tracer
	}
}

// WithoutTx disables transaction management.
// This is useful for testing or read-only operations.
func WithoutTx() TxManagerOption {
	return func(tm *TxManager) {
		tm.enableTx = false
	}
}

// NewTxManager creates a new transaction manager.
func NewTxManager(pool *pgxpool.Pool, opts ...TxManagerOption) *TxManager {
	tm := &TxManager{
		pool:     pool,
		retryCfg: DefaultRetryConfig(),
		enableTx: true,
	}

	for _, opt := range opts {
		opt(tm)
	}

	// Create default tracer if not provided
	if tm.tracer == nil {
		tm.tracer = otel.Tracer("pgxtx")
	}

	return tm
}

// TxFunc is a function that executes within a transaction.
type TxFunc func(ctx context.Context) error

// WithTx executes the given function within a database transaction.
//
// If a transaction already exists in the context, it will be reused (nested call).
// Otherwise, a new transaction is created.
//
// The transaction is automatically committed on success or rolled back on error.
// Serialization errors (40001) and deadlock errors (40P01) are automatically retried.
func (tm *TxManager) WithTx(ctx context.Context, fn TxFunc) error {
	if !tm.enableTx {
		return fn(ctx)
	}

	// Check if we're already in a transaction (nested call)
	if HasTransaction(ctx) {
		recordNestedTransaction(ctx)

		ctx, span := tm.tracer.Start(ctx, "pgxtx.WithTx.nested",
			trace.WithSpanKind(trace.SpanKindInternal),
			trace.WithAttributes(attribute.Bool("pgxtx.transaction.nested", true)),
		)
		defer span.End()

		return fn(ctx)
	}

	return WithRetry(ctx, tm.retryCfg, func(ctx context.Context) error {
		return tm.executeWithNewTx(ctx, fn)
	})
}

func (tm *TxManager) executeWithNewTx(ctx context.Context, fn TxFunc) error {
	// Generate transaction ID for tracing
	txID := uuid.New().String()

	ctx, span := tm.tracer.Start(ctx, "pgxtx.WithTx",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("pgxtx.transaction.id", txID),
			attribute.Bool("pgxtx.transaction.nested", false),
		),
	)
	defer span.End()

	startTime := time.Now()

	tx, err := tm.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to begin transaction")
		recordTransactionTotal(ctx, attrStatusError)
		return err
	}

	// Store transaction in context so repositories can use it automatically
	ctxWithTx := TxToContext(ctx, tx)

	span.AddEvent("pgxtx.transaction.started",
		trace.WithAttributes(attribute.String("pgxtx.transaction.id", txID)),
	)

	defer func() {
		if p := recover(); p != nil {
			span.AddEvent("pgxtx.transaction.panic",
				trace.WithAttributes(
					attribute.String("pgxtx.transaction.panic_value", stringifyPanic(p)),
				),
			)
			_ = tx.Rollback(ctx)
			recordTransactionTotal(ctx, attrStatusRolledBack)
			recordTransactionDuration(ctx, time.Since(startTime), attrStatusRolledBack)
			panic(p)
		}
	}()

	err = fn(ctxWithTx)
	if err != nil {
		span.AddEvent("pgxtx.transaction.rollback",
			trace.WithAttributes(attribute.String("pgxtx.error", err.Error())),
		)
		span.RecordError(err)
		span.SetStatus(codes.Error, "transaction rolled back")

		if rbErr := tx.Rollback(ctx); rbErr != nil {
			// Check if transaction is already done (committed or rolled back)
			// SQL state 25P01 means "transaction is not in progress"
			if pgErr, ok := errors.AsType[*pgconn.PgError](rbErr); !ok || pgErr.Code != "25P01" {
				err = errors.Join(err, rbErr)
			}
		}
		recordTransactionTotal(ctx, attrStatusRolledBack)
		recordTransactionDuration(ctx, time.Since(startTime), attrStatusRolledBack)
		return err
	}

	span.AddEvent("pgxtx.transaction.commit")
	if err := tx.Commit(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to commit transaction")
		recordTransactionTotal(ctx, attrStatusError)
		recordTransactionDuration(ctx, time.Since(startTime), attrStatusError)
		return err
	}

	span.SetStatus(codes.Ok, "transaction committed")
	recordTransactionTotal(ctx, attrStatusCommitted)
	recordTransactionDuration(ctx, time.Since(startTime), attrStatusCommitted)

	return nil
}

// GetExecutor returns an Executor for the given context.
// If a transaction exists in the context, it returns the transaction.
// Otherwise, it returns the pool (for direct queries without transaction).
//
// Repositories should use this method to obtain an Executor instead of
// storing a reference to the pool or transaction directly.
func (tm *TxManager) GetExecutor(ctx context.Context) Executor {
	if tx := TxFromContext(ctx); tx != nil {
		return tx
	}
	return tm.pool
}

// GetPool returns the underlying pool.
// Use this only for operations that must not be part of a transaction.
func (tm *TxManager) GetPool() *pgxpool.Pool {
	return tm.pool
}

// ExecInTx is a helper that executes a function in a transaction if one exists,
// otherwise uses the pool directly. This is useful for queries that don't need
// transaction guarantees but should participate in a transaction if available.
func (tm *TxManager) ExecInTx(ctx context.Context, fn func(ctx context.Context, exec Executor) error) error {
	exec := tm.GetExecutor(ctx)
	wrapper := func(ctx context.Context) error {
		return fn(ctx, exec)
	}

	if HasTransaction(ctx) {
		return wrapper(ctx)
	}

	return tm.WithTx(ctx, wrapper)
}

// stringifyPanic converts a panic value to a string safely.
func stringifyPanic(p interface{}) string {
	if s, ok := p.(string); ok {
		return s
	}
	return string("panic")
}
