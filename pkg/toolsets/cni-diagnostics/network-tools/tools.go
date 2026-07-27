package network_tools

import (
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/cni-diagnostics/utils"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	ovnknetmcp "github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/network-tools/mcp"
	"github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/network-tools/types"

	"k8s.io/utils/ptr"
)

func InitNetworkTools() []api.ServerTool {
	return []api.ServerTool{
		initTcpdumpTool(),
		initPwruTool(),
	}
}

// initTcpdumpTool creates the tcpdump tool.
func initTcpdumpTool() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "tcpdump",
			Description: "Capture network packets on a node or inside a pod with BPF filtering. Creates a specialized debug pod for node-level captures. IMPORTANT: Use restrictive BPF filters and low packet counts to avoid performance impact. Maximum 1000 packets.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"target_type": {
						Type:        "string",
						Description: "Capture target: 'node' (node-level) or 'pod' (pod network namespace)",
						Enum:        []interface{}{"node", "pod"},
					},
					"name": {
						Type:        "string",
						Description: "Name of the target (node or pod)",
					},
					"namespace": {
						Type:        "string",
						Description: "Namespace of the target (node or pod). Required when target_type is 'pod'. Optional when target_type is 'node' and defaults to 'default'.",
					},
					"container_name": {
						Type:        "string",
						Description: "Name of the container in the pod when target_type is 'pod' (optional, uses default container if not specified)",
					},
					"interface": {
						Type:        "string",
						Description: "Network interface name or 'any' (optional, captures on all interfaces if not specified)",
					},
					"packet_count": {
						Type:        "integer",
						Description: fmt.Sprintf("Number of packets to capture (default: %d, max: %d)", ovnknetmcp.DefaultPacketCount, ovnknetmcp.MaxPacketCount),
						Minimum:     ptr.To(float64(0)),
						Maximum:     ptr.To(float64(ovnknetmcp.MaxPacketCount)),
					},
					"bpf_filter": {
						Type:        "string",
						Description: "BPF filter expression (optional, e.g., 'tcp and dst port 8080', 'host 10.0.0.1')",
					},
					"snaplen": {
						Type:        "integer",
						Description: fmt.Sprintf("Snapshot length in bytes (default: %d, max: %d). Use %d for headers only, %d for full packets.", ovnknetmcp.DefaultSnaplen, ovnknetmcp.MaxSnaplen, ovnknetmcp.DefaultSnaplen, ovnknetmcp.MaxSnaplen),
						Minimum:     ptr.To(float64(0)),
						Maximum:     ptr.To(float64(ovnknetmcp.MaxSnaplen)),
					},
					"timeout_seconds": utils.TimeoutSecondsSchema(),
				},
				Required: []string{"target_type", "name"},
			},
			Annotations: utils.ReadOnlyAnnotations("CNI-Diagnostics: Network Packet Capture", false),
		},
		Handler: tcpdumpHandler,
	}
}

func tcpdumpHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p := api.WrapParams(params)

	// Extract parameters
	targetType := p.RequiredString("target_type")
	name := p.RequiredString("name")
	namespace := p.OptionalString("namespace", "")
	containerName := p.OptionalString("container_name", "")
	iface := p.OptionalString("interface", "")
	packetCount := int(p.OptionalInt64("packet_count", 0))
	bpfFilter := p.OptionalString("bpf_filter", "")
	snaplen := int(p.OptionalInt64("snaplen", 0))
	timeoutSecs := p.OptionalInt64("timeout_seconds", 0)

	if result, err, ok := utils.ParamsErrorResult(p); ok {
		return result, err
	}

	// Validate target type
	if targetType != "node" && targetType != "pod" {
		return utils.ValidationErrorResult("target_type must be 'node' or 'pod'")
	}
	// Validate name
	if name == "" {
		return utils.ValidationErrorResult(fmt.Sprintf("name is required when target_type is '%s'", targetType))
	}
	// Validate namespace
	if targetType == "pod" && namespace == "" {
		return utils.ValidationErrorResult("namespace is required when target_type is 'pod'")
	}

	// Create upstream parameters
	upstreamParams := types.TcpdumpParams{
		BaseNetworkDiagParams: types.BaseNetworkDiagParams{
			BPFFilter: bpfFilter,
		},
		TargetType:    targetType,
		Name:          name,
		Namespace:     namespace,
		ContainerName: containerName,
		Interface:     iface,
	}

	// Validate packet count
	var validationResult *api.ToolCallResult
	var validationErr error
	packetCount, validationResult, validationErr = utils.NormalizeBoundedInt("packet_count", packetCount, ovnknetmcp.MaxPacketCount, ovnknetmcp.DefaultPacketCount)
	if validationResult != nil {
		return validationResult, validationErr
	}
	upstreamParams.PacketCount = packetCount

	// Validate snaplen
	snaplen, validationResult, validationErr = utils.NormalizeBoundedInt("snaplen", snaplen, ovnknetmcp.MaxSnaplen, ovnknetmcp.DefaultSnaplen)
	if validationResult != nil {
		return validationResult, validationErr
	}
	upstreamParams.Snaplen = snaplen

	if validationResult, validationErr = utils.ValidateTimeout(&upstreamParams.TimeoutParams, timeoutSecs); validationResult != nil {
		return validationResult, validationErr
	}

	// Create upstream MCP server with our adapters
	upstreamServer, err := utils.NewNetworkToolsMCPServer(params, ovnknetmcp.Config{
		TcpdumpImage: utils.TcpdumpImage(params),
	})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create MCP server: %w", err)), nil
	}

	// Call upstream handler
	_, result, err := upstreamServer.Tcpdump(
		params.Context,
		&mcp.CallToolRequest{},
		upstreamParams,
	)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to capture packets: %w", err)), nil
	}

	return api.NewToolCallResult(utils.FormatCommandOutput(result.Output, result.Stderr), nil), nil
}

// initPwruTool creates the pwru tool.
func initPwruTool() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "pwru",
			Description: "Trace packets through the Linux kernel networking stack using eBPF. pwru (packet, where are you?) shows which kernel functions process a packet, helping debug packet drops and routing issues. Creates a specialized debug pod with eBPF capabilities.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"node_name": {
						Type:        "string",
						Description: "Name of the node to run pwru on",
					},
					"node_pod_namespace": {
						Type:        "string",
						Description: "Namespace of the debug pod on which the command is expected to be executed (optional, defaults to 'default')",
					},
					"bpf_filter": {
						Type:        "string",
						Description: "BPF filter expression to match packets (optional, e.g., 'tcp and dst port 8080', 'host 10.0.0.1')",
					},
					"output_limit_lines": {
						Type:        "integer",
						Description: fmt.Sprintf("Maximum number of trace events to capture (default: %d, max: %d)", ovnknetmcp.DefaultOutputLimitLines, ovnknetmcp.MaxOutputLimitLines),
						Minimum:     ptr.To(float64(0)),
						Maximum:     ptr.To(float64(ovnknetmcp.MaxOutputLimitLines)),
					},
					"timeout_seconds": utils.TimeoutSecondsSchema(),
				},
				Required: []string{"node_name"},
			},
			Annotations: utils.ReadOnlyAnnotations("CNI-Diagnostics: Network eBPF Packet Tracing", false),
		},
		Handler: pwruHandler,
	}
}

func pwruHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p := api.WrapParams(params)

	// Extract parameters
	nodeName := p.RequiredString("node_name")
	nodePodNamespace := p.OptionalString("node_pod_namespace", "")
	bpfFilter := p.OptionalString("bpf_filter", "")
	outputLimitLines := int(p.OptionalInt64("output_limit_lines", 0))
	timeoutSecs := p.OptionalInt64("timeout_seconds", 0)

	if result, err, ok := utils.ParamsErrorResult(p); ok {
		return result, err
	}

	// Create upstream parameters
	upstreamParams := types.PwruParams{
		BaseNetworkDiagParams: types.BaseNetworkDiagParams{
			BPFFilter: bpfFilter,
		},
		NodeName:         nodeName,
		NodePodNamespace: nodePodNamespace,
	}

	// Validate output limit lines
	var validationResult *api.ToolCallResult
	var validationErr error
	outputLimitLines, validationResult, validationErr = utils.NormalizeBoundedInt(
		"output_limit_lines",
		outputLimitLines,
		ovnknetmcp.MaxOutputLimitLines,
		ovnknetmcp.DefaultOutputLimitLines,
	)
	if validationResult != nil {
		return validationResult, validationErr
	}
	upstreamParams.OutputLimitLines = outputLimitLines

	if validationResult, validationErr = utils.ValidateTimeout(&upstreamParams.TimeoutParams, timeoutSecs); validationResult != nil {
		return validationResult, validationErr
	}

	// Create upstream MCP server with our adapters
	upstreamServer, err := utils.NewNetworkToolsMCPServer(params, ovnknetmcp.Config{
		PwruImage: utils.PwruImage(params),
	})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create MCP server: %w", err)), nil
	}

	// Call upstream handler
	_, result, err := upstreamServer.Pwru(
		params.Context,
		&mcp.CallToolRequest{},
		upstreamParams,
	)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to trace packets: %w", err)), nil
	}

	return api.NewToolCallResult(utils.FormatCommandOutput(result.Output, result.Stderr), nil), nil
}
