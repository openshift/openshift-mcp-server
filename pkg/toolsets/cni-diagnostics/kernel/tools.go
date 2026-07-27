package kernel

import (
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/cni-diagnostics/utils"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	ovnkkernelmcp "github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/kernel/mcp"
	"github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/kernel/types"
)

func InitKernelTools() []api.ServerTool {
	return []api.ServerTool{
		initConntrackTool(),
		initIPtablesTool(),
		initNFTTool(),
		initIPTool(),
	}
}

// initConntrackTool creates the get-conntrack tool.
func initConntrackTool() api.ServerTool {
	props := map[string]*jsonschema.Schema{
		"node": {
			Type:        "string",
			Description: "Name of the node from where conntrack entries are expected to be extracted",
		},
		"namespace": {
			Type:        "string",
			Description: "Namespace of the debug pod from where conntrack entries are expected to be extracted (optional, defaults to 'default')",
		},
		"command": {
			Type: "string",
			Description: `These options specify the particular operation to perform. These options can only be used if configured image has 'conntrack' utility available.
							-L, --dump : List connection tracking table.
							-C, --count: Show the table counter.
							-S, --stats: Show the in-kernel connection tracking system statistics.`,
			Enum: []interface{}{"-L", "--dump", "-C", "--count", "-S", "--stats"},
		},
		"filter_parameters": {
			Type: "string",
			Description: `These parameters are useful to filter certain entries from the whole table:
							-s, --src, --orig-src IP_ADDRESS : Match only entries whose source address in the original direction equals to mentioned IP.
							-d, --dst, --orig-dst IP_ADDRESS : Match only entries whose destination address in the original direction equals to mentioned IP.
							-p, --proto PROTO                : Specify layer four (TCP, UDP, ...) protocol.
							--sport, --orig-port-src PORT    : Source port in original direction.
							--dport, --orig-port-dst PORT    : Destination port in original direction.`,
		},
	}
	utils.AddHeadTailProperties(props, fmt.Sprintf("Return only first N lines. Default: %d lines if tail is not specified", ovnkkernelmcp.DefaultMaxOutputLines))
	utils.AddTimeoutSecondsProperty(props)

	return api.ServerTool{
		Tool: api.Tool{
			Name:        "get-conntrack",
			Description: "Interact with the connection tracking system on a Kubernetes node. Lists, counts, or shows statistics for tracked connections. Connection tracking shows active network connections and their state (ESTABLISHED, TIME_WAIT, etc.).",
			InputSchema: &jsonschema.Schema{
				Type:       "object",
				Properties: props,
				Required:   []string{"node"},
			},
			Annotations: utils.ReadOnlyAnnotations("CNI-Diagnostics: Kernel Connection Tracking", true),
		},
		Handler: conntrackHandler,
	}
}

func conntrackHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p := api.WrapParams(params)

	// Extract parameters
	node := p.RequiredString("node")
	namespace := p.OptionalString("namespace", "")
	command := p.OptionalString("command", "")
	filterParams := p.OptionalString("filter_parameters", "")
	head := int(p.OptionalInt64("head", 0))
	tail := int(p.OptionalInt64("tail", 0))
	applyTailFirst := p.OptionalBool("apply_tail_first", false)
	timeoutSecs := p.OptionalInt64("timeout_seconds", 0)

	if result, err, ok := utils.ParamsErrorResult(p); ok {
		return result, err
	}

	// Create upstream parameters
	upstreamParams := types.ListConntrackParams{
		CommonParams: types.CommonParams{
			Node:      node,
			Namespace: namespace,
		},
		Command:          command,
		FilterParameters: filterParams,
	}

	if result, err := utils.ValidateHeadTail(&upstreamParams.HeadTailParams, head, tail, applyTailFirst); result != nil {
		return result, err
	}
	if result, err := utils.ValidateTimeout(&upstreamParams.TimeoutParams, timeoutSecs); result != nil {
		return result, err
	}

	upstreamServer, err := utils.NewKernelMCPServer(params, utils.KernelDebugImage(params))
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create MCP server: %w", err)), nil
	}

	// Call upstream handler
	_, result, err := upstreamServer.GetConntrack(
		params.Context,
		&mcp.CallToolRequest{}, // MCP request not used by handler
		upstreamParams,
	)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get conntrack data: %w", err)), nil
	}

	return api.NewToolCallResult(result.Data, nil), nil
}

// initIPtablesTool creates the get-iptables tool.
func initIPtablesTool() api.ServerTool {
	props := map[string]*jsonschema.Schema{
		"node": {
			Type:        "string",
			Description: "Name of the node from where packet filter rules are expected to be extracted",
		},
		"namespace": {
			Type:        "string",
			Description: "Namespace of the debug pod from where packet filter rules are expected to be extracted (optional, defaults to 'default')",
		},
		"table": {
			Type: "string",
			Description: `There are currently five independent tables (which tables are present at any time depends on the kernel configuration options and which modules are present).
							filter	: This is the default table
							nat   	: This  table is consulted when a packet that creates a new connection is encountered.
							mangle	: This table is used for specialized packet alteration.
							raw   	: This table is used mainly for configuring exemptions from connection tracking in combination with the NOTRACK target.
							security: This table is used for Mandatory Access Control (MAC) networking rules.`,
			Enum: []interface{}{"filter", "nat", "mangle", "raw", "security"},
		},
		"command": {
			Type: "string",
			Description: `These options specify the desired action to perform. Only one of them can be specified on the command line unless otherwise stated below.
							-L, --list [chain]       : List all rules in the selected chain. If no chain is selected, all chains are listed.
							-S, --list-rules [chain] : Print all rules in the selected chain. If no chain is selected, all chains are printed like iptables-save.`,
		},
		"filter_parameters": {
			Type: "string",
			Description: `These parameters are useful to filter certain entries from the whole table:
							-s, --source address[/mask]      : Source specification. Address can be either a network name, a hostname, a network IP address (with /mask), or a  plain  IP  address.
							-d, --destination address[/mask] : Destination  specification.
							-v, --verbose					 : Verbose output.
							-n, --numeric                    : Numeric  output.   IP  addresses  and port numbers will be printed in numeric format.
							-p, --protocol protocol          : The protocol of the rule or of the packet to check.
							-4, --ipv4                       : IPv4
							-6, --ipv6                       : IPv6`,
		},
	}
	utils.AddHeadTailProperties(props, fmt.Sprintf("Return only first N lines. Default: %d lines if tail is not specified", ovnkkernelmcp.DefaultMaxOutputLines))
	utils.AddTimeoutSecondsProperty(props)

	return api.ServerTool{
		Tool: api.Tool{
			Name:        "get-iptables",
			Description: "List packet filter rules using iptables or ip6tables on a Kubernetes node. Shows rules for specific tables (filter, nat, mangle, raw, security). Use this to inspect firewall rules, NAT configuration, and packet filtering on nodes.",
			InputSchema: &jsonschema.Schema{
				Type:       "object",
				Properties: props,
				Required:   []string{"node"},
			},
			Annotations: utils.ReadOnlyAnnotations("CNI-Diagnostics: Kernel IPtables", true),
		},
		Handler: iptablesHandler,
	}
}

func iptablesHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p := api.WrapParams(params)

	// Extract parameters
	node := p.RequiredString("node")
	namespace := p.OptionalString("namespace", "")
	command := p.OptionalString("command", "-L")
	table := p.OptionalString("table", "")
	filterParams := p.OptionalString("filter_parameters", "")
	head := int(p.OptionalInt64("head", 0))
	tail := int(p.OptionalInt64("tail", 0))
	applyTailFirst := p.OptionalBool("apply_tail_first", false)
	timeoutSecs := p.OptionalInt64("timeout_seconds", 0)

	if result, err, ok := utils.ParamsErrorResult(p); ok {
		return result, err
	}

	// Create upstream parameters
	upstreamParams := types.ListIPTablesParams{
		CommonParams: types.CommonParams{
			Node:      node,
			Namespace: namespace,
		},
		Table:            table,
		Command:          command,
		FilterParameters: filterParams,
	}

	if result, err := utils.ValidateHeadTail(&upstreamParams.HeadTailParams, head, tail, applyTailFirst); result != nil {
		return result, err
	}
	if result, err := utils.ValidateTimeout(&upstreamParams.TimeoutParams, timeoutSecs); result != nil {
		return result, err
	}

	upstreamServer, err := utils.NewKernelMCPServer(params, utils.KernelDebugImage(params))
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create MCP server: %w", err)), nil
	}

	// Call upstream handler
	_, result, err := upstreamServer.GetIptables(
		params.Context,
		&mcp.CallToolRequest{},
		upstreamParams,
	)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get iptables data: %w", err)), nil
	}

	return api.NewToolCallResult(result.Data, nil), nil
}

// initNFTTool creates the get-nft tool.
func initNFTTool() api.ServerTool {
	props := map[string]*jsonschema.Schema{
		"node": {
			Type:        "string",
			Description: "Name of the node from where packet filtering and classification rules are expected to be extracted",
		},
		"namespace": {
			Type:        "string",
			Description: "Namespace of the debug pod from where packet filtering and classification rules are expected to be extracted (optional, defaults to 'default')",
		},
		"command": {
			Type: "string",
			Description: `These options specify the desired action to perform. Only one of them can be specified on the command line unless otherwise stated below.
                    					- list ruleset   : The ruleset keyword is used to identify the whole set of tables, chains, etc. Print the ruleset in human-readable format.
										- list tables    : List all chains and rules of the specified table.
										- list chains    : List all rules of the specified chain.
										- list sets      : Display the elements in the specified set.
										- list maps      : Display the elements in the specified map.
										- list flowtables: List all flowtables.`,
			Enum: []interface{}{"list ruleset", "list tables", "list chains", "list sets", "list maps", "list flowtables"},
		},
		"address_families": {
			Type: "string",
			Description: `Address families determine the type of packets which are processed. For each address family, the kernel contains so called hooks at specific stages of
       						   the packet processing paths, which invoke nftables if rules for these hooks exist.
							   - ip       IPv4 address family.
                               - ip6      IPv6 address family.
                               - inet     Internet (IPv4/IPv6) address family.
                               - arp      ARP address family, handling IPv4 ARP packets.
                               - bridge   Bridge address family, handling packets which traverse a bridge device.
                               - netdev   Netdev address family, handling packets on ingress and egress.`,
			Enum: []interface{}{"ip", "ip6", "inet", "arp", "bridge", "netdev"},
		},
	}
	utils.AddHeadTailProperties(props, fmt.Sprintf("Return only first N lines. Default: %d lines if tail is not specified", ovnkkernelmcp.DefaultMaxOutputLines))
	utils.AddTimeoutSecondsProperty(props)

	return api.ServerTool{
		Tool: api.Tool{
			Name:        "get-nft",
			Description: "List nftables packet filtering and classification rules on a Kubernetes node. nftables is the modern replacement for iptables. Use this to inspect firewall rules, packet filtering, and network address translation.",
			InputSchema: &jsonschema.Schema{
				Type:       "object",
				Properties: props,
				Required:   []string{"node", "command"},
			},
			Annotations: utils.ReadOnlyAnnotations("CNI-Diagnostics: Kernel NFtables", true),
		},
		Handler: nftHandler,
	}
}

func nftHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p := api.WrapParams(params)

	// Extract parameters
	node := p.RequiredString("node")
	namespace := p.OptionalString("namespace", "")
	command := p.RequiredString("command")
	addressFamilies := p.OptionalString("address_families", "")
	head := int(p.OptionalInt64("head", 0))
	tail := int(p.OptionalInt64("tail", 0))
	applyTailFirst := p.OptionalBool("apply_tail_first", false)
	timeoutSecs := p.OptionalInt64("timeout_seconds", 0)

	if result, err, ok := utils.ParamsErrorResult(p); ok {
		return result, err
	}

	// Create upstream parameters
	upstreamParams := types.ListNFTParams{
		CommonParams: types.CommonParams{
			Node:      node,
			Namespace: namespace,
		},
		Command:         command,
		AddressFamilies: addressFamilies,
	}

	if result, err := utils.ValidateHeadTail(&upstreamParams.HeadTailParams, head, tail, applyTailFirst); result != nil {
		return result, err
	}
	if result, err := utils.ValidateTimeout(&upstreamParams.TimeoutParams, timeoutSecs); result != nil {
		return result, err
	}

	upstreamServer, err := utils.NewKernelMCPServer(params, utils.KernelDebugImage(params))
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create MCP server: %w", err)), nil
	}

	// Call upstream handler
	_, result, err := upstreamServer.GetNFT(
		params.Context,
		&mcp.CallToolRequest{},
		upstreamParams,
	)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get nftables data: %w", err)), nil
	}

	return api.NewToolCallResult(result.Data, nil), nil
}

// initIPTool creates the get-ip tool.
func initIPTool() api.ServerTool {
	props := map[string]*jsonschema.Schema{
		"node": {
			Type:        "string",
			Description: "Name of the node on which ip command is expected to be executed",
		},
		"namespace": {
			Type:        "string",
			Description: "Namespace of the debug pod on which ip command is expected to be executed (optional, defaults to 'default')",
		},
		"options": {
			Type: "string",
			Description: `These options helps in providing more details or formattig output data.
                      					-d, -details        : Output more detailed information.
					  					-4                  : shortcut for -family inet.
					  					-6                  : shortcut for -family inet6.
					  					-r, -resolve        : use the system's name resolver to print DNS names instead of host addresses.
					  					-n, -netns <NETNS>  : switches ip to the specified network namespace NETNS.
					  -a, -all            : executes specified command over all objects, it depends if command supports this option.`,
		},
		"command": {
			Type: "string",
			Description: `These options specify the desired action to perform. Only one of them can be specified on the command line unless otherwise stated below.
                      - address show     : protocol (IP or IPv6) address on a device.
					  - link show        : network device.
					  - neighbour show   : manage ARP or NDISC cache entries.
					  - netns show       : manage network namespaces.
					  - route show       : routing table entry.
					  - rule show        : rule in routing policy database.
					  - vrf show         : manage virtual routing and forwarding devices. 
					  - xfrm state list  : show Security Association Database.
					  - xfrm policy list : show Security Policy Database.`,
			Enum: []interface{}{"address show", "link show", "neighbour show", "netns show", "route show", "rule show", "vrf show", "xfrm state list", "xfrm policy list"},
		},
		"filter_parameters": {
			Type: "string",
			Description: `This allows to mention sub command to get more filtered data. Available sub command varies and supportability depends on what is 
                          already supported with 'ip' utility.`,
		},
	}
	utils.AddHeadTailProperties(props, fmt.Sprintf("Return only first N lines. Default: %d lines if tail is not specified", ovnkkernelmcp.DefaultMaxOutputLines))
	utils.AddTimeoutSecondsProperty(props)

	return api.ServerTool{
		Tool: api.Tool{
			Name:        "get-ip",
			Description: "Execute ip commands on a Kubernetes node to show routing, network devices, interfaces, and network namespaces. Part of the iproute2 suite for network configuration inspection.",
			InputSchema: &jsonschema.Schema{
				Type:       "object",
				Properties: props,
				Required:   []string{"node", "command"},
			},
			Annotations: utils.ReadOnlyAnnotations("CNI-Diagnostics: Kernel IP Command", true),
		},
		Handler: ipHandler,
	}
}

func ipHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p := api.WrapParams(params)

	// Extract parameters
	node := p.RequiredString("node")
	namespace := p.OptionalString("namespace", "")
	command := p.RequiredString("command")
	options := p.OptionalString("options", "")
	filterParams := p.OptionalString("filter_parameters", "")
	head := int(p.OptionalInt64("head", 0))
	tail := int(p.OptionalInt64("tail", 0))
	applyTailFirst := p.OptionalBool("apply_tail_first", false)
	timeoutSecs := p.OptionalInt64("timeout_seconds", 0)

	if result, err, ok := utils.ParamsErrorResult(p); ok {
		return result, err
	}

	// Create upstream parameters
	upstreamParams := types.ListIPParams{
		CommonParams: types.CommonParams{
			Node:      node,
			Namespace: namespace,
		},
		Options:          options,
		Command:          command,
		FilterParameters: filterParams,
	}

	if result, err := utils.ValidateHeadTail(&upstreamParams.HeadTailParams, head, tail, applyTailFirst); result != nil {
		return result, err
	}
	if result, err := utils.ValidateTimeout(&upstreamParams.TimeoutParams, timeoutSecs); result != nil {
		return result, err
	}

	upstreamServer, err := utils.NewKernelMCPServer(params, utils.KernelDebugImage(params))
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create MCP server: %w", err)), nil
	}

	// Call upstream handler
	_, result, err := upstreamServer.GetIPCommandOutput(
		params.Context,
		&mcp.CallToolRequest{},
		upstreamParams,
	)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get ip command output: %w", err)), nil
	}

	return api.NewToolCallResult(result.Data, nil), nil
}
