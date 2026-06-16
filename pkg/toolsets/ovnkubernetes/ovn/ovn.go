package ovn

import (
	"context"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	k8stypes "github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/kubernetes/types"
	ovnmcp "github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/ovn/mcp"
	ovntypes "github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/ovn/types"
	"github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/utils/headtail"
	"github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/utils/pattern"
	"k8s.io/utils/ptr"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
)

func InitOVNTools() []api.ServerTool {
	return []api.ServerTool{
		{Tool: api.Tool{
			Name: "ovn_show",
			Description: `Display a comprehensive overview of OVN configuration from either the Northbound or Southbound database.

For Northbound (nbdb): Runs 'ovn-nbctl show' and displays logical switches, logical routers,
their ports, and connections between them.

For Southbound (sbdb): Runs 'ovn-sbctl show' and displays chassis information, port bindings,
and their relationships. Returns 100 lines by default; use head/tail to adjust.

Example output for nbdb:
{
  "database": "nbdb",
  "output": "switch 1234-5678 (node1)\n    port node1-k8s\n        addresses: [\"00:00:00:00:00:01\"]\n..."
}`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Default:     api.ToRawMessage("openshift-ovn-kubernetes"),
						Description: `Kubernetes namespace of the OVN pod (e.g., "openshift-ovn-kubernetes")`,
					},
					"name": {
						Type:        "string",
						Description: `Name of the pod running OVN (e.g., "ovnkube-node-xxxxx")`,
					},
					"database": {
						Type:        "string",
						Enum:        []any{"nbdb", "sbdb"},
						Description: `OVN database to query - "nbdb" for Northbound or "sbdb" for Southbound`,
					},
					"head": {
						Type:        "integer",
						Description: fmt.Sprintf("Return only first N lines. Default: %d lines if tail is not specified", ovnmcp.DefaultMaxLines),
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
				Required: []string{"name", "database"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OVN: Show",
				ReadOnlyHint:    ptr.To(true),
				DestructiveHint: ptr.To(false),
				IdempotentHint:  ptr.To(true),
				OpenWorldHint:   ptr.To(true),
			},
		}, Handler: ovnShow},
		{Tool: api.Tool{
			Name: "ovn_get",
			Description: `Query records from an OVN database table with flexible filtering.

This is a versatile command that can:
1. List all records in a table (when no record specified)
2. Get a specific record (when record specified)

Common Northbound tables: Logical_Switch, Logical_Router, Logical_Switch_Port, 
Logical_Router_Port, ACL, Address_Set, Port_Group, Load_Balancer, NAT

Common Southbound tables: Chassis, Port_Binding, Datapath_Binding, Logical_Flow,
MAC_Binding, Multicast_Group, SB_Global

Returns 100 lines by default; use head/tail to adjust.

Example listing all records:
{
  "database": "nbdb",
  "table": "Port_Group",
  "output": "_uuid: 1234-5678\nname: \"pg_default\"\nports: [...]\n\n_uuid: abcd-efgh\n..."
}

Example getting a specific record:
{
  "database": "nbdb",
  "table": "Logical_Router",
  "record": "ovn_cluster_router",
  "output": "_uuid: 4c4a0a35-348c-41cc-8417-53a618e0c383\nname: ovn_cluster_router\nports: [...]"
}

Example getting specific columns:
{
  "database": "nbdb",
  "table": "Logical_Switch",
  "columns": "name,ports",
  "output": "name: ovn-worker\nports: [uuid1, uuid2]\n\nname: join\nports: [uuid3]"
}`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Default:     api.ToRawMessage("openshift-ovn-kubernetes"),
						Description: "Kubernetes namespace of the OVN pod",
					},
					"name": {
						Type:        "string",
						Description: "Name of the pod running OVN",
					},
					"database": {
						Type:        "string",
						Enum:        []any{"nbdb", "sbdb"},
						Description: `OVN database to query - "nbdb" for Northbound or "sbdb" for Southbound`,
					},
					"table": {
						Type:        "string",
						Description: `Name of the table (e.g., "Logical_Switch", "Port_Binding")`,
					},
					"record": {
						Type:        "string",
						Description: `Record identifier (UUID or name). If not specified, lists all records`,
					},
					"columns": {
						Type:        "string",
						Description: `Comma-separated list of columns to display (e.g., "name,_uuid,ports")`,
					},
					"pattern": {
						Type:        "string",
						Description: `Regex pattern to filter results. Only applies when listing all records.`,
					},
					"head": {
						Type:        "integer",
						Description: fmt.Sprintf("Return only first N lines. Default: %d lines if tail is not specified", ovnmcp.DefaultMaxLines),
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
				Required: []string{"name", "database", "table"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OVN: Get",
				ReadOnlyHint:    ptr.To(true),
				DestructiveHint: ptr.To(false),
				IdempotentHint:  ptr.To(true),
				OpenWorldHint:   ptr.To(true),
			},
		}, Handler: ovnGet},
		{Tool: api.Tool{
			Name: "ovn_lflow_list",
			Description: `List logical flows from the OVN Southbound database.

Runs 'ovn-sbctl lflow-list' to retrieve logical flows which represent the compiled
logical network pipeline. This is essential for debugging packet forwarding.
Returns 100 lines by default; use head/tail to adjust.

Example output:
{
  "datapath": "node1",
  "flows": [
    "table=0 (ls_in_port_sec_l2), priority=100, match=(inport == \"pod1\"), action=(next;)",
    "table=1 (ls_in_port_sec_ip), priority=90, match=(ip4), action=(next;)"
  ]
}`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Default:     api.ToRawMessage("openshift-ovn-kubernetes"),
						Description: "Kubernetes namespace of the OVN pod",
					},
					"name": {
						Type:        "string",
						Description: "Name of the pod running OVN",
					},
					"datapath": {
						Type:        "string",
						Description: "Datapath name or UUID to filter flows for a specific logical switch/router",
					},
					"pattern": {
						Type:        "string",
						Description: "Regex pattern to filter flows",
					},
					"head": {
						Type:        "integer",
						Description: fmt.Sprintf("Return only first N lines. Default: %d lines if tail is not specified", ovnmcp.DefaultMaxLines),
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
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OVN: Logical Flow List",
				ReadOnlyHint:    ptr.To(true),
				DestructiveHint: ptr.To(false),
				IdempotentHint:  ptr.To(true),
				OpenWorldHint:   ptr.To(true),
			},
		}, Handler: ovnLFlowList},
		{Tool: api.Tool{
			Name: "ovn_trace",
			Description: `Trace a packet through the OVN logical network.

Runs 'ovn-trace' to simulate packet processing through the logical network pipeline.
This shows which logical flows match, what actions are taken, and the final disposition.

The trace is essential for debugging connectivity issues and understanding how traffic
flows through the OVN logical network. Returns 100 lines by default; use head/tail to adjust.

Microflow specification examples:
- inport=="pod1" && eth.src==00:00:00:00:00:01 && ip4.src==10.244.0.5 && ip4.dst==10.244.1.5
- inport=="pod1" && eth.src==00:00:00:00:00:01 && icmp && ip4.src==10.244.0.5 && ip4.dst==8.8.8.8

Example output:
{
  "datapath": "node1",
  "microflow": "inport==\"pod1\" && ...",
  "output": "ingress(dp=\"node1\", inport=\"pod1\")\n  0. ls_in_port_sec_l2: inport == \"pod1\", priority 50, uuid 1234\n     next;\n..."
}`,
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Default:     api.ToRawMessage("openshift-ovn-kubernetes"),
						Description: "Kubernetes namespace of the OVN pod",
					},
					"name": {
						Type:        "string",
						Description: "Name of the pod running OVN",
					},
					"datapath": {
						Type:        "string",
						Description: "Name of the logical switch or router to start the trace",
					},
					"microflow": {
						Type:        "string",
						Description: `Microflow specification describing the packet (e.g., "inport==\"pod1\" && eth.src==00:00:00:00:00:01 && ip4.src==10.244.0.5 && ip4.dst==10.244.1.5")`,
					},
					"mode": {
						Type:        "string",
						Enum:        []any{"detailed", "summary", "minimal"},
						Default:     api.ToRawMessage("detailed"),
						Description: `Output verbosity mode - "detailed" (default), "summary", or "minimal"`,
					},
					"pattern": {
						Type:        "string",
						Description: "Regex pattern to filter trace output",
					},
					"head": {
						Type:        "integer",
						Description: fmt.Sprintf("Return only first N lines. Default: %d lines if tail is not specified", ovnmcp.DefaultMaxLines),
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
				Required: []string{"name", "datapath", "microflow"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OVN: Trace",
				ReadOnlyHint:    ptr.To(true),
				DestructiveHint: ptr.To(false),
				IdempotentHint:  ptr.To(true),
				OpenWorldHint:   ptr.To(true),
			},
		}, Handler: ovnTrace},
	}
}

func ovnShow(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p := api.WrapParams(params)
	namespace := p.OptionalString("namespace", "openshift-ovn-kubernetes")
	name := p.RequiredString("name")
	database := p.RequiredString("database")
	head := int(p.OptionalInt64("head", 0))
	tail := int(p.OptionalInt64("tail", 0))
	applyTailFirst := p.OptionalBool("apply_tail_first", false)
	if err := p.Err(); err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to run ovn show: %w", err)), nil
	}

	server, err := newOVNServer(kubernetes.NewCore(params), containerForDatabase(database))
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	_, result, err := server.Show(params.Context, nil, ovntypes.ShowParams{
		NamespacedNameParams: namespacedName(namespace, name),
		Database:             ovntypes.Database(database),
		HeadTailParams:       headtail.HeadTailParams{Head: head, Tail: tail, ApplyTailFirst: applyTailFirst},
	})
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	return api.NewToolCallResultStructured(result, nil), nil
}

func ovnGet(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p := api.WrapParams(params)
	namespace := p.OptionalString("namespace", "openshift-ovn-kubernetes")
	name := p.RequiredString("name")
	database := p.RequiredString("database")
	table := p.RequiredString("table")
	record := p.OptionalString("record", "")
	columns := p.OptionalString("columns", "")
	pat := p.OptionalString("pattern", "")
	head := int(p.OptionalInt64("head", 0))
	tail := int(p.OptionalInt64("tail", 0))
	applyTailFirst := p.OptionalBool("apply_tail_first", false)
	if err := p.Err(); err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to run ovn get: %w", err)), nil
	}

	server, err := newOVNServer(kubernetes.NewCore(params), containerForDatabase(database))
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	_, result, err := server.Get(params.Context, nil, ovntypes.GetParams{
		NamespacedNameParams: namespacedName(namespace, name),
		Database:             ovntypes.Database(database),
		Table:                table,
		Record:               record,
		Columns:              columns,
		PatternParams:        pattern.PatternParams{Pattern: pat},
		HeadTailParams:       headtail.HeadTailParams{Head: head, Tail: tail, ApplyTailFirst: applyTailFirst},
	})
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	return api.NewToolCallResultStructured(result, nil), nil
}

func ovnLFlowList(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p := api.WrapParams(params)
	namespace := p.OptionalString("namespace", "openshift-ovn-kubernetes")
	name := p.RequiredString("name")
	datapath := p.OptionalString("datapath", "")
	pat := p.OptionalString("pattern", "")
	head := int(p.OptionalInt64("head", 0))
	tail := int(p.OptionalInt64("tail", 0))
	applyTailFirst := p.OptionalBool("apply_tail_first", false)
	if err := p.Err(); err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to run ovn lflow-list: %w", err)), nil
	}

	server, err := newOVNServer(kubernetes.NewCore(params), "sbdb")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	_, result, err := server.ListLogicalFlows(params.Context, nil, ovntypes.LogicalFlowListParams{
		NamespacedNameParams: namespacedName(namespace, name),
		Datapath:             datapath,
		PatternParams:        pattern.PatternParams{Pattern: pat},
		HeadTailParams:       headtail.HeadTailParams{Head: head, Tail: tail, ApplyTailFirst: applyTailFirst},
	})
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	return api.NewToolCallResultStructured(result, nil), nil
}

func ovnTrace(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p := api.WrapParams(params)
	namespace := p.OptionalString("namespace", "openshift-ovn-kubernetes")
	name := p.RequiredString("name")
	datapath := p.RequiredString("datapath")
	microflow := p.RequiredString("microflow")
	mode := p.OptionalString("mode", "")
	pat := p.OptionalString("pattern", "")
	head := int(p.OptionalInt64("head", 0))
	tail := int(p.OptionalInt64("tail", 0))
	applyTailFirst := p.OptionalBool("apply_tail_first", false)
	if err := p.Err(); err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to run ovn trace: %w", err)), nil
	}

	server, err := newOVNServer(kubernetes.NewCore(params), "northd")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	_, result, err := server.Trace(params.Context, nil, ovntypes.OVNTraceParams{
		NamespacedNameParams: namespacedName(namespace, name),
		Datapath:             datapath,
		Microflow:            microflow,
		Mode:                 ovntypes.TraceMode(mode),
		PatternParams:        pattern.PatternParams{Pattern: pat},
		HeadTailParams:       headtail.HeadTailParams{Head: head, Tail: tail, ApplyTailFirst: applyTailFirst},
	})
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	return api.NewToolCallResultStructured(result, nil), nil
}

// newOVNServer creates an ovnmcp.MCPServer that executes OVN CLI commands
// inside the given container of an OVN pod.
func newOVNServer(core *kubernetes.Core, container string) (*ovnmcp.MCPServer, error) {
	return ovnmcp.NewMCPServer(func(ctx context.Context, namespace, name, _ string, command []string) (string, string, error) {
		return core.PodsExec(ctx, namespace, name, container, command)
	})
}

// containerForDatabase maps the database parameter to the container that has
// the corresponding OVN CLI tool. On OpenShift, nbdb and sbdb are separate
// containers in the ovnkube-node pod.
func containerForDatabase(database string) string {
	if database == string(ovntypes.SouthboundDB) {
		return "sbdb"
	}
	return "nbdb"
}

func namespacedName(namespace, name string) k8stypes.NamespacedNameParams {
	return k8stypes.NamespacedNameParams{Namespace: namespace, Name: name}
}
