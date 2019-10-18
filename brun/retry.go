package brun

import (
	"context"
	"time"
)

// GapRetry wraps a Group.Add call with an automatic retry system. If fn
// terminates, GapRetry with restart the function after at least "gap" amount of
// time has passed since the last execution began.
//
// Note that the inner function has no error return. If the user of this
// function wishes to handle an error, it must be done within the function body.
// Panics will still propagate back to the group.
func GapRetry(
	gap time.Duration,
	fn func(ctx context.Context),
) func(ctx context.Context) error {
	if gap < 10*time.Millisecond {
		gap = 10 * time.Millisecond
	}

	return func(ctx context.Context) error {
		for {
			start := time.Now()
			execFnInContext(ctx, fn)
			end := time.Now()
			if ctx.Err() != nil {
				return ctx.Err()
			}
			retryIn := start.Sub(end) + gap
			select {
			case <-time.After(retryIn):
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

// ExpBackoffRetry offers a similar API to GapRetry, but instead of fixed-sized
// gaps between runs it uses exponential backoff.
//
// To determine if a run "failed", it will compare the time of the execution to
// min - if the runtime was less than min it's considered a failure and will
// trigger backoff; if it's more than min it's considered a success, and the
// wait time will be min.
func ExpBackoffRetry(
	min, max time.Duration,
	fn func(ctx context.Context),
) func(ctx context.Context) error {
	if min < 10*time.Millisecond {
		min = 10 * time.Millisecond
	}
	if max < min {
		max = min
	}

	return func(ctx context.Context) error {
		backoffIndex := 0
		for {
			start := time.Now()
			execFnInContext(ctx, fn)
			end := time.Now()
			runTime := end.Sub(start)
			if ctx.Err() != nil {
				return ctx.Err()
			}

			if runTime > min {
				backoffIndex = 0
			} else {
				backoffIndex++
			}
			gap := min
			for i := 1; i < backoffIndex; i++ {
				nextGap := gap * 2
				if nextGap > max {
					break
				}
				gap = nextGap
			}
			retryIn := start.Sub(end) + gap

			select {
			case <-time.After(retryIn):
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}
