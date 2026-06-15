package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	ovntypes "github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/ovn/types"
	"github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/utils"
	"github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/utils/ovndb"
)

type RunPodExecCommandFuncType func(ctx context.Context, namespace, name, container string, command []string) (string, string, error)

// MCPServer provides OVN layer analysis tools
type MCPServer struct {
	runPodExecCommand RunPodExecCommandFuncType
}

// NewMCPServer creates a new OVN MCP server
func NewMCPServer(runPodExecCommand RunPodExecCommandFuncType) (*MCPServer, error) {
	if runPodExecCommand == nil {
		return nil, fmt.Errorf("function to run pod exec command is nil")
	}
	return &MCPServer{
		runPodExecCommand: runPodExecCommand,
	}, nil
}

// AddTools registers OVN tools with the MCP server
func (s *MCPServer) AddTools(server *mcp.Server) {
	mcp.AddTool(server,
		&mcp.Tool{
			Name: "ovn-show",
			Description: fmt.Sprintf(`Display a comprehensive overview of OVN configuration from either the Northbound or Southbound database.

For Northbound (nbdb): Runs 'ovn-nbctl show' and displays logical switches, logical routers, 
their ports, and connections between them.

For Southbound (sbdb): Runs 'ovn-sbctl show' and displays chassis information, port bindings,
and their relationships.

Parameters:
- namespace: Kubernetes namespace of the OVN pod (e.g., "ovn-kubernetes")
- name: Name of the pod running OVN (e.g., "ovnkube-node-xxxxx")
- database: OVN database to query - "nbdb" for Northbound or "sbdb" for Southbound
- head (optional): Return only first N lines. Default: %d lines if tail is not specified
- tail (optional): Return only last N lines
- apply_tail_first (optional): If both head and tail are set and apply_tail_first is true,
apply tail before head. Default: false

Example output for nbdb:
{
  "database": "nbdb",
  "output": "switch 1234-5678 (node1)\n    port node1-k8s\n        addresses: [\"00:00:00:00:00:01\"]\n..."
}`, DefaultMaxLines),
		}, s.Show)

	mcp.AddTool(server,
		&mcp.Tool{
			Name: "ovn-get",
			Description: fmt.Sprintf(`Query records from an OVN database table with flexible filtering.

This is a versatile command that can:
1. List all records in a table (when no record specified)
2. Get a specific record (when record specified)

Common Northbound tables: Logical_Switch, Logical_Router, Logical_Switch_Port, 
Logical_Router_Port, ACL, Address_Set, Port_Group, Load_Balancer, NAT

Common Southbound tables: Chassis, Port_Binding, Datapath_Binding, Logical_Flow,
MAC_Binding, Multicast_Group, SB_Global

Parameters:
- namespace: Kubernetes namespace of the OVN pod
- name: Name of the pod running OVN
- database: OVN database to query - "nbdb" for Northbound or "sbdb" for Southbound
- table: Name of the table (e.g., "Logical_Switch", "Port_Binding")
- record (optional): Record identifier (UUID or name). If not specified, lists all records
- columns (optional): Comma-separated list of columns to display (e.g., "name,_uuid,ports")
- pattern (optional): Regex pattern to filter results. Only applies when listing all records.
- head (optional): Return only first N lines. Default: %d lines if tail is not specified
- tail (optional): Return only last N lines
- apply_tail_first (optional): If both head and tail are set and apply_tail_first is true,
apply tail before head. Default: false

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
}`, DefaultMaxLines),
		}, s.Get)

	mcp.AddTool(server,
		&mcp.Tool{
			Name: "ovn-lflow-list",
			Description: fmt.Sprintf(`List logical flows from the OVN Southbound database.

Runs 'ovn-sbctl lflow-list' to retrieve logical flows which represent the compiled 
logical network pipeline. This is essential for debugging packet forwarding.

Parameters:
- namespace: Kubernetes namespace of the OVN pod
- name: Name of the pod running OVN
- datapath (optional): Datapath name or UUID to filter flows for a specific logical switch/router
- pattern (optional): Regex pattern to filter flows
- head (optional): Return only first N lines. Default: %d lines if tail is not specified
- tail (optional): Return only last N lines
- apply_tail_first (optional): If both head and tail are set and apply_tail_first is true,
apply tail before head. Default: false

Example output:
{
  "datapath": "node1",
  "flows": [
    "table=0 (ls_in_port_sec_l2), priority=100, match=(inport == \"pod1\"), action=(next;)",
    "table=1 (ls_in_port_sec_ip), priority=90, match=(ip4), action=(next;)"
  ]
}`, DefaultMaxLines),
		}, s.ListLogicalFlows)

	mcp.AddTool(server,
		&mcp.Tool{
			Name: "ovn-trace",
			Description: fmt.Sprintf(`Trace a packet through the OVN logical network.

Runs 'ovn-trace' to simulate packet processing through the logical network pipeline.
This shows which logical flows match, what actions are taken, and the final disposition.

The trace is essential for debugging connectivity issues and understanding how traffic
flows through the OVN logical network.

Parameters:
- namespace: Kubernetes namespace of the OVN pod
- name: Name of the pod running OVN
- datapath: Name of the logical switch or router to start the trace
- microflow: Microflow specification describing the packet (e.g., "inport==\"pod1\" && eth.src==00:00:00:00:00:01 && ip4.src==10.244.0.5 && ip4.dst==10.244.1.5")
- mode (optional): Output verbosity mode - "detailed" (default), "summary", or "minimal"
- pattern (optional): Regex pattern to filter trace output
- head (optional): Return only first N lines. Default: %d lines if tail is not specified
- tail (optional): Return only last N lines
- apply_tail_first (optional): If both head and tail are set and apply_tail_first is true,
apply tail before head. Default: false

Microflow specification examples:
- inport=="pod1" && eth.src==00:00:00:00:00:01 && ip4.src==10.244.0.5 && ip4.dst==10.244.1.5
- inport=="pod1" && eth.src==00:00:00:00:00:01 && icmp && ip4.src==10.244.0.5 && ip4.dst==8.8.8.8

Example output:
{
  "datapath": "node1",
  "microflow": "inport==\"pod1\" && ...",
  "output": "ingress(dp=\"node1\", inport=\"pod1\")\n  0. ls_in_port_sec_l2: inport == \"pod1\", priority 50, uuid 1234\n     next;\n..."
}`, DefaultMaxLines),
		}, s.Trace)
}

// Show displays a comprehensive overview of OVN configuration.
func (s *MCPServer) Show(ctx context.Context, req *mcp.CallToolRequest,
	in ovntypes.ShowParams) (*mcp.CallToolResult, ovntypes.ShowResult, error) {
	result := ovntypes.ShowResult{
		Database: in.Database,
	}

	// Validate database
	if err := validateDatabase(in.Database); err != nil {
		return nil, result, err
	}

	// Build command
	cmd := getDBCommand(in.Database)
	stdout, stderr, err := s.runPodExecCommand(ctx, in.Namespace, in.Name, "", []string{cmd, "show"})
	if err != nil {
		return nil, result, fmt.Errorf("failed to retrieve OVN configuration from pod %s/%s: %w",
			in.Namespace, in.Name, err)
	}
	if stderr != "" {
		return nil, result, fmt.Errorf("failed to retrieve OVN configuration from pod %s/%s: %s",
			in.Namespace, in.Name, stderr)
	}
	lines := utils.StripEmptyLines(strings.Split(stdout, "\n"))

	// Apply the head and tail parameters to the lines
	lines = in.HeadTailParams.Apply(lines, DefaultMaxLines)

	// Join all lines into a single output string
	result.Output = strings.Join(lines, "\n")
	return nil, result, nil
}

// Get queries records from an OVN table with flexible filtering.
// Supports two modes:
// 1. List all records (when Record is empty)
// 2. Get specific record (when Record is set)
// Both modes support filtering columns with the Columns parameter.
func (s *MCPServer) Get(ctx context.Context, req *mcp.CallToolRequest,
	in ovntypes.GetParams) (*mcp.CallToolResult, ovntypes.GetResult, error) {
	result := ovntypes.GetResult{
		Database: in.Database,
		Table:    in.Table,
		Record:   in.Record,
	}

	// Validate inputs
	if err := validateDatabase(in.Database); err != nil {
		return nil, result, err
	}
	if err := ovndb.ValidateOVNTableName(in.Table); err != nil {
		return nil, result, err
	}
	if err := validateColumnSpec(in.Columns); err != nil {
		return nil, result, err
	}

	cmd := getDBCommand(in.Database)
	cmdArgs := []string{cmd}

	// Add columns filter if specified
	if in.Columns != "" {
		cmdArgs = append(cmdArgs, "--columns="+in.Columns)
	}

	if in.Record == "" {
		// Mode 1: List all records in the table
		cmdArgs = append(cmdArgs, "list", in.Table)
	} else {
		// Mode 2: Get specific record
		if err := validateRecordName(in.Record); err != nil {
			return nil, result, err
		}
		cmdArgs = append(cmdArgs, "list", in.Table, in.Record)
	}

	// Match the pattern to the get results if in list mode
	lines, err := in.PatternParams.ExecuteWithMatch(func() ([]string, error) {
		stdout, stderr, err := s.runPodExecCommand(ctx, in.Namespace, in.Name, "", cmdArgs)
		if err != nil {
			if in.Record != "" {
				return nil, fmt.Errorf("failed to get record %s from table %s on pod %s/%s: %w",
					in.Record, in.Table, in.Namespace, in.Name, err)
			}
			return nil, fmt.Errorf("failed to list table %s from pod %s/%s: %w",
				in.Table, in.Namespace, in.Name, err)
		}
		if stderr != "" {
			if in.Record != "" {
				return nil, fmt.Errorf("failed to get record %s from table %s on pod %s/%s: %s",
					in.Record, in.Table, in.Namespace, in.Name, stderr)
			}
			return nil, fmt.Errorf("failed to list table %s from pod %s/%s: %s",
				in.Table, in.Namespace, in.Name, stderr)
		}
		lines := utils.StripEmptyLines(strings.Split(stdout, "\n"))
		return lines, nil
	}, in.Record == "")
	if err != nil {
		return nil, result, err
	}

	// Apply the head and tail parameters to the lines
	lines = in.HeadTailParams.Apply(lines, DefaultMaxLines)

	result.Output = strings.Join(lines, "\n")
	return nil, result, nil
}

// ListLogicalFlows lists logical flows from the Southbound database.
func (s *MCPServer) ListLogicalFlows(ctx context.Context, req *mcp.CallToolRequest,
	in ovntypes.LogicalFlowListParams) (*mcp.CallToolResult, ovntypes.LogicalFlowListResult, error) {
	result := ovntypes.LogicalFlowListResult{
		Datapath: in.Datapath,
		Flows:    []string{},
	}

	// Validate datapath if provided
	if in.Datapath != "" {
		if err := validateDatapath(in.Datapath); err != nil {
			return nil, result, err
		}
	}

	// Build command
	cmdArgs := []string{"ovn-sbctl", "lflow-list"}
	if in.Datapath != "" {
		cmdArgs = append(cmdArgs, in.Datapath)
	}

	// Match the pattern to the logical flows
	lines, err := in.PatternParams.ExecuteWithMatch(func() ([]string, error) {
		stdout, stderr, err := s.runPodExecCommand(ctx, in.Namespace, in.Name, "", cmdArgs)
		if err != nil {
			return nil, fmt.Errorf("failed to list logical flows from pod %s/%s: %w",
				in.Namespace, in.Name, err)
		}
		if stderr != "" {
			return nil, fmt.Errorf("failed to list logical flows from pod %s/%s: %s",
				in.Namespace, in.Name, stderr)
		}
		lines := utils.StripEmptyLines(strings.Split(stdout, "\n"))
		return lines, nil
	}, true)
	if err != nil {
		return nil, result, err
	}

	// Apply the head and tail parameters to the lines
	lines = in.HeadTailParams.Apply(lines, DefaultMaxLines)

	result.Flows = lines
	return nil, result, nil
}

// Trace traces a packet through the OVN logical network.
func (s *MCPServer) Trace(ctx context.Context, req *mcp.CallToolRequest,
	in ovntypes.OVNTraceParams) (*mcp.CallToolResult, ovntypes.OVNTraceResult, error) {
	result := ovntypes.OVNTraceResult{
		Datapath:  in.Datapath,
		Microflow: in.Microflow,
	}

	// Validate inputs
	if err := validateDatapath(in.Datapath); err != nil {
		return nil, result, err
	}
	if err := validateMicroflow(in.Microflow); err != nil {
		return nil, result, err
	}

	// Build command: ovn-trace <datapath> '<microflow>'
	cmdArgs := []string{"ovn-trace"}

	// Add output format flag based on mode (default to detailed)
	switch in.Mode {
	case ovntypes.TraceModeSummary:
		cmdArgs = append(cmdArgs, "--summary")
	case ovntypes.TraceModeMinimal:
		cmdArgs = append(cmdArgs, "--minimal")
	case ovntypes.TraceModeDetailed, "":
		cmdArgs = append(cmdArgs, "--detailed")
	}

	cmdArgs = append(cmdArgs, in.Datapath, in.Microflow)

	// Match the pattern to the trace output
	lines, err := in.PatternParams.ExecuteWithMatch(func() ([]string, error) {
		stdout, stderr, err := s.runPodExecCommand(ctx, in.Namespace, in.Name, "", cmdArgs)
		if err != nil {
			return nil, fmt.Errorf("failed to trace packet on pod %s/%s: %w",
				in.Namespace, in.Name, err)
		}
		if stderr != "" {
			return nil, fmt.Errorf("failed to trace packet on pod %s/%s: %s",
				in.Namespace, in.Name, stderr)
		}
		lines := utils.StripEmptyLines(strings.Split(stdout, "\n"))
		return lines, nil
	}, true)
	if err != nil {
		return nil, result, err
	}

	// Apply the head and tail parameters to the lines
	lines = in.HeadTailParams.Apply(lines, DefaultMaxLines)

	result.Output = strings.Join(lines, "\n")
	return nil, result, nil
}
