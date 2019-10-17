package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/gosimple/slug"
	"github.com/wcharczuk/go-chart"
)

func main() {

	if err := os.MkdirAll(filepath.Join(".", "img"), 0777); err != nil {
		log.Fatal(err)
	}

	graphs := []*chart.Chart{
		// benchMutexVsRWMutex2Writer10kSize(),
		// benchMutexVsDeferMutex2Writer10kSize(),
		// benchMutexVsSemiAtomic2Writer10kSize(),
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
