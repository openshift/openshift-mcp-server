package nodes

import (
	"context"
	"testing"
	"time"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/ocp/nodes"
)

type staticRequest struct {
	args map[string]any
}

func (s staticRequest) GetArguments() map[string]any {
	return s.args
}

func TestNodesDebugExecHandlerValidatesInput(t *testing.T) {
	t.Run("missing node", func(t *testing.T) {
		params := api.ToolHandlerParams{
			Context:         context.Background(),
			ToolCallRequest: staticRequest{args: map[string]any{}},
		}
		result, err := nodesDebugExec(params)
		if err != nil {
			t.Fatalf("handler returned error: %v", err)
		}
		if result.Error == nil || result.Error.Error() != "missing required argument: node" {
			t.Fatalf("unexpected error: %v", result.Error)
		}
	})

	t.Run("invalid command type", func(t *testing.T) {
		params := api.ToolHandlerParams{
			Context: context.Background(),
			ToolCallRequest: staticRequest{args: map[string]any{
				"node":    "worker-0",
				"command": "ls -la",
			}},
		}
		result, err := nodesDebugExec(params)
		if err != nil {
			t.Fatalf("handler returned error: %v", err)
		}
		if result.Error == nil || result.Error.Error() != "invalid command argument: command must be an array of strings" {
			t.Fatalf("unexpected error: %v", result.Error)
		}
	})
}

func TestNodesDebugExecHandlerExecutesCommand(t *testing.T) {
	env := nodes.NewNodeDebugTestEnv(t)
	env.Pods.Logs = "done"

	// Call NodesDebugExec directly instead of going through the handler
	// This avoids the need to mock the full kubernetes.Kubernetes type
	output, err := nodes.NodesDebugExec(
		context.Background(),
		env.Kubernetes,
		"debug",
		"infra-node",
		"registry.local/debug:latest",
		[]string{"systemctl", "status", "kubelet"},
		15*time.Second,
	)

	if err != nil {
		t.Fatalf("NodesDebugExec returned error: %v", err)
	}
	if output != "done" {
		t.Fatalf("unexpected output: %q", output)
	}

	created := env.Pods.Created
	if created == nil {
		t.Fatalf("expected pod creation")
	}
	if created.Namespace != "debug" {
		t.Fatalf("expected namespace override, got %q", created.Namespace)
	}
	if created.Spec.Containers[0].Image != "registry.local/debug:latest" {
		t.Fatalf("expected custom image, got %q", created.Spec.Containers[0].Image)
	}
	expectedCommand := []string{"systemctl", "status", "kubelet"}
	if len(created.Spec.Containers[0].Command) != len(expectedCommand) {
		t.Fatalf("unexpected command length: %v", created.Spec.Containers[0].Command)
	}
	for i, part := range expectedCommand {
		if created.Spec.Containers[0].Command[i] != part {
			t.Fatalf("command[%d]=%q expected %q", i, created.Spec.Containers[0].Command[i], part)
		}
	}
}
