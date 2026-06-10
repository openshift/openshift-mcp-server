package ovs

import (
	"encoding/json"
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/kubernetes/types"
	ovsmcp "github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/ovs/mcp"
	ovstypes "github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/ovs/types"
	"github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/utils/headtail"
	"github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/utils/pattern"
	"k8s.io/utils/ptr"
)

const defaultMaxLines = 100

// Tools returns all OVS layer tools for the ovn-kubernetes toolset.
func Tools() []api.ServerTool {
	return []api.ServerTool{
		{Tool: api.Tool{
			Name: "ovs_vsctl_show",
			Description: `Display a comprehensive overview of OVS configuration.

Runs 'ovs-vsctl show' command and returns detailed information about bridges, ports, interfaces,
controllers, and their configurations in a hierarchical format.

Parameters:
- namespace: Kubernetes namespace of the OVS pod
- name: Name of the pod running OVS
- head (optional): Return only first N lines. Default: 100 lines if tail is not specified
- tail (optional): Return only last N lines
- apply_tail_first (optional): If both head and tail are set and apply_tail_first is true, apply tail before head`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: `Kubernetes namespace of the OVS pod (e.g., "openshift-ovn-kubernetes")`,
					},
					"name": {
						Type:        "string",
						Description: `Name of the pod running OVS (e.g., "ovnkube-node-xxxxx")`,
					},
					"head": {
						Type:        "integer",
						Description: fmt.Sprintf(`Return only first N lines. Default: %d lines if tail is not specified`, defaultMaxLines),
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
				Required: []string{"namespace", "name"},
			},
			Annotations: api.ToolAnnotations{
				Title:         "OVN-Kubernetes: OVS Show",
				ReadOnlyHint:  ptr.To(true),
				OpenWorldHint: ptr.To(true),
			},
		}, Handler: ovsShow},

		{Tool: api.Tool{
			Name: "ovs_list_br",
			Description: `List all OVS bridges on a specific pod.

Runs 'ovs-vsctl list-br' command and returns the names of all configured bridges.

Parameters:
- namespace: Kubernetes namespace of the OVS pod
- name: Name of the pod running OVS`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: `Kubernetes namespace of the OVS pod (e.g., "openshift-ovn-kubernetes")`,
					},
					"name": {
						Type:        "string",
						Description: `Name of the pod running OVS (e.g., "ovnkube-node-xxxxx")`,
					},
				},
				Required: []string{"namespace", "name"},
			},
			Annotations: api.ToolAnnotations{
				Title:         "OVN-Kubernetes: OVS List Bridges",
				ReadOnlyHint:  ptr.To(true),
				OpenWorldHint: ptr.To(true),
			},
		}, Handler: ovsListBr},

		{Tool: api.Tool{
			Name: "ovs_list_ports",
			Description: `List all ports on a specific OVS bridge.

Runs 'ovs-vsctl list-ports' command and returns the names of all ports attached to the specified bridge.

Parameters:
- namespace: Kubernetes namespace of the OVS pod
- name: Name of the pod running OVS
- bridge: Name of the OVS bridge (e.g., "br-int")`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: `Kubernetes namespace of the OVS pod (e.g., "openshift-ovn-kubernetes")`,
					},
					"name": {
						Type:        "string",
						Description: `Name of the pod running OVS (e.g., "ovnkube-node-xxxxx")`,
					},
					"bridge": {
						Type:        "string",
						Description: `Name of the OVS bridge (e.g., "br-int")`,
					},
				},
				Required: []string{"namespace", "name", "bridge"},
			},
			Annotations: api.ToolAnnotations{
				Title:         "OVN-Kubernetes: OVS List Ports",
				ReadOnlyHint:  ptr.To(true),
				OpenWorldHint: ptr.To(true),
			},
		}, Handler: ovsListPorts},

		{Tool: api.Tool{
			Name: "ovs_list_ifaces",
			Description: `List all interfaces on a specific OVS bridge.

Runs 'ovs-vsctl list-ifaces' command and returns the names of all interfaces attached to the specified bridge.

Parameters:
- namespace: Kubernetes namespace of the OVS pod
- name: Name of the pod running OVS
- bridge: Name of the OVS bridge (e.g., "br-int")`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: `Kubernetes namespace of the OVS pod (e.g., "openshift-ovn-kubernetes")`,
					},
					"name": {
						Type:        "string",
						Description: `Name of the pod running OVS (e.g., "ovnkube-node-xxxxx")`,
					},
					"bridge": {
						Type:        "string",
						Description: `Name of the OVS bridge (e.g., "br-int")`,
					},
				},
				Required: []string{"namespace", "name", "bridge"},
			},
			Annotations: api.ToolAnnotations{
				Title:         "OVN-Kubernetes: OVS List Interfaces",
				ReadOnlyHint:  ptr.To(true),
				OpenWorldHint: ptr.To(true),
			},
		}, Handler: ovsListIfaces},

		{Tool: api.Tool{
			Name: "ovs_ofctl_dump_flows",
			Description: `Dump OpenFlow flows from a specific OVS bridge.

Runs 'ovs-ofctl dump-flows' command on the specified bridge and returns the flow entries.

Parameters:
- namespace: Kubernetes namespace of the OVS pod
- name: Name of the pod running OVS
- bridge: Name of the OVS bridge (e.g., "br-int")
- filter (optional): Regex pattern to filter flows
- head (optional): Return only first N lines
- tail (optional): Return only last N lines`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: `Kubernetes namespace of the OVS pod (e.g., "openshift-ovn-kubernetes")`,
					},
					"name": {
						Type:        "string",
						Description: `Name of the pod running OVS (e.g., "ovnkube-node-xxxxx")`,
					},
					"bridge": {
						Type:        "string",
						Description: `Name of the OVS bridge (e.g., "br-int")`,
					},
					"filter": {
						Type:        "string",
						Description: "Regex pattern to filter flows",
					},
					"head": {
						Type:        "integer",
						Description: fmt.Sprintf(`Return only first N lines. Default: %d lines if tail is not specified`, defaultMaxLines),
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
				Required: []string{"namespace", "name", "bridge"},
			},
			Annotations: api.ToolAnnotations{
				Title:         "OVN-Kubernetes: OVS Dump Flows",
				ReadOnlyHint:  ptr.To(true),
				OpenWorldHint: ptr.To(true),
			},
		}, Handler: ovsDumpFlows},

		{Tool: api.Tool{
			Name: "ovs_appctl_dump_conntrack",
			Description: `Dump connection tracking entries from OVS datapath.

Runs 'ovs-appctl dpctl/dump-conntrack' command and returns the conntrack entries.

Parameters:
- namespace: Kubernetes namespace of the OVS pod
- name: Name of the pod running OVS
- filter (optional): Regex pattern to filter conntrack entries
- additional_params (optional): Additional parameters (e.g., ["zone=5"])
- head (optional): Return only first N lines
- tail (optional): Return only last N lines`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: `Kubernetes namespace of the OVS pod (e.g., "openshift-ovn-kubernetes")`,
					},
					"name": {
						Type:        "string",
						Description: `Name of the pod running OVS (e.g., "ovnkube-node-xxxxx")`,
					},
					"filter": {
						Type:        "string",
						Description: "Regex pattern to filter conntrack entries",
					},
					"additional_params": {
						Type: "array",
						Items: &jsonschema.Schema{
							Type: "string",
						},
						Description: `Additional parameters to pass to dpctl/dump-conntrack (e.g., ["zone=5"])`,
					},
					"head": {
						Type:        "integer",
						Description: fmt.Sprintf(`Return only first N lines. Default: %d lines if tail is not specified`, defaultMaxLines),
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
				Required: []string{"namespace", "name"},
			},
			Annotations: api.ToolAnnotations{
				Title:         "OVN-Kubernetes: OVS Dump Conntrack",
				ReadOnlyHint:  ptr.To(true),
				OpenWorldHint: ptr.To(true),
			},
		}, Handler: ovsDumpConntrack},

		{Tool: api.Tool{
			Name: "ovs_appctl_ofproto_trace",
			Description: `Trace a packet through the OpenFlow pipeline.

Runs 'ovs-appctl ofproto/trace' command to simulate packet processing through OpenFlow tables.

Parameters:
- namespace: Kubernetes namespace of the OVS pod
- name: Name of the pod running OVS
- bridge: Name of the OVS bridge (e.g., "br-int")
- flow: Flow specification (e.g., "in_port=1,ip,nw_src=10.244.0.5,nw_dst=10.96.0.1")
- filter (optional): Regex pattern to filter trace output
- head (optional): Return only first N lines
- tail (optional): Return only last N lines`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: `Kubernetes namespace of the OVS pod (e.g., "openshift-ovn-kubernetes")`,
					},
					"name": {
						Type:        "string",
						Description: `Name of the pod running OVS (e.g., "ovnkube-node-xxxxx")`,
					},
					"bridge": {
						Type:        "string",
						Description: `Name of the OVS bridge (e.g., "br-int")`,
					},
					"flow": {
						Type:        "string",
						Description: `Flow specification describing the packet to trace (e.g., "in_port=1,ip,nw_src=10.244.0.5,nw_dst=10.96.0.1")`,
					},
					"filter": {
						Type:        "string",
						Description: "Regex pattern to filter trace output",
					},
					"head": {
						Type:        "integer",
						Description: fmt.Sprintf(`Return only first N lines. Default: %d lines if tail is not specified`, defaultMaxLines),
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
				Required: []string{"namespace", "name", "bridge", "flow"},
			},
			Annotations: api.ToolAnnotations{
				Title:         "OVN-Kubernetes: OVS Ofproto Trace",
				ReadOnlyHint:  ptr.To(true),
				OpenWorldHint: ptr.To(true),
			},
		}, Handler: ovsDumpOfprotoTrace},
	}
}

// newOVSServer builds an OVS MCP server backed by pod exec.
// NOTE: this execs in the pod's default container (PodsExec is passed the
// empty container by the upstream handlers). If OVS commands must run in a
// specific container, wrap PodsExec in a closure that injects the container
// name instead of passing the method value directly.
func newOVSServer(params api.ToolHandlerParams) (*ovsmcp.MCPServer, error) {
	return ovsmcp.NewMCPServer(kubernetes.NewCore(params).PodsExec)
}

// optionalStringSlice extracts an optional []string tool argument. JSON-decoded
// arguments arrive as []any, so each element is asserted to string; non-strings
// are skipped. A missing or non-array value returns nil.
func optionalStringSlice(params api.ToolHandlerParams, key string) []string {
	raw, ok := params.GetArguments()[key].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func ovsShow(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p := api.WrapParams(params)
	showParams := ovstypes.ShowParams{
		NamespacedNameParams: types.NamespacedNameParams{
			Namespace: p.RequiredString("namespace"),
			Name:      p.RequiredString("name"),
		},
		HeadTailParams: headtail.HeadTailParams{
			Head:           int(p.OptionalInt64("head", 0)),
			Tail:           int(p.OptionalInt64("tail", 0)),
			ApplyTailFirst: p.OptionalBool("apply_tail_first", false),
		},
	}
	if err := p.Err(); err != nil {
		return api.NewToolCallResult("", err), nil
	}
	srv, err := newOVSServer(params)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	_, result, err := srv.Show(p.Context, nil, showParams)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	out, err := json.Marshal(result)
	return api.NewToolCallResult(string(out), err), nil
}

func ovsListBr(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p := api.WrapParams(params)
	nameParams := types.NamespacedNameParams{
		Namespace: p.RequiredString("namespace"),
		Name:      p.RequiredString("name"),
	}
	if err := p.Err(); err != nil {
		return api.NewToolCallResult("", err), nil
	}
	srv, err := newOVSServer(params)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	_, result, err := srv.ListBridges(p.Context, nil, nameParams)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	out, err := json.Marshal(result)
	return api.NewToolCallResult(string(out), err), nil
}

func ovsListPorts(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p := api.WrapParams(params)
	cmdParams := ovstypes.GetOVSCommandParams{
		NamespacedNameParams: types.NamespacedNameParams{
			Namespace: p.RequiredString("namespace"),
			Name:      p.RequiredString("name"),
		},
		Bridge: p.RequiredString("bridge"),
	}
	if err := p.Err(); err != nil {
		return api.NewToolCallResult("", err), nil
	}
	srv, err := newOVSServer(params)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	_, result, err := srv.ListPorts(p.Context, nil, cmdParams)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	out, err := json.Marshal(result)
	return api.NewToolCallResult(string(out), err), nil
}

func ovsListIfaces(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p := api.WrapParams(params)
	cmdParams := ovstypes.GetOVSCommandParams{
		NamespacedNameParams: types.NamespacedNameParams{
			Namespace: p.RequiredString("namespace"),
			Name:      p.RequiredString("name"),
		},
		Bridge: p.RequiredString("bridge"),
	}
	if err := p.Err(); err != nil {
		return api.NewToolCallResult("", err), nil
	}
	srv, err := newOVSServer(params)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	_, result, err := srv.ListInterfaces(p.Context, nil, cmdParams)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	out, err := json.Marshal(result)
	return api.NewToolCallResult(string(out), err), nil
}

func ovsDumpFlows(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p := api.WrapParams(params)
	cmdParams := ovstypes.GetOVSCommandParams{
		NamespacedNameParams: types.NamespacedNameParams{
			Namespace: p.RequiredString("namespace"),
			Name:      p.RequiredString("name"),
		},
		Bridge:        p.RequiredString("bridge"),
		PatternParams: pattern.PatternParams{Pattern: p.OptionalString("filter", "")},
		HeadTailParams: headtail.HeadTailParams{
			Head:           int(p.OptionalInt64("head", 0)),
			Tail:           int(p.OptionalInt64("tail", 0)),
			ApplyTailFirst: p.OptionalBool("apply_tail_first", false),
		},
	}
	if err := p.Err(); err != nil {
		return api.NewToolCallResult("", err), nil
	}
	srv, err := newOVSServer(params)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	_, result, err := srv.DumpFlows(p.Context, nil, cmdParams)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	out, err := json.Marshal(result)
	return api.NewToolCallResult(string(out), err), nil
}

func ovsDumpConntrack(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p := api.WrapParams(params)
	conntrackParams := ovstypes.DumpConntrackParams{
		NamespacedNameParams: types.NamespacedNameParams{
			Namespace: p.RequiredString("namespace"),
			Name:      p.RequiredString("name"),
		},
		AdditionalParams: optionalStringSlice(params, "additional_params"),
		PatternParams:    pattern.PatternParams{Pattern: p.OptionalString("filter", "")},
		HeadTailParams: headtail.HeadTailParams{
			Head:           int(p.OptionalInt64("head", 0)),
			Tail:           int(p.OptionalInt64("tail", 0)),
			ApplyTailFirst: p.OptionalBool("apply_tail_first", false),
		},
	}
	if err := p.Err(); err != nil {
		return api.NewToolCallResult("", err), nil
	}
	srv, err := newOVSServer(params)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	_, result, err := srv.DumpConntrack(p.Context, nil, conntrackParams)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	out, err := json.Marshal(result)
	return api.NewToolCallResult(string(out), err), nil
}

func ovsDumpOfprotoTrace(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p := api.WrapParams(params)
	traceParams := ovstypes.OfprotoTraceParams{
		NamespacedNameParams: types.NamespacedNameParams{
			Namespace: p.RequiredString("namespace"),
			Name:      p.RequiredString("name"),
		},
		Bridge:        p.RequiredString("bridge"),
		Flow:          p.RequiredString("flow"),
		PatternParams: pattern.PatternParams{Pattern: p.OptionalString("filter", "")},
		HeadTailParams: headtail.HeadTailParams{
			Head:           int(p.OptionalInt64("head", 0)),
			Tail:           int(p.OptionalInt64("tail", 0)),
			ApplyTailFirst: p.OptionalBool("apply_tail_first", false),
		},
	}
	if err := p.Err(); err != nil {
		return api.NewToolCallResult("", err), nil
	}
	srv, err := newOVSServer(params)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	_, result, err := srv.DumpOfprotoTrace(p.Context, nil, traceParams)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	out, err := json.Marshal(result)
	return api.NewToolCallResult(string(out), err), nil
}
