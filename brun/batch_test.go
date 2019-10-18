package brun

import (
	"context"
	"errors"
	"testing"
	"time"
)

func Test_emptyBatch(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	l := Batch{}

	if err := l.Run(ctx); err != nil {
		t.Fatalf("Error in latch run: %s", err)
	}
}

func Test_simpleBatch(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	l := Batch{}

	v1 := 0
	l.Add(func(ctx context.Context) error {
		v1 = 1
		return nil
	})

	v2 := 0
	l.Add(func(ctx context.Context) error {
		v2 = 2
		return nil
	})

	if err := l.Run(ctx); err != nil {
		t.Fatalf("Error in latch run: %s", err)
	}
	if v1 != 1 || v2 != 2 {
		t.Fatal("Run failed to set value")
	}
}

func Test_errorLatch(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	l := Batch{}

	baseErr := errors.New("an error")
	l.Add(func(ctx context.Context) error {
		return baseErr
	})

	// todo (bs): should test to ensure that an error in one queued function
	// causes the others to be cancelled.

	if err := l.Run(ctx); err != baseErr {
		t.Fatalf("Expected error to propagate in exec; got %s", err)
	}
}

func Test_cancelBatch(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	l := Batch{}

	// todo (bs): this is a pretty shallow test of cancellation; should go deeper
	// here.

	finishedChan := make(chan struct{}, 8)

	l.Add(func(ctx context.Context) error {
		<-ctx.Done()
		finishedChan <- struct{}{}
		return ctx.Err()
	})

	cancel()

	if err := l.Run(ctx); err != context.Canceled {
		t.Fatalf("Expected canceled error; got %s", err)
	}

	<-finishedChan
}

func Test_panicBatch(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	b := Batch{}

	batchPanic := "panic message"
	b.Add(func(ctx context.Context) error {
		panic(batchPanic)
	})
	b.Add(func(ctx context.Context) error {
		select {
		case <-time.After(100 * time.Millisecond):
			return errors.New("expected cancel")
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	execPanic := getPanic(ctx, func(ctx context.Context) {
		t.Fatal(b.Run(ctx))
	})

	if execPanic != batchPanic {
		t.Fatal("Expected panic to be propagated; got ", execPanic)
	}
}
