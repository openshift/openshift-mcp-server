package kiali

import (
	"fmt"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"k8s.io/utils/ptr"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	internalkiali "github.com/containers/kubernetes-mcp-server/pkg/kiali"
	internalk8s "github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
)

func initValidations() []api.ServerTool {
	ret := make([]api.ServerTool, 0)
	ret = append(ret, api.ServerTool{
		Tool: api.Tool{
			Name:        "validations_list",
			Description: "List all the validations in the current cluster from all namespaces",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Optional Namespace to retrieve the namespaced resource from (ignored in case of cluster scoped resources). If not provided, will get resource from configured namespace",
					},
				},
				Required: []string{},
			},
			Annotations: api.ToolAnnotations{
				Title:           "Validations: List",
				ReadOnlyHint:    ptr.To(true),
				DestructiveHint: ptr.To(false),
				IdempotentHint:  ptr.To(false),
				OpenWorldHint:   ptr.To(true),
			},
		}, Handler: validationsList,
	})
	return ret
}

func validationsList(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	// Extract the Authorization header from context
	authHeader, _ := params.Context.Value(internalk8s.OAuthAuthorizationHeader).(string)
	if strings.TrimSpace(authHeader) == "" {
		// Fall back to using the same token that the Kubernetes client is using
		if params.Kubernetes != nil {
			authHeader = params.Kubernetes.CurrentAuthorizationHeader()
		}
	}
	// Build a Kiali client from static config
	kialiClient := internalkiali.NewFromConfig(params.Kubernetes.StaticConfig())
	// Parse optional arguments
	namespace := ""
	if v, ok := params.GetArguments()["namespace"].(string); ok {
		namespace = v
	}
	allNamespaces := strings.TrimSpace(namespace) == ""
	content, err := kialiClient.ValidationsList(params.Context, authHeader, namespace, allNamespaces)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list validations: %v", err)), nil
	}
	return api.NewToolCallResult(content, nil), nil
}
