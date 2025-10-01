package core

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"k8s.io/utils/ptr"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	internalk8s "github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
)

func initNodes(_ internalk8s.Openshift) []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        "nodes_debug_exec",
				Description: "Run commands on an OpenShift node using a privileged debug pod (output is truncated to the most recent 100 lines, so prefer filters like grep when expecting large logs).",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"node": {
							Type:        "string",
							Description: "Name of the node to debug (e.g. worker-0).",
						},
						"command": {
							Type:        "array",
							Description: "Command to execute on the node via chroot. Provide each argument as a separate array item (e.g. ['systemctl', 'status', 'kubelet']).",
							Items:       &jsonschema.Schema{Type: "string"},
						},
						"namespace": {
							Type:        "string",
							Description: "Namespace to create the temporary debug pod in (optional, defaults to the current namespace or 'default').",
						},
						"image": {
							Type:        "string",
							Description: "Container image to use for the debug pod (optional). Defaults to a Fedora-based utility image that includes chroot.",
						},
						"timeout_seconds": {
							Type:        "integer",
							Description: "Maximum time to wait for the command to complete before timing out (optional, defaults to 300 seconds).",
							Minimum:     ptr.To(float64(1)),
						},
					},
					Required: []string{"node", "command"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Nodes: Debug Exec",
					ReadOnlyHint:    ptr.To(false),
					DestructiveHint: ptr.To(true),
					IdempotentHint:  ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: nodesDebugExec,
		},
	}
}

func nodesDebugExec(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	nodeArg := params.GetArguments()["node"]
	nodeName, ok := nodeArg.(string)
	if nodeArg == nil || !ok || nodeName == "" {
		return api.NewToolCallResult("", errors.New("missing required argument: node")), nil
	}

	commandArg := params.GetArguments()["command"]
	command, err := toStringSlice(commandArg)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("invalid command argument: %w", err)), nil
	}

	namespace := ""
	if nsArg, ok := params.GetArguments()["namespace"].(string); ok {
		namespace = nsArg
	}

	image := ""
	if imageArg, ok := params.GetArguments()["image"].(string); ok {
		image = imageArg
	}

	var timeout time.Duration
	if timeoutRaw, exists := params.GetArguments()["timeout_seconds"]; exists && timeoutRaw != nil {
		switch v := timeoutRaw.(type) {
		case float64:
			timeout = time.Duration(int64(v)) * time.Second
		case int:
			timeout = time.Duration(v) * time.Second
		case int64:
			timeout = time.Duration(v) * time.Second
		default:
			return api.NewToolCallResult("", errors.New("timeout_seconds must be a numeric value")), nil
		}
	}

	output, execErr := params.NodesDebugExec(params.Context, namespace, nodeName, image, command, timeout)
	if output == "" && execErr == nil {
		output = fmt.Sprintf("Command executed successfully on node %s but produced no output.", nodeName)
	}
	return api.NewToolCallResult(output, execErr), nil
}

func toStringSlice(arg any) ([]string, error) {
	if arg == nil {
		return nil, errors.New("command is required")
	}
	raw, ok := arg.([]interface{})
	if !ok {
		return nil, errors.New("command must be an array of strings")
	}
	if len(raw) == 0 {
		return nil, errors.New("command array cannot be empty")
	}
	command := make([]string, 0, len(raw))
	for _, item := range raw {
		str, ok := item.(string)
		if !ok {
			return nil, errors.New("command items must be strings")
		}
		command = append(command, str)
	}
	return command, nil
}
