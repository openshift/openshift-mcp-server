package kiali

import (
	"fmt"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"k8s.io/utils/ptr"

	"github.com/kiali/kiali-mcp-server/pkg/api"
	internalkiali "github.com/kiali/kiali-mcp-server/pkg/kiali"
	internalk8s "github.com/kiali/kiali-mcp-server/pkg/kubernetes"
)

func initNamespaces() []api.ServerTool {
	ret := make([]api.ServerTool, 0)
	ret = append(ret, api.ServerTool{
		Tool: api.Tool{
			Name:        "namespaces",
			Description: "Get all namespaces in the mesh that the user has access to",
			InputSchema: &jsonschema.Schema{
				Type: "object",
			},
			Annotations: api.ToolAnnotations{
				Title:           "Namespaces: List",
				ReadOnlyHint:    ptr.To(true),
				DestructiveHint: ptr.To(false),
				IdempotentHint:  ptr.To(true),
				OpenWorldHint:   ptr.To(true),
			},
		}, Handler: namespacesHandler,
	})
	return ret
}

func namespacesHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
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

	content, err := kialiClient.MeshNamespaces(params.Context, authHeader)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list namespaces: %v", err)), nil
	}
	return api.NewToolCallResult(content, nil), nil
}
