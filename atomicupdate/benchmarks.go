package main

import (
	"testing"
	"time"

	"github.com/wcharczuk/go-chart"
)

func benchMutexVsRWMutex2Writer10kSize() *chart.Chart {
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
		factory ThreadsafeArrayFactory,
		numReaders int,
	) (points []ChartPoint) {
		for _, writesPerSec := range writeVals {
			points = append(points,
				calculatePoint(
					float64(writesPerSec),
					prepareBench(updatesBenchConfig{
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

	return graphBenchmarks(
		ChartConfig{
			Title:     "Mutex vs RWMutex - 2 Writers, 10K Size",
			XTitle:    "Writes/Sec",
			YTitle:    "Reads/Sec",
			ZeroBasis: true,
		},
		ChartSeries{
			Name:   "Mutex-2",
			Points: getPoints(NewMutexArray, 2),
		},
		ChartSeries{
			Name:   "RWMutex-2",
			Points: getPoints(NewRWMutexArray, 2),
		},
		ChartSeries{
			Name:   "Mutex-4",
			Points: getPoints(NewMutexArray, 4),
		},
		ChartSeries{
			Name:   "RWMutex-4",
			Points: getPoints(NewRWMutexArray, 4),
		},
		ChartSeries{
			Name:   "Mutex-6",
			Points: getPoints(NewMutexArray, 6),
		},
		ChartSeries{
			Name:   "RWMutex-6",
			Points: getPoints(NewRWMutexArray, 6),
		},
	)
}

func benchMutexVsDeferMutex2Writer10kSize() *chart.Chart {
	writeVals := []int{
		100,
		5_000,
		10_000,
		20_000,
	}

	getPoints := func(
		factory ThreadsafeArrayFactory,
	) (points []ChartPoint) {
		for _, writesPerSec := range writeVals {
			points = append(points,
				calculatePoint(
					float64(writesPerSec),
					prepareBench(updatesBenchConfig{
						NumWriters:   2,
						NumReaders:   2,
						ArraySize:    10_000,
						WritesPerSec: writesPerSec,
						Factory:      factory,
					}),
				))
		}
		return
	}

	return graphBenchmarks(
		ChartConfig{
			Title:     "Mutex vs Defer Mutex - 2 Writers, 10K Size",
			XTitle:    "Writes/Sec",
			YTitle:    "Reads/Sec",
			ZeroBasis: true,
		},
		ChartSeries{
			Name:   "Mutex-2",
			Points: getPoints(NewMutexArray),
		},
		ChartSeries{
			Name:   "DeferMutex-2",
			Points: getPoints(NewDeferMutexArray),
		},
	)
}

func benchMutexVsSemiAtomic2Writer10kSize() *chart.Chart {

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
		factory ThreadsafeArrayFactory,
		numReaders int,
	) (points []ChartPoint) {
		for _, writesPerSec := range writeVals {
			points = append(points,
				calculatePoint(
					float64(writesPerSec),
					prepareBench(updatesBenchConfig{
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

	return graphBenchmarks(
		ChartConfig{
			Title:     "Mutex vs SemiAtomic - 2 Writers, 10K Size",
			XTitle:    "Writes/Sec",
			YTitle:    "Reads/Sec",
			ZeroBasis: true,
		},
		ChartSeries{
			Name:   "Mutex-2",
			Points: getPoints(NewMutexArray, 2),
		},
		ChartSeries{
			Name:   "Atomic-2",
			Points: getPoints(NewSemiAtomicArray, 2),
		},
		ChartSeries{
			Name:   "Mutex-4",
			Points: getPoints(NewMutexArray, 4),
		},
		ChartSeries{
			Name:   "Atomic-4",
			Points: getPoints(NewSemiAtomicArray, 4),
		},
		ChartSeries{
			Name:   "Mutex-6",
			Points: getPoints(NewMutexArray, 6),
		},
		ChartSeries{
			Name:   "Atomic-6",
			Points: getPoints(NewSemiAtomicArray, 6),
		},
	)
}

func calculatePoint(x float64, bFn func(b *testing.B) float64) ChartPoint {
	var y float64
	res := testing.Benchmark(func(b *testing.B) {
		y = bFn(b)
	})
	// adjusts the final rate so it's a per-second value
	adjustment := float64(time.Second) / float64(res.T)
	return ChartPoint{
		X: approxFloat3(x),
		Y: approxFloat3(y * adjustment),
	}
}
