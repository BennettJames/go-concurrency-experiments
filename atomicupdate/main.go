package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gosimple/slug"
	"github.com/wcharczuk/go-chart"
)

func main() {
	// Basic operation:
	//
	// This entry point will execute benchmarks, gather data, and output graphs
	// based on the results. Rather than perform benchmarks through the go tool's
	// testing facility, this invokes testing.Benchmark directly as to control and
	// analyze execution.
	//
	// Set's of data are prepared, normalized, labelled, then fed into the
	// go-chart library. This is used to generate simple line graphs of the data,
	// which is then output to the img/ directory.
	//
	// The control for what to graph and bench is not particularly elegant - just
	// comment/uncomment values in the "graphs" array.

	if err := os.MkdirAll(filepath.Join(".", "img"), 0777); err != nil {
		log.Fatal(err)
	}

	graphs := []*chart.Chart{
		benchMutexVsRWMutex2Writer10kSize(),
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
}
