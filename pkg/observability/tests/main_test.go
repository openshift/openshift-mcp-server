//go:build observability_e2e

package observability

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// ensurePrometheusK8sToken mints PROMETHEUS_K8S_TOKEN when unset. Call only from
// metrics/route HTTP tests — stacks/otelcol tests use kubeconfig and must not
// require token-creation permissions.
func ensurePrometheusK8sToken(t *testing.T) {
	t.Helper()
	if strings.TrimSpace(os.Getenv("PROMETHEUS_K8S_TOKEN")) != "" {
		return
	}

	cli := "oc"
	if _, err := exec.LookPath(cli); err != nil {
		cli = "kubectl"
		if _, err := exec.LookPath(cli); err != nil {
			t.Fatal("PROMETHEUS_K8S_TOKEN unset and neither oc nor kubectl found in PATH")
		}
	}

	cmd := exec.Command(cli, "create", "token", "prometheus-k8s",
		"-n", "openshift-monitoring",
		"--duration=20m",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("PROMETHEUS_K8S_TOKEN unset and %s create token failed: %v\n%s", cli, err, strings.TrimSpace(string(out)))
	}
	token := strings.TrimSpace(string(out))
	if token == "" {
		t.Fatalf("PROMETHEUS_K8S_TOKEN unset and %s create token returned empty output", cli)
	}
	if err := os.Setenv("PROMETHEUS_K8S_TOKEN", token); err != nil {
		t.Fatal(err)
	}
	t.Logf("exported PROMETHEUS_K8S_TOKEN via %s create token prometheus-k8s -n openshift-monitoring", cli)
}
