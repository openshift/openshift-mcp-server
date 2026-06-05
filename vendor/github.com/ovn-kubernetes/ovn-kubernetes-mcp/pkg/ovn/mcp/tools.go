package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	gosdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"k8s.io/utils/ptr"

	ovntypes "github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/ovn/types"
)

// ExecTarget represents the logical execution target for an OVN tool.
// Upstream defines what role it needs; downstream maps to concrete containers.
type ExecTarget string

const (
	TargetNBDB   ExecTarget = "nbdb"
	TargetSBDB   ExecTarget = "sbdb"
	TargetNorthd ExecTarget = "northd"
)

// ExecuteFunc is a unified execution function for OVN tools.
type ExecuteFunc func(s *MCPServer, ctx context.Context, args map[string]any) (any, error)

// ToolRegistration pairs a tool definition with its execution logic and
// routing metadata.
type ToolRegistration struct {
	Tool           *gosdk.Tool
	Execute        ExecuteFunc
	TargetSelector func(args map[string]any) ExecTarget
}

// MakeExecute creates an ExecuteFunc from a typed MCPServer method.
// Passes a synthetic *CallToolRequest (with populated Arguments) to avoid
// nil-dereference risk if upstream ever uses req.
func MakeExecute[P any, R any](
	method func(*MCPServer, context.Context, *gosdk.CallToolRequest, P) (*gosdk.CallToolResult, R, error),
) ExecuteFunc {
	return func(s *MCPServer, ctx context.Context, args map[string]any) (any, error) {
		var p P
		if err := unmarshalArgs(args, &p); err != nil {
			return nil, err
		}
		data, err := json.Marshal(args)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal arguments for request: %w", err)
		}
		req := &gosdk.CallToolRequest{Params: &gosdk.CallToolParamsRaw{Arguments: data}}
		_, result, err := method(s, ctx, req, p)
		return result, err
	}
}

// Exported tool definitions, shared between AddTools and AllToolRegistrations.
var (
	ShowTool      = showTool()
	GetTool       = getTool()
	LFlowListTool = lflowListTool()
	TraceTool     = traceTool()
)

// AllToolRegistrations returns all OVN tool registrations.
func AllToolRegistrations() []ToolRegistration {
	return []ToolRegistration{
		{Tool: ShowTool, Execute: MakeExecute((*MCPServer).Show), TargetSelector: dbTarget},
		{Tool: GetTool, Execute: MakeExecute((*MCPServer).Get), TargetSelector: dbTarget},
		{Tool: LFlowListTool, Execute: MakeExecute((*MCPServer).ListLogicalFlows), TargetSelector: staticTarget(TargetSBDB)},
		{Tool: TraceTool, Execute: MakeExecute((*MCPServer).Trace), TargetSelector: staticTarget(TargetNorthd)},
	}
}

func dbTarget(args map[string]any) ExecTarget {
	if db, _ := args["database"].(string); db == string(ovntypes.SouthboundDB) {
		return TargetSBDB
	}
	return TargetNBDB
}

func staticTarget(t ExecTarget) func(map[string]any) ExecTarget {
	return func(_ map[string]any) ExecTarget { return t }
}

func unmarshalArgs(args map[string]any, dest any) error {
	data, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("failed to marshal arguments: %w", err)
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("failed to unmarshal arguments: %w", err)
	}
	return nil
}

var headTailSchema = map[string]*jsonschema.Schema{
	"head": {
		Type:        "integer",
		Description: "Return only first N lines (default: 100 if tail is not specified)",
		Minimum:     ptr.To(float64(1)),
	},
	"tail": {
		Type:        "integer",
		Description: "Return only last N lines",
		Minimum:     ptr.To(float64(1)),
	},
	"apply_tail_first": {
		Type:        "boolean",
		Description: "If both head and tail are set and apply_tail_first is true, apply tail before head (default: false)",
	},
}

func mergeProps(base map[string]*jsonschema.Schema, extra map[string]*jsonschema.Schema) map[string]*jsonschema.Schema {
	for k, v := range extra {
		base[k] = v
	}
	return base
}

func showTool() *gosdk.Tool {
	return &gosdk.Tool{
		Name: "ovn_show",
		Description: `Display a comprehensive overview of OVN configuration from either the Northbound or Southbound database.

For Northbound (nbdb): Runs 'ovn-nbctl show' and displays logical switches, logical routers, their ports, and connections between them.
For Southbound (sbdb): Runs 'ovn-sbctl show' and displays chassis information, port bindings, and their relationships.`,
		Title: "OVN: Show",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: mergeProps(map[string]*jsonschema.Schema{
				"namespace": {
					Type:        "string",
					Description: "Kubernetes namespace of the OVN pod (e.g., \"openshift-ovn-kubernetes\")",
				},
				"name": {
					Type:        "string",
					Description: "Name of the pod running OVN (e.g., \"ovnkube-node-xxxxx\")",
				},
				"database": {
					Type:        "string",
					Description: `OVN database to query - "nbdb" for Northbound or "sbdb" for Southbound`,
					Enum:        []any{string(ovntypes.NorthboundDB), string(ovntypes.SouthboundDB)},
				},
			}, headTailSchema),
			Required: []string{"namespace", "name", "database"},
		},
		Annotations: &gosdk.ToolAnnotations{
			ReadOnlyHint:   true,
			IdempotentHint: true,
			OpenWorldHint:  ptr.To(true),
		},
	}
}

func getTool() *gosdk.Tool {
	return &gosdk.Tool{
		Name: "ovn_get",
		Description: `Query records from an OVN database table with flexible filtering.

Can list all records in a table (when no record specified) or get a specific record (when record specified).

Common Northbound tables: Logical_Switch, Logical_Router, Logical_Switch_Port, Logical_Router_Port, ACL, Address_Set, Port_Group, Load_Balancer, NAT.
Common Southbound tables: Chassis, Port_Binding, Datapath_Binding, Logical_Flow, MAC_Binding, Multicast_Group, SB_Global.`,
		Title: "OVN: Get",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: mergeProps(map[string]*jsonschema.Schema{
				"namespace": {
					Type:        "string",
					Description: "Kubernetes namespace of the OVN pod",
				},
				"name": {
					Type:        "string",
					Description: "Name of the pod running OVN",
				},
				"database": {
					Type:        "string",
					Description: `OVN database to query - "nbdb" for Northbound or "sbdb" for Southbound`,
					Enum:        []any{string(ovntypes.NorthboundDB), string(ovntypes.SouthboundDB)},
				},
				"table": {
					Type:        "string",
					Description: "Name of the OVN table (e.g., \"Logical_Switch\", \"Port_Binding\")",
				},
				"record": {
					Type:        "string",
					Description: "Record identifier (UUID or name). If not specified, lists all records",
				},
				"columns": {
					Type:        "string",
					Description: "Comma-separated list of columns to display (e.g., \"name,_uuid,ports\")",
				},
				"pattern": {
					Type:        "string",
					Description: "Regex pattern to filter results (only applies when listing all records)",
				},
			}, headTailSchema),
			Required: []string{"namespace", "name", "database", "table"},
		},
		Annotations: &gosdk.ToolAnnotations{
			ReadOnlyHint:   true,
			IdempotentHint: true,
			OpenWorldHint:  ptr.To(true),
		},
	}
}

func lflowListTool() *gosdk.Tool {
	return &gosdk.Tool{
		Name: "ovn_lflow_list",
		Description: `List logical flows from the OVN Southbound database.

Runs 'ovn-sbctl lflow-list' to retrieve logical flows which represent the compiled logical network pipeline. Essential for debugging packet forwarding.`,
		Title: "OVN: Logical Flow List",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: mergeProps(map[string]*jsonschema.Schema{
				"namespace": {
					Type:        "string",
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
			}, headTailSchema),
			Required: []string{"namespace", "name"},
		},
		Annotations: &gosdk.ToolAnnotations{
			ReadOnlyHint:   true,
			IdempotentHint: true,
			OpenWorldHint:  ptr.To(true),
		},
	}
}

func traceTool() *gosdk.Tool {
	return &gosdk.Tool{
		Name: "ovn_trace",
		Description: `Trace a packet through the OVN logical network.

Runs 'ovn-trace' to simulate packet processing through the logical network pipeline. Shows which logical flows match, what actions are taken, and the final disposition. Essential for debugging connectivity issues.

Microflow examples:
- inport=="pod1" && eth.src==00:00:00:00:00:01 && ip4.src==10.244.0.5 && ip4.dst==10.244.1.5
- inport=="pod1" && eth.src==00:00:00:00:00:01 && icmp && ip4.src==10.244.0.5 && ip4.dst==8.8.8.8`,
		Title: "OVN: Trace",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: mergeProps(map[string]*jsonschema.Schema{
				"namespace": {
					Type:        "string",
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
					Description: "Microflow specification describing the packet to trace",
				},
				"mode": {
					Type:        "string",
					Description: "Output verbosity mode (default: \"detailed\")",
					Enum:        []any{"detailed", "summary", "minimal"},
				},
				"pattern": {
					Type:        "string",
					Description: "Regex pattern to filter trace output",
				},
			}, headTailSchema),
			Required: []string{"namespace", "name", "datapath", "microflow"},
		},
		Annotations: &gosdk.ToolAnnotations{
			ReadOnlyHint:   true,
			IdempotentHint: true,
			OpenWorldHint:  ptr.To(true),
		},
	}
}
