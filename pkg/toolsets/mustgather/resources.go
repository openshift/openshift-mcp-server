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
				Name:        "mustgather_resources_get",
				Description: "Get a specific Kubernetes resource from the must-gather archive by kind, name, and optional namespace",
				Annotations: api.ToolAnnotations{
					Title:        "Get Resource",
					ReadOnlyHint: ptr.To(true),
				},
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"kind":       {Type: "string", Description: "Resource kind (e.g., Pod, Deployment, Service)"},
						"name":       {Type: "string", Description: "Resource name"},
						"namespace":  {Type: "string", Description: "Resource namespace (omit for cluster-scoped resources)"},
						"apiVersion": {Type: "string", Description: "API version (default: v1)"},
					},
					Required: []string{"kind", "name"},
				},
			},
			Handler:      mustgatherResourcesGet,
			ClusterAware: ptr.To(false),
		},
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

func mustgatherResourcesGet(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p, err := getProvider()
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	args := params.GetArguments()
	kind := getString(args, "kind", "")
	name := getString(args, "name", "")
	namespace := getString(args, "namespace", "")
	apiVersion := getString(args, "apiVersion", "v1")

	if kind == "" || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("kind and name are required")), nil
	}

	gvk := parseGVK(apiVersion, kind)
	obj := p.GetResource(gvk, name, namespace)
	if obj == nil {
		return api.NewToolCallResult("", fmt.Errorf("resource %s/%s not found", kind, name)), nil
	}

	yamlBytes, err := yaml.Marshal(obj.Object)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal resource: %w", err)), nil
	}

	return api.NewToolCallResult(string(yamlBytes), nil), nil
}

func mustgatherResourcesList(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p, err := getProvider()
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	args := params.GetArguments()
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
	list := p.ListResources(params.Context, gvk, namespace, mg.ListOptions{
		LabelSelector: labelSelector,
		FieldSelector: fieldSelector,
		Limit:         limit,
	})

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
