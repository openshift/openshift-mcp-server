package certmanager

import (
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	"k8s.io/utils/ptr"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/certmanager"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/containers/kubernetes-mcp-server/pkg/output"
)

func initIssuers() []api.ServerTool {
	return []api.ServerTool{
		// issuers_list
		{
			Tool: api.Tool{
				Name:        "certmanager_issuers_list",
				Description: "List all cert-manager Issuers in the cluster. Issuers are namespaced resources that represent a certificate authority.",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"namespace": {
							Type:        "string",
							Description: "Optional namespace to list Issuers from. If not provided, lists from all namespaces",
						},
					},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Cert-Manager: List Issuers",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: issuersList,
		},
		// issuer_get
		{
			Tool: api.Tool{
				Name:        "certmanager_issuer_get",
				Description: "Get a specific cert-manager Issuer with its status and configuration.",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"namespace": {
							Type:        "string",
							Description: "Namespace of the Issuer",
						},
						"name": {
							Type:        "string",
							Description: "Name of the Issuer",
						},
					},
					Required: []string{"name", "namespace"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Cert-Manager: Get Issuer",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: issuerGet,
		},
		// clusterissuers_list
		{
			Tool: api.Tool{
				Name:        "certmanager_clusterissuers_list",
				Description: "List all cert-manager ClusterIssuers in the cluster. ClusterIssuers are cluster-scoped resources that can issue certificates in any namespace.",
				InputSchema: &jsonschema.Schema{
					Type: "object",
				},
				Annotations: api.ToolAnnotations{
					Title:           "Cert-Manager: List ClusterIssuers",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: clusterIssuersList,
		},
		// clusterissuer_get
		{
			Tool: api.Tool{
				Name:        "certmanager_clusterissuer_get",
				Description: "Get a specific cert-manager ClusterIssuer with its status and configuration.",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"name": {
							Type:        "string",
							Description: "Name of the ClusterIssuer",
						},
					},
					Required: []string{"name"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Cert-Manager: Get ClusterIssuer",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: clusterIssuerGet,
		},
	}
}

func issuersList(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := ""
	if ns := params.GetArguments()["namespace"]; ns != nil {
		namespace = ns.(string)
	}

	listOptions := kubernetes.ResourceListOptions{
		AsTable: params.ListOutput.AsTable(),
	}

	ret, err := params.ResourcesList(params.Context, &certmanager.IssuerGVK, namespace, listOptions)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list issuers: %v", err)), nil
	}
	return api.NewToolCallResult(params.ListOutput.PrintObj(ret)), nil
}

func issuerGet(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := params.GetArguments()["namespace"].(string)
	name := params.GetArguments()["name"].(string)

	ret, err := params.ResourcesGet(params.Context, &certmanager.IssuerGVK, namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get issuer %s/%s: %v", namespace, name, err)), nil
	}
	return api.NewToolCallResult(output.MarshalYaml(ret)), nil
}

func clusterIssuersList(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	listOptions := kubernetes.ResourceListOptions{
		AsTable: params.ListOutput.AsTable(),
	}

	ret, err := params.ResourcesList(params.Context, &certmanager.ClusterIssuerGVK, "", listOptions)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list cluster issuers: %v", err)), nil
	}
	return api.NewToolCallResult(params.ListOutput.PrintObj(ret)), nil
}

func clusterIssuerGet(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	name := params.GetArguments()["name"].(string)

	ret, err := params.ResourcesGet(params.Context, &certmanager.ClusterIssuerGVK, "", name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get cluster issuer %s: %v", name, err)), nil
	}
	return api.NewToolCallResult(output.MarshalYaml(ret)), nil
}
