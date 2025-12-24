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

func initSecretTools() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name: "external_secrets_list",
				Description: `List ExternalSecrets and/or ClusterExternalSecrets in the cluster.
ExternalSecret is a namespaced resource that defines what secret data to fetch from a SecretStore.
ClusterExternalSecret can create ExternalSecrets across multiple namespaces.
Reference: https://external-secrets.io/latest/api/externalsecret/`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"namespace": {
							Type:        "string",
							Description: "Namespace to list ExternalSecrets from (optional, lists from all namespaces if not provided)",
						},
						"cluster_scoped": {
							Type:        "boolean",
							Description: "If true, list ClusterExternalSecrets instead of ExternalSecrets (default: false)",
							Default:     api.ToRawMessage(false),
						},
					},
				},
				Annotations: api.ToolAnnotations{
					Title:           "External Secrets: List Secrets",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: secretList,
		},
		{
			Tool: api.Tool{
				Name: "external_secrets_get",
				Description: `Get details of an ExternalSecret or ClusterExternalSecret.
Returns the full specification, sync status, and any error conditions.
Reference: https://external-secrets.io/latest/api/externalsecret/`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"name": {
							Type:        "string",
							Description: "Name of the ExternalSecret or ClusterExternalSecret",
						},
						"namespace": {
							Type:        "string",
							Description: "Namespace of the ExternalSecret (not needed for ClusterExternalSecret)",
						},
						"cluster_scoped": {
							Type:        "boolean",
							Description: "If true, get a ClusterExternalSecret instead of ExternalSecret (default: false)",
							Default:     api.ToRawMessage(false),
						},
					},
					Required: []string{"name"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "External Secrets: Get Secret",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: secretGet,
		},
		{
			Tool: api.Tool{
				Name: "external_secrets_create",
				Description: `Create or update an ExternalSecret or ClusterExternalSecret.
ExternalSecret defines how to fetch secret data from a provider and create a Kubernetes Secret.

Example ExternalSecret:
  apiVersion: external-secrets.io/v1
  kind: ExternalSecret
  metadata:
    name: my-secret
    namespace: my-namespace
  spec:
    refreshInterval: 1h
    secretStoreRef:
      name: aws-secretsmanager
      kind: SecretStore
    target:
      name: my-k8s-secret
      creationPolicy: Owner
    data:
    - secretKey: password
      remoteRef:
        key: my-aws-secret
        property: password

Reference: https://external-secrets.io/latest/api/externalsecret/`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"secret": {
							Type:        "string",
							Description: "YAML or JSON representation of the ExternalSecret or ClusterExternalSecret resource",
						},
					},
					Required: []string{"secret"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "External Secrets: Create/Update Secret",
					ReadOnlyHint:    ptr.To(false),
					DestructiveHint: ptr.To(false),
					IdempotentHint:  ptr.To(true),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: secretCreate,
		},
		{
			Tool: api.Tool{
				Name: "external_secrets_delete",
				Description: `Delete an ExternalSecret or ClusterExternalSecret.
Note: By default, the associated Kubernetes Secret will also be deleted (depending on creationPolicy).`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"name": {
							Type:        "string",
							Description: "Name of the ExternalSecret or ClusterExternalSecret to delete",
						},
						"namespace": {
							Type:        "string",
							Description: "Namespace of the ExternalSecret (not needed for ClusterExternalSecret)",
						},
						"cluster_scoped": {
							Type:        "boolean",
							Description: "If true, delete a ClusterExternalSecret instead of ExternalSecret (default: false)",
							Default:     api.ToRawMessage(false),
						},
					},
					Required: []string{"name"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "External Secrets: Delete Secret",
					ReadOnlyHint:    ptr.To(false),
					DestructiveHint: ptr.To(true),
					IdempotentHint:  ptr.To(true),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: secretDelete,
		},
		{
			Tool: api.Tool{
				Name: "external_secrets_sync_status",
				Description: `Check the synchronization status of ExternalSecrets.
Returns a summary of sync health including:
- Whether secrets are synced successfully
- Last sync time and refresh interval
- Any sync errors or issues
Use this to quickly identify ExternalSecrets with sync problems.`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"namespace": {
							Type:        "string",
							Description: "Namespace to check ExternalSecrets in (optional, checks all namespaces if not provided)",
						},
						"name": {
							Type:        "string",
							Description: "Specific ExternalSecret name to check (optional, checks all if not provided)",
						},
					},
				},
				Annotations: api.ToolAnnotations{
					Title:           "External Secrets: Sync Status",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: secretSyncStatus,
		},
		{
			Tool: api.Tool{
				Name: "external_secrets_refresh",
				Description: `Trigger a refresh of an ExternalSecret to immediately sync from the provider.
This adds an annotation to force the controller to re-sync the secret data.
Useful when you've updated the secret in the provider and want immediate sync.`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"name": {
							Type:        "string",
							Description: "Name of the ExternalSecret to refresh",
						},
						"namespace": {
							Type:        "string",
							Description: "Namespace of the ExternalSecret",
						},
					},
					Required: []string{"name"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "External Secrets: Refresh",
					ReadOnlyHint:    ptr.To(false),
					DestructiveHint: ptr.To(false),
					IdempotentHint:  ptr.To(true),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: secretRefresh,
		},
	}
}

func secretList(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := getStringArg(params, "namespace", "")
	clusterScoped := getBoolArg(params, "cluster_scoped", false)

	kind := "ExternalSecret"
	if clusterScoped {
		kind = "ClusterExternalSecret"
		namespace = ""
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

func secretGet(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	name := getStringArg(params, "name", "")
	if name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name argument is required")), nil
	}

	namespace := getStringArg(params, "namespace", "")
	clusterScoped := getBoolArg(params, "cluster_scoped", false)

	kind := "ExternalSecret"
	if clusterScoped {
		kind = "ClusterExternalSecret"
		namespace = ""
	}

	gvk := &schema.GroupVersionKind{
		Group:   externalSecretsAPIGroup,
		Version: "v1",
		Kind:    kind,
	}

	secret, err := params.ResourcesGet(params, gvk, namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get %s '%s': %w", kind, name, err)), nil
	}

	secretYaml, _ := output.MarshalYaml(secret)
	return api.NewToolCallResult(
		fmt.Sprintf("# %s: %s\n```yaml\n%s```", kind, name, secretYaml),
		nil,
	), nil
}

func secretCreate(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	secret, ok := params.GetArguments()["secret"].(string)
	if !ok || secret == "" {
		return api.NewToolCallResult("", fmt.Errorf("secret argument is required")), nil
	}

	result, err := params.ResourcesCreateOrUpdate(params, secret)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to create/update ExternalSecret: %w", err)), nil
	}

	marshalledYaml, err := output.MarshalYaml(result)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal result: %w", err)), nil
	}

	return api.NewToolCallResult(
		"# ExternalSecret created/updated successfully\n```yaml\n"+marshalledYaml+"```\n\n"+
			"Use 'external_secrets_sync_status' to verify the secret is syncing correctly.",
		nil,
	), nil
}

func secretDelete(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	name := getStringArg(params, "name", "")
	if name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name argument is required")), nil
	}

	namespace := getStringArg(params, "namespace", "")
	clusterScoped := getBoolArg(params, "cluster_scoped", false)

	kind := "ExternalSecret"
	if clusterScoped {
		kind = "ClusterExternalSecret"
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
		fmt.Sprintf("# %s '%s' deleted successfully\n\nNote: The associated Kubernetes Secret may also be deleted depending on the creationPolicy.", kind, name),
		nil,
	), nil
}

func secretSyncStatus(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := getStringArg(params, "namespace", "")
	name := getStringArg(params, "name", "")

	var results []string
	results = append(results, "# ExternalSecret Sync Status Report\n")

	gvk := &schema.GroupVersionKind{
		Group:   externalSecretsAPIGroup,
		Version: "v1",
		Kind:    "ExternalSecret",
	}

	if name != "" {
		// Get specific ExternalSecret
		secret, err := params.ResourcesGet(params, gvk, namespace, name)
		if err != nil {
			return api.NewToolCallResult("", fmt.Errorf("failed to get ExternalSecret '%s': %w", name, err)), nil
		}
		results = append(results, extractSecretSyncStatus(secret, true))
	} else {
		// List all ExternalSecrets
		list, err := params.ResourcesList(params, gvk, namespace, kubernetes.ResourceListOptions{AsTable: false})
		if err != nil {
			return api.NewToolCallResult("", fmt.Errorf("failed to list ExternalSecrets: %w", err)), nil
		}
		results = append(results, extractSecretListSyncStatus(list))
	}

	return api.NewToolCallResult(strings.Join(results, "\n"), nil), nil
}

func secretRefresh(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	name := getStringArg(params, "name", "")
	if name == "" {
		return api.NewToolCallResult("", fmt.Errorf("name argument is required")), nil
	}

	namespace := getStringArg(params, "namespace", "")

	gvk := &schema.GroupVersionKind{
		Group:   externalSecretsAPIGroup,
		Version: "v1",
		Kind:    "ExternalSecret",
	}

	// Get current ExternalSecret
	secret, err := params.ResourcesGet(params, gvk, namespace, name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get ExternalSecret '%s': %w", name, err)), nil
	}

	// Add/update the force-sync annotation
	secretMap := secret.UnstructuredContent()
	metadata, ok := secretMap["metadata"].(map[string]interface{})
	if !ok {
		metadata = make(map[string]interface{})
		secretMap["metadata"] = metadata
	}

	annotations, ok := metadata["annotations"].(map[string]interface{})
	if !ok {
		annotations = make(map[string]interface{})
		metadata["annotations"] = annotations
	}

	// Use the reconcile annotation to trigger a sync
	annotations["force.external-secrets.io/sync"] = fmt.Sprintf("%d", getCurrentTimestamp())
	secret.SetUnstructuredContent(secretMap)

	// Apply the updated resource
	secretYaml, _ := output.MarshalYaml(secret)
	result, err := params.ResourcesCreateOrUpdate(params, secretYaml)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to trigger refresh for ExternalSecret '%s': %w", name, err)), nil
	}

	marshalledYaml, err := output.MarshalYaml(result)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal result: %w", err)), nil
	}

	return api.NewToolCallResult(
		fmt.Sprintf("# ExternalSecret '%s' refresh triggered\n\n", name)+
			"The controller will now re-sync the secret from the provider.\n"+
			"Use 'external_secrets_sync_status' to monitor the sync progress.\n\n"+
			"```yaml\n"+marshalledYaml+"```",
		nil,
	), nil
}

// extractSecretSyncStatus extracts sync status from a single ExternalSecret
func extractSecretSyncStatus(secret interface{}, detailed bool) string {
	var results []string

	secretMap, ok := secret.(map[string]interface{})
	if !ok {
		// Try unstructured.Unstructured
		type unstructuredLike interface {
			UnstructuredContent() map[string]interface{}
		}
		if u, ok := secret.(unstructuredLike); ok {
			secretMap = u.UnstructuredContent()
		}
	}

	if secretMap == nil {
		return "No data available"
	}

	name := "unknown"
	namespace := "-"
	if metadata, ok := secretMap["metadata"].(map[string]interface{}); ok {
		if n, ok := metadata["name"].(string); ok {
			name = n
		}
		if ns, ok := metadata["namespace"].(string); ok {
			namespace = ns
		}
	}

	results = append(results, fmt.Sprintf("## %s (namespace: %s)", name, namespace))

	if statusMap, ok := secretMap["status"].(map[string]interface{}); ok {
		// Sync status
		if syncStatus, ok := statusMap["syncedResourceVersion"].(string); ok {
			results = append(results, fmt.Sprintf("- **Synced Resource Version**: %s", syncStatus))
		}

		// Refresh time
		if refreshTime, ok := statusMap["refreshTime"].(string); ok {
			results = append(results, fmt.Sprintf("- **Last Refresh**: %s", refreshTime))
		}

		// Binding
		if binding, ok := statusMap["binding"].(map[string]interface{}); ok {
			if bindingName, ok := binding["name"].(string); ok {
				results = append(results, fmt.Sprintf("- **Target Secret**: %s", bindingName))
			}
		}

		// Conditions
		if conditions, ok := statusMap["conditions"].([]interface{}); ok {
			results = append(results, "\n### Conditions")
			for _, cond := range conditions {
				if condMap, ok := cond.(map[string]interface{}); ok {
					condType := condMap["type"]
					condStatus := condMap["status"]
					reason := condMap["reason"]
					message := condMap["message"]
					lastTransition := condMap["lastTransitionTime"]

					statusIcon := "❓"
					switch condStatus {
					case "True":
						statusIcon = "✅"
					case "False":
						statusIcon = "❌"
					}

					results = append(results, fmt.Sprintf("- %s **%v** (%v)", statusIcon, condType, reason))
					if message != nil && message != "" {
						results = append(results, fmt.Sprintf("  - Message: %v", message))
					}
					if detailed && lastTransition != nil {
						results = append(results, fmt.Sprintf("  - Last Transition: %v", lastTransition))
					}
				}
			}
		}
	}

	return strings.Join(results, "\n")
}

// extractSecretListSyncStatus extracts sync status from a list of ExternalSecrets
func extractSecretListSyncStatus(list interface{}) string {
	var results []string

	listMap, ok := list.(map[string]interface{})
	if !ok {
		type unstructuredLike interface {
			UnstructuredContent() map[string]interface{}
		}
		if u, ok := list.(unstructuredLike); ok {
			listMap = u.UnstructuredContent()
		}
	}

	if listMap == nil {
		return "No data available"
	}

	items, ok := listMap["items"].([]interface{})
	if !ok || len(items) == 0 {
		return "No ExternalSecrets found"
	}

	results = append(results, "| Name | Namespace | Status | Target Secret | Last Refresh |")
	results = append(results, "|------|-----------|--------|---------------|--------------|")

	for _, item := range items {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		name := "unknown"
		namespace := "-"
		status := "Unknown"
		targetSecret := "-"
		lastRefresh := "-"

		if metadata, ok := itemMap["metadata"].(map[string]interface{}); ok {
			if n, ok := metadata["name"].(string); ok {
				name = n
			}
			if ns, ok := metadata["namespace"].(string); ok {
				namespace = ns
			}
		}

		if statusMap, ok := itemMap["status"].(map[string]interface{}); ok {
			if binding, ok := statusMap["binding"].(map[string]interface{}); ok {
				if bindingName, ok := binding["name"].(string); ok {
					targetSecret = bindingName
				}
			}
			if refreshTime, ok := statusMap["refreshTime"].(string); ok {
				lastRefresh = refreshTime
			}

			if conditions, ok := statusMap["conditions"].([]interface{}); ok {
				for _, cond := range conditions {
					if condMap, ok := cond.(map[string]interface{}); ok {
						if condType, ok := condMap["type"].(string); ok && condType == "Ready" {
							if condStatus, ok := condMap["status"].(string); ok {
								if condStatus == "True" {
									status = "✅ Synced"
								} else {
									status = "❌ Failed"
									if reason, ok := condMap["reason"].(string); ok {
										status += " (" + reason + ")"
									}
								}
							}
						}
					}
				}
			}
		}

		results = append(results, fmt.Sprintf("| %s | %s | %s | %s | %s |", name, namespace, status, targetSecret, lastRefresh))
	}

	return strings.Join(results, "\n")
}
