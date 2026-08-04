package fencing

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/utils/ptr"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/cluster-diagnostics/nodesdebug"
)

const defaultSTONITHTimeout = 120 * time.Second

var stonithDiagnosticScript = strings.Join([]string{
	"echo '===PCS_STATUS==='",
	"pcs status 2>&1",
	"echo '===PCS_STONITH_CONFIG==='",
	"pcs stonith config 2>&1",
	"echo '===PCS_STONITH_STATUS==='",
	"pcs stonith status 2>&1",
	"echo '===PCS_STONITH_HISTORY==='",
	"pcs stonith history 2>&1",
	"echo '===PCS_QUORUM_STATUS==='",
	"corosync-quorumtool 2>&1",
	"echo '===PCS_PROPERTY==='",
	"{ pcs property config 2>&1 || pcs property list 2>&1; }",
	"echo '===END==='",
}, "; ")

func InitSTONITH() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name: "tnf_check_stonith_status",
				Description: "Check STONITH and pacemaker fencing status on a Two-Node Fencing (TNF) cluster. " +
					"Creates a temporary privileged debug pod on a control-plane node to run pcs diagnostic " +
					"commands. Returns pacemaker cluster state, STONITH device configuration, quorum status, " +
					"and recent fencing history. The debug pod is automatically cleaned up after execution.",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"node": {
							Type:        "string",
							Description: "Name of the node to run diagnostics on. If omitted, auto-detects the first control-plane node.",
						},
						"namespace": {
							Type:        "string",
							Description: "Namespace to create the temporary debug pod in (optional, defaults to 'default').",
						},
						"timeout_seconds": {
							Type:        "integer",
							Description: "Maximum time in seconds to wait for the diagnostic commands to complete (optional, defaults to 120).",
							Minimum:     ptr.To(float64(1)),
						},
					},
				},
				Annotations: api.ToolAnnotations{
					Title:           "TNF: Check STONITH Status",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					IdempotentHint:  ptr.To(true),
					OpenWorldHint:   ptr.To(false),
				},
			},
			Handler: checkSTONITHStatus,
		},
	}
}

func checkSTONITHStatus(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	if IsAPIUnreachable(params.KubernetesClient) {
		return api.NewToolCallResult(APIUnreachableGuide(), nil), nil
	}

	p := api.WrapParams(params)
	nodeName := p.OptionalString("node", "")
	namespace := p.OptionalString("namespace", "")
	if err := p.Err(); err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to check STONITH status: %w", err)), nil
	}

	timeout := defaultSTONITHTimeout
	if timeoutRaw, exists := params.GetArguments()["timeout_seconds"]; exists && timeoutRaw != nil {
		t, err := parseTimeout(timeoutRaw)
		if err != nil {
			return api.NewToolCallResult("", fmt.Errorf("failed to check STONITH status: %w", err)), nil
		}
		timeout = t
	}

	report, issues, err := RunSTONITHDiagnostics(params.Context, params.KubernetesClient, params.CoreV1(), nodeName, namespace, timeout)
	if err != nil {
		return api.NewToolCallResult(report, err), nil
	}

	var reportBuilder strings.Builder
	reportBuilder.WriteString(report)

	reportBuilder.WriteString("\n## Summary\n\n")
	if len(issues) == 0 {
		reportBuilder.WriteString("No STONITH issues detected.\n")
	} else {
		fmt.Fprintf(&reportBuilder, "Found %d issue(s):\n\n", len(issues))
		for i, issue := range issues {
			fmt.Fprintf(&reportBuilder, "%d. %s\n", i+1, issue)
		}
	}

	return api.NewToolCallResult(reportBuilder.String(), nil), nil
}

// RunSTONITHDiagnostics runs pcs diagnostics via a debug pod and returns a
// structured report and list of issues. If nodeName is empty, auto-detects
// the first control-plane node.
func RunSTONITHDiagnostics(ctx context.Context, k8sClient api.KubernetesClient, coreClient corev1client.CoreV1Interface, nodeName, namespace string, timeout time.Duration) (string, []string, error) {
	if nodeName == "" {
		detected, err := DetectControlPlaneNode(ctx, coreClient)
		if err != nil {
			return "", nil, fmt.Errorf("failed to auto-detect control-plane node: %w", err)
		}
		nodeName = detected
	}

	client := nodesdebug.NewNodeDebug(k8sClient)
	command := []string{"chroot", "/host", "sh", "-c", stonithDiagnosticScript}

	stdout, stderr, execErr := client.NodesDebugExec(
		ctx, namespace, nodeName, "", command, "", "", timeout,
	)

	if execErr != nil {
		output := stderr
		if stdout != "" {
			output = fmt.Sprintf("Partial output:\n%s\n\nError: %v", stdout, execErr)
		}
		return output, nil, execErr
	}

	report, issues := buildSTONITHReport(nodeName, stdout)

	if stderr != "" {
		slog.Debug("STONITH diagnostic stderr", "node", nodeName, "stderr", stderr)
	}

	return report, issues, nil
}

// DetectControlPlaneNode finds the first control-plane node by label.
// Checks both node-role.kubernetes.io/control-plane and the legacy
// node-role.kubernetes.io/master label.
func DetectControlPlaneNode(ctx context.Context, client corev1client.CoreV1Interface) (string, error) {
	for _, label := range []string{"node-role.kubernetes.io/control-plane", "node-role.kubernetes.io/master"} {
		nodes, err := client.Nodes().List(ctx, metav1.ListOptions{
			LabelSelector: label,
		})
		if err != nil {
			return "", fmt.Errorf("failed to list control-plane nodes: %w", err)
		}
		if len(nodes.Items) > 0 {
			return nodes.Items[0].Name, nil
		}
	}
	return "", errors.New("no control-plane nodes found")
}

func buildSTONITHReport(nodeName, rawOutput string) (string, []string) {
	sections := parseSections(rawOutput)
	var report strings.Builder
	var issues []string

	fmt.Fprintf(&report, "# TNF STONITH Status (from node: %s)\n\n", nodeName)

	pcsIssues := writePCSStatusSection(&report, sections["PCS_STATUS"])
	issues = append(issues, pcsIssues...)

	propIssues := writePropertySection(&report, sections["PCS_PROPERTY"])
	issues = append(issues, propIssues...)

	deviceIssues := writeStonithDeviceSection(&report, sections["PCS_STONITH_CONFIG"], sections["PCS_STONITH_STATUS"])
	issues = append(issues, deviceIssues...)

	quorumIssues := writeQuorumSection(&report, sections["PCS_QUORUM_STATUS"])
	issues = append(issues, quorumIssues...)

	writeHistorySection(&report, sections["PCS_STONITH_HISTORY"])

	return report.String(), issues
}

func parseSections(raw string) map[string]string {
	sections := make(map[string]string)
	markers := []string{"PCS_STATUS", "PCS_STONITH_CONFIG", "PCS_STONITH_STATUS",
		"PCS_STONITH_HISTORY", "PCS_QUORUM_STATUS", "PCS_PROPERTY", "END"}

	for i, marker := range markers {
		start := strings.Index(raw, "==="+marker+"===")
		if start == -1 {
			continue
		}
		contentStart := start + len("==="+marker+"===")
		contentEnd := len(raw)
		for j := i + 1; j < len(markers); j++ {
			nextStart := strings.Index(raw, "==="+markers[j]+"===")
			if nextStart != -1 {
				contentEnd = nextStart
				break
			}
		}
		if contentStart < contentEnd {
			sections[marker] = strings.TrimSpace(raw[contentStart:contentEnd])
		}
	}

	return sections
}

func writePCSStatusSection(report *strings.Builder, section string) []string {
	var issues []string
	report.WriteString("## Pacemaker Cluster\n\n")

	if section == "" {
		report.WriteString("- pcs status: not available\n\n")
		issues = append(issues, "pcs status command returned no output")
		return issues
	}

	var onlineNodes, offlineNodes, standbyNodes []string

	for _, line := range strings.Split(section, "\n") {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "Cluster name:") {
			fmt.Fprintf(report, "- %s\n", trimmed)
		}

		if strings.Contains(trimmed, "Online:") && strings.Contains(trimmed, "[") {
			nodes := extractBracketedList(trimmed)
			onlineNodes = nodes
			if len(nodes) > 0 {
				fmt.Fprintf(report, "- Nodes Online: %s\n", strings.Join(nodes, ", "))
			}
		}
		if strings.Contains(trimmed, "OFFLINE:") && strings.Contains(trimmed, "[") {
			nodes := extractBracketedList(trimmed)
			offlineNodes = nodes
			if len(nodes) > 0 {
				fmt.Fprintf(report, "- Nodes Offline: %s\n", strings.Join(nodes, ", "))
				issues = append(issues, fmt.Sprintf("nodes offline: %s", strings.Join(nodes, ", ")))
			}
		}
		if strings.Contains(trimmed, "Standby:") && strings.Contains(trimmed, "[") {
			nodes := extractBracketedList(trimmed)
			standbyNodes = append(standbyNodes, nodes...)
		}
		if strings.HasPrefix(trimmed, "* Node") && strings.Contains(trimmed, "standby") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 3 {
				name := strings.TrimSuffix(parts[2], ":")
				standbyNodes = append(standbyNodes, name)
			}
		}

		if strings.Contains(trimmed, "STONITH") && strings.Contains(trimmed, "enabled") {
			fmt.Fprintf(report, "- %s\n", trimmed)
		}

		if strings.Contains(trimmed, "Failed") && strings.Contains(trimmed, "Resource Actions") {
			fmt.Fprintf(report, "- **%s**\n", trimmed)
		}
	}

	if len(standbyNodes) > 0 {
		fmt.Fprintf(report, "- Nodes Standby: %s\n", strings.Join(standbyNodes, ", "))
		issues = append(issues, fmt.Sprintf("nodes in standby: %s", strings.Join(standbyNodes, ", ")))
	}

	if len(onlineNodes) == 0 && len(offlineNodes) == 0 && len(standbyNodes) == 0 {
		report.WriteString("- (could not parse node status from pcs output)\n")
	}

	report.WriteString("\n")

	if strings.Contains(section, "Failed Resource Actions") || strings.Contains(section, "Failed Actions") {
		failedSection := extractAfter(section, "Failed")
		if failedSection != "" && !strings.Contains(failedSection, "0 resource") {
			issues = append(issues, "pcs reports failed resource actions")
		}
	}

	return issues
}

func writePropertySection(report *strings.Builder, section string) []string {
	var issues []string

	if section == "" {
		return issues
	}

	for _, line := range strings.Split(section, "\n") {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)

		if strings.HasPrefix(lower, "stonith-enabled:") {
			val := strings.TrimSpace(strings.SplitN(trimmed, ":", 2)[1])
			if strings.EqualFold(val, "false") {
				issues = append(issues, "STONITH is disabled (stonith-enabled: false)")
			}
		}
		if strings.HasPrefix(lower, "no-quorum-policy:") {
			val := strings.TrimSpace(strings.SplitN(trimmed, ":", 2)[1])
			fmt.Fprintf(report, "- No-Quorum Policy: %s\n", val)
		}
	}

	return issues
}

func writeStonithDeviceSection(report *strings.Builder, configSection, statusSection string) []string {
	var issues []string
	report.WriteString("## STONITH Devices\n\n")

	if configSection == "" {
		report.WriteString("No STONITH device configuration found.\n\n")
		issues = append(issues, "no STONITH devices configured")
		return issues
	}

	if strings.Contains(configSection, "NO stonith devices") || strings.Contains(configSection, "No stonith") {
		report.WriteString("No STONITH devices configured.\n\n")
		issues = append(issues, "no STONITH devices configured")
		return issues
	}

	report.WriteString("### Configuration\n\n")
	report.WriteString("```\n")
	report.WriteString(configSection)
	report.WriteString("\n```\n\n")

	if statusSection != "" {
		report.WriteString("### Status\n\n")
		for _, line := range strings.Split(statusSection, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			fmt.Fprintf(report, "- %s\n", trimmed)

			lower := strings.ToLower(trimmed)
			if strings.Contains(lower, "stopped") {
				issues = append(issues, fmt.Sprintf("STONITH device stopped: %s", trimmed))
			}
			if strings.Contains(lower, "failed") {
				issues = append(issues, fmt.Sprintf("STONITH device failed: %s", trimmed))
			}
		}
		report.WriteString("\n")
	}

	return issues
}

func writeQuorumSection(report *strings.Builder, section string) []string {
	var issues []string
	report.WriteString("## Quorum\n\n")

	if section == "" {
		report.WriteString("- corosync-quorumtool: not available\n\n")
		return issues
	}

	for _, line := range strings.Split(section, "\n") {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)

		if strings.HasPrefix(lower, "quorate:") {
			fmt.Fprintf(report, "- %s\n", trimmed)
			val := strings.TrimSpace(strings.SplitN(trimmed, ":", 2)[1])
			if strings.EqualFold(val, "No") {
				issues = append(issues, "cluster is NOT quorate")
			}
		}
		if strings.HasPrefix(lower, "expected votes:") || strings.HasPrefix(lower, "total votes:") {
			fmt.Fprintf(report, "- %s\n", trimmed)
		}
		if strings.HasPrefix(lower, "flags:") || strings.Contains(lower, "two_node") || strings.Contains(lower, "two node") {
			fmt.Fprintf(report, "- %s\n", trimmed)
		}
	}

	report.WriteString("\n")
	return issues
}

func writeHistorySection(report *strings.Builder, section string) {
	report.WriteString("## Fencing History\n\n")

	if section == "" || strings.Contains(section, "No fencing") || strings.Contains(section, "no fencing") {
		report.WriteString("No fencing events recorded.\n\n")
		return
	}

	for _, line := range strings.Split(section, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		fmt.Fprintf(report, "- %s\n", trimmed)
	}
	report.WriteString("\n")
}

func extractBracketedList(line string) []string {
	start := strings.Index(line, "[")
	end := strings.Index(line, "]")
	if start == -1 || end == -1 || end <= start {
		return nil
	}
	inner := strings.TrimSpace(line[start+1 : end])
	if inner == "" {
		return nil
	}
	return strings.Fields(inner)
}

func extractAfter(text, marker string) string {
	idx := strings.Index(text, marker)
	if idx == -1 {
		return ""
	}
	return text[idx:]
}

func parseTimeout(raw interface{}) (time.Duration, error) {
	switch v := raw.(type) {
	case float64:
		if v < 1 || v != math.Trunc(v) {
			return 0, errors.New("timeout_seconds must be an integer >= 1")
		}
		return time.Duration(int64(v)) * time.Second, nil
	case int:
		if v < 1 {
			return 0, errors.New("timeout_seconds must be >= 1")
		}
		return time.Duration(v) * time.Second, nil
	case int64:
		if v < 1 {
			return 0, errors.New("timeout_seconds must be >= 1")
		}
		return time.Duration(v) * time.Second, nil
	default:
		return 0, errors.New("timeout_seconds must be a numeric value")
	}
}
