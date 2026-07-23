package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/utils/timeout"
)

type RunDebugNodeCommandFuncType func(ctx context.Context, namespace string, nodeName string, image string, command []string, hostPath string, mountPath string, timeout time.Duration) (string, string, error)
type RunPodExecCommandFuncType func(ctx context.Context, namespace, name, container string, command []string) (string, string, error)

// Config contains the configuration for the network tools MCP server.
type Config struct {
	// PwruImage is the container image to use for running the pwru command on the node.
	PwruImage string
	// TcpdumpImage is the container image to use for running the tcpdump command on the node.
	TcpdumpImage string
}

// MCPServer provides MCP server functionality for network tools operations.
type MCPServer struct {
	runDebugNodeCommand RunDebugNodeCommandFuncType
	runPodExecCommand   RunPodExecCommandFuncType
	cfg                 Config
}

// NewMCPServer creates a new MCP server instance
func NewMCPServer(runNodeCommand RunDebugNodeCommandFuncType, runPodExecCommand RunPodExecCommandFuncType, cfg Config) (*MCPServer, error) {
	if runNodeCommand == nil {
		return nil, fmt.Errorf("function to run debug node command is nil")
	}
	if runPodExecCommand == nil {
		return nil, fmt.Errorf("function to run pod exec command is nil")
	}
	return &MCPServer{
		runDebugNodeCommand: runNodeCommand,
		runPodExecCommand:   runPodExecCommand,
		cfg:                 cfg,
	}, nil
}

// AddTools registers network tools with the MCP server
func (s *MCPServer) AddTools(server *mcp.Server) {
	mcp.AddTool(server,
		&mcp.Tool{
			Name: "tcpdump",
			Description: fmt.Sprintf(`Capture network packets on a node or inside a pod with strict safety controls.

Supports both node-level and pod-level packet capture with BPF filtering.

This tool creates a specialized debug pod on the specified node for node-level captures.

Parameters:
- target_type: 'node' or 'pod' (required)
- name: Name of the target (node or pod) (required)
- namespace: Namespace of the target (node or pod). Required when target_type is 'pod'. Optional when target_type is 'node' and defaults to 'default'.
- container_name: Name of the container in the pod when target_type is 'pod' (optional, uses default container if not specified)
- interface: Network interface name or 'any' (optional, uses default if not specified)
- packet_count: Number of packets to capture (default: 100, max: 1000)
- bpf_filter: BPF filter expression to match packets (optional, e.g., "tcp and dst port 8080", "host 10.0.0.1")
- snaplen: Snapshot length in bytes (default: 96, max: 1500)
- timeout_seconds (optional): Timeout in seconds for the command execution. If not specified, server default timeout is used. The maximum value is %d seconds.

Examples:
- Capture on node: {"target_type": "node", "name": "worker-1", "interface": "eth0", "packet_count": 100, "bpf_filter": "tcp port 80"}
- Capture in pod: {"target_type": "pod", "name": "my-pod", "namespace": "default", "interface": "eth0", "packet_count": 100, "bpf_filter": "host 10.0.0.1"}
- Capture DNS: {"target_type": "node", "name": "worker-1", "interface": "any", "packet_count": 50, "bpf_filter": "port 53"}`,
				int(timeout.MaxTimeout.Seconds())),
		}, s.Tcpdump)
	mcp.AddTool(server,
		&mcp.Tool{
			Name: "pwru",
			Description: fmt.Sprintf(`Trace packets through the Linux kernel networking stack using eBPF.

pwru (packet, where are you?) shows which kernel functions process a packet, helping debug packet
drops, routing issues, and understanding the kernel's packet processing path.

This tool creates a specialized debug pod on the specified node with necessary eBPF capabilities
to trace packets through kernel networking functions.

Parameters:
- node_name: Name of the node to run pwru on (required)
- node_pod_namespace (optional): Namespace of the debug pod on which the command is expected to be executed. Default: 'default'
- bpf_filter: BPF filter expression to match packets (optional, e.g., "tcp and dst port 8080", "host 10.0.0.1")
- output_limit_lines: Maximum number of trace events to capture (default: 100, max: 1000)
- timeout_seconds (optional): Timeout in seconds for the command execution. If not specified, server default timeout is used. The maximum value is %d seconds.

Examples:
- Basic trace: {"node_name": "worker-1", "bpf_filter": "host 10.244.0.5", "output_limit_lines": 100}
- TCP traffic: {"node_name": "worker-1", "bpf_filter": "tcp and dst port 8080", "output_limit_lines": 50}
- ICMP packets: {"node_name": "worker-1", "bpf_filter": "icmp", "output_limit_lines": 100}`,
				int(timeout.MaxTimeout.Seconds())),
		}, s.Pwru)
}
