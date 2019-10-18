package brun

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func Test_Set(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var sum int64
	incrSum := func() {
		atomic.AddInt64(&sum, 1)
	}
	getSum := func() int64 {
		return atomic.LoadInt64(&sum)
	}

	set := NewSet()

	for i := 0; i < 5; i++ {
		set.Add(func(context.Context) {
			incrSum()
		})
	}

	go func() {
		set.Run(ctx)
	}()
	time.Sleep(10 * time.Millisecond)

	if v := getSum(); v != 5 {
		t.Fatalf("Expected 5 additions; got %d", v)
	}

	for i := 0; i < 5; i++ {
		set.Add(func(context.Context) {
			incrSum()
		})
	}
	time.Sleep(10 * time.Millisecond)

	if v := getSum(); v != 10 {
		t.Fatalf("Expected 10 additions; got %d", v)
	}
}
