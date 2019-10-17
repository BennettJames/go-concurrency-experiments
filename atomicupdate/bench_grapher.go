package main

import (
	"fmt"
	"math"

	"github.com/wcharczuk/go-chart"
)

type (
	// ChartConfig allows for the configuration of a benchmark chart. Many values
	// will be assigned defaults when used to render a graph.
	ChartConfig struct {
		// Title is the title of the chart. No title will be given if this is not
		// set.
		Title string

		// XTitle is the title of the x-axis. No title will appear if this is not
		// set.
		XTitle string

		// YTitle is the title of the y-axis. No title will appear if this is not
		// set.
		YTitle string

		// ZeroBasis will make zero be the "basis" of the y-axis; if possible. If
		// all values are greater or equal to zero, then the y-axis will start at
		// zero. If all values are less than or equal to zero, then the top of the
		// y-axis will be zero. If there are values greater and less than zero, this
		// value is ignored.
		ZeroBasis bool

		// todo (bs): see about the possibility of allowing values to be
		// logarithmic.
	}

	// ChartSeries represents a set of data that will correspond to a line on the
	// chart.
	ChartSeries struct {
		// The name of the series, as it will appear in the legend.
		Name string

		// Points is the set of X/Y values that comprise the line. X-values should
		// be given in a linearly increasing fashion.
		Points []ChartPoint
	}

	// ChartPoint is a single point as it appears on the chart.
	ChartPoint struct {
		X, Y float64
	}
)

func graphBenchmarks(config ChartConfig, series ...ChartSeries) *chart.Chart {
	c := &charter{
		config: config,
		series: series,
	}
	return c.Get()
}

type charter struct {
	config ChartConfig
	series []ChartSeries
}

func (c *charter) Get() *chart.Chart {

	yRange := c.yRange()

	// todo (bs): see if the density of the axis can be scaled down to deal with

	var xFormatter chart.ValueFormatter = func(v interface{}) string {
		// fixme (bs): this is overly-clumsy. The formatting here should be
		// sensitive to smaller x values than this.
		return fmt.Sprintf("%.0f", approxFloat2(v.(float64)))
	}
	var yFormatter chart.ValueFormatter = func(v interface{}) string {
		if yRange.GetDelta() >= 100_000 {
			return fmt.Sprintf("%.1e", v)
		}
		return fmt.Sprintf("%v", v)
	}

	graph := &chart.Chart{
		Title:      c.config.Title,
		Background: c.bgStyle(),
		XAxis: chart.XAxis{
			Name:           c.config.XTitle,
			ValueFormatter: xFormatter,
		},
		YAxis: chart.YAxis{
			Name:           c.config.YTitle,
			ValueFormatter: yFormatter,
			Range:          yRange,
		},
		Series: c.chartSeries(),
	}
	graph.Elements = []chart.Renderable{
		chart.LegendLeft(graph),
	}
	return graph
}

func (c *charter) bgStyle() chart.Style {
	return chart.Style{
		Padding: chart.Box{
			Top:    20,
			Left:   110,
			Right:  20,
			Bottom: 20,
		},
	}
}

func (c *charter) yRange() chart.Range {
	minY, maxY := c.yMinMax()
	return &chart.ContinuousRange{
		Min: minY,
		Max: maxY,
	}
}

func (c *charter) yMinMax() (minY, maxY float64) {
	// get the min/max y values from the provided points
	minY, maxY = math.MaxFloat64, math.SmallestNonzeroFloat64
	pCount := 0
	for _, s := range c.series {
		for _, p := range s.Points {
			minY = math.Min(minY, p.Y)
			maxY = math.Max(maxY, p.Y)
			pCount++
		}
	}
	if pCount == 0 {
		minY, maxY = 0, 0
	}

	yPad := 0.0
	if minY == maxY {
		// go to a default pad of +- 1 if there is no variance
		yPad = 1
	} else {
		// by default, do a 5% pad over the range
		yPad = (maxY - minY) * 0.05
	}

	// If there is a zero basis; try to center the graph around 0
	if c.config.ZeroBasis {
		if minY >= 0 && maxY >= 0 {
			minY = 0
			maxY += yPad
			return
		} else if minY <= 0 && maxY <= 0 {
			maxY = 0
			minY -= yPad
			return
		}
	}

	// if no zero basis; pad equally in both directions
	//
	// todo (bs): strongly consider making it so this won't "cross zero".
	minY -= yPad
	maxY += yPad
	return
}

func (c *charter) chartSeries() (series []chart.Series) {
	for i, s := range c.series {
		series = append(series, chart.ContinuousSeries{
			Name: defaultStr(s.Name, fmt.Sprintf("Series-%d", i)),
			Style: chart.Style{
				DotWidth:    2,
				StrokeWidth: 1,
				// todo (bs): strongly consider setting my own dot/stroke colors
			},
			XValues: pointXVals(s.Points),
			YValues: pointYVals(s.Points),
		})
	}
	return
}

func pointXVals(points []ChartPoint) (xVals []float64) {
	for _, p := range points {
		xVals = append(xVals, p.X)
	}
	return
}

func pointYVals(points []ChartPoint) (yVals []float64) {
	for _, p := range points {
		yVals = append(yVals, p.Y)
	}
	return
}
