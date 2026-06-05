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

// AddTools registers OVN tools with the MCP server.
func (s *MCPServer) AddTools(server *mcp.Server) {
	mcp.AddTool(server, ShowTool, s.Show)
	mcp.AddTool(server, GetTool, s.Get)
	mcp.AddTool(server, LFlowListTool, s.ListLogicalFlows)
	mcp.AddTool(server, TraceTool, s.Trace)
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
	lines = in.HeadTailParams.Apply(lines, defaultMaxLines)

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
	lines = in.HeadTailParams.Apply(lines, defaultMaxLines)

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
	lines = in.HeadTailParams.Apply(lines, defaultMaxLines)

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
	lines = in.HeadTailParams.Apply(lines, defaultMaxLines)

	result.Output = strings.Join(lines, "\n")
	return nil, result, nil
}
