package netedge

import (
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/google/jsonschema-go/jsonschema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"
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

	// Deep-copy the route so we can redact sensitive TLS fields without
	// mutating the original object returned by the API server.
	sanitizedRoute := route.DeepCopy()
	tlsSensitiveFields := []string{"key", "certificate", "caCertificate"}
	for _, field := range tlsSensitiveFields {
		if _, found, _ := unstructured.NestedString(sanitizedRoute.Object, "spec", "tls", field); found {
			if err := unstructured.SetNestedField(sanitizedRoute.Object, "<redacted>", "spec", "tls", field); err != nil {
				return api.NewToolCallResult("", fmt.Errorf("failed to redact tls.%s: %w", field, err)), nil
			}
		}
	}

	keyFields := map[string]interface{}{
		"Name":      sanitizedRoute.GetName(),
		"Namespace": sanitizedRoute.GetNamespace(),
	}

	if host, found, err := unstructured.NestedString(sanitizedRoute.Object, "spec", "host"); found && err == nil {
		keyFields["Host"] = host
	}

	if tls, found, err := unstructured.NestedMap(sanitizedRoute.Object, "spec", "tls"); found && err == nil {
		keyFields["TLS"] = tls
	}

	if to, found, err := unstructured.NestedMap(sanitizedRoute.Object, "spec", "to"); found && err == nil {
		keyFields["To"] = to
	}

	if port, found, err := unstructured.NestedMap(sanitizedRoute.Object, "spec", "port"); found && err == nil {
		keyFields["Port"] = port
	}

	if ingress, found, err := unstructured.NestedSlice(sanitizedRoute.Object, "status", "ingress"); found && err == nil {
		keyFields["IngressStatus"] = ingress
	}

	resultObj := map[string]interface{}{
		"KeyFields": keyFields,
		"RawRoute":  sanitizedRoute.Object,
	}

	data, err := yaml.Marshal(resultObj)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal route as yaml: %w", err)), nil
	}

	return api.NewToolCallResult(string(data), nil), nil
}
