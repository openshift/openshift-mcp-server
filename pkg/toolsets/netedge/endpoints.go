package netedge

import (
	"encoding/json"
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/google/jsonschema-go/jsonschema"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func initEndpoints() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        "get_service_endpoints",
				Description: "Return Endpoints object for a Service to verify backend pod availability.",
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

	endpoints := &corev1.Endpoints{}
	err = cl.Get(params.Context, types.NamespacedName{Name: serviceName, Namespace: namespace}, endpoints)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get Endpoints for service %s/%s: %w", namespace, serviceName, err)), nil
	}

	data, err := json.MarshalIndent(endpoints, "", "  ")
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal endpoints: %w", err)), nil
	}

	return api.NewToolCallResult(string(data), nil), nil
}
