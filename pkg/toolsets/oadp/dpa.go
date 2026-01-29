package oadp

import (
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/oadp"
	"github.com/google/jsonschema-go/jsonschema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func initDPATools() []api.ServerTool {
	return []api.ServerTool{
		initDPAList(),
		initDPAGet(),
	}
}

func initDPAList() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_dpa_list",
			Description: "List all DataProtectionApplication instances (OADP operator configuration)",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace containing DPAs (default: openshift-adp)",
					},
				},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: List DataProtectionApplications",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: dpaListHandler,
	}
}

func dpaListHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	dpas, err := oadp.ListDataProtectionApplications(params.Context, params.DynamicClient(), namespace, metav1.ListOptions{})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list DataProtectionApplications: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(dpas)), nil
}

func initDPAGet() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_dpa_get",
			Description: "Get detailed information about a DataProtectionApplication including configuration and status conditions",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the DPA (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the DataProtectionApplication",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: Get DataProtectionApplication",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: dpaGetHandler,
	}
}

func dpaGetHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	dpa, err := oadp.GetDataProtectionApplication(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get DataProtectionApplication: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(dpa)), nil
}
