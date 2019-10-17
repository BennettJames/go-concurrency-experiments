package main

import (
	"github.com/wcharczuk/go-chart"
)

func benchMutexVsRWMutex2Writer10kSize() *chart.Chart {

	// ques (bs): would it make sense to explicitly curry here, given that you are
	// in some sense creating top-down fanout of configurable properties? Maybe?
	// On the other hand, could just take the perspective that you want to save
	// some repetition, some copying's fine, and I should get on with it.

	// (factory) -> (numReaders) -> (writeVals) -> points

	writeVals := []int{
		100,
		1_000,
		5_000,
		10_000,
		15_000,
		20_000,
		25_000,
		30_000,
		35_000,
	}

	getPoints := func(
		numReaders int,
		factory ThreadsafeArrayFactory,
	) (points []ChartPoint) {
		for _, writesPerSec := range writeVals {
			points = append(points,
				calculatePoint(
					float64(writesPerSec),
					prepareBench(&updatesBenchConfig{
						NumWriters:   2,
						NumReaders:   numReaders,
						ArraySize:    10_000,
						WritesPerSec: writesPerSec,
						Factory:      factory,
					}),
				))
		}
		return
	}

	series := []ChartSeries{
		ChartSeries{
			Name:   "Mutex-2",
			Points: getPoints(2, NewMutexArray),
		},
		ChartSeries{
			Name:   "RWMutex-2",
			Points: getPoints(2, NewRWMutexArray),
		},
		ChartSeries{
			Name:   "Mutex-4",
			Points: getPoints(4, NewMutexArray),
		},
		ChartSeries{
			Name:   "RWMutex-4",
			Points: getPoints(4, NewRWMutexArray),
		},
		ChartSeries{
			Name:   "Mutex-6",
			Points: getPoints(6, NewMutexArray),
		},
		ChartSeries{
			Name:   "RWMutex-6",
			Points: getPoints(6, NewRWMutexArray),
		},
	}

	// note (bs): a lot of these interface boundaries are pretty rough; they need
	// to be revisited. Also - I think the file names need to be locally bound in
	// some sense. Perhaps I should just kinda cheat and use token munging.
	// Alternatively, I could just stick with doing fanout-without-returning for
	// the time being.
	c := &charter{
		config: normalizeChartConfig(&ChartConfig{
			Title:  "Mutex vs RWMutex - 2 Writers, 10K Size",
			XTitle: "Writes/Sec",
			YTitle: "Reads/Sec",
		}),
		series: series,
	}
	return c.Get()
}
