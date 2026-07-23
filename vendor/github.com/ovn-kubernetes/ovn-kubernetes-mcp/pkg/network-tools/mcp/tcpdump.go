package mcp

import (
	"context"
	"fmt"
	"strconv"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/network-tools/types"
	"github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/utils/commandbuilder"
)

const (
	DefaultPacketCount = 100
	MaxPacketCount     = 1000
	// DefaultSnaplen is set to 96 bytes to capture headers (Ethernet + IP + TCP/UDP + some payload)
	// This is sufficient for most diagnostic purposes while minimizing data capture
	DefaultSnaplen = 96
	// MaxSnaplen is set to 1500 bytes (standard Ethernet MTU) to allow full packet capture
	// when needed for deeper analysis. Users can set snaplen parameter to capture complete packets.
	MaxSnaplen = 1500
)

// Tcpdump executes the tcpdump packet capture tool on a node or inside a pod.
func (s *MCPServer) Tcpdump(ctx context.Context, req *mcp.CallToolRequest, in types.TcpdumpParams) (*mcp.CallToolResult, types.CommandResult, error) {
	if in.TargetType == "" || (in.TargetType != "node" && in.TargetType != "pod") {
		return nil, types.CommandResult{}, fmt.Errorf("target_type is required and must be 'node' or 'pod'")
	}
	if in.Name == "" {
		return nil, types.CommandResult{}, fmt.Errorf("name is required when target_type is '%s'", in.TargetType)
	}
	if in.TargetType == "pod" && in.Namespace == "" {
		return nil, types.CommandResult{}, fmt.Errorf("namespace is required when target_type is 'pod'")
	}
	if err := validateInterface(in.Interface); err != nil {
		return nil, types.CommandResult{}, err
	}
	if err := validatePacketFilter(in.BPFFilter); err != nil {
		return nil, types.CommandResult{}, err
	}

	packetCount := in.PacketCount
	if packetCount == 0 {
		packetCount = DefaultPacketCount
	}
	if err := validateIntMax(packetCount, MaxPacketCount, "packet_count", ""); err != nil {
		return nil, types.CommandResult{}, err
	}

	snaplen := in.Snaplen
	if snaplen == 0 {
		snaplen = DefaultSnaplen
	}
	if err := validateIntMax(snaplen, MaxSnaplen, "snaplen", "bytes"); err != nil {
		return nil, types.CommandResult{}, err
	}

	cmd := commandbuilder.NewCommand("tcpdump", "-n", "-v",
		"-s", strconv.Itoa(snaplen),
		"-c", strconv.Itoa(packetCount))
	cmd.AddIfNotEmpty(in.Interface, "-i", in.Interface)
	cmd.AddIfNotEmpty(in.BPFFilter, in.BPFFilter)

	// If timeout is specified, create a new context with timeout
	var cancel context.CancelFunc
	ctx, cancel = in.TimeoutParams.WithTimeout(ctx)
	if cancel != nil {
		defer cancel()
	}

	switch in.TargetType {
	case "node":
		stdout, stderr, err := s.runDebugNodeCommand(ctx, in.Namespace, in.Name, s.cfg.TcpdumpImage, cmd.Build(), "", "", 0)
		if err != nil {
			return nil, types.CommandResult{}, err
		}
		return nil, types.CommandResult{Output: stdout, Stderr: stderr}, nil
	case "pod":
		stdout, stderr, err := s.runPodExecCommand(ctx, in.Namespace, in.Name, in.ContainerName, cmd.Build())
		if err != nil {
			return nil, types.CommandResult{}, err
		}
		return nil, types.CommandResult{Output: stdout, Stderr: stderr}, nil
	default:
		return nil, types.CommandResult{}, fmt.Errorf("invalid target_type: %s (must be 'node' or 'pod')", in.TargetType)
	}
}
