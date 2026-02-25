package mustgather

import (
	"fmt"
	"sort"
	"strings"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/google/jsonschema-go/jsonschema"
	"k8s.io/utils/ptr"
)

func initNamespaces() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        "mustgather_namespaces_list",
				Description: "List all namespaces found in the must-gather archive",
				Annotations: api.ToolAnnotations{
					Title:        "List Namespaces",
					ReadOnlyHint: ptr.To(true),
				},
				InputSchema: &jsonschema.Schema{
					Type: "object",
				},
			},
			Handler:      mustgatherNamespacesList,
			ClusterAware: ptr.To(false),
		},
	}
}

func mustgatherNamespacesList(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p, err := getProvider()
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	namespaces := p.ListNamespaces()
	sort.Strings(namespaces)

	output := fmt.Sprintf("Found %d namespaces:\n\n", len(namespaces))
	output += strings.Join(namespaces, "\n")
	output += "\n"

	return api.NewToolCallResult(output, nil), nil
}
