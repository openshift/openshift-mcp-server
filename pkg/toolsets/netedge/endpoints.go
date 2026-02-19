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

func initEndpoints() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        "get_service_endpoints",
				Description: "Return EndpointSlice objects for a Service to verify backend pod availability.",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"namespace": {
							Type:        "string",
							Description: "Service namespace",
						},
						"service": {
							Type:        "string",
							Description: "Service name",
						},
					},
					Required: []string{"namespace", "service"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Get Service Endpoints",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: getServiceEndpoints,
		},
	}
}

func getServiceEndpoints(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace, err := api.RequiredString(params, "namespace")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	serviceName, err := api.RequiredString(params, "service")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	gvr := schema.GroupVersionResource{
		Group:    "discovery.k8s.io",
		Version:  "v1",
		Resource: "endpointslices",
	}

	// EndpointSlices are linked to a service via the "kubernetes.io/service-name" label
	labelSelector := "kubernetes.io/service-name=" + serviceName

	list, err := params.DynamicClient().Resource(gvr).Namespace(namespace).List(params.Context, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list EndpointSlices for service %s/%s: %w", namespace, serviceName, err)), nil
	}

	if len(list.Items) == 0 {
		return api.NewToolCallResult("", fmt.Errorf("no EndpointSlices found for service %s/%s", namespace, serviceName)), nil
	}

	// Dynamic client returns Unstructured items. We can marshal them directly.
	// We might want to extract just the item list or marshal the whole list.
	// The original code marshaled `endpointSlices.Items`.

	data, err := json.MarshalIndent(list.Items, "", "  ")
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal endpoint slices: %w", err)), nil
	}

	return api.NewToolCallResult(string(data), nil), nil
}
