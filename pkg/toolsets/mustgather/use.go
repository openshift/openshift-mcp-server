package mustgather

import (
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	mg "github.com/containers/kubernetes-mcp-server/pkg/ocp/mustgather"
	"github.com/google/jsonschema-go/jsonschema"
	"k8s.io/utils/ptr"
)

func initUse() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        "mustgather_use",
				Description: "Load a must-gather archive from a given filesystem path for analysis. Must be called before any other mustgather_* tools.",
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

	p, err := mg.NewProvider(path)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to load must-gather archive: %w", err)), nil
	}

	setProvider(p)

	metadata := p.GetMetadata()

	// Register MCP resource
	if params.ResourceRegistrar != nil {
		resourceURI := "must-gather://" + path
		content := fmt.Sprintf("Must-Gather Archive\n"+
			"Path: %s\n"+
			"Version: %s\n"+
			"Timestamp: %s\n"+
			"Resources: %d\n"+
			"Namespaces: %d\n",
			metadata.Path, metadata.Version, metadata.Timestamp,
			metadata.ResourceCount, metadata.NamespaceCount)
		params.ResourceRegistrar.AddResource(
			resourceURI,
			"must-gather",
			"Loaded must-gather archive metadata",
			"text/plain",
			content,
		)
	}

	// Build summary
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
