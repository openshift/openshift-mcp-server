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

func initDPATools() []api.ServerTool {
	return []api.ServerTool{
		initDPAList(),
		initDPAGet(),
		initDPACreate(),
		initDPAUpdate(),
		initDPADelete(),
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

func initDPACreate() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_dpa_create",
			Description: "Create a DataProtectionApplication to configure OADP operator",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace for the DPA (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name for the DataProtectionApplication",
					},
					"backupLocationProvider": {
						Type:        "string",
						Description: "Provider for backup storage (e.g., aws, azure, gcp)",
					},
					"backupLocationBucket": {
						Type:        "string",
						Description: "Bucket name for backup storage",
					},
					"backupLocationRegion": {
						Type:        "string",
						Description: "Region for backup storage",
					},
					"backupLocationCredentialName": {
						Type:        "string",
						Description: "Name of the secret containing backup storage credentials",
					},
					"snapshotLocationProvider": {
						Type:        "string",
						Description: "Provider for volume snapshots (e.g., aws, azure, gcp)",
					},
					"snapshotLocationRegion": {
						Type:        "string",
						Description: "Region for volume snapshots",
					},
					"enableRestic": {
						Type:        "boolean",
						Description: "Enable Restic for file-system backups (deprecated, use nodeAgent)",
					},
					"enableNodeAgent": {
						Type:        "boolean",
						Description: "Enable NodeAgent for file-system backups",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Create DataProtectionApplication",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(false),
				IdempotentHint:  ptr.To(false),
			},
		},
		Handler: dpaCreateHandler,
	}
}

func dpaCreateHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	spec := map[string]any{}

	// Build backup locations
	if provider, ok := params.GetArguments()["backupLocationProvider"].(string); ok && provider != "" {
		bsl := map[string]any{
			"velero": map[string]any{
				"provider": provider,
			},
		}

		velero := bsl["velero"].(map[string]any)

		if bucket, ok := params.GetArguments()["backupLocationBucket"].(string); ok && bucket != "" {
			velero["objectStorage"] = map[string]any{
				"bucket": bucket,
			}
		}

		if region, ok := params.GetArguments()["backupLocationRegion"].(string); ok && region != "" {
			velero["config"] = map[string]any{
				"region": region,
			}
		}

		if credName, ok := params.GetArguments()["backupLocationCredentialName"].(string); ok && credName != "" {
			velero["credential"] = map[string]any{
				"name": credName,
				"key":  "cloud",
			}
		}

		spec["backupLocations"] = []any{bsl}
	}

	// Build snapshot locations
	if provider, ok := params.GetArguments()["snapshotLocationProvider"].(string); ok && provider != "" {
		vsl := map[string]any{
			"velero": map[string]any{
				"provider": provider,
			},
		}

		velero := vsl["velero"].(map[string]any)

		if region, ok := params.GetArguments()["snapshotLocationRegion"].(string); ok && region != "" {
			velero["config"] = map[string]any{
				"region": region,
			}
		}

		spec["snapshotLocations"] = []any{vsl}
	}

	// Configure NodeAgent/Restic
	if enableRestic, ok := params.GetArguments()["enableRestic"].(bool); ok && enableRestic {
		spec["configuration"] = map[string]any{
			"restic": map[string]any{
				"enable": true,
			},
		}
	}

	if enableNodeAgent, ok := params.GetArguments()["enableNodeAgent"].(bool); ok && enableNodeAgent {
		if spec["configuration"] == nil {
			spec["configuration"] = map[string]any{}
		}
		spec["configuration"].(map[string]any)["nodeAgent"] = map[string]any{
			"enable": true,
		}
	}

	dpa := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": oadp.OADPGroup + "/" + oadp.OADPVersion,
			"kind":       "DataProtectionApplication",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": spec,
		},
	}

	created, err := oadp.CreateDataProtectionApplication(params.Context, params.DynamicClient(), dpa)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create DataProtectionApplication: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(created)), nil
}

func initDPAUpdate() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_dpa_update",
			Description: "Update a DataProtectionApplication configuration",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the DPA (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the DataProtectionApplication to update",
					},
					"enableNodeAgent": {
						Type:        "boolean",
						Description: "Enable or disable NodeAgent",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Update DataProtectionApplication",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(false),
				IdempotentHint:  ptr.To(true),
			},
		},
		Handler: dpaUpdateHandler,
	}
}

func dpaUpdateHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	// Get the existing DPA
	dpa, err := oadp.GetDataProtectionApplication(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get DataProtectionApplication: %w", err)), nil
	}

	// Apply updates
	if enableNodeAgent, ok := params.GetArguments()["enableNodeAgent"].(bool); ok {
		if err := unstructured.SetNestedField(dpa.Object, enableNodeAgent, "spec", "configuration", "nodeAgent", "enable"); err != nil {
			return api.NewToolCallResult("", fmt.Errorf("failed to set nodeAgent enable field: %w", err)), nil
		}
	}

	updated, err := oadp.UpdateDataProtectionApplication(params.Context, params.DynamicClient(), dpa)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to update DataProtectionApplication: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(updated)), nil
}

func initDPADelete() api.ServerTool {
	return api.ServerTool{
		Tool: api.Tool{
			Name:        "oadp_dpa_delete",
			Description: "Delete a DataProtectionApplication. Warning: This will remove the OADP operator configuration.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"namespace": {
						Type:        "string",
						Description: "Namespace of the DPA (default: openshift-adp)",
					},
					"name": {
						Type:        "string",
						Description: "Name of the DataProtectionApplication to delete",
					},
				},
				Required: []string{"name"},
			},
			Annotations: api.ToolAnnotations{
				Title:           "OADP: Delete DataProtectionApplication",
				ReadOnlyHint:    ptr.To(false),
				DestructiveHint: ptr.To(true),
				IdempotentHint:  ptr.To(true),
			},
		},
		Handler: dpaDeleteHandler,
	}
}

func dpaDeleteHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := oadp.DefaultOADPNamespace
	if v, ok := params.GetArguments()["namespace"].(string); ok && v != "" {
		namespace = v
	}

	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required")), nil
	}

	err := oadp.DeleteDataProtectionApplication(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to delete DataProtectionApplication: %w", err)), nil
	}

	return api.NewToolCallResult(fmt.Sprintf("DataProtectionApplication %s/%s deleted", namespace, name), nil), nil
}
