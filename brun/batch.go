package brun

import (
	"context"
)

// Batch provides a means by which to execute several goroutines in parallel.
type Batch struct {
	queue fnQueue
}

// Add a job to the batch that will be called when `Exec` is invoked.
func (b *Batch) Add(fn func(ctx context.Context) error) {
	b.queue.push(fn)
}

// Run performs all queued actions. Any errors from a queued function will
// immediately cancel all jobs and return.
func (b *Batch) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// todo (bs): consider setting a value here to ensure no double-runs.

	// note (bs): should do a little more research to guarantee this is safe. I'd
	// bet it is, but now that waitgroup is not used my original research into the
	// matter is not valid.

	queue := b.queue.get()
	errChan := make(chan runErr, len(queue))

	for _, queuedFn := range queue {
		fn := queuedFn
		go func() {
			var err error
			defer func() {
				if r := recover(); r != nil {
					errChan <- runErr{
						panic: r,
					}
				} else {
					errChan <- runErr{
						err: err,
					}
				}
			}()
			err = execErrFnInContext(ctx, fn)
		}()
	}

	var firstErr error
	var firstPanic interface{}
	for range queue {
		select {
		case re := <-errChan:
			if re.panic != nil {
				cancel()
				if firstPanic == nil {
					firstPanic = re.panic
				} else {
					// todo (bs): consider logging the returned value here; perhaps with a
					// globally configurable logger (that could of course be set to mute
					// logs)
				}
			} else if re.err != nil {
				cancel()
				if firstErr == nil {
					firstErr = re.err
				} else if re.err != context.Canceled {
					// todo (bs): if the error is not a context error and a default logger
					// has been configured, consider using the default logger here.
				}
			}
		}
	}
	if firstPanic != nil {
		panic(firstPanic)
	}
	if firstErr != nil {
		return firstErr
	}
	return ctx.Err()
}
