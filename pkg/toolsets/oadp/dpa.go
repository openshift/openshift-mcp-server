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

// DPAAction represents the action to perform on DataProtectionApplications
type DPAAction string

const (
	DPAActionList   DPAAction = "list"
	DPAActionGet    DPAAction = "get"
	DPAActionCreate DPAAction = "create"
	DPAActionUpdate DPAAction = "update"
	DPAActionDelete DPAAction = "delete"
)

func initDPATools() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        "oadp_dpa",
				Description: "Manage OADP DataProtectionApplication resources: list, get, create, update, or delete",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"action": {
							Type:        "string",
							Enum:        []any{string(DPAActionList), string(DPAActionGet), string(DPAActionCreate), string(DPAActionUpdate), string(DPAActionDelete)},
							Description: "Action to perform: 'list', 'get', 'create', 'update', or 'delete'",
						},
						"namespace": {
							Type:        "string",
							Description: "Namespace containing DPAs (default: openshift-adp)",
						},
						"name": {
							Type:        "string",
							Description: "Name of the DPA (required for get, create, update, delete)",
						},
						"backupLocationProvider": {
							Type:        "string",
							Description: "Provider for backup storage e.g., aws, azure, gcp (for create)",
						},
						"backupLocationBucket": {
							Type:        "string",
							Description: "Bucket name for backup storage (for create)",
						},
						"backupLocationRegion": {
							Type:        "string",
							Description: "Region for backup storage (for create)",
						},
						"backupLocationCredentialName": {
							Type:        "string",
							Description: "Secret name containing backup storage credentials (for create)",
						},
						"snapshotLocationProvider": {
							Type:        "string",
							Description: "Provider for volume snapshots (for create)",
						},
						"snapshotLocationRegion": {
							Type:        "string",
							Description: "Region for volume snapshots (for create)",
						},
						"enableNodeAgent": {
							Type:        "boolean",
							Description: "Enable NodeAgent for file-system backups (for create/update)",
						},
					},
					Required: []string{"action"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "OADP: DataProtectionApplication",
					ReadOnlyHint:    ptr.To(false),
					DestructiveHint: ptr.To(false),
				},
			},
			Handler: dpaHandler,
		},
	}
}

func dpaHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	action, err := api.RequiredString(params, "action")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	namespace := api.OptionalString(params, "namespace", oadp.DefaultOADPNamespace)

	switch DPAAction(action) {
	case DPAActionList:
		return handleDPAList(params, namespace)
	case DPAActionGet:
		return handleDPAGet(params, namespace)
	case DPAActionCreate:
		return handleDPACreate(params, namespace)
	case DPAActionUpdate:
		return handleDPAUpdate(params, namespace)
	case DPAActionDelete:
		return handleDPADelete(params, namespace)
	default:
		return api.NewToolCallResult("", fmt.Errorf("invalid action '%s': must be one of 'list', 'get', 'create', 'update', 'delete'", action)), nil
	}
}

func handleDPAList(params api.ToolHandlerParams, namespace string) (*api.ToolCallResult, error) {
	dpas, err := oadp.ListDataProtectionApplications(params.Context, params.DynamicClient(), namespace, metav1.ListOptions{})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list DataProtectionApplications: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(dpas)), nil
}

func handleDPAGet(params api.ToolHandlerParams, namespace string) (*api.ToolCallResult, error) {
	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required for get action")), nil
	}

	dpa, err := oadp.GetDataProtectionApplication(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get DataProtectionApplication: %w", err)), nil
	}

	return api.NewToolCallResult(params.ListOutput.PrintObj(dpa)), nil
}

func handleDPACreate(params api.ToolHandlerParams, namespace string) (*api.ToolCallResult, error) {
	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required for create action")), nil
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

	// Configure NodeAgent
	if enableNodeAgent, ok := params.GetArguments()["enableNodeAgent"].(bool); ok && enableNodeAgent {
		spec["configuration"] = map[string]any{
			"nodeAgent": map[string]any{
				"enable": true,
			},
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

func handleDPAUpdate(params api.ToolHandlerParams, namespace string) (*api.ToolCallResult, error) {
	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required for update action")), nil
	}

	dpa, err := oadp.GetDataProtectionApplication(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get DataProtectionApplication: %w", err)), nil
	}

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

func handleDPADelete(params api.ToolHandlerParams, namespace string) (*api.ToolCallResult, error) {
	name, ok := params.GetArguments()["name"].(string)
	if !ok || name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name is required for delete action")), nil
	}

	err := oadp.DeleteDataProtectionApplication(params.Context, params.DynamicClient(), namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to delete DataProtectionApplication: %w", err)), nil
	}

	return api.NewToolCallResult(fmt.Sprintf("DataProtectionApplication %s/%s deleted", namespace, name), nil), nil
}
