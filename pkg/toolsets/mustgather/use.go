package mustgather

import (
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/google/jsonschema-go/jsonschema"
	"k8s.io/utils/ptr"
)

func initUse() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        "mustgather_use",
				Description: "Load a must-gather archive from a given filesystem path for analysis. Other mustgather_* tools can also accept a path argument directly.",
				Annotations: api.ToolAnnotations{
					Title:        "Load Must-Gather Archive",
					ReadOnlyHint: ptr.To(true),
				},
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"path": {
							Type:        "string",
							Description: "Absolute path to the must-gather archive directory",
						},
					},
					Required: []string{"path"},
				},
			},
			Handler:      mustgatherUse,
			ClusterAware: ptr.To(false),
		},
	}
}

func mustgatherUse(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()
	path := getString(args, "path", "")
	if path == "" {
		return api.NewToolCallResult("", fmt.Errorf("path is required")), nil
	}

	p, err := getOrInitProvider(params.Context, path)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to load must-gather archive: %w", err)), nil
	}

	metadata := p.GetMetadata()

	output := "Must-gather archive loaded successfully\n"
	output += "=======================================\n\n"
	output += fmt.Sprintf("Path: %s\n", metadata.Path)
	if metadata.Version != "" {
		output += fmt.Sprintf("Version: %s\n", metadata.Version)
	}
	if metadata.Timestamp != "" {
		output += fmt.Sprintf("Timestamp: %s\n", metadata.Timestamp)
	}
	output += fmt.Sprintf("Resources indexed: %d\n", metadata.ResourceCount)
	output += fmt.Sprintf("Namespaces: %d\n", metadata.NamespaceCount)

	return api.NewToolCallResult(output, nil), nil
}
