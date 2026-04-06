package mustgather

import (
	"fmt"
	"strings"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	mg "github.com/containers/kubernetes-mcp-server/pkg/ocp/mustgather"
	"github.com/google/jsonschema-go/jsonschema"
	"k8s.io/utils/ptr"
)

func initPodLogs() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        "mustgather_pod_logs_get",
				Description: "Get container logs for a specific pod from the must-gather archive. Returns current or previous logs.",
				Annotations: api.ToolAnnotations{
					Title:        "Get Pod Logs",
					ReadOnlyHint: ptr.To(true),
				},
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"namespace": {Type: "string", Description: "Pod namespace"},
						"pod":       {Type: "string", Description: "Pod name"},
						"container": {Type: "string", Description: "Container name (uses first container if not specified)"},
						"previous":  {Type: "boolean", Description: "Get previous container logs (from crash/restart)"},
						"tail":      {Type: "integer", Description: "Number of lines from end of logs (0 for all)"},
					},
					Required: []string{"namespace", "pod"},
				},
			},
			Handler:      mustgatherPodLogsGet,
			ClusterAware: ptr.To(false),
		},
		{
			Tool: api.Tool{
				Name:        "mustgather_pod_logs_grep",
				Description: "Filter pod container logs by a search string. Returns only matching lines from the must-gather archive.",
				Annotations: api.ToolAnnotations{
					Title:        "Pod Logs Grep",
					ReadOnlyHint: ptr.To(true),
				},
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"namespace":       {Type: "string", Description: "Pod namespace"},
						"pod":             {Type: "string", Description: "Pod name"},
						"container":       {Type: "string", Description: "Container name (uses first container if not specified)"},
						"filter":          {Type: "string", Description: "String to search for in log lines"},
						"previous":        {Type: "boolean", Description: "Search previous container logs (from crash/restart)"},
						"tail":            {Type: "integer", Description: "Maximum number of matching lines to return (0 for all)"},
						"caseInsensitive": {Type: "boolean", Description: "Perform case-insensitive search (default: false)"},
					},
					Required: []string{"namespace", "pod", "filter"},
				},
			},
			Handler:      mustgatherPodLogsGrep,
			ClusterAware: ptr.To(false),
		},
	}
}

func mustgatherPodLogsGet(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p, err := getProvider()
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	args := params.GetArguments()
	namespace := getString(args, "namespace", "")
	pod := getString(args, "pod", "")
	container := getString(args, "container", "")
	previous := getBool(args, "previous", false)
	tail := getInt(args, "tail", 0)

	if namespace == "" || pod == "" {
		return api.NewToolCallResult("", fmt.Errorf("namespace and pod are required")), nil
	}

	logType := mg.LogTypeCurrent
	if previous {
		logType = mg.LogTypePrevious
	}

	logs, err := p.GetPodLog(mg.PodLogOptions{
		Namespace: namespace,
		Pod:       pod,
		Container: container,
		LogType:   logType,
		TailLines: tail,
	})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get pod logs: %w", err)), nil
	}

	header := fmt.Sprintf("Logs for pod %s/%s", namespace, pod)
	if container != "" {
		header += fmt.Sprintf(", container %s", container)
	}
	if previous {
		header += " (previous)"
	}
	if tail > 0 {
		header += fmt.Sprintf(" (last %d lines)", tail)
	}
	header += ":\n\n"

	return api.NewToolCallResult(header+logs, nil), nil
}

func mustgatherPodLogsGrep(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p, err := getProvider()
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	args := params.GetArguments()
	namespace := getString(args, "namespace", "")
	pod := getString(args, "pod", "")
	container := getString(args, "container", "")
	filter := getString(args, "filter", "")
	previous := getBool(args, "previous", false)
	tail := getInt(args, "tail", 0)
	caseInsensitive := getBool(args, "caseInsensitive", false)

	if namespace == "" || pod == "" {
		return api.NewToolCallResult("", fmt.Errorf("namespace and pod are required")), nil
	}
	if filter == "" {
		return api.NewToolCallResult("", fmt.Errorf("filter string is required")), nil
	}

	logType := mg.LogTypeCurrent
	if previous {
		logType = mg.LogTypePrevious
	}

	logs, err := p.GetPodLog(mg.PodLogOptions{
		Namespace: namespace,
		Pod:       pod,
		Container: container,
		LogType:   logType,
	})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get pod logs: %w", err)), nil
	}

	lines := strings.Split(logs, "\n")
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

	if tail > 0 && len(matchingLines) > tail {
		matchingLines = matchingLines[len(matchingLines)-tail:]
	}

	header := fmt.Sprintf("Logs for pod %s/%s filtered by '%s'", namespace, pod, filter)
	if container != "" {
		header += fmt.Sprintf(", container %s", container)
	}
	if previous {
		header += " (previous)"
	}
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
