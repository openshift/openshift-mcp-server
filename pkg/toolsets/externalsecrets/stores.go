package externalsecrets

import (
	"fmt"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/containers/kubernetes-mcp-server/pkg/output"
)

const (
	externalSecretsAPIGroup = "external-secrets.io"
)

func initStoreTools() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name: "external_secrets_store_list",
				Description: `List SecretStores and/or ClusterSecretStores in the cluster.
SecretStore is a namespaced resource that specifies how to access a secret provider (AWS, GCP, Azure, Vault, etc.).
ClusterSecretStore is a cluster-scoped variant that can be referenced from any namespace.
Reference: https://external-secrets.io/latest/api/secretstore/`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"namespace": {
							Type:        "string",
							Description: "Namespace to list SecretStores from (optional, lists from all namespaces if not provided)",
						},
						"cluster_scoped": {
							Type:        "boolean",
							Description: "If true, list ClusterSecretStores instead of SecretStores (default: false)",
							Default:     api.ToRawMessage(false),
						},
					},
				},
				Annotations: api.ToolAnnotations{
					Title:           "External Secrets: List Stores",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: storeList,
		},
		{
			Tool: api.Tool{
				Name: "external_secrets_store_get",
				Description: `Get details of a SecretStore or ClusterSecretStore.
Returns the full specification and current status including validation state and capabilities.
Reference: https://external-secrets.io/latest/api/secretstore/`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"name": {
							Type:        "string",
							Description: "Name of the SecretStore or ClusterSecretStore",
						},
						"namespace": {
							Type:        "string",
							Description: "Namespace of the SecretStore (not needed for ClusterSecretStore)",
						},
						"cluster_scoped": {
							Type:        "boolean",
							Description: "If true, get a ClusterSecretStore instead of SecretStore (default: false)",
							Default:     api.ToRawMessage(false),
						},
					},
					Required: []string{"name"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "External Secrets: Get Store",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: storeGet,
		},
		{
			Tool: api.Tool{
				Name: "external_secrets_store_create",
				Description: `Create or update a SecretStore or ClusterSecretStore.
Supports various providers: AWS Secrets Manager, GCP Secret Manager, Azure Key Vault, HashiCorp Vault, 
Kubernetes Secrets, Bitwarden, 1Password, and many more.

Example SecretStore for AWS Secrets Manager:
  apiVersion: external-secrets.io/v1
  kind: SecretStore
  metadata:
    name: aws-secretsmanager
    namespace: my-namespace
  spec:
    provider:
      aws:
        service: SecretsManager
        region: us-east-1
        auth:
          secretRef:
            accessKeyIDSecretRef:
              name: aws-credentials
              key: access-key
            secretAccessKeySecretRef:
              name: aws-credentials
              key: secret-access-key

Reference: https://external-secrets.io/latest/provider/aws-secrets-manager/`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"store": {
							Type:        "string",
							Description: "YAML or JSON representation of the SecretStore or ClusterSecretStore resource",
						},
					},
					Required: []string{"store"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "External Secrets: Create/Update Store",
					ReadOnlyHint:    ptr.To(false),
					DestructiveHint: ptr.To(false),
					IdempotentHint:  ptr.To(true),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: storeCreate,
		},
		{
			Tool: api.Tool{
				Name: "external_secrets_store_delete",
				Description: `Delete a SecretStore or ClusterSecretStore.
WARNING: Deleting a store will cause ExternalSecrets referencing it to fail syncing.`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"name": {
							Type:        "string",
							Description: "Name of the SecretStore or ClusterSecretStore to delete",
						},
						"namespace": {
							Type:        "string",
							Description: "Namespace of the SecretStore (not needed for ClusterSecretStore)",
						},
						"cluster_scoped": {
							Type:        "boolean",
							Description: "If true, delete a ClusterSecretStore instead of SecretStore (default: false)",
							Default:     api.ToRawMessage(false),
						},
					},
					Required: []string{"name"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "External Secrets: Delete Store",
					ReadOnlyHint:    ptr.To(false),
					DestructiveHint: ptr.To(true),
					IdempotentHint:  ptr.To(true),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: storeDelete,
		},
		{
			Tool: api.Tool{
				Name: "external_secrets_store_validate",
				Description: `Check the validation status of SecretStores and/or ClusterSecretStores.
Returns a summary of store health including:
- Whether the store is valid and ready
- Capabilities (ReadOnly, ReadWrite)
- Any error conditions
Use this to quickly identify stores with configuration issues.`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"namespace": {
							Type:        "string",
							Description: "Namespace to check SecretStores in (optional, checks all namespaces if not provided)",
						},
						"include_cluster_stores": {
							Type:        "boolean",
							Description: "Also include ClusterSecretStores in the validation check (default: true)",
							Default:     api.ToRawMessage(true),
						},
					},
				},
				Annotations: api.ToolAnnotations{
					Title:           "External Secrets: Validate Stores",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: storeValidate,
		},
	}
}

func storeList(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := getStringArg(params, "namespace", "")
	clusterScoped := getBoolArg(params, "cluster_scoped", false)

	kind := "SecretStore"
	if clusterScoped {
		kind = "ClusterSecretStore"
		namespace = "" // ClusterSecretStore is cluster-scoped
	}

	gvk := &schema.GroupVersionKind{
		Group:   externalSecretsAPIGroup,
		Version: "v1",
		Kind:    kind,
	}

	list, err := params.ResourcesList(params, gvk, namespace, kubernetes.ResourceListOptions{AsTable: true})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list %s: %w", kind, err)), nil
	}

	result, _ := params.ListOutput.PrintObj(list)
	title := fmt.Sprintf("# %s List", kind)
	if namespace != "" {
		title += fmt.Sprintf(" (namespace: %s)", namespace)
	}

	return api.NewToolCallResult(title+"\n"+result, nil), nil
}

func storeGet(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	name := getStringArg(params, "name", "")
	if name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name argument is required")), nil
	}

	namespace := getStringArg(params, "namespace", "")
	clusterScoped := getBoolArg(params, "cluster_scoped", false)

	kind := "SecretStore"
	if clusterScoped {
		kind = "ClusterSecretStore"
		namespace = ""
	}

	gvk := &schema.GroupVersionKind{
		Group:   externalSecretsAPIGroup,
		Version: "v1",
		Kind:    kind,
	}

	store, err := params.ResourcesGet(params, gvk, namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get %s '%s': %w", kind, name, err)), nil
	}

	storeYaml, _ := output.MarshalYaml(store)
	return api.NewToolCallResult(
		fmt.Sprintf("# %s: %s\n```yaml\n%s```", kind, name, storeYaml),
		nil,
	), nil
}

func storeCreate(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	store, ok := params.GetArguments()["store"].(string)
	if !ok || store == "" {
		return api.NewToolCallResult("", fmt.Errorf("store argument is required")), nil
	}

	result, err := params.ResourcesCreateOrUpdate(params, store)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create/update store: %w", err)), nil
	}

	marshalledYaml, err := output.MarshalYaml(result)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal result: %w", err)), nil
	}

	return api.NewToolCallResult(
		"# Store created/updated successfully\n```yaml\n"+marshalledYaml+"```\n\n"+
			"Use 'external_secrets_store_validate' to verify the store is working correctly.",
		nil,
	), nil
}

func storeDelete(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	name := getStringArg(params, "name", "")
	if name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name argument is required")), nil
	}

	namespace := getStringArg(params, "namespace", "")
	clusterScoped := getBoolArg(params, "cluster_scoped", false)

	kind := "SecretStore"
	if clusterScoped {
		kind = "ClusterSecretStore"
		namespace = ""
	}

	gvk := &schema.GroupVersionKind{
		Group:   externalSecretsAPIGroup,
		Version: "v1",
		Kind:    kind,
	}

	err := params.ResourcesDelete(params, gvk, namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to delete %s '%s': %w", kind, name, err)), nil
	}

	return api.NewToolCallResult(
		fmt.Sprintf("# %s '%s' deleted successfully", kind, name),
		nil,
	), nil
}

func storeValidate(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := getStringArg(params, "namespace", "")
	includeClusterStores := getBoolArg(params, "include_cluster_stores", true)

	var results []string
	results = append(results, "# SecretStore Validation Report\n")

	// Check SecretStores
	secretStoreGVK := &schema.GroupVersionKind{
		Group:   externalSecretsAPIGroup,
		Version: "v1",
		Kind:    "SecretStore",
	}

	storeList, err := params.ResourcesList(params, secretStoreGVK, namespace, kubernetes.ResourceListOptions{AsTable: false})
	if err != nil {
		results = append(results, fmt.Sprintf("## SecretStores\nError listing: %v\n", err))
	} else {
		storeResults := extractStoreStatus(storeList, "SecretStore")
		results = append(results, storeResults)
	}

	// Check ClusterSecretStores
	if includeClusterStores {
		clusterStoreGVK := &schema.GroupVersionKind{
			Group:   externalSecretsAPIGroup,
			Version: "v1",
			Kind:    "ClusterSecretStore",
		}

		clusterStoreList, err := params.ResourcesList(params, clusterStoreGVK, "", kubernetes.ResourceListOptions{AsTable: false})
		if err != nil {
			results = append(results, fmt.Sprintf("## ClusterSecretStores\nError listing: %v\n", err))
		} else {
			clusterStoreResults := extractStoreStatus(clusterStoreList, "ClusterSecretStore")
			results = append(results, clusterStoreResults)
		}
	}

	return api.NewToolCallResult(strings.Join(results, "\n"), nil), nil
}

// extractStoreStatus processes a list of stores and returns a formatted status summary
func extractStoreStatus(list interface{}, kind string) string {
	var results []string
	results = append(results, fmt.Sprintf("## %ss", kind))

	// Type assertion for unstructured list
	listMap, ok := list.(map[string]interface{})
	if !ok {
		// Try unstructured.Unstructured
		type unstructuredLike interface {
			UnstructuredContent() map[string]interface{}
		}
		if u, ok := list.(unstructuredLike); ok {
			listMap = u.UnstructuredContent()
		}
	}

	if listMap == nil {
		return fmt.Sprintf("## %ss\nNo data available\n", kind)
	}

	items, ok := listMap["items"].([]interface{})
	if !ok || len(items) == 0 {
		return fmt.Sprintf("## %ss\nNo %ss found\n", kind, kind)
	}

	results = append(results, "| Name | Namespace | Status | Capabilities | Message |")
	results = append(results, "|------|-----------|--------|--------------|---------|")

	for _, item := range items {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		name := "unknown"
		namespace := "-"
		status := "Unknown"
		capabilities := "-"
		message := "-"

		if metadata, ok := itemMap["metadata"].(map[string]interface{}); ok {
			if n, ok := metadata["name"].(string); ok {
				name = n
			}
			if ns, ok := metadata["namespace"].(string); ok {
				namespace = ns
			}
		}

		if statusMap, ok := itemMap["status"].(map[string]interface{}); ok {
			if cap, ok := statusMap["capabilities"].(string); ok {
				capabilities = cap
			}
			if conditions, ok := statusMap["conditions"].([]interface{}); ok {
				for _, cond := range conditions {
					if condMap, ok := cond.(map[string]interface{}); ok {
						if condType, ok := condMap["type"].(string); ok && condType == "Ready" {
							if condStatus, ok := condMap["status"].(string); ok {
								if condStatus == "True" {
									status = "✅ Valid"
								} else {
									status = "❌ Invalid"
								}
							}
							if msg, ok := condMap["message"].(string); ok {
								message = msg
								if len(message) > 50 {
									message = message[:50] + "..."
								}
							}
						}
					}
				}
			}
		}

		results = append(results, fmt.Sprintf("| %s | %s | %s | %s | %s |", name, namespace, status, capabilities, message))
	}

	return strings.Join(results, "\n") + "\n"
}
