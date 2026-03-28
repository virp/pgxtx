//revive:disable:exported
package mocks

import (
	"context"
	"fmt"
	"reflect"

	"github.com/virp/pgxtx"
)

type testingT interface {
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
	Cleanup(func())
}

type ExecutorProvider struct {
	t                       testingT
	getExecutorExpectations []*ExecutorProviderGetExecutorCall
}

type ExecutorProviderExpecter struct {
	mock *ExecutorProvider
}

type ExecutorProviderGetExecutorCall struct {
	mock       *ExecutorProvider
	ctx        any
	returnExec pgxtx.Executor
	returnFn   func(context.Context) pgxtx.Executor
	run        func(context.Context)
	once       bool
	called     bool
}

func NewExecutorProvider(t testingT) *ExecutorProvider {
	m := &ExecutorProvider{t: t}
	t.Cleanup(func() { m.assertExpectations() })
	return m
}

func (m *ExecutorProvider) EXPECT() *ExecutorProviderExpecter {
	return &ExecutorProviderExpecter{mock: m}
}

func (m *ExecutorProvider) GetExecutor(ctx context.Context) pgxtx.Executor {
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

func (e *ExecutorProviderExpecter) GetExecutor(ctx any) *ExecutorProviderGetExecutorCall {
	call := &ExecutorProviderGetExecutorCall{mock: e.mock, ctx: ctx}
	e.mock.getExecutorExpectations = append(e.mock.getExecutorExpectations, call)
	return call
}

func (c *ExecutorProviderGetExecutorCall) Return(exec pgxtx.Executor) *ExecutorProviderGetExecutorCall {
	c.returnExec = exec
	return c
}

func (c *ExecutorProviderGetExecutorCall) Run(run func(context.Context)) *ExecutorProviderGetExecutorCall {
	c.run = run
	return c
}

func (c *ExecutorProviderGetExecutorCall) RunAndReturn(fn func(context.Context) pgxtx.Executor) *ExecutorProviderGetExecutorCall {
	c.returnFn = fn
	return c
}

func (c *ExecutorProviderGetExecutorCall) Once() *ExecutorProviderGetExecutorCall {
	c.once = true
	return c
}

func (m *ExecutorProvider) findGetExecutorExpectation(ctx context.Context) *ExecutorProviderGetExecutorCall {
	for _, call := range m.getExecutorExpectations {
		if call.once && call.called {
			continue
		}
		if matchesArg(call.ctx, ctx) {
			return call
		}
	}
	m.t.Fatalf("unexpected GetExecutor call with ctx=%v", ctx)
	return nil
}

func (m *ExecutorProvider) assertExpectations() {
	for _, call := range m.getExecutorExpectations {
		if !call.called {
			m.t.Errorf("expected GetExecutor(%s) to be called", formatArg(call.ctx))
		}
	}
}

func matchesArg(expected, actual any) bool {
	if _, ok := expected.(anythingMatcher); ok {
		return true
	}
	return reflect.DeepEqual(expected, actual)
}

func formatArg(v any) string {
	if _, ok := v.(anythingMatcher); ok {
		return "Anything"
	}
	return fmt.Sprintf("%#v", v)
}

//revive:enable:exported
