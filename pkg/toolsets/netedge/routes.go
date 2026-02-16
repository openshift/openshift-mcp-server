package netedge

import (
	"encoding/json"
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/google/jsonschema-go/jsonschema"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func initRoutes() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        "inspect_route",
				Description: "Inspect an OpenShift Route to view its configuration, status, and related services.",
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

	cfg := params.RESTConfig()
	if cfg == nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get REST config")), nil
	}

	cl, err := newClientFunc(cfg, client.Options{Scheme: kubernetes.Scheme})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create controller-runtime client: %w", err)), nil
	}

	// Use Unstructured for Route
	route := &unstructured.Unstructured{}
	route.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "route.openshift.io",
		Version: "v1",
		Kind:    "Route",
	})

	err = cl.Get(params.Context, client.ObjectKey{Namespace: namespace, Name: routeName}, route)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get route %s/%s: %w", namespace, routeName, err)), nil
	}

	data, err := json.MarshalIndent(route.Object, "", "  ")
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal route: %w", err)), nil
	}

	return api.NewToolCallResult(string(data), nil), nil
}
