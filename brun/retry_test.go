package brun

import (
	"context"
	"testing"
	"time"
)

func Test_GapRetry(t *testing.T) {

	t.Run("waitsToRetry", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		g := &Group{}
		execTimes := []time.Time{}
		gap := 20 * time.Millisecond
		g.Add(GapRetry(gap, func(ctx context.Context) {
			execTimes = append(execTimes, time.Now())
			time.Sleep(5 * time.Millisecond)
		}))
		err := g.Run(ctx)
		if err != context.DeadlineExceeded {
			t.Fatalf("unexpected error: %s\n", err)
		}
		var lastTime time.Time
		for i, thisTime := range execTimes {
			if i > 0 {
				if !approximatelyApartBy(lastTime, thisTime, gap) {
					t.Fatalf("Execution at index %d not sufficiently far apart", i)
				}
			}
			lastTime = thisTime
		}
	})

	t.Run("retriesImmediatelyAfterGap", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		g := &Group{}
		execTimes := []time.Time{}
		runTime := 40 * time.Millisecond
		g.Add(GapRetry(20*time.Millisecond, func(ctx context.Context) {
			execTimes = append(execTimes, time.Now())
			time.Sleep(runTime)
		}))
		err := g.Run(ctx)
		if err != context.DeadlineExceeded {
			t.Fatalf("unexpected error: %s\n", err)
		}
		var lastTime time.Time
		for i, thisTime := range execTimes {
			if i > 0 {
				if !approximatelyApartBy(lastTime, thisTime, runTime) {
					t.Fatalf(
						"Execution at index %d not sufficiently far apart (diff: %s, expected: %s)",
						i, thisTime.Sub(lastTime), runTime)
				}
			}
			lastTime = thisTime
		}
	})
}

func Test_ExpBackoffRetry(t *testing.T) {

	t.Run("gapsGrowToLimit", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer cancel()

		g := &Group{}
		execTimes := []time.Time{}
		min, max := 20*time.Millisecond, 80*time.Millisecond
		g.Add(ExpBackoffRetry(min, max, func(ctx context.Context) {
			execTimes = append(execTimes, time.Now())
			time.Sleep(5 * time.Millisecond)
		}))
		err := g.Run(ctx)
		if err != context.DeadlineExceeded {
			t.Fatalf("unexpected error: %s\n", err)
		}

		if len(execTimes) < 5 {
			t.Fatalf("Expected at least 5 retries; got %d", len(execTimes))
		}
		expectations := []time.Duration{
			20 * time.Millisecond,
			40 * time.Millisecond,
			80 * time.Millisecond,
			80 * time.Millisecond,
		}
		for i, gap := range expectations {
			elapsed := execTimes[i+1].Sub(execTimes[i])
			if !approximatelyApartBy(execTimes[i], execTimes[i+1], gap) {
				t.Fatalf(
					"Execution at index %d not sufficiently far apart (diff: %s, expected: %s)",
					i, elapsed, gap)
			}
		}
	})

	t.Run("resetsIfRunsSufficientlyFarApart", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer cancel()

		g := &Group{}
		execTimes := []time.Time{}
		min, max := 20*time.Millisecond, 80*time.Millisecond
		g.Add(ExpBackoffRetry(min, max, func(ctx context.Context) {
			execTimes = append(execTimes, time.Now())
			time.Sleep(30 * time.Millisecond)
		}))
		err := g.Run(ctx)
		if err != context.DeadlineExceeded {
			t.Fatalf("unexpected error: %s\n", err)
		}

		if len(execTimes) < 5 {
			t.Fatalf("Expected at least 5 retries; got %d", len(execTimes))
		}
		expectations := []time.Duration{
			30 * time.Millisecond,
			30 * time.Millisecond,
			30 * time.Millisecond,
			30 * time.Millisecond,
		}
		for i, gap := range expectations {
			elapsed := execTimes[i+1].Sub(execTimes[i])
			if !approximatelyApartBy(execTimes[i], execTimes[i+1], gap) {
				t.Fatalf(
					"Execution at index %d not sufficiently far apart (diff: %s, expected: %s)",
					i, elapsed, gap)
			}
		}
	})
}

type printFunc func()

// approximatelyApartBy indicates if the two times are about "apart" within
// ~5ms.
func approximatelyApartBy(t1, t2 time.Time, apart time.Duration) bool {
	diff := t2.Sub(t1)
	return diff >= apart-6*time.Millisecond && diff <= apart+6*time.Millisecond
}
