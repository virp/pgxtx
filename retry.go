package pgxtx

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// RetryConfig holds configuration for transaction retry logic.
type RetryConfig struct {
	// MaxRetries is the maximum number of retry attempts.
	// Default is 3 if not specified.
	MaxRetries int

	// InitialInterval is the initial interval between retries.
	// Default is 100ms if not specified.
	InitialInterval time.Duration

	// MaxInterval is the maximum interval between retries.
	// Default is 1s if not specified.
	MaxInterval time.Duration

	// Multiplier is the exponential backoff multiplier.
	// Default is 2.0 if not specified.
	Multiplier float64
}

// DefaultRetryConfig returns the default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:      3,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
	}
}

// WithRetry executes the given function with retry logic for serialization errors.
// If the function returns a serialization error, it will be retried with exponential backoff.
func WithRetry(ctx context.Context, config RetryConfig, fn func(ctx context.Context) error) error {
	if config.MaxRetries == 0 {
		cfg := DefaultRetryConfig()
		config.MaxRetries = cfg.MaxRetries
		config.InitialInterval = cfg.InitialInterval
		config.MaxInterval = cfg.MaxInterval
		config.Multiplier = cfg.Multiplier
	}

	tracer := otel.Tracer("pgxtx")
	ctx, span := tracer.Start(ctx, "pgxtx.WithRetry", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	b := backoff.NewExponentialBackOff()
	b.InitialInterval = config.InitialInterval
	b.MaxInterval = config.MaxInterval
	b.Multiplier = config.Multiplier

	var lastErr error
	var retryCount int
	startTime := time.Now()

	for attempt := 1; attempt <= config.MaxRetries+1; attempt++ {
		lastErr = fn(ctx)
		if lastErr == nil {
			if retryCount > 0 {
				recordRetry(ctx, attrStatusSuccess)
				recordRetryDuration(ctx, time.Since(startTime))
			}
			span.SetAttributes(attribute.Int("pgxtx.retry.count", retryCount))
			return nil
		}

		if !isRetryableError(lastErr) {
			recordRetry(ctx, attrStatusFailed)
			span.SetAttributes(
				attribute.Int("pgxtx.retry.count", retryCount),
				attribute.String("pgxtx.error", lastErr.Error()),
			)
			return lastErr
		}

		// Record serialization error metric
		recordSerializationError(ctx)
		retryCount++

		// Don't sleep after the last attempt
		if attempt > config.MaxRetries {
			break
		}

		nextBackoff := b.NextBackOff()
		if nextBackoff == backoff.Stop {
			// Use default interval if backoff is stopped
			nextBackoff = config.InitialInterval * time.Duration(attempt)
			if nextBackoff > config.MaxInterval {
				nextBackoff = config.MaxInterval
			}
		}

		// Record retry attempt
		recordRetry(ctx, attrStatusFailed)

		span.AddEvent("pgxtx.retry.waiting",
			trace.WithAttributes(
				attribute.Int("pgxtx.retry.attempt", attempt),
				attribute.String("pgxtx.retry.backoff", nextBackoff.String()),
				attribute.String("pgxtx.error", lastErr.Error()),
			),
		)

		select {
		case <-ctx.Done():
			recordRetry(ctx, attrStatusFailed)
			span.RecordError(ctx.Err())
			return ctx.Err()
		case <-time.After(nextBackoff):
			// Continue to next retry
		}
	}

	recordRetry(ctx, attrStatusFailed)
	recordRetryDuration(ctx, time.Since(startTime))
	span.SetAttributes(
		attribute.Int("pgxtx.retry.count", retryCount),
		attribute.String("pgxtx.error", lastErr.Error()),
	)
	span.RecordError(lastErr)

	return lastErr
}
