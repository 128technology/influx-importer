package client

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

// AnalyticRequest represents a request for an analytic
// given a metric name, the transform, any parameters and a time window.
type AnalyticRequest struct {
	Metric     string              `json:"metric"`
	Transform  string              `json:"transform"`
	Parameters []AnalyticParameter `json:"parameters"`
	Window     AnalyticWindow      `json:"window"`
}
