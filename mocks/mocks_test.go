package mocks

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/virp/pgxtx"
)

type testUser struct {
	ID   int64
	Name string
}

type testUserRepository struct {
	ep pgxtx.ExecutorProvider
}

func (r *testUserRepository) List(ctx context.Context) ([]testUser, error) {
	exec := r.ep.GetExecutor(ctx)

	rows, err := exec.Query(ctx, "SELECT id, name FROM users ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []testUser
	for rows.Next() {
		var user testUser
		if err := rows.Scan(&user.ID, &user.Name); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func TestManagerExpectGetExecutor(t *testing.T) {
	mgr := NewManager(t)
	exec := NewExecutor(t)

	mgr.EXPECT().GetExecutor(Anything).Return(exec).Once()

	assert.Same(t, exec, mgr.GetExecutor(t.Context()))
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

	err := mgr.WithTx(t.Context(), func(context.Context) error {
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

	err := mgr.ExecInTx(t.Context(), func(ctx context.Context, exec pgxtx.Executor) error {
		_, err := exec.Exec(ctx, "SELECT 1")
		return err
	})

	assert.ErrorIs(t, err, expectedErr)
}

func TestExecutorExpectQueryWithRows(t *testing.T) {
	ep := NewExecutorProvider(t)
	exec := NewExecutor(t)
	repo := &testUserRepository{ep: ep}

	rows := NewRows(t).
		AddRow(int64(1), "John").
		AddRow(int64(2), "Jane")

	ep.EXPECT().
		GetExecutor(Anything).
		Return(exec).
		Once()

	exec.EXPECT().
		Query(Anything, "SELECT id, name FROM users ORDER BY id").
		Return(rows, nil).
		Once()

	users, err := repo.List(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []testUser{
		{ID: 1, Name: "John"},
		{ID: 2, Name: "Jane"},
	}, users)
}
