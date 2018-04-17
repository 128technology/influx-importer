package client

import (
	"fmt"
	"strings"
)

// AnalyticPoint represents a point in time
type AnalyticPoint struct {
	Value float64 `json:"value"`
	Time  string  `json:"date"`
}

// AnalyticParameter represents a key value parameter
type AnalyticParameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// AnalyticWindow represents a start and end time window
type AnalyticWindow struct {
	End   string `json:"end"`
	Start string `json:"start"`
}

// AnalyticMetricFilter represents a filter map of keys and values
type AnalyticMetricFilter map[string]string

// AnalyticMetricRequest defines the request for a metric
type AnalyticMetricRequest struct {
	ID        string                 `json:"id"`
	Transform string                 `json:"transform"`
	Window    AnalyticWindow         `json:"window"`
	Filters   []AnalyticMetricFilter `json:"filters"`
}

// ToString converts a filter map to a string for debug
func (filter AnalyticMetricFilter) ToString() string {
	var str []string
	for key, val := range filter {
		str = append(str, fmt.Sprintf("%v=%v", key, val))
	}
	return strings.Join(str, ",")
}
