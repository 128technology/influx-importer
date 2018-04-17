package client

// MetricDescriptor describes a metric within the system.
type MetricDescriptor struct {
	ID          string
	Description string
	Keys        []string
}

// MetricPermutation describes a series of parameters and their values
// that can be used to retrieve a metric
type MetricPermutation struct {
	Parameters map[string]string
}