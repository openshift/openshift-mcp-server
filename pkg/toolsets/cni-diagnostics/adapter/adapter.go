package adapter

import (
	"context"
	"time"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/cluster-diagnostics/nodesdebug"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
)

// RunDebugNodeCommandFunc is an alias for the function type required by ovn-kubernetes-mcp.
// This matches ovnkmcp.RunDebugNodeCommandFuncType exactly.
type RunDebugNodeCommandFunc = func(ctx context.Context, namespace string, nodeName string, image string, command []string, hostPath string, mountPath string, timeout time.Duration) (string, string, error)

// RunPodExecCommandFunc is an alias for the function type required by ovn-kubernetes-mcp.
// This matches the network-tools RunPodExecCommandFuncType exactly.
type RunPodExecCommandFunc = func(ctx context.Context, namespace, name, container string, command []string) (string, string, error)

// NewRunDebugNodeCommand creates a RunDebugNodeCommandFunc that uses kubernetes-mcp-server's
// nodesdebug.NodeDebug to execute commands on nodes via privileged debug pods.
func NewRunDebugNodeCommand(k api.KubernetesClient) RunDebugNodeCommandFunc {
	return func(ctx context.Context, namespace string, nodeName string, image string, command []string, hostPath string, mountPath string, timeout time.Duration) (string, string, error) {
		nodeDebug := nodesdebug.NewNodeDebug(k)
		return nodeDebug.NodesDebugExec(ctx, namespace, nodeName, image, command, hostPath, mountPath, timeout)
	}
}

// NewRunPodExecCommand creates a RunPodExecCommandFunc that uses kubernetes-mcp-server's
// kubernetes.Core to execute commands in existing pods.
func NewRunPodExecCommand(k api.KubernetesClient) RunPodExecCommandFunc {
	return func(ctx context.Context, namespace, name, container string, command []string) (string, string, error) {
		core := kubernetes.NewCore(k)
		return core.PodsExec(ctx, namespace, name, container, command)
	}
}
