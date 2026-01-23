package netedge

import "time"

// PrometheusResponse represents the standard response format from Prometheus API
type PrometheusResponse struct {
	Status    string   `json:"status"`
	Data      Data     `json:"data,omitempty"`
	ErrorType string   `json:"errorType,omitempty"`
	Error     string   `json:"error,omitempty"`
	Warnings  []string `json:"warnings,omitempty"`
}

type Data struct {
	ResultType string   `json:"resultType"`
	Result     []Result `json:"result"`
}

type Result struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value,omitempty"`  // For vector
	Values [][]interface{}   `json:"values,omitempty"` // For matrix
}

// Helper types for easier consumption if needed,
// though interface{} for values is flexible for the JSON unmarshaling
type MetricValue struct {
	Timestamp time.Time
	Value     string
}
