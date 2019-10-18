package brun

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func Test_Group(t *testing.T) {
	t.Run("basicBehavior", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		g := Group{}
		var runCount uint64
		g.Add(func(ctx context.Context) error {
			atomic.AddUint64(&runCount, 1)
			<-ctx.Done()
			return ctx.Err()
		})
		g.Add(func(ctx context.Context) error {
			atomic.AddUint64(&runCount, 1)
			<-ctx.Done()
			return ctx.Err()
		})
		err := g.Run(ctx)

		if err != context.DeadlineExceeded {
			t.Fatalf("unexpected error: %s\n", err)
		}
		finalCount := atomic.LoadUint64(&runCount)
		if finalCount != 2 {
			t.Fatalf("bad run count: %d\n", finalCount)
		}
	})

	t.Run("RunGroupFrom", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		var runCount uint64
		err := GroupRun(
			ctx,
			func(ctx context.Context) error {
				atomic.AddUint64(&runCount, 1)
				<-ctx.Done()
				return ctx.Err()
			},
			func(ctx context.Context) error {
				atomic.AddUint64(&runCount, 1)
				<-ctx.Done()
				return ctx.Err()
			},
		)
		if err != context.DeadlineExceeded {
			t.Fatalf("unexpected error: %s\n", err)
		}
		finalCount := atomic.LoadUint64(&runCount)
		if finalCount != 2 {
			t.Fatalf("bad run count: %d\n", finalCount)
		}
	})

	t.Run("emptySet", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		start := time.Now()
		go func() {
			time.Sleep(10 * time.Millisecond)
			cancel()
		}()

		g := &Group{}
		err := g.Run(ctx)
		end := time.Now()
		if err != context.Canceled {
			t.Fatalf("expected cancel error, got %s", err)
		}

		runTime := end.Sub(start)
		if runTime < 10*time.Millisecond {
			t.Fatalf("run should have lasted 10 ms, got %s", runTime)
		}
	})

	t.Run("propagatesUnexpectedError", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		innerErr := errors.New("this is an error")

		g := Group{}
		g.Add(func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		})
		g.Add(func(ctx context.Context) error {
			return innerErr
		})
		runErr := g.Run(ctx)

		if runErr != innerErr {
			t.Fatalf("error from group should be returned, got %s\n", runErr)
		}
	})

	t.Run("propagatesPanic", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		panicVal := "this is my panic"
		g := Group{}
		g.Add(func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		})
		g.Add(func(ctx context.Context) error {
			panic(panicVal)
		})
		runPanic := getPanic(ctx, func(ctx context.Context) {
			t.Fatal(g.Run(ctx))
		})

		if runPanic != panicVal {
			t.Fatalf("panic from group should be returned, got %v\n", runPanic)
		}
	})
}

func getPanic(ctx context.Context, fn plainFn) interface{} {
	panicChan := make(chan interface{}, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				panicChan <- r
			} else {
				panicChan <- nil
			}
		}()
		execFnInContext(ctx, fn)
	}()
	select {
	case p := <-panicChan:
		return p
	case <-ctx.Done():
		return nil
	}
}
