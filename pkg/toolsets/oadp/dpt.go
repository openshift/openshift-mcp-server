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

func initDataProtectionTestTools() []api.ServerTool {
	return []api.ServerTool{
		initDataProtectionTestList(),
		initDataProtectionTestGet(),
		initDataProtectionTestCreate(),
		initDataProtectionTestDelete(),
	}
}

func initDataProtectionTestList() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_data_protection_test_list",
			Description: "List all DataProtectionTests for storage validation",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace containing DataProtectionTests (default: openshift-adp)",
					},
				},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: List Data Protection Tests",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: dataProtectionTestListHandler,
	}
}

func dataProtectionTestListHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	dpts, err := oadp.ListDataProtectionTests(params.Context, params.DynamicClient(), namespace, metav1.ListOptions{})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list data protection tests: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(dpts)), nil
}

func initDataProtectionTestGet() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_data_protection_test_get",
			Description: "Get detailed information about a DataProtectionTest including test results",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the DataProtectionTest (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the DataProtectionTest",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:        "OADP: Get Data Protection Test",
				ReadOnlyHint: ptr.To(true),
			},
		},
		Handler: dataProtectionTestGetHandler,
	}
}

func dataProtectionTestGetHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	dpt, err := oadp.GetDataProtectionTest(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get data protection test: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(dpt)), nil
}

func initDataProtectionTestCreate() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_data_protection_test_create",
			Description: "Create a DataProtectionTest to validate storage connectivity and performance",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace for the DataProtectionTest (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name for the DataProtectionTest",
					},
					"backupLocationName": {
						Type:        "string",
						Description: "Name of the BackupStorageLocation to test",
					},
					"uploadTestFileSize": {
						Type:        "string",
						Description: "Size of test file for upload speed test (e.g., '100MB')",
					},
					"uploadTestTimeout": {
						Type:        "string",
						Description: "Timeout for upload test (e.g., '60s')",
					},
					"skipTLSVerify": {
						Type:        "boolean",
						Description: "Skip TLS certificate verification",
					},
				},
				Required: []string{"name", "backupLocationName"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Create Data Protection Test",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(false),
				IdempotentHint:  ptr.To(false),
			},
		},
		Handler: dataProtectionTestCreateHandler,
	}
}

func dataProtectionTestCreateHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	bslName, ok := params.GetArguments()["backupLocationName"].(string)
	if !ok || bslName == "" {
		return api.NewToolCallResult("", fmt.Errorf("backupLocationName is required")), nil
	}

	spec := map[string]any{
		"backupLocationName": bslName,
	}

	if fileSize, ok := params.GetArguments()["uploadTestFileSize"].(string); ok && fileSize != "" {
		if timeout, ok := params.GetArguments()["uploadTestTimeout"].(string); ok && timeout != "" {
			spec["uploadSpeedTestConfig"] = map[string]any{
				"fileSize": fileSize,
				"timeout":  timeout,
			}
		} else {
			spec["uploadSpeedTestConfig"] = map[string]any{
				"fileSize": fileSize,
			}
		}
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

func initDataProtectionTestDelete() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_data_protection_test_delete",
			Description: "Delete a DataProtectionTest",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the DataProtectionTest (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the DataProtectionTest to delete",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Delete Data Protection Test",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(true),
				IdempotentHint:  ptr.To(true),
			},
		},
		Handler: dataProtectionTestDeleteHandler,
	}
}

func dataProtectionTestDeleteHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	err := oadp.DeleteDataProtectionTest(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to delete data protection test: %w", err)), nil
	}

	return api.NewToolCallResult(fmt.Sprintf("DataProtectionTest %s/%s deleted", namespace, name), nil), nil
}
