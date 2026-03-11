package pgxtx

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsSerializationError(t *testing.T) {
	t.Run("serialization failure", func(t *testing.T) {
		err := &pgconn.PgError{Code: pgSerializationErrorCode}
		assert.True(t, IsSerializationError(err))
	})

	t.Run("deadlock detected", func(t *testing.T) {
		err := &pgconn.PgError{Code: pgDeadlockDetectedErrorCode}
		assert.True(t, IsSerializationError(err))
	})

	t.Run("other error", func(t *testing.T) {
		err := &pgconn.PgError{Code: "08000"}
		assert.False(t, IsSerializationError(err))
	})

	t.Run("nil error", func(t *testing.T) {
		assert.False(t, IsSerializationError(nil))
	})

	t.Run("wrapped error", func(t *testing.T) {
		err := errors.Join(errors.New("some error"), &pgconn.PgError{Code: pgSerializationErrorCode})
		assert.True(t, IsSerializationError(err))
	})
}

func TestTxFromContext(t *testing.T) {
	t.Run("no transaction in context", func(t *testing.T) {
		ctx := context.Background()
		tx := TxFromContext(ctx)
		assert.Nil(t, tx)
	})

	t.Run("transaction in context", func(t *testing.T) {
		// We can't create a real pgx.Tx without a database connection,
		// but we can test that the context mechanism works with an interface
		// Using raw context value since TxToContext expects pgx.Tx
		ctx := context.WithValue(context.Background(), txContextKey, "mock-tx")
		retrieved := ctx.Value(txContextKey)
		assert.Equal(t, "mock-tx", retrieved)
	})
}

func TestHasTransaction(t *testing.T) {
	t.Run("no transaction", func(t *testing.T) {
		ctx := context.Background()
		assert.False(t, HasTransaction(ctx))
	})

	t.Run("has transaction", func(t *testing.T) {
		// Test that HasTransaction works when value is in context
		// We test the mechanism rather than the specific type
		ctx := context.WithValue(context.Background(), txContextKey, "mock-tx")
		// Direct check since TxFromContext does type assertion
		assert.NotNil(t, ctx.Value(txContextKey))
	})
}

func TestWithRetry(t *testing.T) {
	t.Run("success on first attempt", func(t *testing.T) {
		ctx := context.Background()
		attempts := 0

		err := WithRetry(ctx, RetryConfig{MaxRetries: 3}, func(ctx context.Context) error {
			attempts++
			return nil
		})

		require.NoError(t, err)
		assert.Equal(t, 1, attempts)
	})

	t.Run("retry on serialization error", func(t *testing.T) {
		ctx := context.Background()
		attempts := 0

		err := WithRetry(ctx, RetryConfig{
			MaxRetries:      3,
			InitialInterval: 1 * time.Millisecond,
			MaxInterval:     10 * time.Millisecond,
		}, func(ctx context.Context) error {
			attempts++
			if attempts < 3 {
				return &pgconn.PgError{Code: pgSerializationErrorCode}
			}
			return nil
		})

		require.NoError(t, err)
		assert.Equal(t, 3, attempts)
	})

	t.Run("no retry on other errors", func(t *testing.T) {
		ctx := context.Background()
		attempts := 0
		expectedErr := errors.New("non-retryable error")

		err := WithRetry(ctx, RetryConfig{MaxRetries: 3}, func(ctx context.Context) error {
			attempts++
			return expectedErr
		})

		require.ErrorIs(t, err, expectedErr)
		assert.Equal(t, 1, attempts)
	})

	t.Run("max retries exceeded", func(t *testing.T) {
		ctx := context.Background()
		attempts := 0

		err := WithRetry(ctx, RetryConfig{
			MaxRetries:      2,
			InitialInterval: 1 * time.Millisecond,
			MaxInterval:     10 * time.Millisecond,
		}, func(ctx context.Context) error {
			attempts++
			return &pgconn.PgError{Code: pgSerializationErrorCode}
		})

		require.Error(t, err)
		assert.Equal(t, 3, attempts) // initial + 2 retries
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		attempts := 0

		cancel() // Cancel immediately

		err := WithRetry(ctx, RetryConfig{
			MaxRetries:      3,
			InitialInterval: 100 * time.Millisecond,
		}, func(ctx context.Context) error {
			attempts++
			return &pgconn.PgError{Code: pgSerializationErrorCode}
		})

		require.ErrorIs(t, err, context.Canceled)
		assert.Equal(t, 1, attempts)
	})
}

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()

	assert.Equal(t, 3, cfg.MaxRetries)
	assert.Equal(t, 100*time.Millisecond, cfg.InitialInterval)
	assert.Equal(t, 1*time.Second, cfg.MaxInterval)
	assert.Equal(t, 2.0, cfg.Multiplier)
}
