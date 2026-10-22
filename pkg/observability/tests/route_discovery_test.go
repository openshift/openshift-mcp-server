//go:build observability_e2e

package observability

import (
	"context"
	"strings"
	"testing"

	"github.com/containers/kubernetes-mcp-server/pkg/obs/openshift"
	"k8s.io/client-go/dynamic"
)

// Route discovery tests exercise OpenShift Route resolvers against platform
// monitoring routes in openshift-monitoring.
//
// Run with: make test-observability-e2e

func newMetricsResolver(t *testing.T) (*openshift.MetricsRouteResolver, dynamic.Interface) {
	t.Helper()
	return &openshift.MetricsRouteResolver{}, dynamicClient(t)
}

func TestRouteDiscovery_ThanosQuerier(t *testing.T) {
	resolver, dynClient := newMetricsResolver(t)
	// Resolve the named route directly — ResolveMetricsBackend("thanos") can
	// fall back to prometheus-k8s and would still pass if Thanos is missing.
	discoveredURL, err := resolver.ResolveEndpoint(
		context.Background(), dynClient,
		openshift.MonitoringNamespace, openshift.ThanosQuerierRouteName,
	)
	if err != nil {
		t.Fatalf("Failed to discover thanos-querier route: %v", err)
	}
	assertValidRouteURL(t, discoveredURL)
	if !strings.Contains(discoveredURL, "thanos-querier") {
		t.Fatalf("expected thanos-querier host in URL, got %s", discoveredURL)
	}
	t.Logf("Discovered Thanos URL: %s", discoveredURL)
}

func TestRouteDiscovery_PrometheusK8s(t *testing.T) {
	resolver, dynClient := newMetricsResolver(t)
	discoveredURL, err := resolver.ResolveMetricsBackend(context.Background(), dynClient, "prometheus")
	if err != nil {
		t.Fatalf("Failed to discover prometheus-k8s route: %v", err)
	}
	assertValidRouteURL(t, discoveredURL)
	t.Logf("Discovered Prometheus URL: %s", discoveredURL)
}

func TestRouteDiscovery_Alertmanager(t *testing.T) {
	resolver, dynClient := newMetricsResolver(t)
	discoveredURL, err := resolver.ResolveAlertmanager(context.Background(), dynClient)
	if err != nil {
		t.Fatalf("Failed to discover alertmanager-main route: %v", err)
	}
	assertValidRouteURL(t, discoveredURL)
	t.Logf("Discovered Alertmanager URL: %s", discoveredURL)
}

func TestRouteDiscovery_URLsAreReachable(t *testing.T) {
	ensurePrometheusK8sToken(t)
	resolver, dynClient := newMetricsResolver(t)
	client := authenticatedHTTPClient(t)

	tests := []struct {
		name    string
		getURL  func() (string, error)
		apiPath string
		hostSub string // optional host substring that must appear
	}{
		{
			name: "thanos-querier",
			getURL: func() (string, error) {
				return resolver.ResolveEndpoint(
					context.Background(), dynClient,
					openshift.MonitoringNamespace, openshift.ThanosQuerierRouteName,
				)
			},
			apiPath: "/api/v1/query?query=up",
			hostSub: "thanos-querier",
		},
		{
			name: "prometheus-k8s",
			getURL: func() (string, error) {
				return resolver.ResolveMetricsBackend(context.Background(), dynClient, "prometheus")
			},
			apiPath: "/api/v1/query?query=up",
			hostSub: "prometheus-k8s",
		},
		{
			name: "alertmanager-main",
			getURL: func() (string, error) {
				return resolver.ResolveAlertmanager(context.Background(), dynClient)
			},
			apiPath: "/api/v2/status",
			hostSub: "alertmanager",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rawURL, err := tt.getURL()
			if err != nil {
				t.Fatalf("Route discovery failed: %v", err)
			}
			assertValidRouteURL(t, rawURL)
			if tt.hostSub != "" && !strings.Contains(rawURL, tt.hostSub) {
				t.Fatalf("expected %q in URL, got %s", tt.hostSub, rawURL)
			}
			requireHTTPOK(t, client, mustJoinURL(rawURL, tt.apiPath))
		})
	}
}
