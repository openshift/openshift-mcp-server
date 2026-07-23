package types

import "github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/utils/timeout"

// BaseNetworkDiagParams contains common parameters shared across network diagnostic tools.
type BaseNetworkDiagParams struct {
	BPFFilter string `json:"bpf_filter,omitempty"`
}

// TcpdumpParams contains parameters for running tcpdump packet capture.
// Supports both node-level and pod-level packet capture with BPF filtering.
type TcpdumpParams struct {
	BaseNetworkDiagParams

	TargetType string `json:"target_type"`

	// Combined node and pod fields
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`

	// Pod-specific field
	ContainerName string `json:"container_name,omitempty"`

	Interface   string `json:"interface,omitempty"`
	PacketCount int    `json:"packet_count,omitempty"`
	Snaplen     int    `json:"snaplen,omitempty"`

	timeout.TimeoutParams
}

// PwruParams contains parameters for running pwru (packet, where are you?) eBPF-based
// kernel packet tracing. This tool traces packets through the Linux kernel networking stack.
type PwruParams struct {
	BaseNetworkDiagParams

	NodeName         string `json:"node_name"`
	NodePodNamespace string `json:"node_pod_namespace,omitempty"`
	OutputLimitLines int    `json:"output_limit_lines,omitempty"`

	timeout.TimeoutParams
}

// CommandResult represents the output and status of an executed command.
type CommandResult struct {
	Output string `json:"output"`
	Stderr string `json:"stderr,omitempty"`
}
