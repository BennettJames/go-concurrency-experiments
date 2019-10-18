package brun

import (
	"context"
	"sync"
)

type plainFn func(ctx context.Context)
type errFn func(ctx context.Context) error

// runErr is a simple record of a completed execution, and any panics or
// errors that were returned or raised.
type runErr struct {
	err   error
	panic interface{}
}

// execFnInContext runs the given function with a cancel that's executed
// immediately after the function exits.
func execFnInContext(ctx context.Context, fn func(ctx context.Context)) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	fn(ctx)
}

// execErrFnInContext runs the given function with a cancel that's executed
// immediately after the function exits.
func execErrFnInContext(
	ctx context.Context,
	fn func(ctx context.Context) error,
) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	return fn(ctx)
}

// fnQueue is a simple threadsafe way to store and retrieve a set of functions.
type fnQueue struct {
	l     sync.Mutex
	queue []func(ctx context.Context) error
}

func (q *fnQueue) push(fn func(ctx context.Context) error) {
	q.l.Lock()
	defer q.l.Unlock()
	q.queue = append(q.queue, fn)
}

func (q *fnQueue) get() []func(ctx context.Context) error {
	q.l.Lock()
	defer q.l.Unlock()
	return q.queue[:]
}
