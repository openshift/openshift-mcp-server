package ovs

import (
	"context"
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/google/jsonschema-go/jsonschema"
	ovsmcp "github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/ovs/mcp"
	ovstypes "github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/ovs/types"
	"github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/utils/headtail"
	"github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/utils/pattern"
	"k8s.io/utils/ptr"
)

// Tools returns the consolidated OVS layer tools for the ovn-kubernetes toolset.
// Tools are grouped by their underlying CLI binary (ovs-vsctl, ovs-ofctl, ovs-appctl)
// and dispatch to specific commands via the required "action" parameter.
func Tools() []api.ServerTool {
	return []api.ServerTool{
		{Tool: api.Tool{
			Name: "ovs_vsctl",
			Description: `Run an ovs-vsctl command against an ovnkube-node pod.

The 'action' parameter selects the ovs-vsctl subcommand to run.

--- action: "show" ---
Display a comprehensive overview of OVS configuration.

Runs 'ovs-vsctl show' command and returns detailed information about bridges, ports, interfaces,
controllers, and their configurations in a hierarchical format.

This command is useful for getting a complete view of the OVS switch configuration including:
- All bridges and their configurations
- Ports and interfaces attached to each bridge
- Controller connections and status
- Interface types and options
- Port configurations and tags

Example output:
{
  "output": "a1b2c3d4-5678-90ab-cdef-1234567890ab\n    Bridge br-int\n        Port ovn-k8s-mp0\n            Interface ovn-k8s-mp0\n                type: internal\n        Port br-int\n            Interface br-int\n                type: internal\n    ovs_version: \"2.17.0\""
}

--- action: "list-br" ---
List all OVS bridges on a specific pod.

Runs 'ovs-vsctl list-br' command and returns the names of all configured bridges.

Example output:
{
  "bridges": [
    "br-int",
    "br-ex",
    "br-local"
  ]
}

--- action: "list-ports" ---
List all ports on a specific OVS bridge.

Runs 'ovs-vsctl list-ports' command and returns the names of all ports attached to the specified bridge.

Example output:
{
  "ports": [
    "patch-br-int-to-br-ex",
    "veth1234",
    "ovn-k8s-mp0"
  ]
}

--- action: "list-ifaces" ---
List all interfaces on a specific OVS bridge.

Runs 'ovs-vsctl list-ifaces' command and returns the names of all interfaces attached to the specified bridge.

Example output:
{
  "interfaces": [
    "patch-br-int-to-br-ex",
    "veth1234",
    "ovn-k8s-mp0"
  ]
}`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: `Kubernetes namespace of the ovnkube-node pod (e.g., "openshift-ovn-kubernetes")`,
					},
					"name": {
						Type:        "string",
						Description: `Name of the ovnkube-node pod (e.g., "ovnkube-node-xxxxx")`,
					},
					"action": {
						Type: "string",
						Enum: []any{
							string(ovstypes.VsctlShow),
							string(ovstypes.VsctlListBr),
							string(ovstypes.VsctlListPorts),
							string(ovstypes.VsctlListIfaces),
						},
						Description: `The ovs-vsctl subcommand to run: "show", "list-br", "list-ports", or "list-ifaces"`,
					},
					"bridge": {
						Type:        "string",
						Description: `Name of the OVS bridge (required for "list-ports" and "list-ifaces"; e.g., "br-int")`,
					},
					"head": {
						Type:        "integer",
						Description: fmt.Sprintf(`Return only first N lines (only used when action is "show"). Default: %d lines if tail is not specified`, ovsmcp.DefaultMaxLines),
					},
					"tail": {
						Type:        "integer",
						Description: `Return only last N lines (only used when action is "show")`,
					},
					"apply_tail_first": {
						Type:        "boolean",
						Description: `If both head and tail are set and apply_tail_first is true, apply tail before head (only used when action is "show"). Default: false`,
					},
				},
				Required: []string{"namespace", "name", "action"},
			},
			Annotations: api.ToolAnnotations{
				Title:         "OVN-Kubernetes: ovs-vsctl",
				ReadOnlyHint:  ptr.To(true),
				OpenWorldHint: ptr.To(true),
			},
		}, Handler: ovsVsctl},

		{Tool: api.Tool{
			Name: "ovs_ofctl",
			Description: `Run an ovs-ofctl command against an ovnkube-node pod.

The 'action' parameter selects the ovs-ofctl subcommand to run.

--- action: "dump-flows" ---
Dump OpenFlow flows from a specific OVS bridge.

Runs 'ovs-ofctl dump-flows' command on the specified bridge and returns the flow entries.

Example output:
{
  "bridge": "br-int",
  "flows": [
    "cookie=0x0, duration=123.456s, table=0, n_packets=100, n_bytes=10000, priority=100,in_port=1 actions=output:2",
    "cookie=0x0, duration=123.456s, table=0, n_packets=50, n_bytes=5000, priority=90,in_port=2 actions=output:1"
  ]
}`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: `Kubernetes namespace of the ovnkube-node pod (e.g., "openshift-ovn-kubernetes")`,
					},
					"name": {
						Type:        "string",
						Description: `Name of the ovnkube-node pod (e.g., "ovnkube-node-xxxxx")`,
					},
					"action": {
						Type: "string",
						Enum: []any{
							string(ovstypes.OfctlDumpFlows),
						},
						Description: `The ovs-ofctl subcommand to run: "dump-flows"`,
					},
					"bridge": {
						Type:        "string",
						Description: `Name of the OVS bridge (e.g., "br-int")`,
					},
					"pattern": {
						Type:        "string",
						Description: "Regex pattern to filter output lines",
					},
					"head": {
						Type:        "integer",
						Description: fmt.Sprintf(`Return only first N lines. Default: %d lines if tail is not specified`, ovsmcp.DefaultMaxLines),
					},
					"tail": {
						Type:        "integer",
						Description: "Return only last N lines",
					},
					"apply_tail_first": {
						Type:        "boolean",
						Description: "If both head and tail are set and apply_tail_first is true, apply tail before head. Default: false",
					},
				},
				Required: []string{"namespace", "name", "action", "bridge"},
			},
			Annotations: api.ToolAnnotations{
				Title:         "OVN-Kubernetes: ovs-ofctl",
				ReadOnlyHint:  ptr.To(true),
				OpenWorldHint: ptr.To(true),
			},
		}, Handler: ovsOfctl},

		{Tool: api.Tool{
			Name: "ovs_appctl",
			Description: `Run an ovs-appctl command against an ovnkube-node pod.

The 'action' parameter selects the ovs-appctl subcommand to run.

--- action: "dpctl/dump-conntrack" ---
Dump connection tracking entries from OVS datapath.

Runs 'ovs-appctl dpctl/dump-conntrack' command and returns the conntrack entries.

Connection tracking (conntrack) maintains state for stateful firewall rules and NAT.
Each entry shows source/destination IPs, ports, protocol, connection state, and more.

Example output:
{
  "entries": [
    "tcp,orig=(src=10.244.0.5,dst=10.96.0.1,sport=45678,dport=443),reply=(src=10.96.0.1,dst=10.244.0.5,sport=443,dport=45678)",
    "udp,orig=(src=10.244.0.3,dst=8.8.8.8,sport=53214,dport=53),reply=(src=8.8.8.8,dst=10.244.0.3,sport=53,dport=53214)"
  ]
}

--- action: "ofproto/trace" ---
Trace a packet through the OpenFlow pipeline.

Runs 'ovs-appctl ofproto/trace' command to simulate packet processing through OpenFlow tables.
This shows which flows match, what actions are taken, and the final disposition of the packet.

The trace output is essential for debugging flow rules, understanding packet forwarding decisions,
and troubleshooting connectivity issues.

Flow specification examples:
- "in_port=1,icmp"
- "in_port=2,ip,nw_src=192.168.1.10,nw_dst=192.168.1.20"
- "in_port=3,tcp,nw_src=10.0.0.1,nw_dst=10.0.0.2,tp_src=12345,tp_dst=80"

Example output:
{
  "bridge": "br-int",
  "flow": "in_port=1,ip,nw_src=10.244.0.5,nw_dst=10.96.0.1",
  "output": "Flow: ip,in_port=1,nw_src=10.244.0.5,nw_dst=10.96.0.1\n\nbridge(\"br-int\")\n-------------\n 0. priority 100\n    resubmit(,10)\n10. ip,nw_dst=10.96.0.1, priority 200\n    load:0x1->NXM_NX_REG0[]\n    resubmit(,20)\n...\nFinal flow: ...\nDatapath actions: ..."
}`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: `Kubernetes namespace of the ovnkube-node pod (e.g., "openshift-ovn-kubernetes")`,
					},
					"name": {
						Type:        "string",
						Description: `Name of the ovnkube-node pod (e.g., "ovnkube-node-xxxxx")`,
					},
					"action": {
						Type: "string",
						Enum: []any{
							string(ovstypes.AppctlDumpConntrack),
							string(ovstypes.AppctlOfprotoTrace),
						},
						Description: `The ovs-appctl subcommand to run: "dpctl/dump-conntrack" or "ofproto/trace"`,
					},
					"bridge": {
						Type:        "string",
						Description: `Name of the OVS bridge (required for "ofproto/trace"; e.g., "br-int")`,
					},
					"flow": {
						Type:        "string",
						Description: `Flow specification (required for "ofproto/trace"; e.g., "in_port=1,ip,nw_src=10.244.0.5,nw_dst=10.96.0.1")`,
					},
					"additional_params": {
						Type: "array",
						Items: &jsonschema.Schema{
							Type: "string",
						},
						Description: `Additional CLI arguments (only used when action is "dpctl/dump-conntrack"; e.g., ["zone=5"])`,
					},
					"pattern": {
						Type:        "string",
						Description: "Regex pattern to filter output lines",
					},
					"head": {
						Type:        "integer",
						Description: fmt.Sprintf(`Return only first N lines. Default: %d lines if tail is not specified`, ovsmcp.DefaultMaxLines),
					},
					"tail": {
						Type:        "integer",
						Description: "Return only last N lines",
					},
					"apply_tail_first": {
						Type:        "boolean",
						Description: "If both head and tail are set and apply_tail_first is true, apply tail before head. Default: false",
					},
				},
				Required: []string{"namespace", "name", "action"},
			},
			Annotations: api.ToolAnnotations{
				Title:         "OVN-Kubernetes: ovs-appctl",
				ReadOnlyHint:  ptr.To(true),
				OpenWorldHint: ptr.To(true),
			},
		}, Handler: ovsAppctl},
	}
}

// newOVSServer builds an OVS MCP server that execs into the ovn-controller
// container of the ovnkube-node pod, which has the OVS CLI tools and access
// to the OVS daemon socket.
func newOVSServer(params api.ToolHandlerParams) (*ovsmcp.MCPServer, error) {
	core := kubernetes.NewCore(params)
	return ovsmcp.NewMCPServer(func(ctx context.Context, namespace, name, _ string, command []string) (string, string, error) {
		return core.PodsExec(ctx, namespace, name, "ovn-controller", command)
	})
}

// optionalStringSlice extracts an optional []string tool argument. JSON-decoded
// arguments arrive as []any, so each element is asserted to string. A missing
// or non-array value returns (nil, nil). A non-string element returns an error
// to fail fast on malformed input rather than silently dropping it.
func optionalStringSlice(params api.ToolHandlerParams, key string) ([]string, error) {
	raw, ok := params.GetArguments()[key].([]any)
	if !ok {
		return nil, nil
	}
	out := make([]string, 0, len(raw))
	for i, v := range raw {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("%s[%d] must be a string", key, i)
		}
		out = append(out, s)
	}
	return out, nil
}

// ovsVsctl delegates to the exported Vsctl handler in the upstream
// ovn-kubernetes-mcp package. Downstream owns the tool declaration; upstream
// owns the command execution and result shape.
func ovsVsctl(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p := api.WrapParams(params)
	in := ovstypes.VsctlParams{
		Namespace: p.RequiredString("namespace"),
		Name:      p.RequiredString("name"),
		Action:    p.RequiredString("action"),
		Bridge:    p.OptionalString("bridge", ""),
		HeadTailParams: headtail.HeadTailParams{
			Head:           int(p.OptionalInt64("head", 0)),
			Tail:           int(p.OptionalInt64("tail", 0)),
			ApplyTailFirst: p.OptionalBool("apply_tail_first", false),
		},
	}
	if err := p.Err(); err != nil {
		return api.NewToolCallResultStructured(nil, err), nil
	}

	srv, err := newOVSServer(params)
	if err != nil {
		return api.NewToolCallResultStructured(nil, err), nil
	}

	_, result, err := srv.Vsctl(p.Context, nil, in)
	return api.NewToolCallResultStructured(result, err), nil
}

// ovsOfctl delegates to the exported Ofctl handler in the upstream
// ovn-kubernetes-mcp package.
func ovsOfctl(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p := api.WrapParams(params)
	in := ovstypes.OfctlParams{
		Namespace: p.RequiredString("namespace"),
		Name:      p.RequiredString("name"),
		Action:    p.RequiredString("action"),
		Bridge:    p.RequiredString("bridge"),
		PatternParams: pattern.PatternParams{
			Pattern: p.OptionalString("pattern", ""),
		},
		HeadTailParams: headtail.HeadTailParams{
			Head:           int(p.OptionalInt64("head", 0)),
			Tail:           int(p.OptionalInt64("tail", 0)),
			ApplyTailFirst: p.OptionalBool("apply_tail_first", false),
		},
	}
	if err := p.Err(); err != nil {
		return api.NewToolCallResultStructured(nil, err), nil
	}

	srv, err := newOVSServer(params)
	if err != nil {
		return api.NewToolCallResultStructured(nil, err), nil
	}

	_, result, err := srv.Ofctl(p.Context, nil, in)
	return api.NewToolCallResultStructured(result, err), nil
}

// ovsAppctl delegates to the exported Appctl handler in the upstream
// ovn-kubernetes-mcp package.
func ovsAppctl(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p := api.WrapParams(params)
	additionalParams, err := optionalStringSlice(params, "additional_params")
	if err != nil {
		return api.NewToolCallResultStructured(nil, err), nil
	}
	in := ovstypes.AppctlParams{
		Namespace:        p.RequiredString("namespace"),
		Name:             p.RequiredString("name"),
		Action:           p.RequiredString("action"),
		Bridge:           p.OptionalString("bridge", ""),
		Flow:             p.OptionalString("flow", ""),
		AdditionalParams: additionalParams,
		PatternParams: pattern.PatternParams{
			Pattern: p.OptionalString("pattern", ""),
		},
		HeadTailParams: headtail.HeadTailParams{
			Head:           int(p.OptionalInt64("head", 0)),
			Tail:           int(p.OptionalInt64("tail", 0)),
			ApplyTailFirst: p.OptionalBool("apply_tail_first", false),
		},
	}
	if err := p.Err(); err != nil {
		return api.NewToolCallResultStructured(nil, err), nil
	}

	srv, err := newOVSServer(params)
	if err != nil {
		return api.NewToolCallResultStructured(nil, err), nil
	}

	_, result, err := srv.Appctl(p.Context, nil, in)
	return api.NewToolCallResultStructured(result, err), nil
}
