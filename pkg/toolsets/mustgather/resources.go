package mustgather

import (
	"fmt"
	"strings"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	mg "github.com/containers/kubernetes-mcp-server/pkg/ocp/mustgather"
	"github.com/google/jsonschema-go/jsonschema"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
						"archive_id":    archiveIDProperty(),
						"kind":          {Type: "string", Description: "Resource kind (e.g., Pod, Deployment, Service)"},
						"namespace":     {Type: "string", Description: "Filter by namespace"},
						"apiVersion":    {Type: "string", Description: "API version (default: v1)"},
						"labelSelector": {Type: "string", Description: "Label selector (e.g., app=nginx,tier=frontend)"},
						"fieldSelector": {Type: "string", Description: "Field selector (e.g., metadata.name=foo)"},
						"limit":         {Type: "integer", Description: "Maximum number of resources to return (0 for all)"},
					},
					Required: []string{"archive_id", "kind"},
				},
			},
			RBAC:         api.RBACNone(),
			Handler:      mustgatherResourcesList,
			ClusterAware: ptr.To(false),
		},
	}
}

func mustgatherResourcesList(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()
	id := getString(args, "archive_id", "")
	p, err := providerForArchive(params, id)
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
	result := api.NewBuilderWithBudget(api.ResultLimit(params.Context))
	// Leave room for the header, which includes a count known only after the
	// walk. The scratch bounds both the names and optional YAML details.
	body := result.Scratch(len(kind) + 32)
	if body.Truncated() {
		result.WriteString(fmt.Sprintf("Found %s resource(s):\n\n", kind))
		result.WriteString("\n")
		return api.NewToolCallResult(result.String(), nil), nil
	}
	count := 0
	err = p.VisitResources(params.Context, gvk, namespace, mg.ListOptions{
		LabelSelector: labelSelector,
		FieldSelector: fieldSelector,
		Limit:         limit,
	}, func(item *unstructured.Unstructured) bool {
		if !body.Fits(0) {
			body.WriteString("\n")
			return false
		}
		count++
		ns, name := item.GetNamespace(), item.GetName()
		if ns != "" {
			body.WriteString(fmt.Sprintf("- %s/%s\n", ns, name))
		} else {
			body.WriteString(fmt.Sprintf("- %s\n", name))
		}
		return !body.Truncated()
	})
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	if count == 0 {
		return api.NewToolCallResult(fmt.Sprintf("No %s resources found", kind), nil), nil
	}
	if count <= 5 && !body.Truncated() {
		body.WriteString("\n---\n\n")
		if !body.Truncated() {
			err = p.VisitResources(params.Context, gvk, namespace, mg.ListOptions{
				LabelSelector: labelSelector, FieldSelector: fieldSelector, Limit: limit,
			}, func(item *unstructured.Unstructured) bool {
				if !body.Fits(0) {
					body.WriteString("\n")
					return false
				}
				yamlBytes, err := yaml.Marshal(item.Object)
				if err == nil {
					body.WriteString(string(yamlBytes) + "\n---\n\n")
				}
				return !body.Truncated()
			})
			if err != nil {
				return api.NewToolCallResult("", err), nil
			}
		}
	}
	result.WriteString(fmt.Sprintf("Found %d %s resource(s):\n\n", count, kind))
	text, cut := body.Prefix()
	result.WriteCut(text, cut)
	return api.NewToolCallResult(result.String(), nil), nil
}

func parseGVK(apiVersion, kind string) schema.GroupVersionKind {
	parts := strings.SplitN(apiVersion, "/", 2)
	if len(parts) == 2 {
		return schema.GroupVersionKind{Group: parts[0], Version: parts[1], Kind: kind}
	}
	return schema.GroupVersionKind{Group: "", Version: apiVersion, Kind: kind}
}
