package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/netedge"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/rest"
)

// MockConfigProvider partially implements api.ExtendedConfigProvider
type MockConfigProvider struct {
	api.ExtendedConfigProvider // Embed to satisfy interface
	PrometheusURL              string
}

func (m *MockConfigProvider) GetToolsetConfig(name string) (api.ExtendedConfig, bool) {
	if name == "netedge" {
		return &netedge.Config{
			PrometheusURL:      m.PrometheusURL,
			PrometheusInsecure: true,
		}, true
	}
	return nil, false
}

// MockKubernetesClient partially implements api.KubernetesClient
type MockKubernetesClient struct {
	api.KubernetesClient // Embed to satisfy interface
	Token                string
}

func (m *MockKubernetesClient) RESTConfig() *rest.Config {
	return &rest.Config{BearerToken: m.Token}
}

// MockToolCallRequest implements api.ToolCallRequest
type MockToolCallRequest struct {
	Args map[string]any
}

func (m *MockToolCallRequest) GetArguments() map[string]any {
	return m.Args
}

func TestQueryPrometheusHandler_Diagnostics(t *testing.T) {
	// 1. Setup mock Prometheus server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Authentication
		auth := r.Header.Get("Authorization")
		if auth != "Bearer fake-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Check what query we received
		query := r.URL.Query().Get("query")

		// Respond with mock data based on query content
		resp := netedge.PrometheusResponse{
			Status: "success",
			Data: netedge.Data{
				ResultType: "vector",
				Result: []netedge.Result{
					{
						Metric: map[string]string{"foo": "bar"},
						Value:  []interface{}{float64(time.Now().Unix()), "123.45"},
					},
				},
			},
		}

		// Specialize response if needed based on query to distinguish them
		// For Ingress Error Rate
		if strings.Contains(query, "haproxy_server_http_responses_total") {
			resp.Data.Result[0].Metric = map[string]string{"check": "ingress-error-rate"}
		}
		// For DNS Request Rate
		if strings.Contains(query, "coredns_dns_request_count_total") {
			resp.Data.Result[0].Metric = map[string]string{"check": "dns-request-rate"}
		}

		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	tests := []struct {
		name             string
		diagTarget       string
		expectedContains []string
		expectError      bool
	}{
		{
			name:       "Ingress Diagnostics",
			diagTarget: "ingress",
			expectedContains: []string{
				"ingress_error_rate",
				"ingress_active_conns",
				"ingress_reloads_last_day",
				"check", "ingress-error-rate", // Metric from mock
			},
		},
		{
			name:       "DNS Diagnostics",
			diagTarget: "dns",
			expectedContains: []string{
				"dns_request_rate",
				"dns_nxdomain_rate",
				"dns_servfail_rate",
				"dns_panic_recovery",
				"check", "dns-request-rate", // Metric from mock
			},
		},
		{
			name:        "Unknown Target",
			diagTarget:  "foo",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup dependencies
			cfgProvider := &MockConfigProvider{PrometheusURL: ts.URL}
			kubeClient := &MockKubernetesClient{Token: "fake-token"}
			toolReq := &MockToolCallRequest{
				Args: map[string]any{
					"diagnostic_target": tt.diagTarget,
				},
			}

			// Construct params
			params := api.ToolHandlerParams{
				Context:                context.Background(),
				ExtendedConfigProvider: cfgProvider,
				KubernetesClient:       kubeClient,
				ToolCallRequest:        toolReq,
			}

			// Execute Handler (which is queryPrometheusHandler)
			// Need to call queryPrometheusHandler directly.
			// It is not exported but tests are in the same package 'tools'.
			// However `queryPrometheusHandler` is not exported.
			// Wait, the test file is package `tools`. `queryPrometheusHandler` is in package `tools`.
			// So it is accessible.

			result, err := queryPrometheusHandler(params)

			// Validation
			if tt.expectError {
				// The handler swallows errors and returns them in ToolCallResult or returns error?
				// Handler signature: `(*api.ToolCallResult, error)`
				// `queryPrometheusHandler` returns error if something fails?
				// Looking at the code:
				// `return api.NewToolCallResult("", fmt.Errorf("unknown diagnostic target: %s", target)), nil`
				// So it returns NO error, but a ToolCallResult with Error set?
				// Let's check `api.NewToolCallResult` implementation or assumption.
				// Usually `NewToolCallResult` sets the error field.

				// Re-checking query_prometheus.go:
				// `return api.NewToolCallResult("", fmt.Errorf("...")), nil`
				// So `result.Error` should be non-nil.

				assert.NoError(t, err) // Should strictly be no error from function call
				if result != nil {
					assert.Error(t, result.Error)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.NoError(t, result.Error)

				// Check content
				// result.Content is the JSON string
				for _, expected := range tt.expectedContains {
					assert.Contains(t, result.Content, expected)
				}
			}
		})
	}
}
