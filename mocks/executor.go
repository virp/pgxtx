//revive:disable:exported
package mocks

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Executor struct {
	t testingT

	execExpectations     []*ExecutorExecCall
	queryExpectations    []*ExecutorQueryCall
	queryRowExpectations []*ExecutorQueryRowCall
}

type ExecutorExpecter struct {
	mock *Executor
}

type ExecutorExecCall struct {
	mock      *Executor
	ctx       any
	sql       any
	arguments []any
	returnTag pgconn.CommandTag
	returnErr error
	returnFn  func(context.Context, string, ...any) (pgconn.CommandTag, error)
	run       func(context.Context, string, ...any)
	once      bool
	called    bool
}

type ExecutorQueryCall struct {
	mock       *Executor
	ctx        any
	sql        any
	arguments  []any
	returnRows pgx.Rows
	returnErr  error
	returnFn   func(context.Context, string, ...any) (pgx.Rows, error)
	run        func(context.Context, string, ...any)
	once       bool
	called     bool
}

type ExecutorQueryRowCall struct {
	mock      *Executor
	ctx       any
	sql       any
	arguments []any
	returnRow pgx.Row
	returnFn  func(context.Context, string, ...any) pgx.Row
	run       func(context.Context, string, ...any)
	once      bool
	called    bool
}

func NewExecutor(t testingT) *Executor {
	m := &Executor{t: t}
	t.Cleanup(func() { m.assertExpectations() })
	return m
}

func (m *Executor) EXPECT() *ExecutorExpecter {
	return &ExecutorExpecter{mock: m}
}

func (m *Executor) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	call := m.findExecExpectation(ctx, sql, arguments...)
	if call.run != nil {
		call.run(ctx, sql, arguments...)
	}
	call.called = true
	if call.returnFn != nil {
		return call.returnFn(ctx, sql, arguments...)
	}
	return call.returnTag, call.returnErr
}

func (m *Executor) Query(ctx context.Context, sql string, arguments ...any) (pgx.Rows, error) {
	call := m.findQueryExpectation(ctx, sql, arguments...)
	if call.run != nil {
		call.run(ctx, sql, arguments...)
	}
	call.called = true
	if call.returnFn != nil {
		return call.returnFn(ctx, sql, arguments...)
	}
	return call.returnRows, call.returnErr
}

func (m *Executor) QueryRow(ctx context.Context, sql string, arguments ...any) pgx.Row {
	call := m.findQueryRowExpectation(ctx, sql, arguments...)
	if call.run != nil {
		call.run(ctx, sql, arguments...)
	}
	call.called = true
	if call.returnFn != nil {
		return call.returnFn(ctx, sql, arguments...)
	}
	return call.returnRow
}

func (e *ExecutorExpecter) Exec(ctx, sql any, arguments ...any) *ExecutorExecCall {
	call := &ExecutorExecCall{mock: e.mock, ctx: ctx, sql: sql, arguments: arguments}
	e.mock.execExpectations = append(e.mock.execExpectations, call)
	return call
}

func (e *ExecutorExpecter) Query(ctx, sql any, arguments ...any) *ExecutorQueryCall {
	call := &ExecutorQueryCall{mock: e.mock, ctx: ctx, sql: sql, arguments: arguments}
	e.mock.queryExpectations = append(e.mock.queryExpectations, call)
	return call
}

func (e *ExecutorExpecter) QueryRow(ctx, sql any, arguments ...any) *ExecutorQueryRowCall {
	call := &ExecutorQueryRowCall{mock: e.mock, ctx: ctx, sql: sql, arguments: arguments}
	e.mock.queryRowExpectations = append(e.mock.queryRowExpectations, call)
	return call
}

func (c *ExecutorExecCall) Return(tag pgconn.CommandTag, err error) *ExecutorExecCall {
	c.returnTag = tag
	c.returnErr = err
	return c
}

func (c *ExecutorExecCall) Run(run func(context.Context, string, ...any)) *ExecutorExecCall {
	c.run = run
	return c
}

func (c *ExecutorExecCall) RunAndReturn(fn func(context.Context, string, ...any) (pgconn.CommandTag, error)) *ExecutorExecCall {
	c.returnFn = fn
	return c
}

func (c *ExecutorExecCall) Once() *ExecutorExecCall {
	c.once = true
	return c
}

func (c *ExecutorQueryCall) Return(rows pgx.Rows, err error) *ExecutorQueryCall {
	c.returnRows = rows
	c.returnErr = err
	return c
}

func (c *ExecutorQueryCall) Run(run func(context.Context, string, ...any)) *ExecutorQueryCall {
	c.run = run
	return c
}

func (c *ExecutorQueryCall) RunAndReturn(fn func(context.Context, string, ...any) (pgx.Rows, error)) *ExecutorQueryCall {
	c.returnFn = fn
	return c
}

func (c *ExecutorQueryCall) Once() *ExecutorQueryCall {
	c.once = true
	return c
}

func (c *ExecutorQueryRowCall) Return(row pgx.Row) *ExecutorQueryRowCall {
	c.returnRow = row
	return c
}

func (c *ExecutorQueryRowCall) Run(run func(context.Context, string, ...any)) *ExecutorQueryRowCall {
	c.run = run
	return c
}

func (c *ExecutorQueryRowCall) RunAndReturn(fn func(context.Context, string, ...any) pgx.Row) *ExecutorQueryRowCall {
	c.returnFn = fn
	return c
}

func (c *ExecutorQueryRowCall) Once() *ExecutorQueryRowCall {
	c.once = true
	return c
}

func (m *Executor) findExecExpectation(ctx context.Context, sql string, arguments ...any) *ExecutorExecCall {
	for _, call := range m.execExpectations {
		if call.once && call.called {
			continue
		}
		if matchesCall(call.ctx, ctx, call.sql, sql, call.arguments, arguments) {
			return call
		}
	}
	m.t.Fatalf("unexpected Exec call with sql=%q args=%v", sql, arguments)
	return nil
}

func (m *Executor) findQueryExpectation(ctx context.Context, sql string, arguments ...any) *ExecutorQueryCall {
	for _, call := range m.queryExpectations {
		if call.once && call.called {
			continue
		}
		if matchesCall(call.ctx, ctx, call.sql, sql, call.arguments, arguments) {
			return call
		}
	}
	m.t.Fatalf("unexpected Query call with sql=%q args=%v", sql, arguments)
	return nil
}

func (m *Executor) findQueryRowExpectation(ctx context.Context, sql string, arguments ...any) *ExecutorQueryRowCall {
	for _, call := range m.queryRowExpectations {
		if call.once && call.called {
			continue
		}
		if matchesCall(call.ctx, ctx, call.sql, sql, call.arguments, arguments) {
			return call
		}
	}
	m.t.Fatalf("unexpected QueryRow call with sql=%q args=%v", sql, arguments)
	return nil
}

func (m *Executor) assertExpectations() {
	for _, call := range m.execExpectations {
		if !call.called {
			m.t.Errorf("expected Exec(%s, %s, %v) to be called", formatArg(call.ctx), formatArg(call.sql), call.arguments)
		}
	}
	for _, call := range m.queryExpectations {
		if !call.called {
			m.t.Errorf("expected Query(%s, %s, %v) to be called", formatArg(call.ctx), formatArg(call.sql), call.arguments)
		}
	}
	for _, call := range m.queryRowExpectations {
		if !call.called {
			m.t.Errorf("expected QueryRow(%s, %s, %v) to be called", formatArg(call.ctx), formatArg(call.sql), call.arguments)
		}
	}
}

func matchesCall(expectedCtx, actualCtx, expectedSQL, actualSQL any, expectedArgs, actualArgs []any) bool {
	if !matchesArg(expectedCtx, actualCtx) || !matchesArg(expectedSQL, actualSQL) {
		return false
	}
	if len(expectedArgs) != len(actualArgs) {
		return false
	}
	for i := range expectedArgs {
		if !matchesArg(expectedArgs[i], actualArgs[i]) {
			return false
		}
	}
	return true
}

//revive:enable:exported
