package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/prometheus"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/rest"
)

// MockConfigProvider partially implements api.ExtendedConfigProvider
type MockConfigProvider struct {
	api.ExtendedConfigProvider // Embed to satisfy interface
}

func (m *MockConfigProvider) GetToolsetConfig(name string) (api.ExtendedConfig, bool) {
	return nil, false
}

// MockKubernetesClient implements api.KubernetesClient with DynamicClient support
type MockKubernetesClient struct {
	api.KubernetesClient // Embed to satisfy interface
	Token                string
	DynClient            dynamic.Interface
}

func (m *MockKubernetesClient) RESTConfig() *rest.Config {
	return &rest.Config{
		BearerToken: m.Token,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true,
		},
	}
}

func (m *MockKubernetesClient) DynamicClient() dynamic.Interface {
	return m.DynClient
}

// MockToolCallRequest implements api.ToolCallRequest
type MockToolCallRequest struct {
	Args map[string]any
}

func (m *MockToolCallRequest) GetArguments() map[string]any {
	return m.Args
}

func TestQueryPrometheusHandler_Diagnostics(t *testing.T) {
	// 1. Setup mock Prometheus server (TLS)
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Authentication
		auth := r.Header.Get("Authorization")
		if auth != "Bearer fake-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Check what query we received
		query := r.URL.Query().Get("query")

		// Parse the query to ensure we return correct metric names
		// This is a rough simulation
		var metricName string
		if strings.Contains(query, "haproxy_server_http_responses_total") {
			metricName = "haproxy_server_http_responses_total"
		} else if strings.Contains(query, "coredns_dns_request_count_total") {
			metricName = "coredns_dns_request_count_total"
		} else {
			metricName = "result"
		}

		// Respond with mock data
		resp := prometheus.QueryResult{
			Status: "success",
			Data: prometheus.Data{
				ResultType: "vector",
				Result: []prometheus.Result{
					{
						Metric: map[string]string{"__name__": metricName},
						Value:  []interface{}{float64(time.Now().Unix()), "123.45"},
					},
				},
			},
		}

		// Add specific metrics for validation
		if strings.Contains(query, "haproxy_server_http_responses_total") {
			resp.Data.Result[0].Metric["check"] = "ingress-error-rate"
		} else if strings.Contains(query, "coredns_dns_request_count_total") {
			resp.Data.Result[0].Metric["check"] = "dns-request-rate"
		}

		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	// Extract host from URL for Route object
	u, _ := url.Parse(ts.URL)
	host := u.Host

	// 2. Setup Fake Dynamic Client with Route
	route := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "route.openshift.io/v1",
			"kind":       "Route",
			"metadata": map[string]interface{}{
				"name":      "thanos-querier",
				"namespace": "openshift-monitoring",
			},
			"spec": map[string]interface{}{
				"host": host,
			},
		},
	}

	scheme := runtime.NewScheme()
	dynClient := fakedynamic.NewSimpleDynamicClient(scheme, route)

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
				"check", "ingress-error-rate",
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
				"check", "dns-request-rate",
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
			cfgProvider := &MockConfigProvider{}
			kubeClient := &MockKubernetesClient{
				Token:     "fake-token",
				DynClient: dynClient,
			}
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

			result, err := queryPrometheusHandler(params)

			// Validation
			if tt.expectError {
				if err == nil {
					// Check if result has Error field (ToolCallResult)
					if result == nil || result.Error == nil {
						t.Errorf("expected error but got nil")
					}
				} else {
					// Logic error in test assumption: handler returns (result, nil) usually
					// but check if err is returned
					assert.NoError(t, err)
				}
			} else {
				assert.NoError(t, err)
				if assert.NotNil(t, result) {
					assert.NoError(t, result.Error)
					for _, expected := range tt.expectedContains {
						assert.Contains(t, result.Content, expected)
					}
				}
			}
		})
	}
}
