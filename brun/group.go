package brun

import (
	"context"
)

// Group is a way to execute a set of long-running service together.
type Group struct {
	queue fnQueue
}

// Add will include the given function
func (g *Group) Add(fn func(ctx context.Context) error) {
	g.queue.push(fn)
}

// Run executes every stored function in parallel. Upon cancellation or a stored
// function returning/panicing, all stored functions will receive a cancellation
// in their context. Once all functions have returned, this will then complete
// with either a panic, an unexpected error, or a cancellation error, depending
// on how the termination occurs.
func (g *Group) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	queue := g.queue.get()
	return performGroupRun(ctx, queue)
}

func GroupRunner(
	fns ...func(context.Context) error,
) func(context.Context) error {
	return func(ctx context.Context) error {
		return performGroupRun(ctx, fns)
	}
}

func GroupRun(
	ctx context.Context,
	fns ...func(ctx context.Context) error,
) error {
	return performGroupRun(ctx, fns)
}

func performGroupRun(
	ctx context.Context,
	fns []func(ctx context.Context) error,
) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// If the group is empty, use a dummy that exits upon cancellation.
	if len(fns) == 0 {
		fns = []func(ctx context.Context) error{
			func(ctx context.Context) error {
				<-ctx.Done()
				return ctx.Err()
			},
		}
	}

	// errChan is large enough to hold response values from every member of the
	// queue. This ensures that even when not every message is processed, no
	// goroutine is blocked on writing to an orphaned channel.
	//
	// ques (bs): does this still need to be this size now that all returns are
	// mandated to be returned? Possibly not.
	errChan := make(chan runErr, len(fns))

	for _, queuedFn := range fns {
		fn := queuedFn
		go func() {
			var err error
			defer func() {
				if r := recover(); r != nil {
					errChan <- runErr{
						panic: r,
					}
				} else if err != nil {
					errChan <- runErr{
						err: err,
					}
				} else {
					errChan <- runErr{}
				}
			}()
			err = execErrFnInContext(ctx, fn)
		}()
	}

	// todo (bs): while I like the idea of keeping this dependency-free, I think
	// multi-error would make more sense rather than trying to side-load a
	var firstErr error
	var firstPanic interface{}
	for range fns {
		select {
		case re := <-errChan:
			cancel()
			if re.panic != nil {
				if firstPanic == nil {
					firstPanic = re.panic
				} else {
					// todo (bs): consider logging the returned value here; perhaps with a
					// globally configurable logger (that could of course be set to mute
					// logs)
				}
			} else if re.err != nil {
				if firstErr == nil {
					firstErr = re.err
				} else if re.err != ctx.Err() {
					// todo (bs): if the error is not a context error and a default logger
					// has been configured, consider using the default logger here. Also:
					// should this be compared to context.Canceled, or ctx.Err() being
					// given?
				}
			}
		}
	}
	if firstPanic != nil {
		panic(firstPanic)
	}
	// note (bs): is the context is cancelled and the first response from a fn is
	// a non-context error, it would be returned rather than the fn error. Is that
	// ok? I think so, but let's think on it a little
	if firstErr != nil {
		return firstErr
	}
	return ctx.Err()
}
