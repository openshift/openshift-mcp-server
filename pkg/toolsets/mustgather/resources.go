package mustgather

import (
	"fmt"
	"strings"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	mg "github.com/containers/kubernetes-mcp-server/pkg/ocp/mustgather"
	"github.com/google/jsonschema-go/jsonschema"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"
)

func initResources() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        "mustgather_resources_list",
				Description: "List Kubernetes resources from the must-gather archive with optional filtering by namespace, labels, and fields",
				Annotations: api.ToolAnnotations{
					Title:        "List Resources",
					ReadOnlyHint: ptr.To(true),
				},
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"path":          {Type: "string", Description: "Path to the must-gather archive directory (optional if mustgather_use was called earlier)"},
						"kind":          {Type: "string", Description: "Resource kind (e.g., Pod, Deployment, Service)"},
						"namespace":     {Type: "string", Description: "Filter by namespace"},
						"apiVersion":    {Type: "string", Description: "API version (default: v1)"},
						"labelSelector": {Type: "string", Description: "Label selector (e.g., app=nginx,tier=frontend)"},
						"fieldSelector": {Type: "string", Description: "Field selector (e.g., metadata.name=foo)"},
						"limit":         {Type: "integer", Description: "Maximum number of resources to return (0 for all)"},
					},
					Required: []string{"kind"},
				},
			},
			Handler:      mustgatherResourcesList,
			ClusterAware: ptr.To(false),
		},
	}
}

func mustgatherResourcesList(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()
	path := getString(args, "path", "")
	p, err := InitProvider(params.Context, path)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	kind := getString(args, "kind", "")
	namespace := getString(args, "namespace", "")
	apiVersion := getString(args, "apiVersion", "v1")
	labelSelector := getString(args, "labelSelector", "")
	fieldSelector := getString(args, "fieldSelector", "")
	limit := getInt(args, "limit", 0)

	if kind == "" {
		return api.NewToolCallResult("", fmt.Errorf("kind is required")), nil
	}

	gvk := parseGVK(apiVersion, kind)
	list, err := p.ListResources(params.Context, gvk, namespace, mg.ListOptions{
		LabelSelector: labelSelector,
		FieldSelector: fieldSelector,
		Limit:         limit,
	})
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	if len(list.Items) == 0 {
		return api.NewToolCallResult(fmt.Sprintf("No %s resources found", kind), nil), nil
	}

	output := fmt.Sprintf("Found %d %s resource(s):\n\n", len(list.Items), kind)
	for i := range list.Items {
		item := &list.Items[i]
		ns := item.GetNamespace()
		name := item.GetName()
		if ns != "" {
			output += fmt.Sprintf("- %s/%s\n", ns, name)
		} else {
			output += fmt.Sprintf("- %s\n", name)
		}
	}

	// Show full YAML for small result sets
	if len(list.Items) <= 5 {
		output += "\n---\n\n"
		for i := range list.Items {
			yamlBytes, err := yaml.Marshal(list.Items[i].Object)
			if err == nil {
				output += string(yamlBytes) + "\n---\n\n"
			}
		}
	}

	return api.NewToolCallResult(output, nil), nil
}

func parseGVK(apiVersion, kind string) schema.GroupVersionKind {
	parts := strings.SplitN(apiVersion, "/", 2)
	if len(parts) == 2 {
		return schema.GroupVersionKind{Group: parts[0], Version: parts[1], Kind: kind}
	}
	return schema.GroupVersionKind{Group: "", Version: apiVersion, Kind: kind}
}
