package mocks

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/virp/pgxtx"
)

func TestManagerExpectGetExecutor(t *testing.T) {
	mgr := NewManager(t)
	exec := NewExecutor(t)

	mgr.EXPECT().GetExecutor(Anything).Return(exec).Once()

	assert.Same(t, exec, mgr.GetExecutor(context.Background()))
}

func TestManagerExpectWithTx(t *testing.T) {
	mgr := NewManager(t)

	mgr.EXPECT().
		WithTx(Anything, Anything).
		Run(func(_ context.Context, fn pgxtx.TxFunc) {
			assert.NoError(t, fn(context.Background()))
		}).
		Return(nil).
		Once()

	err := mgr.WithTx(context.Background(), func(context.Context) error {
		return nil
	})

	assert.NoError(t, err)
}

func TestManagerExpectExecInTx(t *testing.T) {
	mgr := NewManager(t)
	exec := NewExecutor(t)
	expectedErr := errors.New("boom")

	mgr.EXPECT().
		ExecInTx(Anything, Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context, pgxtx.Executor) error) error {
			return fn(ctx, exec)
		}).
		Once()

	exec.EXPECT().
		Exec(Anything, "SELECT 1").
		Return(pgconn.CommandTag{}, expectedErr).
		Once()

	err := mgr.ExecInTx(context.Background(), func(ctx context.Context, exec pgxtx.Executor) error {
		_, err := exec.Exec(ctx, "SELECT 1")
		return err
	})

	assert.ErrorIs(t, err, expectedErr)
}
