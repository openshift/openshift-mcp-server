//go:build observability_e2e

package observability

import (
	"os"

	"github.com/containers/kubernetes-mcp-server/pkg/config"
)

// applyE2EDefaults pins the provider to the current kubeconfig context only.
func applyE2EDefaults(cfg *config.Config) {
	cfg.ClusterProviderStrategy.SetForTest(config.ClusterProviderDisabled)
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		cfg.KubeConfig.SetForTest(kubeconfig)
	}
}
