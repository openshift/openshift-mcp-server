package netedge

import (
	"encoding/json"
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/google/jsonschema-go/jsonschema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
)

func initRoutes() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        "inspect_route",
				Description: "Inspect an OpenShift Route to view its full configuration and status.",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"namespace": {
							Type:        "string",
							Description: "Route namespace",
						},
						"route": {
							Type:        "string",
							Description: "Route name",
						},
					},
					Required: []string{"namespace", "route"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Inspect Route",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: inspectRoute,
		},
	}
}

func inspectRoute(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace, err := api.RequiredString(params, "namespace")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	routeName, err := api.RequiredString(params, "route")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	gvr := schema.GroupVersionResource{
		Group:    "route.openshift.io",
		Version:  "v1",
		Resource: "routes",
	}

	route, err := params.DynamicClient().Resource(gvr).Namespace(namespace).Get(params.Context, routeName, metav1.GetOptions{})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get route %s/%s: %w", namespace, routeName, err)), nil
	}

	data, err := json.MarshalIndent(route.Object, "", "  ")
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal route: %w", err)), nil
	}

	return api.NewToolCallResult(string(data), nil), nil
}
