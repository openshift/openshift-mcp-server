//go:build observability_e2e

package observability

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/containers/kubernetes-mcp-server/internal/test"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// mcpAuthOptions returns MCP client options that forward PROMETHEUS_K8S_TOKEN as
// Authorization when set. Pair with auth_mode = "header" in toolset config so
// observability backends (Thanos/Alertmanager routes) accept the request.
// Without PROMETHEUS_K8S_TOKEN, callers should use auth_mode = "kubeconfig".
func mcpAuthOptions() []test.McpClientOption {
	if token := os.Getenv("PROMETHEUS_K8S_TOKEN"); token != "" {
		return []test.McpClientOption{
			test.WithHTTPHeaders(map[string]string{
				"Authorization": "Bearer " + token,
			}),
		}
	}
	return nil
}

// metricsAuthMode returns the TOML auth_mode for metrics/logs/traces toolsets.
func metricsAuthMode() string {
	if os.Getenv("PROMETHEUS_K8S_TOKEN") != "" {
		return "header"
	}
	return "kubeconfig"
}

// restConfig loads the default kubeconfig (KUBECONFIG or ~/.kube/config).
func restConfig(t *testing.T) *rest.Config {
	t.Helper()
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, nil)
	cfg, err := kubeConfig.ClientConfig()
	if err != nil {
		t.Fatalf("Failed to load kubeconfig: %v", err)
	}
	return cfg
}

func dynamicClient(t *testing.T) dynamic.Interface {
	t.Helper()
	client, err := dynamic.NewForConfig(restConfig(t))
	if err != nil {
		t.Fatalf("Failed to create dynamic client: %v", err)
	}
	return client
}

// authenticatedHTTPClient returns an HTTP client for OpenShift monitoring Routes.
//
// Prefers PROMETHEUS_K8S_TOKEN because kubeconfigs that use client certificates
// are rejected by the OAuth proxy on monitoring routes (401).
//
// TLS verification is skipped: out-of-cluster e2e clients typically lack the
// cluster ingress CA. This client is test-only and uses a short timeout.
// Matches metrics suite config (insecure = true).
func authenticatedHTTPClient(t *testing.T) *http.Client {
	t.Helper()
	ensurePrometheusK8sToken(t)

	token := os.Getenv("PROMETHEUS_K8S_TOKEN")
	if token == "" {
		t.Fatal("PROMETHEUS_K8S_TOKEN unset after ensurePrometheusK8sToken")
	}
	t.Log("Using PROMETHEUS_K8S_TOKEN for route authentication")
	return &http.Client{
		Timeout: 15 * time.Second,
		Transport: &tokenRoundTripper{
			token: token,
			base: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion:         tls.VersionTLS12,
					InsecureSkipVerify: true, //nolint:gosec // e2e against cluster ingress without bundled CA
				},
			},
		},
	}
}

type tokenRoundTripper struct {
	token string
	base  http.RoundTripper
}

func (t *tokenRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(req)
}

// assertValidRouteURL checks that a discovered route URL is an external OpenShift
// Route host (https, dotted hostname, not in-cluster *.svc DNS).
func assertValidRouteURL(t *testing.T, raw string) {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Errorf("URL is not parseable: %s (%v)", raw, err)
		return
	}
	if parsed.Scheme != "https" {
		t.Errorf("Expected scheme 'https', got %q in URL: %s", parsed.Scheme, raw)
	}
	host := parsed.Hostname()
	if host == "" {
		t.Errorf("URL has no host: %s", raw)
	}
	if !strings.Contains(host, ".") {
		t.Errorf("URL host looks invalid (no dot): %s", host)
	}
	if strings.HasSuffix(host, ".svc") || strings.Contains(host, ".svc.") {
		t.Errorf("URL looks like in-cluster service DNS, expected OpenShift Route host: %s", raw)
	}
}

func mustJoinURL(base, path string) string {
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(path, "/")
}

func requireHTTPOK(t *testing.T, client *http.Client, apiURL string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		t.Fatalf("build GET %s: %v", apiURL, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET %s failed: %v", apiURL, err)
	}
	defer resp.Body.Close()
	t.Logf("%s responded with HTTP %d", apiURL, resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected HTTP 200 from %s, got %d", apiURL, resp.StatusCode)
	}
}
