package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	ovstypes "github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/ovs/types"
	"github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/utils"
	"github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/utils/headtail"
	"github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/utils/pattern"
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
	// ovs-vsctl tool registration
	mcp.AddTool(server,
		&mcp.Tool{
			Name: "ovs-vsctl",
			Description: fmt.Sprintf(`ovs-vsctl allows to run an ovs-vsctl command against an ovnkube-node pod to inspect the OVS switch configuration.
Parameters:
- namespace (required): Kubernetes namespace of the OVS pod
- name (required): Name of the pod running OVS
- action (required): The ovs-vsctl subcommand to run.
                     show         : Display a comprehensive overview of OVS configuration in a hierarchical format (bridges, ports, interfaces, controllers).
                     list-br      : List all OVS bridges on the pod.
                     list-ports   : List all ports on a specific OVS bridge (requires bridge).
                     list-ifaces  : List all interfaces on a specific OVS bridge (requires bridge).
- bridge (required for "list-ports" and "list-ifaces"): Name of the OVS bridge (e.g., "br-int")
- head (optional, only used when action is "show"): Return only first N lines. Default: %d lines if tail is not specified
- tail (optional, only used when action is "show"): Return only last N lines
- apply_tail_first (optional, only used when action is "show"): If both head and tail are set and apply_tail_first is true, apply tail before head. Default: false

Example:
- namespace='ovn-kubernetes', name='ovnkube-node-xxxxx', action='show'
- namespace='ovn-kubernetes', name='ovnkube-node-xxxxx', action='list-br'
- namespace='ovn-kubernetes', name='ovnkube-node-xxxxx', action='list-ports', bridge='br-int'
- namespace='ovn-kubernetes', name='ovnkube-node-xxxxx', action='list-ifaces', bridge='br-int'

Example output (action='show'):
{
  "output": "a1b2c3d4-5678-90ab-cdef-1234567890ab\n    Bridge br-int\n        Port ovn-k8s-mp0\n            Interface ovn-k8s-mp0\n                type: internal\n    ovs_version: \"2.17.0\""
}

Example output (action='list-br'):
{
  "bridges": ["br-int", "br-ex", "br-local"]
}

Example output (action='list-ports'):
{
  "ports": ["patch-br-int-to-br-ex", "veth1234", "ovn-k8s-mp0"]
}

Example output (action='list-ifaces'):
{
  "interfaces": ["patch-br-int-to-br-ex", "veth1234", "ovn-k8s-mp0"]
}
`, DefaultMaxLines),
		}, s.Vsctl)

	// ovs-ofctl tool registration
	mcp.AddTool(server,
		&mcp.Tool{
			Name: "ovs-ofctl",
			Description: fmt.Sprintf(`ovs-ofctl allows to run an ovs-ofctl command against an ovnkube-node pod to inspect the OpenFlow state of an OVS bridge.
Parameters:
- namespace (required): Kubernetes namespace of the OVS pod
- name (required): Name of the pod running OVS
- action (required): The ovs-ofctl subcommand to run.
                     dump-flows : Dump the OpenFlow flow entries programmed on the specified bridge.
- bridge (required): Name of the OVS bridge (e.g., "br-int")
- pattern (optional): Regex pattern to filter output lines
- head (optional): Return only first N lines. Default: %d lines if tail is not specified
- tail (optional): Return only last N lines
- apply_tail_first (optional): If both head and tail are set and apply_tail_first is true, apply tail before head. Default: false

Example:
- namespace='ovn-kubernetes', name='ovnkube-node-xxxxx', action='dump-flows', bridge='br-int'

Example output:
{
  "bridge": "br-int",
  "flows": [
    "cookie=0x0, duration=123.456s, table=0, n_packets=100, n_bytes=10000, priority=100,in_port=1 actions=output:2",
    "cookie=0x0, duration=123.456s, table=0, n_packets=50, n_bytes=5000, priority=90,in_port=2 actions=output:1"
  ]
}
`, DefaultMaxLines),
		}, s.Ofctl)

	// ovs-appctl tool registration
	mcp.AddTool(server,
		&mcp.Tool{
			Name: "ovs-appctl",
			Description: fmt.Sprintf(`ovs-appctl allows to run an ovs-appctl command against an ovnkube-node pod to interact with the OVS daemons for datapath and OpenFlow debugging.
Parameters:
- namespace (required): Kubernetes namespace of the OVS pod
- name (required): Name of the pod running OVS
- action (required): The ovs-appctl subcommand to run.
                     dpctl/dump-conntrack : Dump connection tracking entries from the OVS datapath.
                     ofproto/trace        : Simulate packet processing through the OpenFlow pipeline (requires bridge and flow).
- bridge (required for "ofproto/trace"): Name of the OVS bridge (e.g., "br-int")
- flow (required for "ofproto/trace"): Flow specification describing the packet to trace (e.g., "in_port=1,ip,nw_src=10.244.0.5,nw_dst=10.96.0.1")
- additional_params (optional, only used when action is "dpctl/dump-conntrack"): Additional CLI arguments to pass to dpctl/dump-conntrack (e.g., ["zone=5"])
- pattern (optional): Regex pattern to filter output lines
- head (optional): Return only first N lines. Default: %d lines if tail is not specified
- tail (optional): Return only last N lines
- apply_tail_first (optional): If both head and tail are set and apply_tail_first is true, apply tail before head. Default: false

Example:
- namespace='ovn-kubernetes', name='ovnkube-node-xxxxx', action='dpctl/dump-conntrack'
- namespace='ovn-kubernetes', name='ovnkube-node-xxxxx', action='ofproto/trace', bridge='br-int', flow='in_port=1,ip,nw_src=10.244.0.5,nw_dst=10.96.0.1'

Example output (action='dpctl/dump-conntrack'):
{
  "entries": [
    "tcp,orig=(src=10.244.0.5,dst=10.96.0.1,sport=45678,dport=443),reply=(src=10.96.0.1,dst=10.244.0.5,sport=443,dport=45678)"
  ]
}

Example output (action='ofproto/trace'):
{
  "bridge": "br-int",
  "flow": "in_port=1,ip,nw_src=10.244.0.5,nw_dst=10.96.0.1",
  "output": "Flow: ip,in_port=1,nw_src=10.244.0.5,nw_dst=10.96.0.1\n\nbridge(\"br-int\")\n...\nFinal flow: ...\nDatapath actions: ..."
}
`, DefaultMaxLines),
		}, s.Appctl)
}

// Vsctl dispatches to the appropriate ovs-vsctl subcommand based on the
// required "action" parameter.
func (s *MCPServer) Vsctl(ctx context.Context, req *mcp.CallToolRequest,
	in ovstypes.VsctlParams) (*mcp.CallToolResult, ovstypes.VsctlResult, error) {

	if err := validateVsctlAction(in.Action); err != nil {
		return nil, ovstypes.VsctlResult{}, err
	}

	switch ovstypes.VsctlAction(in.Action) {
	case ovstypes.VsctlShow:
		output, err := s.show(ctx, in.Namespace, in.Name, in.HeadTailParams)
		return nil, ovstypes.VsctlResult{Output: output}, err

	case ovstypes.VsctlListBr:
		bridges, err := s.listBridges(ctx, in.Namespace, in.Name)
		return nil, ovstypes.VsctlResult{Bridges: bridges}, err

	case ovstypes.VsctlListPorts:
		ports, err := s.listPorts(ctx, in.Namespace, in.Name, in.Bridge)
		return nil, ovstypes.VsctlResult{Ports: ports}, err

	case ovstypes.VsctlListIfaces:
		ifaces, err := s.listInterfaces(ctx, in.Namespace, in.Name, in.Bridge)
		return nil, ovstypes.VsctlResult{Interfaces: ifaces}, err

	default:
		return nil, ovstypes.VsctlResult{}, fmt.Errorf(`invalid action %q: must be one of "show", "list-br", "list-ports", "list-ifaces"`, in.Action)
	}
}

// Ofctl dispatches to the appropriate ovs-ofctl subcommand based on the
// required "action" parameter.
func (s *MCPServer) Ofctl(ctx context.Context, req *mcp.CallToolRequest,
	in ovstypes.OfctlParams) (*mcp.CallToolResult, ovstypes.OfctlResult, error) {

	if err := validateOfctlAction(in.Action); err != nil {
		return nil, ovstypes.OfctlResult{}, err
	}

	switch ovstypes.OfctlAction(in.Action) {
	case ovstypes.OfctlDumpFlows:
		flows, err := s.dumpFlows(ctx, in.Namespace, in.Name, in.Bridge, in.PatternParams, in.HeadTailParams)
		return nil, ovstypes.OfctlResult{Bridge: in.Bridge, Flows: flows}, err

	default:
		return nil, ovstypes.OfctlResult{}, fmt.Errorf(`invalid action %q: must be one of "dump-flows"`, in.Action)
	}
}

// Appctl dispatches to the appropriate ovs-appctl subcommand based on the
// required "action" parameter.
func (s *MCPServer) Appctl(ctx context.Context, req *mcp.CallToolRequest,
	in ovstypes.AppctlParams) (*mcp.CallToolResult, ovstypes.AppctlResult, error) {

	if err := validateAppctlAction(in.Action); err != nil {
		return nil, ovstypes.AppctlResult{}, err
	}

	switch ovstypes.AppctlAction(in.Action) {
	case ovstypes.AppctlDumpConntrack:
		entries, err := s.dumpConntrack(ctx, in.Namespace, in.Name, in.AdditionalParams, in.PatternParams, in.HeadTailParams)
		return nil, ovstypes.AppctlResult{Entries: entries}, err

	case ovstypes.AppctlOfprotoTrace:
		output, err := s.dumpOfprotoTrace(ctx, in.Namespace, in.Name, in.Bridge, in.Flow, in.PatternParams, in.HeadTailParams)
		return nil, ovstypes.AppctlResult{Bridge: in.Bridge, Flow: in.Flow, Output: output}, err

	default:
		return nil, ovstypes.AppctlResult{}, fmt.Errorf(`invalid action %q: must be one of "dpctl/dump-conntrack", "ofproto/trace"`, in.Action)
	}
}

// listBridges lists all OVS bridges on the pod using 'ovs-vsctl list-br'.
func (s *MCPServer) listBridges(ctx context.Context, namespace, name string) ([]string, error) {
	stdout, stderr, err := s.runPodExecCommand(ctx, namespace, name, "", []string{"ovs-vsctl", "list-br"})
	if err != nil {
		return []string{}, fmt.Errorf("failed to retrieve ovs bridge from pod %s/%s: %w",
			namespace, name, err)
	}
	if stderr != "" {
		return []string{}, fmt.Errorf("failed to retrieve ovs bridge from pod %s/%s: %s",
			namespace, name, stderr)
	}
	return utils.StripEmptyLines(strings.Split(stdout, "\n")), nil
}

// show returns the comprehensive OVS configuration overview via 'ovs-vsctl show'.
func (s *MCPServer) show(ctx context.Context, namespace, name string, headTailParams headtail.HeadTailParams) (string, error) {
	stdout, stderr, err := s.runPodExecCommand(ctx, namespace, name, "", []string{"ovs-vsctl", "show"})
	if err != nil {
		return "", fmt.Errorf("failed to retrieve ovs configuration from pod %s/%s: %w",
			namespace, name, err)
	}
	if stderr != "" {
		return "", fmt.Errorf("failed to retrieve ovs configuration from pod %s/%s: %s",
			namespace, name, stderr)
	}
	lines := utils.StripEmptyLines(strings.Split(stdout, "\n"))
	lines = headTailParams.Apply(lines, DefaultMaxLines)
	return strings.Join(lines, "\n"), nil
}

// listPorts lists all ports on a specific OVS bridge via 'ovs-vsctl list-ports'.
func (s *MCPServer) listPorts(ctx context.Context, namespace, name, bridge string) ([]string, error) {
	if err := validateBridgeName(bridge); err != nil {
		return []string{}, err
	}
	stdout, stderr, err := s.runPodExecCommand(ctx, namespace, name, "", []string{"ovs-vsctl", "list-ports", bridge})
	if err != nil {
		return []string{}, fmt.Errorf("failed to retrieve ports for bridge %s from pod %s/%s: %w",
			bridge, namespace, name, err)
	}
	if stderr != "" {
		return []string{}, fmt.Errorf("failed to retrieve ports for bridge %s from pod %s/%s: %s",
			bridge, namespace, name, stderr)
	}
	return utils.StripEmptyLines(strings.Split(stdout, "\n")), nil
}

// listInterfaces lists all interfaces on a specific OVS bridge via 'ovs-vsctl list-ifaces'.
func (s *MCPServer) listInterfaces(ctx context.Context, namespace, name, bridge string) ([]string, error) {
	if err := validateBridgeName(bridge); err != nil {
		return []string{}, err
	}
	stdout, stderr, err := s.runPodExecCommand(ctx, namespace, name, "", []string{"ovs-vsctl", "list-ifaces", bridge})
	if err != nil {
		return []string{}, fmt.Errorf("failed to retrieve interfaces for bridge %s from pod %s/%s: %w",
			bridge, namespace, name, err)
	}
	if stderr != "" {
		return []string{}, fmt.Errorf("failed to retrieve interfaces for bridge %s from pod %s/%s: %s",
			bridge, namespace, name, stderr)
	}
	return utils.StripEmptyLines(strings.Split(stdout, "\n")), nil
}

// dumpFlows dumps OpenFlow flows from a specific OVS bridge via 'ovs-ofctl dump-flows'.
func (s *MCPServer) dumpFlows(ctx context.Context, namespace, name, bridge string,
	patternParams pattern.PatternParams, headTailParams headtail.HeadTailParams) ([]string, error) {
	if err := validateBridgeName(bridge); err != nil {
		return []string{}, err
	}
	flows, err := patternParams.ExecuteWithMatch(func() ([]string, error) {
		stdout, stderr, err := s.runPodExecCommand(ctx, namespace, name, "", []string{"ovs-ofctl", "dump-flows", bridge})
		if err != nil {
			return nil, fmt.Errorf("failed to dump flows for bridge %s on pod %s/%s: %w",
				bridge, namespace, name, err)
		}
		if stderr != "" {
			return nil, fmt.Errorf("failed to dump flows for bridge %s on pod %s/%s: %s",
				bridge, namespace, name, stderr)
		}
		return utils.StripEmptyLines(strings.Split(stdout, "\n")), nil
	}, true)
	if err != nil {
		return []string{}, err
	}
	return headTailParams.Apply(flows, DefaultMaxLines), nil
}

// dumpConntrack dumps connection tracking entries from OVS datapath via 'ovs-appctl dpctl/dump-conntrack'.
func (s *MCPServer) dumpConntrack(ctx context.Context, namespace, name string, additionalParams []string,
	patternParams pattern.PatternParams, headTailParams headtail.HeadTailParams) ([]string, error) {
	if len(additionalParams) > 0 {
		if err := validateConntrackParams(additionalParams); err != nil {
			return []string{}, err
		}
	}
	cmd := []string{"ovs-appctl", "dpctl/dump-conntrack"}
	if len(additionalParams) > 0 {
		cmd = append(cmd, additionalParams...)
	}
	entries, err := patternParams.ExecuteWithMatch(func() ([]string, error) {
		stdout, stderr, err := s.runPodExecCommand(ctx, namespace, name, "", cmd)
		if err != nil {
			return nil, fmt.Errorf("failed to dump conntrack on pod %s/%s: %w",
				namespace, name, err)
		}
		if stderr != "" {
			return nil, fmt.Errorf("failed to dump conntrack on pod %s/%s: %s",
				namespace, name, stderr)
		}
		return utils.StripEmptyLines(strings.Split(stdout, "\n")), nil
	}, true)
	if err != nil {
		return []string{}, err
	}
	return headTailParams.Apply(entries, DefaultMaxLines), nil
}

// dumpOfprotoTrace traces a packet through the OpenFlow pipeline via 'ovs-appctl ofproto/trace'.
func (s *MCPServer) dumpOfprotoTrace(ctx context.Context, namespace, name, bridge, flow string,
	patternParams pattern.PatternParams, headTailParams headtail.HeadTailParams) (string, error) {
	if err := validateBridgeName(bridge); err != nil {
		return "", err
	}
	if err := validateFlowSpec(flow); err != nil {
		return "", err
	}
	cmd := []string{"ovs-appctl", "ofproto/trace", bridge, flow}
	lines, err := patternParams.ExecuteWithMatch(func() ([]string, error) {
		stdout, stderr, err := s.runPodExecCommand(ctx, namespace, name, "", cmd)
		if err != nil {
			return nil, fmt.Errorf("failed to trace flow on bridge %s, pod %s/%s: %w",
				bridge, namespace, name, err)
		}
		if stderr != "" {
			return nil, fmt.Errorf("failed to trace flow on bridge %s, pod %s/%s: %s",
				bridge, namespace, name, stderr)
		}
		return utils.StripEmptyLines(strings.Split(stdout, "\n")), nil
	}, true)
	if err != nil {
		return "", err
	}
	lines = headTailParams.Apply(lines, DefaultMaxLines)
	return strings.Join(lines, "\n"), nil
}
