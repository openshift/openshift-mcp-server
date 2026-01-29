// Package prometheus provides a shared HTTP client for Prometheus and Alertmanager APIs.
// It supports flexible authentication (bearer token), TLS configuration (REST config CA,
// custom CA file, or insecure mode), and can be used by multiple toolsets with different
// URL discovery mechanisms.
package prometheus

// QueryResult represents a Prometheus API query response.
type QueryResult struct {
	Status    string   `json:"status"`
	Data      Data     `json:"data"`
	ErrorType string   `json:"errorType,omitempty"`
	Error     string   `json:"error,omitempty"`
	Warnings  []string `json:"warnings,omitempty"`
}

// Data contains the query result data.
type Data struct {
	ResultType string   `json:"resultType"`
	Result     []Result `json:"result"`
}

// Result represents a single result in a query response.
type Result struct {
	Metric map[string]string `json:"metric"`
	// Value is used for instant queries - [timestamp, value]
	Value []any `json:"value,omitempty"`
	// Values is used for range queries - [[timestamp, value], ...]
	Values [][]any `json:"values,omitempty"`
}

// Alert represents an Alertmanager alert.
type Alert struct {
	Annotations  map[string]string `json:"annotations"`
	EndsAt       string            `json:"endsAt"`
	Fingerprint  string            `json:"fingerprint"`
	Receivers    []Receiver        `json:"receivers"`
	StartsAt     string            `json:"startsAt"`
	Status       AlertStatus       `json:"status"`
	UpdatedAt    string            `json:"updatedAt"`
	GeneratorURL string            `json:"generatorURL,omitempty"`
	Labels       map[string]string `json:"labels"`
}

// Receiver represents an Alertmanager receiver.
type Receiver struct {
	Name string `json:"name"`
}

// AlertStatus represents the status of an alert.
type AlertStatus struct {
	InhibitedBy []string `json:"inhibitedBy"`
	SilencedBy  []string `json:"silencedBy"`
	State       string   `json:"state"`
}
