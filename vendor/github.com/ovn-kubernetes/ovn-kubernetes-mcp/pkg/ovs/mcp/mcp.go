package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	k8stypes "github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/kubernetes/types"
	ovstypes "github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/ovs/types"
	"github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/utils"
)

type RunPodExecCommandFuncType func(ctx context.Context, namespace, name, container string, command []string) (string, string, error)

// MCPServer provides OVS layer analysis tools
type MCPServer struct {
	runPodExecCommand RunPodExecCommandFuncType
}

// NewMCPServer creates a new OVS MCP server
func NewMCPServer(runPodExecCommand RunPodExecCommandFuncType) (*MCPServer, error) {
	if runPodExecCommand == nil {
		return nil, fmt.Errorf("function to run pod exec command is nil")
	}
	return &MCPServer{
		runPodExecCommand: runPodExecCommand,
	}, nil
}

// AddTools registers OVS tools with the MCP server
func (s *MCPServer) AddTools(server *mcp.Server) {
	mcp.AddTool(server,
		&mcp.Tool{
			Name: "ovs-list-br",
			Description: `List all OVS bridges on a specific pod.

Runs 'ovs-vsctl list-br' command and returns the names of all configured bridges.

Parameters:
- namespace: Kubernetes namespace of the OVS pod
- name: Name of the pod running OVS

Example output:
{
  "bridges": [
    "br-int",
    "br-ex",
    "br-local"
  ]
}`,
		}, s.ListBridges)

	mcp.AddTool(server,
		&mcp.Tool{
			Name: "ovs-list-ports",
			Description: `List all ports on a specific OVS bridge.

Runs 'ovs-vsctl list-ports' command and returns the names of all ports attached to the specified bridge.

Parameters:
- namespace: Kubernetes namespace of the OVS pod
- name: Name of the pod running OVS
- bridge: Name of the OVS bridge (e.g., "br-int")

Example output:
{
  "ports": [
    "patch-br-int-to-br-ex",
    "veth1234",
    "ovn-k8s-mp0"
  ]
}`,
		}, s.ListPorts)

	mcp.AddTool(server,
		&mcp.Tool{
			Name: "ovs-list-ifaces",
			Description: `List all interfaces on a specific OVS bridge.

Runs 'ovs-vsctl list-ifaces' command and returns the names of all interfaces attached to the specified bridge.

Parameters:
- namespace: Kubernetes namespace of the OVS pod
- name: Name of the pod running OVS
- bridge: Name of the OVS bridge (e.g., "br-int")

Example output:
{
  "interfaces": [
    "patch-br-int-to-br-ex",
    "veth1234",
    "ovn-k8s-mp0"
  ]
}`,
		}, s.ListInterfaces)

	mcp.AddTool(server,
		&mcp.Tool{
			Name: "ovs-vsctl-show",
			Description: fmt.Sprintf(`Display a comprehensive overview of OVS configuration.

Runs 'ovs-vsctl show' command and returns detailed information about bridges, ports, interfaces,
controllers, and their configurations in a hierarchical format.

This command is useful for getting a complete view of the OVS switch configuration including:
- All bridges and their configurations
- Ports and interfaces attached to each bridge
- Controller connections and status
- Interface types and options
- Port configurations and tags

Parameters:
- namespace: Kubernetes namespace of the OVS pod
- name: Name of the pod running OVS
- head (optional): Return only first N lines. Default: %d lines if tail is not specified
- tail (optional): Return only last N lines
- apply_tail_first (optional): If both head and tail are set and apply_tail_first is true, apply tail before head. Default: false

Example output:
{
  "output": "a1b2c3d4-5678-90ab-cdef-1234567890ab\n    Bridge br-int\n        Port ovn-k8s-mp0\n            Interface ovn-k8s-mp0\n                type: internal\n        Port br-int\n            Interface br-int\n                type: internal\n    ovs_version: \"2.17.0\""
}`, defaultMaxLines),
		}, s.Show)

	mcp.AddTool(server,
		&mcp.Tool{
			Name: "ovs-ofctl-dump-flows",
			Description: fmt.Sprintf(`Dump OpenFlow flows from a specific OVS bridge.

Runs 'ovs-ofctl dump-flows' command on the specified bridge and returns the flow entries.

Parameters:
- namespace: Kubernetes namespace of the OVS pod
- name: Name of the pod running OVS
- bridge: Name of the OVS bridge (e.g., "br-int")
- pattern (optional): Regex pattern to filter flows
- head (optional): Return only first N lines. Default: %d lines if tail is not specified
- tail (optional): Return only last N lines
- apply_tail_first (optional): If both head and tail are set and apply_tail_first is true, apply tail before head. Default: false

Example output:
{
  "bridge": "br-int",
  "flows": [
    "cookie=0x0, duration=123.456s, table=0, n_packets=100, n_bytes=10000, priority=100,in_port=1 actions=output:2",
    "cookie=0x0, duration=123.456s, table=0, n_packets=50, n_bytes=5000, priority=90,in_port=2 actions=output:1"
  ]
}`, defaultMaxLines),
		}, s.DumpFlows)

	mcp.AddTool(server,
		&mcp.Tool{
			Name: "ovs-appctl-dump-conntrack",
			Description: fmt.Sprintf(`Dump connection tracking entries from OVS datapath.

Runs 'ovs-appctl dpctl/dump-conntrack' command and returns the conntrack entries.

Connection tracking (conntrack) maintains state for stateful firewall rules and NAT.
Each entry shows source/destination IPs, ports, protocol, connection state, and more.

Parameters:
- namespace: Kubernetes namespace of the OVS pod
- name: Name of the pod running OVS
- pattern (optional): Regex pattern to filter conntrack entries
- head (optional): Return only first N lines. Default: %d lines if tail is not specified
- tail (optional): Return only last N lines
- apply_tail_first (optional): If both head and tail are set and apply_tail_first is true, apply tail before head. Default: false
- additional_params (optional): Additional parameters to pass to dpctl/dump-conntrack command (e.g., ["zone=5"])

Example output:
{
  "entries": [
    "tcp,orig=(src=10.244.0.5,dst=10.96.0.1,sport=45678,dport=443),reply=(src=10.96.0.1,dst=10.244.0.5,sport=443,dport=45678)",
    "udp,orig=(src=10.244.0.3,dst=8.8.8.8,sport=53214,dport=53),reply=(src=8.8.8.8,dst=10.244.0.3,sport=53,dport=53214)"
  ]
}`, defaultMaxLines),
		}, s.DumpConntrack)

	mcp.AddTool(server,
		&mcp.Tool{
			Name: "ovs-appctl-ofproto-trace",
			Description: fmt.Sprintf(`Trace a packet through the OpenFlow pipeline.

Runs 'ovs-appctl ofproto/trace' command to simulate packet processing through OpenFlow tables.
This shows which flows match, what actions are taken, and the final disposition of the packet.

The trace output is essential for debugging flow rules, understanding packet forwarding decisions,
and troubleshooting connectivity issues.

Parameters:
- namespace: Kubernetes namespace of the OVS pod
- name: Name of the pod running OVS
- bridge: Name of the OVS bridge (e.g., "br-int")
- flow: Flow specification describing the packet to trace (e.g., "in_port=1,ip,nw_src=10.244.0.5,nw_dst=10.96.0.1")
- pattern (optional): Regex pattern to filter trace output lines
- head (optional): Return only first N lines. Default: %d lines if tail is not specified
- tail (optional): Return only last N lines
- apply_tail_first (optional): If both head and tail are set and apply_tail_first is true, apply tail before head. Default: false

Flow specification examples:
- "in_port=1,icmp"
- "in_port=2,ip,nw_src=192.168.1.10,nw_dst=192.168.1.20"
- "in_port=3,tcp,nw_src=10.0.0.1,nw_dst=10.0.0.2,tp_src=12345,tp_dst=80"

Example output:
{
  "bridge": "br-int",
  "flow": "in_port=1,ip,nw_src=10.244.0.5,nw_dst=10.96.0.1",
  "output": "Flow: ip,in_port=1,nw_src=10.244.0.5,nw_dst=10.96.0.1\n\nbridge(\"br-int\")\n-------------\n 0. priority 100\n    resubmit(,10)\n10. ip,nw_dst=10.96.0.1, priority 200\n    load:0x1->NXM_NX_REG0[]\n    resubmit(,20)\n...\nFinal flow: ...\nDatapath actions: ..."
}`, defaultMaxLines),
		}, s.DumpOfprotoTrace)
}

func (s *MCPServer) ListBridges(ctx context.Context, req *mcp.CallToolRequest,
	in k8stypes.NamespacedNameParams) (*mcp.CallToolResult, ovstypes.BridgeResult, error) {
	result := ovstypes.BridgeResult{
		Bridges: []string{}, // Initialize with empty slice to ensure valid JSON even on error
	}

	// Run ovs-vsctl list-br command
	stdout, stderr, err := s.runPodExecCommand(ctx, in.Namespace, in.Name, "", []string{"ovs-vsctl", "list-br"})
	if err != nil {
		return nil, result, fmt.Errorf("failed to retrieve ovs bridge from pod %s/%s: %w",
			in.Namespace, in.Name, err)
	}
	if stderr != "" {
		return nil, result, fmt.Errorf("failed to retrieve ovs bridge from pod %s/%s: %s",
			in.Namespace, in.Name, stderr)
	}
	bridgeNames := utils.StripEmptyLines(strings.Split(stdout, "\n"))
	result.Bridges = append(result.Bridges, bridgeNames...)
	return nil, result, nil
}

// Show displays a comprehensive overview of OVS configuration.
func (s *MCPServer) Show(ctx context.Context, req *mcp.CallToolRequest,
	in ovstypes.ShowParams) (*mcp.CallToolResult, ovstypes.ShowResult, error) {
	result := ovstypes.ShowResult{}

	// Run ovs-vsctl show command
	stdout, stderr, err := s.runPodExecCommand(ctx, in.Namespace, in.Name, "", []string{"ovs-vsctl", "show"})
	if err != nil {
		return nil, result, fmt.Errorf("failed to retrieve ovs configuration from pod %s/%s: %w",
			in.Namespace, in.Name, err)
	}
	if stderr != "" {
		return nil, result, fmt.Errorf("failed to retrieve ovs configuration from pod %s/%s: %s",
			in.Namespace, in.Name, stderr)
	}
	lines := utils.StripEmptyLines(strings.Split(stdout, "\n"))

	// Apply the head and tail parameters to the lines
	lines = in.HeadTailParams.Apply(lines, defaultMaxLines)

	// Join all lines into a single output string
	result.Output = strings.Join(lines, "\n")
	return nil, result, nil
}

func (s *MCPServer) ListPorts(ctx context.Context, req *mcp.CallToolRequest,
	in ovstypes.GetOVSCommandParams) (*mcp.CallToolResult, ovstypes.PortResult, error) {
	result := ovstypes.PortResult{
		Ports: []string{}, // Initialize with empty slice to ensure valid JSON even on error
	}

	// Validate bridge name
	if err := validateBridgeName(in.Bridge); err != nil {
		return nil, result, err
	}

	// Run ovs-vsctl list-ports command
	stdout, stderr, err := s.runPodExecCommand(ctx, in.Namespace, in.Name, "", []string{"ovs-vsctl", "list-ports", in.Bridge})
	if err != nil {
		return nil, result, fmt.Errorf("failed to retrieve ports for bridge %s from pod %s/%s: %w",
			in.Bridge, in.Namespace, in.Name, err)
	}
	if stderr != "" {
		return nil, result, fmt.Errorf("failed to retrieve ports for bridge %s from pod %s/%s: %s",
			in.Bridge, in.Namespace, in.Name, stderr)
	}
	ports := utils.StripEmptyLines(strings.Split(stdout, "\n"))
	result.Ports = append(result.Ports, ports...)
	return nil, result, nil
}

func (s *MCPServer) ListInterfaces(ctx context.Context, req *mcp.CallToolRequest,
	in ovstypes.GetOVSCommandParams) (*mcp.CallToolResult, ovstypes.InterfaceResult, error) {
	result := ovstypes.InterfaceResult{
		Interfaces: []string{}, // Initialize with empty slice to ensure valid JSON even on error
	}

	// Validate bridge name
	if err := validateBridgeName(in.Bridge); err != nil {
		return nil, result, err
	}

	// Run ovs-vsctl list-ifaces command
	stdout, stderr, err := s.runPodExecCommand(ctx, in.Namespace, in.Name, "", []string{"ovs-vsctl", "list-ifaces", in.Bridge})
	if err != nil {
		return nil, result, fmt.Errorf("failed to retrieve interfaces for bridge %s from pod %s/%s: %w",
			in.Bridge, in.Namespace, in.Name, err)
	}
	if stderr != "" {
		return nil, result, fmt.Errorf("failed to retrieve interfaces for bridge %s from pod %s/%s: %s",
			in.Bridge, in.Namespace, in.Name, stderr)
	}
	ports := utils.StripEmptyLines(strings.Split(stdout, "\n"))
	result.Interfaces = append(result.Interfaces, ports...)
	return nil, result, nil
}

// DumpFlows dumps flows from a specific OVS bridge.
func (s *MCPServer) DumpFlows(ctx context.Context, req *mcp.CallToolRequest,
	in ovstypes.GetOVSCommandParams) (*mcp.CallToolResult, ovstypes.FlowsResult, error) {
	result := ovstypes.FlowsResult{
		Bridge: in.Bridge,
		Flows:  []string{}, // Initialize with empty slice to ensure valid JSON even on error
	}

	// Validate bridge name
	if err := validateBridgeName(in.Bridge); err != nil {
		return nil, result, err
	}

	// Filter flows by pattern if provided
	flows, err := in.PatternParams.ExecuteWithMatch(func() ([]string, error) {
		// Run ovs-ofctl dump-flows command
		stdout, stderr, err := s.runPodExecCommand(ctx, in.Namespace, in.Name, "", []string{"ovs-ofctl", "dump-flows", in.Bridge})
		if err != nil {
			return nil, fmt.Errorf("failed to dump flows for bridge %s on pod %s/%s: %w",
				in.Bridge, in.Namespace, in.Name, err)
		}
		if stderr != "" {
			return nil, fmt.Errorf("failed to dump flows for bridge %s on pod %s/%s: %s",
				in.Bridge, in.Namespace, in.Name, stderr)
		}
		flows := utils.StripEmptyLines(strings.Split(stdout, "\n"))
		return flows, nil
	}, true)
	if err != nil {
		return nil, result, err
	}

	// Apply the head and tail parameters to the flows
	flows = in.HeadTailParams.Apply(flows, defaultMaxLines)

	result.Flows = flows
	return nil, result, nil
}

// DumpConntrack dumps connection tracking entries from OVS datapath.
func (s *MCPServer) DumpConntrack(ctx context.Context, req *mcp.CallToolRequest,
	in ovstypes.DumpConntrackParams) (*mcp.CallToolResult, ovstypes.ConntrackResult, error) {
	result := ovstypes.ConntrackResult{
		Entries: []string{}, // Initialize with empty slice to ensure valid JSON even on error
	}

	// Validate additional parameters if provided
	if len(in.AdditionalParams) > 0 {
		if err := validateConntrackParams(in.AdditionalParams); err != nil {
			return nil, result, err
		}
	}

	// Build command with additional parameters
	cmd := []string{"ovs-appctl", "dpctl/dump-conntrack"}
	if len(in.AdditionalParams) > 0 {
		cmd = append(cmd, in.AdditionalParams...)
	}

	// Filter entries by pattern if provided
	entries, err := in.PatternParams.ExecuteWithMatch(func() ([]string, error) {
		// Run ovs-appctl dpctl/dump-conntrack command
		stdout, stderr, err := s.runPodExecCommand(ctx, in.Namespace, in.Name, "", cmd)
		if err != nil {
			return nil, fmt.Errorf("failed to dump conntrack on pod %s/%s: %w",
				in.Namespace, in.Name, err)
		}
		if stderr != "" {
			return nil, fmt.Errorf("failed to dump conntrack on pod %s/%s: %s",
				in.Namespace, in.Name, stderr)
		}
		entries := utils.StripEmptyLines(strings.Split(stdout, "\n"))
		return entries, nil
	}, true)
	if err != nil {
		return nil, result, err
	}

	// Apply the head and tail parameters to the entries
	entries = in.HeadTailParams.Apply(entries, defaultMaxLines)

	result.Entries = entries
	return nil, result, nil
}

// DumpOfprotoTrace traces a packet through the OpenFlow pipeline.
func (s *MCPServer) DumpOfprotoTrace(ctx context.Context, req *mcp.CallToolRequest,
	in ovstypes.OfprotoTraceParams) (*mcp.CallToolResult, ovstypes.OfprotoTraceResult, error) {
	result := ovstypes.OfprotoTraceResult{
		Bridge: in.Bridge,
		Flow:   in.Flow,
	}

	// Validate bridge name
	if err := validateBridgeName(in.Bridge); err != nil {
		return nil, result, err
	}

	// Validate flow specification
	if err := validateFlowSpec(in.Flow); err != nil {
		return nil, result, err
	}

	// Build command: ovs-appctl ofproto/trace <bridge> <flow>
	cmd := []string{"ovs-appctl", "ofproto/trace", in.Bridge, in.Flow}

	// Filter lines by pattern if provided
	lines, err := in.PatternParams.ExecuteWithMatch(func() ([]string, error) {
		// Run ovs-appctl ofproto/trace command
		stdout, stderr, err := s.runPodExecCommand(ctx, in.Namespace, in.Name, "", cmd)
		if err != nil {
			return nil, fmt.Errorf("failed to trace flow on bridge %s, pod %s/%s: %w",
				in.Bridge, in.Namespace, in.Name, err)
		}
		if stderr != "" {
			return nil, fmt.Errorf("failed to trace flow on bridge %s, pod %s/%s: %s",
				in.Bridge, in.Namespace, in.Name, stderr)
		}
		lines := utils.StripEmptyLines(strings.Split(stdout, "\n"))
		return lines, nil
	}, true)
	if err != nil {
		return nil, result, err
	}

	// Apply the head and tail parameters to the lines
	lines = in.HeadTailParams.Apply(lines, defaultMaxLines)

	// Join all lines into a single output string
	result.Output = strings.Join(lines, "\n")
	return nil, result, nil
}
