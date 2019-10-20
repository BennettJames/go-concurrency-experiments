package brun

import (
	"context"
	"sync"
)

// Set is a way to handle a dynamic group of goroutines. Unlike Batch or
// Group, Set can have individual goroutines fail and be added while it is
// running. Upon shutdown, all goroutines will exit
type Set struct {
	stack *fnStack
}

// NewSet initializes the set. This must be called to safely initialize the set.
func NewSet() *Set {
	return &Set{
		stack: newFnStack(),
	}
}

// Run blocks and runs every function added to the set in a distinct goroutine.
// Anything added before or after this being called will run until the
// subfunction returns, or this is cancelled.
func (s *Set) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for {
		nextFn, err := s.stack.Next(ctx)
		if err != nil {
			return err
		}
		// todo (bs): I know errors are suppressed here, but I think panics should
		// likely still bubble back. Also, the other ones do a full wait for this to
		go func() {
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			nextFn(ctx)
		}()
	}
}

// Add will enqueue the given subfunction in the set. The function will run as
// long as the set is active. If the set is closed, this will successfully
// enqueue it but will never be run.
func (s *Set) Add(fn func(context.Context)) {
	s.stack.Add(fn)
}

// fnStack is a basic, (mostly) threadsafe stack for holding functions added to
// a set. Functions can be added, and be retrieved via `Next`. While there are
// no limitations on how many writers there can be at a time, the expectation is
// that there will only be one reader at a time.
type fnStack struct {
	l      sync.Mutex
	stack  []func(context.Context)
	notify chan (struct{})
}

// newFnStack initializes a new fnStack. This must be called to safely
// initialize the stack.
func newFnStack() *fnStack {
	return &fnStack{
		notify: make(chan struct{}, 1),
	}
}

// Add will put the given function on the stack, and notify any waiters that the
// queue state has changed.
func (s *fnStack) Add(fn func(context.Context)) {
	s.l.Lock()
	defer s.l.Unlock()

	s.stack = append(s.stack, fn)

	// Adds a notification to the queue. Falls through if it's full: the stack is
	// designed to only be read by one thread at a time
	select {
	case s.notify <- struct{}{}:
	default:
	}
}

// Next will wait until a value become available on the queue, or the provided
// context is cancelled. Will only return an error on cancellation (in which
// case the context error is returned).
func (s *fnStack) Next(
	ctx context.Context,
) (fn func(context.Context), err error) {
	for {
		if next, ok := s.tryPop(); ok {
			return next, nil
		}
		select {
		case <-s.notify:
			continue
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// tryPop will get and return the next element in the stack if it's available,
// and return false if the stack is empty.
func (s *fnStack) tryPop() (fn func(context.Context), ok bool) {
	s.l.Lock()
	defer s.l.Unlock()
	stackSize := len(s.stack)
	if stackSize == 0 {
		return nil, false
	}
	v := s.stack[stackSize-1]
	s.stack = s.stack[:stackSize-1]
	return v, true
}
