package netedge

import (
	"encoding/json"
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/google/jsonschema-go/jsonschema"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	cfg := params.RESTConfig()
	if cfg == nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get REST config")), nil
	}

	cl, err := newClientFunc(cfg, client.Options{Scheme: kubernetes.Scheme})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create controller-runtime client: %w", err)), nil
	}

	// Use EndpointSlices as Endpoints is deprecated
	endpointSlices := &discoveryv1.EndpointSliceList{}
	// EndpointSlices are linked to a service via the "kubernetes.io/service-name" label
	labelSelector := client.MatchingLabels{
		"kubernetes.io/service-name": serviceName,
	}
	err = cl.List(params.Context, endpointSlices, client.InNamespace(namespace), labelSelector)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list EndpointSlices for service %s/%s: %w", namespace, serviceName, err)), nil
	}

	if len(endpointSlices.Items) == 0 {
		return api.NewToolCallResult("", fmt.Errorf("no EndpointSlices found for service %s/%s", namespace, serviceName)), nil
	}

	data, err := json.MarshalIndent(endpointSlices.Items, "", "  ")
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal endpoint slices: %w", err)), nil
	}

	return api.NewToolCallResult(string(data), nil), nil
}
