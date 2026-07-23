package mcp

import (
	"context"
	"strconv"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/network-tools/types"
	"github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/utils/commandbuilder"
)

const (
	DefaultOutputLimitLines = 100
	MaxOutputLimitLines     = 1000
)

// Pwru executes the pwru (packet, where are you?) tool to trace packets through the Linux kernel.
// It creates a specialized debug pod with eBPF capabilities and traces packet processing paths.
// This is useful for debugging packet drops, routing issues, and understanding kernel networking behavior.
func (s *MCPServer) Pwru(ctx context.Context, req *mcp.CallToolRequest, in types.PwruParams) (*mcp.CallToolResult, types.CommandResult, error) {
	outputLimitLines := in.OutputLimitLines
	if outputLimitLines == 0 {
		outputLimitLines = DefaultOutputLimitLines
	}
	if err := validateIntMax(outputLimitLines, MaxOutputLimitLines, "output_limit_lines", ""); err != nil {
		return nil, types.CommandResult{}, err
	}

	if err := validatePacketFilter(in.BPFFilter); err != nil {
		return nil, types.CommandResult{}, err
	}

	cmd := commandbuilder.NewCommand("pwru", "--output-limit-lines", strconv.Itoa(outputLimitLines))
	// pwru accepts pcap filter as positional argument(s)
	cmd.AddIfNotEmpty(in.BPFFilter, in.BPFFilter)

	// If timeout is specified, create a new context with timeout
	var cancel context.CancelFunc
	ctx, cancel = in.TimeoutParams.WithTimeout(ctx)
	if cancel != nil {
		defer cancel()
	}

	stdout, stderr, err := s.runDebugNodeCommand(ctx, in.NodePodNamespace, in.NodeName, s.cfg.PwruImage, cmd.Build(), "/sys/kernel/debug", "/sys/kernel/debug", 0)
	if err != nil {
		return nil, types.CommandResult{}, err
	}
	return nil, types.CommandResult{Output: stdout, Stderr: stderr}, nil
}
