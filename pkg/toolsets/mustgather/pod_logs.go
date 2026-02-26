package mustgather

import (
	"fmt"

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
