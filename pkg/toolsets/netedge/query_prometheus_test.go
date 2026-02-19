package netedge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"time"

	"github.com/containers/kubernetes-mcp-server/pkg/prometheus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	fakedynamic "k8s.io/client-go/dynamic/fake"
)

func (s *NetEdgeTestSuite) TestQueryPrometheusHandler_Diagnostics() {
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
		} else if strings.Contains(query, "ALERTS") {
			metricName = "ALERTS"
		} else if strings.Contains(query, "up") {
			metricName = "up"
		} else if strings.Contains(query, "coredns_plugin_rewrite_request_count_total") {
			metricName = "coredns_plugin_rewrite_request_count_total"
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
		} else if strings.Contains(query, "ALERTS") {
			resp.Data.Result[0].Metric["check"] = "operators-alerts"
		}

		_ = json.NewEncoder(w).Encode(resp)
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
				"ingress_top_error_routes",
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
				"dns_error_breakdown",
				"dns_rewrite_count",
				"check", "dns-request-rate",
			},
		},
		{
			name:       "Operators Diagnostics",
			diagTarget: "operators",
			expectedContains: []string{
				"active_alerts",
				"operator_up",
				"check", "operators-alerts",
			},
		},
		{
			name:        "Unknown Target",
			diagTarget:  "foo",
			expectError: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Ensure Insecure is true for prometheus insecure test handling
			s.mockClient.restConfig.BearerToken = "fake-token"
			s.mockClient.restConfig.Insecure = true

			s.SetDynamicClient(dynClient)
			s.SetArgs(map[string]any{
				"diagnostic_target": tt.diagTarget,
			})

			result, err := queryPrometheusHandler(s.params)

			// Validation
			if tt.expectError {
				if err == nil {
					if result == nil || result.Error == nil {
						s.Fail("expected error but got nil")
					}
				} else {
					s.NoError(err)
				}
			} else {
				s.NoError(err)
				if s.NotNil(result) {
					s.NoError(result.Error)
					for _, expected := range tt.expectedContains {
						s.Contains(result.Content, expected)
					}
				}
			}
		})
	}
}
