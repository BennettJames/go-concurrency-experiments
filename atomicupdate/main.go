package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/gosimple/slug"
	"github.com/wcharczuk/go-chart"
)

func main() {

	if err := os.MkdirAll(filepath.Join(".", "img"), 0777); err != nil {
		log.Fatal(err)
	}

	graphs := []*chart.Chart{
		// benchMutexVsRWMutex2Writer10kSize(),
	}

	for i, g := range graphs {
		name := fmt.Sprintf("graphs1-%d.png", i)
		slugTitle := slug.Make(g.Title)
		if len(slugTitle) > 0 {
			name = fmt.Sprintf("%s.png", slugTitle)
		}
		f, copyErr := os.Create(filepath.Join("img", name))
		if copyErr != nil {
			log.Fatal(copyErr)
		}
		g.Render(chart.PNG, f)
	}

	os.Exit(0)

	f, copyErr := os.Create("output.png")
	if copyErr != nil {
		log.Fatal(copyErr)
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
		log.Fatal(copyErr)
	}

}

func calculatePoint(x float64, bFn func(b *testing.B) float64) ChartPoint {
	var y float64
	res := testing.Benchmark(func(b *testing.B) {
		y = bFn(b)
	})
	adjustment := float64(time.Second) / float64(res.T)

	// note (bs): I'm kind of cheating by flooring here. What I really want is to
	// round significant digits.
	return ChartPoint{
		X: approxFloat3(x * adjustment),
		Y: approxFloat3(y * adjustment),
	}
}

func approxFloat2(x float64) float64 {
	if v, err := strconv.ParseFloat(fmt.Sprintf("%.2g", x), 64); err == nil {
		return v
	}
	return x
}

func approxFloat3(x float64) float64 {
	if v, err := strconv.ParseFloat(fmt.Sprintf("%.3g", x), 64); err == nil {
		return v
	}
	return x
}
