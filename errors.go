package pgxtx

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// pgSerializationErrorCode is the SQL state code for serialization failure.
const pgSerializationErrorCode = "40001"

// pgDeadlockDetectedErrorCode is the SQL state code for deadlock detected.
const pgDeadlockDetectedErrorCode = "40P01"

// ErrSerialization is returned when a transaction fails due to serialization failure or deadlock.
var ErrSerialization = errors.New("transaction serialization failure")

// IsSerializationError checks if the error is a serialization failure or deadlock.
// These errors are retryable.
func IsSerializationError(err error) bool {
	if err == nil {
		return false
	}

	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		return pgErr.Code == pgSerializationErrorCode || pgErr.Code == pgDeadlockDetectedErrorCode
	}

	return false
}

// isRetryableError checks if the error is retryable in the context of database transactions.
func isRetryableError(err error) bool {
	return IsSerializationError(err)
}
