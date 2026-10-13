//go:build observability_e2e

package observability

import (
	"os"
	"testing"

	"github.com/containers/kubernetes-mcp-server/internal/test"
	"github.com/containers/kubernetes-mcp-server/pkg/config"
	internalk8s "github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	mcpserver "github.com/containers/kubernetes-mcp-server/pkg/mcp"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// applyE2EDefaults pins the provider to the current kubeconfig context only.
// Without this, the default kubeconfig strategy probes every context for GVK
// filtering and floods logs with unreachable stale clusters.
func applyE2EDefaults(cfg *config.Config) {
	cfg.ClusterProviderStrategy.SetForTest(config.ClusterProviderDisabled)
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		cfg.KubeConfig.SetForTest(kubeconfig)
	}
}

// startServerFromTOML creates an in-process MCP server + client from TOML config.
// clientOpts are passed to the MCP test client (e.g. mcpAuthOptions() for
// monitoring Route bearer auth). Omit them so Kubernetes API calls use
// kubeconfig credentials — required for TempoStack/LokiStack list tools,
// since PROMETHEUS_K8S_TOKEN is the prometheus-k8s SA and lacks those RBAC verbs.
func startServerFromTOML(t *testing.T, tomlCfg string, clientOpts ...test.McpClientOption) (*mcpserver.Server, *test.McpClient) {
	t.Helper()
	cfg, err := config.ReadToml(t.Context(), []byte(tomlCfg))
	require.NoError(t, err, "parse test config")
	applyE2EDefaults(cfg)
	provider, err := internalk8s.NewProvider(t.Context(), cfg)
	require.NoError(t, err, "create k8s provider")
	server, err := mcpserver.NewServer(t.Context(), mcpserver.Configuration{Config: cfg}, provider)
	require.NoError(t, err, "create MCP server")
	client := test.NewMcpClient(t, server.ServeHTTP(), clientOpts...)
	t.Cleanup(func() {
		client.Close()
		server.Close()
	})
	return server, client
}

func crdExists(t *testing.T, pluralGroup string) bool {
	t.Helper()
	// pluralGroup e.g. "tempostacks.tempo.grafana.com"
	dyn := dynamicClient(t)
	gvr := schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1",
		Resource: "customresourcedefinitions",
	}
	_, err := dyn.Resource(gvr).Get(t.Context(), pluralGroup, metav1.GetOptions{})
	if err == nil {
		return true
	}
	if apierrors.IsNotFound(err) {
		return false
	}
	t.Fatalf("looking up CRD %s: %v", pluralGroup, err)
	return false
}

// namedResourceExists reports whether a namespaced CR exists.
func namedResourceExists(t *testing.T, gvr schema.GroupVersionResource, namespace, name string) bool {
	t.Helper()
	dyn := dynamicClient(t)
	_, err := dyn.Resource(gvr).Namespace(namespace).Get(t.Context(), name, metav1.GetOptions{})
	if err == nil {
		return true
	}
	if apierrors.IsNotFound(err) {
		return false
	}
	t.Fatalf("looking up %s %s/%s: %v", gvr.Resource, namespace, name, err)
	return false
}

func structuredMap(t *testing.T, result any) map[string]any {
	t.Helper()
	if result == nil {
		t.Fatal("structuredContent is nil")
	}
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("structuredContent type %T, want map[string]any", result)
	}
	return m
}
