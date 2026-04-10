package mustgather

import (
	"fmt"
	"strings"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	mg "github.com/containers/kubernetes-mcp-server/pkg/ocp/mustgather"
	"github.com/google/jsonschema-go/jsonschema"
	"k8s.io/utils/ptr"
)

func initNodes() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        "mustgather_node_diagnostics_get",
				Description: "Get comprehensive diagnostic information for a specific node including kubelet logs, system info, CPU/IRQ affinities, and hardware details",
				Annotations: api.ToolAnnotations{
					Title:        "Node Diagnostics",
					ReadOnlyHint: ptr.To(true),
				},
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"node":        {Type: "string", Description: "Node name"},
						"include":     {Type: "string", Description: "Comma-separated diagnostics to include: kubelet,sysinfo,cpu,irq,pods,lscpu,lspci,dmesg,cmdline (default: all)"},
						"kubeletTail": {Type: "integer", Description: "Number of lines from end of kubelet log (0 for all, default: 100)"},
					},
					Required: []string{"node"},
				},
			},
			Handler:      mustgatherNodeDiagnosticsGet,
			ClusterAware: ptr.To(false),
		},
		{
			Tool: api.Tool{
				Name:        "mustgather_node_kubelet_logs",
				Description: "Get kubelet logs for a specific node (decompressed from .gz file)",
				Annotations: api.ToolAnnotations{
					Title:        "Kubelet Logs",
					ReadOnlyHint: ptr.To(true),
				},
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"node": {Type: "string", Description: "Node name"},
						"tail": {Type: "integer", Description: "Number of lines from end (0 for all)"},
					},
					Required: []string{"node"},
				},
			},
			Handler:      mustgatherNodeKubeletLogs,
			ClusterAware: ptr.To(false),
		},
		{
			Tool: api.Tool{
				Name:        "mustgather_node_kubelet_logs_grep",
				Description: "Filter kubelet logs for a specific node by a search string. Returns only matching lines.",
				Annotations: api.ToolAnnotations{
					Title:        "Kubelet Logs Grep",
					ReadOnlyHint: ptr.To(true),
				},
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"node":            {Type: "string", Description: "Node name"},
						"filter":          {Type: "string", Description: "String to search for in log lines"},
						"tail":            {Type: "integer", Description: "Maximum number of matching lines to return (0 for all)"},
						"caseInsensitive": {Type: "boolean", Description: "Perform case-insensitive search (default: false)"},
					},
					Required: []string{"node", "filter"},
				},
			},
			Handler:      mustgatherNodeKubeletLogsGrep,
			ClusterAware: ptr.To(false),
		},
	}
}

func mustgatherNodeDiagnosticsGet(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p, err := getProvider()
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	args := params.GetArguments()
	node := getString(args, "node", "")
	include := getString(args, "include", "all")
	kubeletTail := getInt(args, "kubeletTail", 100)

	if node == "" {
		return api.NewToolCallResult("", fmt.Errorf("node is required")), nil
	}

	diag, err := p.GetNodeDiagnostics(node)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get node diagnostics: %w", err)), nil
	}

	includeAll := include == "all"
	includeMap := make(map[string]bool)
	if !includeAll {
		for _, item := range strings.Split(include, ",") {
			includeMap[strings.TrimSpace(item)] = true
		}
	}
	shouldInclude := func(name string) bool {
		return includeAll || includeMap[name]
	}

	output := fmt.Sprintf("Node Diagnostics for %s\n", node)
	output += strings.Repeat("=", 80) + "\n\n"

	if shouldInclude("kubelet") && diag.KubeletLog != "" {
		output += "## Kubelet Logs"
		log := diag.KubeletLog
		if kubeletTail > 0 {
			output += fmt.Sprintf(" (last %d lines)", kubeletTail)
			log = mg.TailLines(log, kubeletTail)
		}
		output += "\n\n" + log + "\n\n"
	}

	sections := []struct {
		key, title, content string
	}{
		{"sysinfo", "System Info", diag.SysInfo},
		{"lscpu", "CPU Info (lscpu)", diag.Lscpu},
		{"cpu", "CPU Affinities", diag.CPUAffinities},
		{"irq", "IRQ Affinities", diag.IRQAffinities},
		{"lspci", "PCI Devices (lspci)", diag.Lspci},
		{"dmesg", "Kernel Messages (dmesg)", diag.Dmesg},
		{"cmdline", "Kernel Boot Parameters", diag.ProcCmdline},
		{"pods", "Pods Info", diag.PodsInfo},
		{"pods", "Pod Resources", diag.PodResources},
	}

	for _, s := range sections {
		if shouldInclude(s.key) && s.content != "" {
			output += fmt.Sprintf("## %s\n\n%s\n\n", s.title, s.content)
		}
	}

	return api.NewToolCallResult(output, nil), nil
}

func mustgatherNodeKubeletLogs(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p, err := getProvider()
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	args := params.GetArguments()
	node := getString(args, "node", "")
	tail := getInt(args, "tail", 0)

	if node == "" {
		return api.NewToolCallResult("", fmt.Errorf("node is required")), nil
	}

	diag, err := p.GetNodeDiagnostics(node)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get node diagnostics: %w", err)), nil
	}

	if diag.KubeletLog == "" {
		return api.NewToolCallResult("", fmt.Errorf("kubelet log not found for node %s", node)), nil
	}

	logs := diag.KubeletLog
	if tail > 0 {
		logs = mg.TailLines(logs, tail)
	}

	header := fmt.Sprintf("Kubelet logs for node %s", node)
	if tail > 0 {
		header += fmt.Sprintf(" (last %d lines)", tail)
	}
	header += ":\n\n"

	return api.NewToolCallResult(header+logs, nil), nil
}

func mustgatherNodeKubeletLogsGrep(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p, err := getProvider()
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	args := params.GetArguments()
	node := getString(args, "node", "")
	filter := getString(args, "filter", "")
	tail := getInt(args, "tail", 0)
	caseInsensitive := getBool(args, "caseInsensitive", false)

	if node == "" {
		return api.NewToolCallResult("", fmt.Errorf("node is required")), nil
	}
	if filter == "" {
		return api.NewToolCallResult("", fmt.Errorf("filter string is required")), nil
	}

	diag, err := p.GetNodeDiagnostics(node)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get node diagnostics: %w", err)), nil
	}

	if diag.KubeletLog == "" {
		return api.NewToolCallResult("", fmt.Errorf("kubelet log not found for node %s", node)), nil
	}

	lines := strings.Split(diag.KubeletLog, "\n")
	searchFilter := filter
	if caseInsensitive {
		searchFilter = strings.ToLower(filter)
	}

	var matchingLines []string
	for _, line := range lines {
		compareLine := line
		if caseInsensitive {
			compareLine = strings.ToLower(line)
		}
		if strings.Contains(compareLine, searchFilter) {
			matchingLines = append(matchingLines, line)
		}
	}

	// Apply tail from the end of matches
	if tail > 0 && len(matchingLines) > tail {
		matchingLines = matchingLines[len(matchingLines)-tail:]
	}

	header := fmt.Sprintf("Kubelet logs for node %s filtered by '%s'", node, filter)
	if caseInsensitive {
		header += " (case-insensitive)"
	}
	if tail > 0 {
		header += fmt.Sprintf(" (last %d matches)", tail)
	}
	header += fmt.Sprintf(":\n\nFound %d matching line(s)\n\n", len(matchingLines))

	if len(matchingLines) == 0 {
		return api.NewToolCallResult(header+"No matching lines found.", nil), nil
	}

	return api.NewToolCallResult(header+strings.Join(matchingLines, "\n"), nil), nil
}
