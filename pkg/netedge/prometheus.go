package netedge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	PrometheusQueryEndpoint = "/api/v1/query"
)

// QueryPrometheus executes a PromQL query against the configured Prometheus URL
func (c *NetEdgeClient) QueryPrometheus(ctx context.Context, query string) (*PrometheusResponse, error) {
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	queryParams := map[string]string{
		"query": query,
	}

	// Calculate endpoint URL
	// Note: executeRequest handles the base URL joining

	// Reuse the execute method from netedge.go
	// We need to export or make executeRequest internal but accessible.
	// Since they are in the same package 'netedge', it is accessible.

	respBody, err := c.executeRequest(ctx, http.MethodGet, PrometheusQueryEndpoint, queryParams)
	if err != nil {
		return nil, err
	}

	var promResp PrometheusResponse
	if err := json.Unmarshal([]byte(respBody), &promResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal prometheus response: %w", err)
	}

	if promResp.Status != "success" {
		return &promResp, fmt.Errorf("prometheus query failed: %s - %s", promResp.ErrorType, promResp.Error)
	}

	return &promResp, nil
}
