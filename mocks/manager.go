//revive:disable:exported
package mocks

import (
	"context"

	"github.com/virp/pgxtx"
)

type Manager struct {
	t testingT

	getExecutorExpectations []*ManagerGetExecutorCall
	withTxExpectations      []*ManagerWithTxCall
	execInTxExpectations    []*ManagerExecInTxCall
}

type ManagerExpecter struct {
	mock *Manager
}

type ManagerGetExecutorCall struct {
	mock       *Manager
	ctx        any
	returnExec pgxtx.Executor
	returnFn   func(context.Context) pgxtx.Executor
	run        func(context.Context)
	once       bool
	called     bool
}

type ManagerWithTxCall struct {
	mock      *Manager
	ctx       any
	fn        any
	returnErr error
	returnFn  func(context.Context, pgxtx.TxFunc) error
	run       func(context.Context, pgxtx.TxFunc)
	once      bool
	called    bool
}

type ManagerExecInTxCall struct {
	mock      *Manager
	ctx       any
	fn        any
	returnErr error
	returnFn  func(context.Context, func(context.Context, pgxtx.Executor) error) error
	run       func(context.Context, func(context.Context, pgxtx.Executor) error)
	once      bool
	called    bool
}

func NewManager(t testingT) *Manager {
	m := &Manager{t: t}
	t.Cleanup(func() { m.assertExpectations() })
	return m
}

func (m *Manager) EXPECT() *ManagerExpecter {
	return &ManagerExpecter{mock: m}
}

func (m *Manager) GetExecutor(ctx context.Context) pgxtx.Executor {
	call := m.findGetExecutorExpectation(ctx)
	if call.run != nil {
		call.run(ctx)
	}
	call.called = true
	if call.returnFn != nil {
		return call.returnFn(ctx)
	}
	return call.returnExec
}

func (m *Manager) WithTx(ctx context.Context, fn pgxtx.TxFunc) error {
	call := m.findWithTxExpectation(ctx, fn)
	if call.run != nil {
		call.run(ctx, fn)
	}
	call.called = true
	if call.returnFn != nil {
		return call.returnFn(ctx, fn)
	}
	return call.returnErr
}

func (m *Manager) ExecInTx(ctx context.Context, fn func(context.Context, pgxtx.Executor) error) error {
	call := m.findExecInTxExpectation(ctx, fn)
	if call.run != nil {
		call.run(ctx, fn)
	}
	call.called = true
	if call.returnFn != nil {
		return call.returnFn(ctx, fn)
	}
	return call.returnErr
}

func (e *ManagerExpecter) GetExecutor(ctx any) *ManagerGetExecutorCall {
	call := &ManagerGetExecutorCall{mock: e.mock, ctx: ctx}
	e.mock.getExecutorExpectations = append(e.mock.getExecutorExpectations, call)
	return call
}

func (e *ManagerExpecter) WithTx(ctx, fn any) *ManagerWithTxCall {
	call := &ManagerWithTxCall{mock: e.mock, ctx: ctx, fn: fn}
	e.mock.withTxExpectations = append(e.mock.withTxExpectations, call)
	return call
}

func (e *ManagerExpecter) ExecInTx(ctx, fn any) *ManagerExecInTxCall {
	call := &ManagerExecInTxCall{mock: e.mock, ctx: ctx, fn: fn}
	e.mock.execInTxExpectations = append(e.mock.execInTxExpectations, call)
	return call
}

func (c *ManagerGetExecutorCall) Return(exec pgxtx.Executor) *ManagerGetExecutorCall {
	c.returnExec = exec
	return c
}

func (c *ManagerGetExecutorCall) Run(run func(context.Context)) *ManagerGetExecutorCall {
	c.run = run
	return c
}

func (c *ManagerGetExecutorCall) RunAndReturn(fn func(context.Context) pgxtx.Executor) *ManagerGetExecutorCall {
	c.returnFn = fn
	return c
}

func (c *ManagerGetExecutorCall) Once() *ManagerGetExecutorCall {
	c.once = true
	return c
}

func (c *ManagerWithTxCall) Return(err error) *ManagerWithTxCall {
	c.returnErr = err
	return c
}

func (c *ManagerWithTxCall) Run(run func(context.Context, pgxtx.TxFunc)) *ManagerWithTxCall {
	c.run = run
	return c
}

func (c *ManagerWithTxCall) RunAndReturn(fn func(context.Context, pgxtx.TxFunc) error) *ManagerWithTxCall {
	c.returnFn = fn
	return c
}

func (c *ManagerWithTxCall) Once() *ManagerWithTxCall {
	c.once = true
	return c
}

func (c *ManagerExecInTxCall) Return(err error) *ManagerExecInTxCall {
	c.returnErr = err
	return c
}

func (c *ManagerExecInTxCall) Run(run func(context.Context, func(context.Context, pgxtx.Executor) error)) *ManagerExecInTxCall {
	c.run = run
	return c
}

func (c *ManagerExecInTxCall) RunAndReturn(fn func(context.Context, func(context.Context, pgxtx.Executor) error) error) *ManagerExecInTxCall {
	c.returnFn = fn
	return c
}

func (c *ManagerExecInTxCall) Once() *ManagerExecInTxCall {
	c.once = true
	return c
}

func (m *Manager) findGetExecutorExpectation(ctx context.Context) *ManagerGetExecutorCall {
	for _, call := range m.getExecutorExpectations {
		if call.once && call.called {
			continue
		}
		if matchesArg(call.ctx, ctx) {
			return call
		}
	}
	m.t.Fatalf("unexpected GetExecutor call")
	return nil
}

func (m *Manager) findWithTxExpectation(ctx context.Context, fn pgxtx.TxFunc) *ManagerWithTxCall {
	for _, call := range m.withTxExpectations {
		if call.once && call.called {
			continue
		}
		if matchesArg(call.ctx, ctx) && matchesArg(call.fn, fn) {
			return call
		}
	}
	m.t.Fatalf("unexpected WithTx call")
	return nil
}

func (m *Manager) findExecInTxExpectation(ctx context.Context, fn func(context.Context, pgxtx.Executor) error) *ManagerExecInTxCall {
	for _, call := range m.execInTxExpectations {
		if call.once && call.called {
			continue
		}
		if matchesArg(call.ctx, ctx) && matchesArg(call.fn, fn) {
			return call
		}
	}
	m.t.Fatalf("unexpected ExecInTx call")
	return nil
}

func (m *Manager) assertExpectations() {
	for _, call := range m.getExecutorExpectations {
		if !call.called {
			m.t.Errorf("expected GetExecutor(%s) to be called", formatArg(call.ctx))
		}
	}
	for _, call := range m.withTxExpectations {
		if !call.called {
			m.t.Errorf("expected WithTx(%s, %s) to be called", formatArg(call.ctx), formatArg(call.fn))
		}
	}
	for _, call := range m.execInTxExpectations {
		if !call.called {
			m.t.Errorf("expected ExecInTx(%s, %s) to be called", formatArg(call.ctx), formatArg(call.fn))
		}
	}
}

//revive:enable:exported
