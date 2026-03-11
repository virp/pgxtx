package pgxtx

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	// Transaction metrics
	transactionDuration  metric.Float64Histogram
	transactionTotal     metric.Int64Counter
	nestedTransactionTot metric.Int64Counter

	// Retry metrics
	retryTotal        metric.Int64Counter
	retryDuration     metric.Float64Histogram
	serializationErrs metric.Int64Counter
)

// Status attributes for transaction metrics
var (
	attrStatusCommitted  = attribute.String("status", "committed")
	attrStatusRolledBack = attribute.String("status", "rolled_back")
	attrStatusError      = attribute.String("status", "error")
	attrStatusSuccess    = attribute.String("status", "success")
	attrStatusFailed     = attribute.String("status", "failed")
)

func init() {
	meter := otel.Meter("pgxtx")
	var err error

	// Transaction duration histogram (in seconds)
	transactionDuration, err = meter.Float64Histogram(
		"pgxtx_transaction_duration",
		metric.WithDescription("Duration of database transactions in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		panic(fmt.Errorf("error while creating pgxtx_transaction_duration metric: %w", err))
	}

	// Transaction total counter
	transactionTotal, err = meter.Int64Counter(
		"pgxtx_transaction_total",
		metric.WithDescription("Total number of database transactions"),
		metric.WithUnit("1"),
	)
	if err != nil {
		panic(fmt.Errorf("error while creating pgxtx_transaction_total metric: %w", err))
	}

	// Nested transaction counter
	nestedTransactionTot, err = meter.Int64Counter(
		"pgxtx_nested_transaction_total",
		metric.WithDescription("Total number of nested database transaction calls"),
		metric.WithUnit("1"),
	)
	if err != nil {
		panic(fmt.Errorf("error while creating pgxtx_nested_transaction_total metric: %w", err))
	}

	// Retry total counter
	retryTotal, err = meter.Int64Counter(
		"pgxtx_retry_total",
		metric.WithDescription("Total number of transaction retry attempts"),
		metric.WithUnit("1"),
	)
	if err != nil {
		panic(fmt.Errorf("error while creating pgxtx_retry_total metric: %w", err))
	}

	// Retry duration histogram
	retryDuration, err = meter.Float64Histogram(
		"pgxtx_retry_duration",
		metric.WithDescription("Duration of transaction retry attempts in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		panic(fmt.Errorf("error while creating pgxtx_retry_duration metric: %w", err))
	}

	// Serialization errors counter
	serializationErrs, err = meter.Int64Counter(
		"pgxtx_serialization_errors_total",
		metric.WithDescription("Total number of serialization errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		panic(fmt.Errorf("error while creating pgxtx_serialization_errors_total metric: %w", err))
	}
}

// recordTransactionDuration records the duration of a transaction.
func recordTransactionDuration(ctx context.Context, duration time.Duration, status attribute.KeyValue) {
	transactionDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(status))
}

// recordTransactionTotal increments the transaction counter with the given status.
func recordTransactionTotal(ctx context.Context, status attribute.KeyValue) {
	transactionTotal.Add(ctx, 1, metric.WithAttributes(status))
}

// recordNestedTransaction increments the nested transaction counter.
func recordNestedTransaction(ctx context.Context) {
	nestedTransactionTot.Add(ctx, 1)
}

// recordRetry increments the retry counter with the given status.
func recordRetry(ctx context.Context, status attribute.KeyValue) {
	retryTotal.Add(ctx, 1, metric.WithAttributes(status))
}

// recordRetryDuration records the duration of a retry attempt.
func recordRetryDuration(ctx context.Context, duration time.Duration) {
	retryDuration.Record(ctx, duration.Seconds())
}

// recordSerializationError increments the serialization error counter.
func recordSerializationError(ctx context.Context) {
	serializationErrs.Add(ctx, 1)
}
