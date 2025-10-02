package core

import (
	"context"
	"testing"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	kubernetestest "github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
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
	env := kubernetestest.NewNodeDebugTestEnv(t)
	env.Pods.Logs = "done"

	params := api.ToolHandlerParams{
		Context:    context.Background(),
		Kubernetes: env.Kubernetes,
		ToolCallRequest: staticRequest{args: map[string]any{
			"node":            "infra-node",
			"command":         []any{"systemctl", "status", "kubelet"},
			"namespace":       "debug",
			"image":           "registry.local/debug:latest",
			"timeout_seconds": float64(15),
		}},
	}

	result, err := nodesDebugExec(params)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if result.Error != nil {
		t.Fatalf("expected no tool error, got %v", result.Error)
	}
	if result.Content != "done" {
		t.Fatalf("unexpected tool output: %q", result.Content)
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
	expectedCommand := []string{"chroot", "/host", "systemctl", "status", "kubelet"}
	if len(created.Spec.Containers[0].Command) != len(expectedCommand) {
		t.Fatalf("unexpected command length: %v", created.Spec.Containers[0].Command)
	}
	for i, part := range expectedCommand {
		if created.Spec.Containers[0].Command[i] != part {
			t.Fatalf("command[%d]=%q expected %q", i, created.Spec.Containers[0].Command[i], part)
		}
	}
}
