package mustgather

import (
	"fmt"
	"sort"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/google/jsonschema-go/jsonschema"
	"k8s.io/utils/ptr"
)

func initEtcd() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        "mustgather_etcd_health",
				Description: "Get ETCD cluster health status including endpoint health and active alarms from the must-gather archive",
				Annotations: api.ToolAnnotations{
					Title:        "ETCD Health",
					ReadOnlyHint: ptr.To(true),
				},
				InputSchema: &jsonschema.Schema{
					Type: "object",
				},
			},
			Handler:      mustgatherETCDHealth,
			ClusterAware: ptr.To(false),
		},
		{
			Tool: api.Tool{
				Name:        "mustgather_etcd_members_list",
				Description: "List ETCD cluster members from the must-gather archive",
				Annotations: api.ToolAnnotations{
					Title:        "ETCD Members",
					ReadOnlyHint: ptr.To(true),
				},
				InputSchema: &jsonschema.Schema{
					Type: "object",
				},
			},
			Handler:      mustgatherETCDMembersList,
			ClusterAware: ptr.To(false),
		},
		{
			Tool: api.Tool{
				Name:        "mustgather_etcd_endpoint_status",
				Description: "Get ETCD endpoint status including database size, raft index, and quota from the must-gather archive",
				Annotations: api.ToolAnnotations{
					Title:        "ETCD Endpoint Status",
					ReadOnlyHint: ptr.To(true),
				},
				InputSchema: &jsonschema.Schema{
					Type: "object",
				},
			},
			Handler:      mustgatherETCDEndpointStatus,
			ClusterAware: ptr.To(false),
		},
		{
			Tool: api.Tool{
				Name:        "mustgather_etcd_object_count",
				Description: "Get ETCD object counts by resource type from the must-gather archive",
				Annotations: api.ToolAnnotations{
					Title:        "ETCD Object Count",
					ReadOnlyHint: ptr.To(true),
				},
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"limit": {Type: "integer", Description: "Maximum number of resource types to show (default: 50, sorted by count descending)"},
					},
				},
			},
			Handler:      mustgatherETCDObjectCount,
			ClusterAware: ptr.To(false),
		},
	}
}

func mustgatherETCDHealth(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p, err := getProvider()
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	health, err := p.GetETCDHealth()
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get ETCD health: %w", err)), nil
	}

	output := "## ETCD Cluster Health\n\n"
	if health.Healthy {
		output += "Status: [OK] Healthy\n\n"
	} else {
		output += "Status: [FAIL] Unhealthy\n\n"
	}

	output += "### Endpoints\n\n"
	for _, ep := range health.Endpoints {
		output += fmt.Sprintf("- %s %s\n", healthSymbol(ep.Health), ep.Address)
	}

	if len(health.Alarms) > 0 {
		output += "\n### Active Alarms\n\n"
		for _, alarm := range health.Alarms {
			output += fmt.Sprintf("- [WARNING] %s\n", alarm)
		}
	} else {
		output += "\nNo active alarms.\n"
	}

	return api.NewToolCallResult(output, nil), nil
}

func mustgatherETCDMembersList(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p, err := getProvider()
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	data, err := p.ReadETCDFile("member_list.json")
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to read ETCD member list: %w", err)), nil
	}

	output := "## ETCD Members\n\n"
	output += string(data) + "\n"

	return api.NewToolCallResult(output, nil), nil
}

func mustgatherETCDEndpointStatus(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p, err := getProvider()
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	data, err := p.ReadETCDFile("endpoint_status.json")
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to read ETCD endpoint status: %w", err)), nil
	}

	output := "## ETCD Endpoint Status\n\n"
	output += string(data) + "\n"

	return api.NewToolCallResult(output, nil), nil
}

func mustgatherETCDObjectCount(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p, err := getProvider()
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	limit := getInt(params.GetArguments(), "limit", 50)

	counts, err := p.GetETCDObjectCount()
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get ETCD object counts: %w", err)), nil
	}

	// Sort by count descending
	type entry struct {
		resource string
		count    int64
	}
	entries := make([]entry, 0, len(counts))
	var total int64
	for k, v := range counts {
		entries = append(entries, entry{resource: k, count: v})
		total += v
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].count > entries[j].count
	})

	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}

	output := fmt.Sprintf("## ETCD Object Counts\n\nTotal objects: %s across %d resource types\n\n", formatNumber(total), len(counts))

	// Find max resource name length for alignment
	maxLen := 0
	for _, e := range entries {
		if len(e.resource) > maxLen {
			maxLen = len(e.resource)
		}
	}

	for _, e := range entries {
		output += fmt.Sprintf("%-*s  %s\n", maxLen, e.resource, formatNumber(e.count))
	}

	if limit > 0 && len(counts) > limit {
		output += fmt.Sprintf("\n... showing top %d of %d resource types\n", limit, len(counts))
	}

	return api.NewToolCallResult(output, nil), nil
}
