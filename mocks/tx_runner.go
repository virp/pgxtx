//revive:disable:exported
package mocks

import (
	"context"

	"github.com/virp/pgxtx"
)

type TxRunner struct {
	t testingT

	withTxExpectations   []*TxRunnerWithTxCall
	execInTxExpectations []*TxRunnerExecInTxCall
}

type TxRunnerExpecter struct {
	mock *TxRunner
}

type TxRunnerWithTxCall struct {
	mock      *TxRunner
	ctx       any
	fn        any
	returnErr error
	returnFn  func(context.Context, pgxtx.TxFunc) error
	run       func(context.Context, pgxtx.TxFunc)
	once      bool
	called    bool
}

type TxRunnerExecInTxCall struct {
	mock      *TxRunner
	ctx       any
	fn        any
	returnErr error
	returnFn  func(context.Context, func(context.Context, pgxtx.Executor) error) error
	run       func(context.Context, func(context.Context, pgxtx.Executor) error)
	once      bool
	called    bool
}

func NewTxRunner(t testingT) *TxRunner {
	m := &TxRunner{t: t}
	t.Cleanup(func() { m.assertExpectations() })
	return m
}

func (m *TxRunner) EXPECT() *TxRunnerExpecter {
	return &TxRunnerExpecter{mock: m}
}

func (m *TxRunner) WithTx(ctx context.Context, fn pgxtx.TxFunc) error {
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

func (m *TxRunner) ExecInTx(ctx context.Context, fn func(context.Context, pgxtx.Executor) error) error {
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

func (e *TxRunnerExpecter) WithTx(ctx, fn any) *TxRunnerWithTxCall {
	call := &TxRunnerWithTxCall{mock: e.mock, ctx: ctx, fn: fn}
	e.mock.withTxExpectations = append(e.mock.withTxExpectations, call)
	return call
}

func (e *TxRunnerExpecter) ExecInTx(ctx, fn any) *TxRunnerExecInTxCall {
	call := &TxRunnerExecInTxCall{mock: e.mock, ctx: ctx, fn: fn}
	e.mock.execInTxExpectations = append(e.mock.execInTxExpectations, call)
	return call
}

func (c *TxRunnerWithTxCall) Return(err error) *TxRunnerWithTxCall {
	c.returnErr = err
	return c
}

func (c *TxRunnerWithTxCall) Run(run func(context.Context, pgxtx.TxFunc)) *TxRunnerWithTxCall {
	c.run = run
	return c
}

func (c *TxRunnerWithTxCall) RunAndReturn(fn func(context.Context, pgxtx.TxFunc) error) *TxRunnerWithTxCall {
	c.returnFn = fn
	return c
}

func (c *TxRunnerWithTxCall) Once() *TxRunnerWithTxCall {
	c.once = true
	return c
}

func (c *TxRunnerExecInTxCall) Return(err error) *TxRunnerExecInTxCall {
	c.returnErr = err
	return c
}

func (c *TxRunnerExecInTxCall) Run(run func(context.Context, func(context.Context, pgxtx.Executor) error)) *TxRunnerExecInTxCall {
	c.run = run
	return c
}

func (c *TxRunnerExecInTxCall) RunAndReturn(fn func(context.Context, func(context.Context, pgxtx.Executor) error) error) *TxRunnerExecInTxCall {
	c.returnFn = fn
	return c
}

func (c *TxRunnerExecInTxCall) Once() *TxRunnerExecInTxCall {
	c.once = true
	return c
}

func (m *TxRunner) findWithTxExpectation(ctx context.Context, fn pgxtx.TxFunc) *TxRunnerWithTxCall {
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

func (m *TxRunner) findExecInTxExpectation(ctx context.Context, fn func(context.Context, pgxtx.Executor) error) *TxRunnerExecInTxCall {
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

func (m *TxRunner) assertExpectations() {
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
