package main

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/uber-go/atomic"
)

type (
	updatesBenchConfig struct {
		NumWriters   int
		NumReaders   int
		ArraySize    int
		WritesPerSec int
	}

	updatesBencher struct {
		config updatesBenchConfig
	}
)

func newUpdatesBencher(
	config *updatesBenchConfig,
) *updatesBencher {
	return &updatesBencher{
		config: defaultConfig(config),
	}
}

func defaultConfig(config *updatesBenchConfig) updatesBenchConfig {
	if config == nil {
		config = &updatesBenchConfig{}
	}

	return updatesBenchConfig{
		NumWriters:   defaultInt(config.NumWriters, 1),
		NumReaders:   defaultInt(config.NumReaders, 1),
		ArraySize:    defaultInt(config.ArraySize, 128),
		WritesPerSec: defaultInt(config.WritesPerSec, 1024),
	}
}

func (ub *updatesBencher) Bench(
	b *testing.B,
	name string,
	factory ThreadsafeArrayFactory,
) {
	b.Run(name, func(b *testing.B) {
		ub.runBench(b, factory)
	})
}

func (ub *updatesBencher) runBench(
	b *testing.B,
	factory ThreadsafeArrayFactory,
) {
	ary := factory(ub.config.ArraySize)
	numAdds := atomic.NewInt64(0)

	doneFlag := atomic.NewBool(false)
	doneChan := make(chan struct{}, 1)
	signalDone := func() {
		doneFlag.Store(true)
		select {
		case doneChan <- struct{}{}:
		default:
		}
	}

	start := time.Now()
	b.StartTimer()

	for i := 0; i < ub.config.NumWriters; i++ {
		go func() {
			writesPerSec := float64(ub.config.WritesPerSec) / float64(ub.config.NumWriters)
			start := time.Now()
			timePerOp := float64(time.Second) / writesPerSec
			for i := 0; !doneFlag.Load(); i++ {
				ary.Add(1)
				numAdds.Add(1)
				now := time.Now()
				targetOps := float64(now.Sub(start)) / float64(time.Second) * writesPerSec
				opsOver := float64(i) - targetOps
				if opsOver > 0 {
					time.Sleep(time.Duration(opsOver * timePerOp))
				} else if i%128 == 0 {
					// pause briefly no matter what periodically to allow for fair
					// scheduling.
					time.Sleep(time.Nanosecond)
				}
			}
		}()
	}

	getsPerWriter := b.N / ub.config.NumReaders
	for i := 0; i < ub.config.NumReaders; i++ {
		go func() {
			for i := 0; i < getsPerWriter; i++ {
				safeUpdatesRef = ary.Get()
				if i%128 == 0 {
					// pause briefly no matter what periodically to allow for fair
					// scheduling.
					time.Sleep(time.Nanosecond)
				}
			}
			signalDone()
		}()
	}

	<-doneChan
	b.StopTimer()

	// ques (bs): any way to use the benchmark timer for this?
	end := time.Now()

	diff := end.Sub(start)
	adjusted := float64(time.Second) / float64(diff)

	b.ReportMetric(float64(b.N)*adjusted, "getsPerSec")
	b.ReportMetric(float64(numAdds.Load())*adjusted, "addsPerSec")
	b.ReportMetric(float64(diff)/float64(time.Second), "numSec")
	b.ReportMetric(float64(b.N), "numGets")
	b.ReportMetric(float64(numAdds.Load()), "numAdds")

}

// todo (bs): let's see if I can reliably extract this s.t. the updater can self
// host it. I'd guess there are other ways to guarantee this w/in a function.
var safeUpdatesRef []int

func Benchmark_safeUpdates(b *testing.B) {
	bencher := newUpdatesBencher(&updatesBenchConfig{
		NumWriters:   2,
		NumReaders:   2,
		ArraySize:    10_000,
		WritesPerSec: 10_000,
	})

	// so - this isn't a bad API per se. I will note that for the sake of
	// variance, I'd like to still perhaps have a better way to vary
	//
	// arguably, using a class here for performing the benchmarking is the wrong
	// approach - perhaps should figure out a way to inject values on a per-run
	// basis. Note that I will also need a way to inject these values within a
	// charting substructure eventually; but I needn't do that yet.
	//
	// also - note that while I'm reporting gross numbers; it does seem likely the
	// wrong value - the benchmark tries to tune itself so the numbers are all
	// comparable; but there can still be oddities. I'd guess a final adjustment
	// w.r.t. to time would be appropriate.
	//
	// I may wish to modify this so it actually is not run directly; but as part
	// of testing.Benchmark. That would give me more control over the output. I
	// think in practical terms though I'd have to move to a system of having a

	bencher.Bench(b, "MutexArray", NewMutexArray)
	bencher.Bench(b, "RMutexArray", NewRWMutexArray)
	bencher.Bench(b, "AtomicArray", NewSemiAtomicArray)
	bencher.Bench(b, "NoOpArray", NewNoOpArray)
}

func Test_chartFunctionality(t *testing.T) {
	f, copyErr := os.Create("output.png")
	if copyErr != nil {
		t.Fatal(copyErr)
	}

	r := graphBenchmarks(&ChartConfig{
		Title:  "It's a Title",
		XTitle: "The X Axis",
		YTitle: "The Y Axis",
	},
		ChartSeries{
			Name: "Sample #1",
			Points: []ChartPoint{
				ChartPoint{X: 1, Y: 7},
				ChartPoint{X: 2, Y: 4},
				ChartPoint{X: 3, Y: 3},
				ChartPoint{X: 4, Y: 2},
				ChartPoint{X: 7, Y: 2},
			},
		},
		ChartSeries{
			Name: "Sample #2",
			Points: []ChartPoint{
				ChartPoint{X: 1, Y: 1},
				ChartPoint{X: 2, Y: 2},
				ChartPoint{X: 3, Y: 3},
				ChartPoint{X: 4, Y: 4},
				ChartPoint{X: 7, Y: 7},
			},
		},
	)
	if _, copyErr := io.Copy(f, r); copyErr != nil {
		t.Fatal(copyErr)
	}
}
