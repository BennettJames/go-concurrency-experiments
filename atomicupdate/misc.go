package main

import (
	"fmt"
	"strconv"
)

func defaultInt(v, def int) int {
	if v == 0 {
		return def
	}
	return v
}

func defaultStr(v, def string) string {
	if v == "" {
		return def
	}
	return v
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
