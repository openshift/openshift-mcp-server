package oadp

import (
	"fmt"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/oadp"
	"github.com/google/jsonschema-go/jsonschema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/ptr"
)

// DPTAction represents the action to perform on DataProtectionTest resources
type DPTAction string

const (
	DPTActionList   DPTAction = "list"
	DPTActionGet    DPTAction = "get"
	DPTActionCreate DPTAction = "create"
	DPTActionDelete DPTAction = "delete"
)

func initDataProtectionTestTools() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        "oadp_data_protection_test",
				Description: "Manage OADP DataProtectionTest resources for validating storage connectivity: list, get, create, or delete",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"action": {
							Type:        "string",
							Enum:        []any{string(DPTActionList), string(DPTActionGet), string(DPTActionCreate), string(DPTActionDelete)},
							Description: "Action to perform: 'list', 'get', 'create', or 'delete'",
						},
						"namespace": {
							Type:        "string",
							Description: "Namespace containing resources (default: openshift-adp)",
						},
						"name": {
							Type:        "string",
							Description: "Name of the test (required for get, create, delete)",
						},
						"backupLocationName": {
							Type:        "string",
							Description: "Name of the BackupStorageLocation to test (for create)",
						},
						"uploadTestFileSize": {
							Type:        "string",
							Description: "Size of test file for upload speed test e.g., '100MB' (for create)",
						},
						"uploadTestTimeout": {
							Type:        "string",
							Description: "Timeout for upload test e.g., '60s' (for create)",
						},
						"skipTLSVerify": {
							Type:        "boolean",
							Description: "Skip TLS certificate verification (for create)",
						},
					},
					Required: []string{"action"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "OADP: Data Protection Test",
					ReadOnlyHint:    ptr.To(false),
					DestructiveHint: ptr.To(false),
				},
			},
			Handler: dptHandler,
		},
	}
}

func dptHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	action, err := api.RequiredString(params, "action")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	switch DPTAction(action) {
	case DPTActionList:
		dpts, err := oadp.ListDataProtectionTests(params.Context, params.DynamicClient(), namespace, metav1.ListOptions{})
		if err != nil {
			return api.NewToolCallResult("", fmt.Errorf("failed to list data protection tests: %w", err)), nil
		}
		return api.NewToolCallResult(params.ListOutput.PrintObj(dpts)), nil

	case DPTActionGet:
		name, ok := params.GetArguments()["name"].(string)
		if !ok || name == "" {
			return api.NewToolCallResult("", fmt.Errorf("name is required for get action")), nil
		}
		dpt, err := oadp.GetDataProtectionTest(params.Context, params.DynamicClient(), namespace, name)
		if err != nil {
			return api.NewToolCallResult("", fmt.Errorf("failed to get data protection test: %w", err)), nil
		}
		return api.NewToolCallResult(params.ListOutput.PrintObj(dpt)), nil

	case DPTActionCreate:
		return handleDPTCreate(params, namespace)

	case DPTActionDelete:
		name, ok := params.GetArguments()["name"].(string)
		if !ok || name == "" {
			return api.NewToolCallResult("", fmt.Errorf("name is required for delete action")), nil
		}
		err := oadp.DeleteDataProtectionTest(params.Context, params.DynamicClient(), namespace, name)
		if err != nil {
			return api.NewToolCallResult("", fmt.Errorf("failed to delete data protection test: %w", err)), nil
		}
		return api.NewToolCallResult(fmt.Sprintf("DataProtectionTest %s/%s deleted", namespace, name), nil), nil

	default:
		return api.NewToolCallResult("", fmt.Errorf("invalid action '%s': must be one of 'list', 'get', 'create', 'delete'", action)), nil
	}
}

func handleDPTCreate(params api.ToolHandlerParams, namespace string) (*api.ToolCallResult, error) {
	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required for create action")), nil
	}

	bslName, ok := params.GetArguments()["backupLocationName"].(string)
	if !ok || bslName == "" {
		return api.NewToolCallResult("", fmt.Errorf("backupLocationName is required for create action")), nil
	}

	spec := map[string]any{
		"backupLocationName": bslName,
	}

	if fileSize, ok := params.GetArguments()["uploadTestFileSize"].(string); ok && fileSize != "" {
		uploadConfig := map[string]any{
			"fileSize": fileSize,
		}
		if timeout, ok := params.GetArguments()["uploadTestTimeout"].(string); ok && timeout != "" {
			uploadConfig["timeout"] = timeout
		}
		spec["uploadSpeedTestConfig"] = uploadConfig
	}

	if skipTLS, ok := params.GetArguments()["skipTLSVerify"].(bool); ok {
		spec["skipTLSVerify"] = skipTLS
	}

	dpt := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": oadp.OADPGroup + "/" + oadp.OADPVersion,
			"kind":       "DataProtectionTest",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": spec,
		},
	}

	created, err := oadp.CreateDataProtectionTest(params.Context, params.DynamicClient(), dpt)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create data protection test: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(created)), nil
}
