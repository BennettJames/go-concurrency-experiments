package main

import (
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
		Factory      ThreadsafeArrayFactory
	}
)

// todo (bs): let's see if I can reliably extract this s.t. the updater can self
// host it. I'd guess there are other ways to guarantee this w/in a function.
var safeUpdatesRef []int

// prepareBench sets up a benchmark with the given configurations. Upon
// execution, it will run and return the number of executions it was able to
// perform.
func prepareBench(
	configOptions updatesBenchConfig,
) func(b *testing.B) float64 {
	config := normalizeBenchConfig(configOptions)
	return func(b *testing.B) float64 {
		ary := config.Factory(config.ArraySize)
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

		for i := 0; i < config.NumWriters; i++ {
			go func() {
				writesPerSec := float64(config.WritesPerSec) / float64(config.NumWriters)
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

		getsPerWriter := b.N / config.NumReaders
		for i := 0; i < config.NumReaders; i++ {
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
		return float64(b.N)
	}
}

func normalizeBenchConfig(config updatesBenchConfig) updatesBenchConfig {
	return updatesBenchConfig{
		NumWriters:   defaultInt(config.NumWriters, 1),
		NumReaders:   defaultInt(config.NumReaders, 1),
		ArraySize:    defaultInt(config.ArraySize, 128),
		WritesPerSec: defaultInt(config.WritesPerSec, 1024),
		Factory:      config.Factory,
	}
}
