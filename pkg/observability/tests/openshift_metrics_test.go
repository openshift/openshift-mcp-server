//go:build observability_e2e

package observability

import (
	"encoding/json"
	"strings"
)

// TestOpenShiftMetricsPresent verifies the metrics toolset can see an
// OpenShift-specific metric from platform monitoring (cluster-monitoring-operator).
func (s *MetricsE2ESuite) TestOpenShiftMetricsPresent() {
	const metric = "cluster_monitoring_operator_reconcile_attempts_total"

	toolResult, err := s.CallTool("list_metrics", map[string]any{
		"name_regex": metric,
	})
	s.Require().NoError(err)
	s.Require().False(toolResult.IsError, "tool returned error: %s", textContent(toolResult))

	body := textContent(toolResult)
	if !strings.Contains(body, metric) {
		s.T().Fatalf("OpenShift-specific metric %q not found — is the server pointed at OpenShift monitoring?\nResult: %s", metric, body)
	}

	var output listMetricsOutput
	s.Require().NoError(json.Unmarshal([]byte(body), &output))
	s.Contains(output.Metrics, metric)
	s.T().Logf("OpenShift metric %q confirmed present", metric)
}
