package externalsecrets

import (
	"fmt"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
)

func initStatusTools() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name: "external_secrets_debug",
				Description: `Comprehensive debugging tool for External Secrets Operator issues.
Collects diagnostic information including:
- Operator deployment status and logs
- ExternalSecretsConfig status
- SecretStore/ClusterSecretStore validation status
- ExternalSecret sync status and errors
- Related Kubernetes events
Use this when troubleshooting sync failures or operator issues.
Reference: https://docs.redhat.com/en/documentation/openshift_container_platform/latest/html/security_and_compliance/external-secrets-operator-for-red-hat-openshift`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"namespace": {
							Type:        "string",
							Description: "Namespace to focus debugging on (optional, collects cluster-wide info if not provided)",
						},
						"include_logs": {
							Type:        "boolean",
							Description: "Include operator pod logs in the debug output (default: true)",
							Default:     api.ToRawMessage(true),
						},
						"log_tail_lines": {
							Type:        "integer",
							Description: "Number of log lines to include (default: 50)",
							Default:     api.ToRawMessage(50),
						},
					},
				},
				Annotations: api.ToolAnnotations{
					Title:           "External Secrets: Debug",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: debugHandler,
		},
		{
			Tool: api.Tool{
				Name: "external_secrets_events",
				Description: `Get Kubernetes events related to External Secrets resources.
Filters events for ExternalSecret, SecretStore, ClusterSecretStore, and ClusterExternalSecret resources.
Useful for troubleshooting sync failures and understanding what's happening.`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"namespace": {
							Type:        "string",
							Description: "Namespace to get events from (optional, gets from all namespaces if not provided)",
						},
						"resource_name": {
							Type:        "string",
							Description: "Filter events for a specific resource name (optional)",
						},
					},
				},
				Annotations: api.ToolAnnotations{
					Title:           "External Secrets: Events",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: eventsHandler,
		},
		{
			Tool: api.Tool{
				Name: "external_secrets_logs",
				Description: `Get logs from the External Secrets Operator pods.
Retrieves logs from the operator controller, webhook, and cert-controller pods.
Useful for diagnosing operator-level issues.`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"tail_lines": {
							Type:        "integer",
							Description: "Number of lines to retrieve from the end of the logs (default: 100)",
							Default:     api.ToRawMessage(100),
						},
						"container": {
							Type:        "string",
							Description: "Specific container to get logs from (optional, gets all containers if not provided)",
						},
						"previous": {
							Type:        "boolean",
							Description: "Get logs from previous container instance (default: false)",
							Default:     api.ToRawMessage(false),
						},
					},
				},
				Annotations: api.ToolAnnotations{
					Title:           "External Secrets: Operator Logs",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: logsHandler,
		},
		{
			Tool: api.Tool{
				Name: "external_secrets_health",
				Description: `Quick health check for the External Secrets Operator and resources.
Returns a summary of:
- Operator installation status
- Number of healthy/unhealthy SecretStores
- Number of synced/failed ExternalSecrets
- Any critical issues detected
Use this for a quick overview of the External Secrets health.`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"namespace": {
							Type:        "string",
							Description: "Namespace to check health for (optional, checks cluster-wide if not provided)",
						},
					},
				},
				Annotations: api.ToolAnnotations{
					Title:           "External Secrets: Health Check",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: healthHandler,
		},
		{
			Tool: api.Tool{
				Name: "external_secrets_guide",
				Description: `Get guidance and examples for using External Secrets Operator.
Provides documentation, examples, and best practices for:
- Setting up different secret providers (AWS, GCP, Azure, Vault, etc.)
- Creating SecretStores and ExternalSecrets
- Troubleshooting common issues
- Security best practices`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"topic": {
							Type:        "string",
							Description: "Topic to get guidance on: 'providers', 'secretstore', 'externalsecret', 'troubleshooting', 'security', or 'overview' (default: 'overview')",
							Default:     api.ToRawMessage("overview"),
							Enum:        []interface{}{"overview", "providers", "secretstore", "externalsecret", "troubleshooting", "security"},
						},
						"provider": {
							Type:        "string",
							Description: "Specific provider to get examples for: 'aws', 'gcp', 'azure', 'vault', 'kubernetes' (only used when topic is 'providers')",
							Enum:        []interface{}{"aws", "gcp", "azure", "vault", "kubernetes"},
						},
					},
				},
				Annotations: api.ToolAnnotations{
					Title:           "External Secrets: Guide",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(false),
				},
			},
			Handler: guideHandler,
		},
	}
}

func debugHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := getStringArg(params, "namespace", "")
	includeLogs := getBoolArg(params, "include_logs", true)
	tailLines := getIntArg(params, "log_tail_lines", 50)

	var results []string
	results = append(results, "# External Secrets Operator Debug Report\n")

	// 1. Operator Status
	results = append(results, "## 1. Operator Status")
	operatorResult, _ := operatorStatus(params)
	if operatorResult != nil {
		results = append(results, operatorResult.Content)
	}

	// 2. SecretStore Validation
	results = append(results, "\n## 2. SecretStore Validation")
	storeValidateResult, _ := storeValidate(params)
	if storeValidateResult != nil {
		results = append(results, storeValidateResult.Content)
	}

	// 3. ExternalSecret Sync Status
	results = append(results, "\n## 3. ExternalSecret Sync Status")
	syncStatusResult, _ := secretSyncStatus(params)
	if syncStatusResult != nil {
		results = append(results, syncStatusResult.Content)
	}

	// 4. Events
	results = append(results, "\n## 4. Related Events")
	eventsResult, _ := eventsHandler(params)
	if eventsResult != nil {
		results = append(results, eventsResult.Content)
	}

	// 5. Operator Logs (if requested)
	if includeLogs {
		results = append(results, "\n## 5. Operator Logs")
		// Get operator pods
		podGVK := &schema.GroupVersionKind{
			Group:   "",
			Version: "v1",
			Kind:    "Pod",
		}
		pods, err := params.ResourcesList(params, podGVK, externalSecretsOperatorNamespace, kubernetes.ResourceListOptions{
			ListOptions: metav1.ListOptions{
				LabelSelector: "app.kubernetes.io/name=external-secrets",
			},
			AsTable: false,
		})
		if err != nil {
			results = append(results, fmt.Sprintf("Error getting operator pods: %v", err))
		} else {
			podsMap := pods.UnstructuredContent()
			if items, ok := podsMap["items"].([]interface{}); ok {
				for _, item := range items {
					if itemMap, ok := item.(map[string]interface{}); ok {
						if metadata, ok := itemMap["metadata"].(map[string]interface{}); ok {
							if podName, ok := metadata["name"].(string); ok {
								results = append(results, fmt.Sprintf("\n### Pod: %s", podName))
								logs, err := params.PodsLog(params.Context, externalSecretsOperatorNamespace, podName, "", false, int64(tailLines))
								if err != nil {
									results = append(results, fmt.Sprintf("Error getting logs: %v", err))
								} else {
									results = append(results, "```")
									results = append(results, logs)
									results = append(results, "```")
								}
							}
						}
					}
				}
			}
		}
	}

	// 6. Diagnostic Summary
	results = append(results, "\n## 6. Diagnostic Summary")
	results = append(results, "For more detailed troubleshooting, use:")
	results = append(results, "- `external_secrets_logs` - Get detailed operator logs")
	results = append(results, "- `external_secrets_store_get` - Inspect specific SecretStore")
	results = append(results, "- `external_secrets_get` - Inspect specific ExternalSecret")
	results = append(results, "- `external_secrets_guide topic=troubleshooting` - Get troubleshooting guidance")

	if namespace != "" {
		results = append(results, fmt.Sprintf("\nNote: Debug focused on namespace: %s", namespace))
	}

	return api.NewToolCallResult(strings.Join(results, "\n"), nil), nil
}

func eventsHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := getStringArg(params, "namespace", "")
	resourceName := getStringArg(params, "resource_name", "")

	eventGVK := &schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Event",
	}

	events, err := params.ResourcesList(params, eventGVK, namespace, kubernetes.ResourceListOptions{AsTable: false})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list events: %w", err)), nil
	}

	var results []string
	results = append(results, "# External Secrets Related Events\n")
	results = append(results, "| Time | Type | Reason | Object | Message |")
	results = append(results, "|------|------|--------|--------|---------|")

	eventsMap := events.UnstructuredContent()
	items, ok := eventsMap["items"].([]interface{})
	if !ok || len(items) == 0 {
		return api.NewToolCallResult("No events found", nil), nil
	}

	externalSecretsKinds := map[string]bool{
		"ExternalSecret":        true,
		"SecretStore":           true,
		"ClusterSecretStore":    true,
		"ClusterExternalSecret": true,
	}

	eventCount := 0
	for _, item := range items {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		// Check if this event is related to External Secrets
		involvedObject, ok := itemMap["involvedObject"].(map[string]interface{})
		if !ok {
			continue
		}

		kind, _ := involvedObject["kind"].(string)
		objName, _ := involvedObject["name"].(string)

		if !externalSecretsKinds[kind] {
			continue
		}

		if resourceName != "" && objName != resourceName {
			continue
		}

		eventType, _ := itemMap["type"].(string)
		reason, _ := itemMap["reason"].(string)
		message, _ := itemMap["message"].(string)
		lastTimestamp, _ := itemMap["lastTimestamp"].(string)

		if len(message) > 60 {
			message = message[:60] + "..."
		}

		objectRef := fmt.Sprintf("%s/%s", kind, objName)
		results = append(results, fmt.Sprintf("| %s | %s | %s | %s | %s |", lastTimestamp, eventType, reason, objectRef, message))
		eventCount++
	}

	if eventCount == 0 {
		return api.NewToolCallResult("No External Secrets related events found", nil), nil
	}

	return api.NewToolCallResult(strings.Join(results, "\n"), nil), nil
}

func logsHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	tailLines := getIntArg(params, "tail_lines", 100)
	container := getStringArg(params, "container", "")
	previous := getBoolArg(params, "previous", false)

	var results []string
	results = append(results, "# External Secrets Operator Logs\n")

	// Get operator pods
	podGVK := &schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Pod",
	}

	pods, err := params.ResourcesList(params, podGVK, externalSecretsOperatorNamespace, kubernetes.ResourceListOptions{
		AsTable: false,
	})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list operator pods: %w", err)), nil
	}

	podsMap := pods.UnstructuredContent()
	items, ok := podsMap["items"].([]interface{})
	if !ok || len(items) == 0 {
		return api.NewToolCallResult("No operator pods found. Is the External Secrets Operator installed?", nil), nil
	}

	for _, item := range items {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		metadata, ok := itemMap["metadata"].(map[string]interface{})
		if !ok {
			continue
		}

		podName, ok := metadata["name"].(string)
		if !ok {
			continue
		}

		results = append(results, fmt.Sprintf("## Pod: %s", podName))

		logs, err := params.PodsLog(params.Context, externalSecretsOperatorNamespace, podName, container, previous, int64(tailLines))
		if err != nil {
			results = append(results, fmt.Sprintf("Error getting logs: %v\n", err))
		} else if logs == "" {
			results = append(results, "No logs available\n")
		} else {
			results = append(results, "```")
			results = append(results, logs)
			results = append(results, "```\n")
		}
	}

	return api.NewToolCallResult(strings.Join(results, "\n"), nil), nil
}

func healthHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	namespace := getStringArg(params, "namespace", "")

	var results []string
	results = append(results, "# External Secrets Health Check\n")

	// Overall status
	operatorHealthy := true
	storesHealthy := 0
	storesUnhealthy := 0
	secretsSynced := 0
	secretsFailed := 0
	var issues []string

	// Check operator
	results = append(results, "## Operator Status")
	subscriptionGVK := &schema.GroupVersionKind{
		Group:   operatorsAPIGroup,
		Version: "v1alpha1",
		Kind:    "Subscription",
	}
	_, err := params.ResourcesGet(params, subscriptionGVK, externalSecretsOperatorNamespace, externalSecretsOperatorName)
	if err != nil {
		operatorHealthy = false
		issues = append(issues, "❌ Operator not installed or subscription not found")
		results = append(results, "- ❌ **Not Installed**")
	} else {
		results = append(results, "- ✅ **Installed**")

		// Check if pods are running
		podGVK := &schema.GroupVersionKind{
			Group:   "",
			Version: "v1",
			Kind:    "Pod",
		}
		pods, err := params.ResourcesList(params, podGVK, externalSecretsOperatorNamespace, kubernetes.ResourceListOptions{AsTable: false})
		if err == nil {
			podsMap := pods.UnstructuredContent()
			if items, ok := podsMap["items"].([]interface{}); ok && len(items) > 0 {
				runningPods := 0
				for _, item := range items {
					if itemMap, ok := item.(map[string]interface{}); ok {
						if status, ok := itemMap["status"].(map[string]interface{}); ok {
							if phase, ok := status["phase"].(string); ok && phase == "Running" {
								runningPods++
							}
						}
					}
				}
				results = append(results, fmt.Sprintf("- ✅ **%d pods running**", runningPods))
			}
		}
	}

	// Check SecretStores
	results = append(results, "\n## SecretStore Status")
	secretStoreGVK := &schema.GroupVersionKind{
		Group:   externalSecretsAPIGroup,
		Version: "v1",
		Kind:    "SecretStore",
	}
	stores, err := params.ResourcesList(params, secretStoreGVK, namespace, kubernetes.ResourceListOptions{AsTable: false})
	if err == nil {
		storesMap := stores.UnstructuredContent()
		if items, ok := storesMap["items"].([]interface{}); ok {
			for _, item := range items {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if status, ok := itemMap["status"].(map[string]interface{}); ok {
						if conditions, ok := status["conditions"].([]interface{}); ok {
							for _, cond := range conditions {
								if condMap, ok := cond.(map[string]interface{}); ok {
									if condType, ok := condMap["type"].(string); ok && condType == "Ready" {
										if condStatus, ok := condMap["status"].(string); ok {
											if condStatus == "True" {
												storesHealthy++
											} else {
												storesUnhealthy++
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Also check ClusterSecretStores
	clusterStoreGVK := &schema.GroupVersionKind{
		Group:   externalSecretsAPIGroup,
		Version: "v1",
		Kind:    "ClusterSecretStore",
	}
	clusterStores, err := params.ResourcesList(params, clusterStoreGVK, "", kubernetes.ResourceListOptions{AsTable: false})
	if err == nil {
		storesMap := clusterStores.UnstructuredContent()
		if items, ok := storesMap["items"].([]interface{}); ok {
			for _, item := range items {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if status, ok := itemMap["status"].(map[string]interface{}); ok {
						if conditions, ok := status["conditions"].([]interface{}); ok {
							for _, cond := range conditions {
								if condMap, ok := cond.(map[string]interface{}); ok {
									if condType, ok := condMap["type"].(string); ok && condType == "Ready" {
										if condStatus, ok := condMap["status"].(string); ok {
											if condStatus == "True" {
												storesHealthy++
											} else {
												storesUnhealthy++
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	results = append(results, fmt.Sprintf("- ✅ Healthy: %d", storesHealthy))
	results = append(results, fmt.Sprintf("- ❌ Unhealthy: %d", storesUnhealthy))
	if storesUnhealthy > 0 {
		issues = append(issues, fmt.Sprintf("❌ %d SecretStore(s) have validation issues", storesUnhealthy))
	}

	// Check ExternalSecrets
	results = append(results, "\n## ExternalSecret Status")
	externalSecretGVK := &schema.GroupVersionKind{
		Group:   externalSecretsAPIGroup,
		Version: "v1",
		Kind:    "ExternalSecret",
	}
	secrets, err := params.ResourcesList(params, externalSecretGVK, namespace, kubernetes.ResourceListOptions{AsTable: false})
	if err == nil {
		secretsMap := secrets.UnstructuredContent()
		if items, ok := secretsMap["items"].([]interface{}); ok {
			for _, item := range items {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if status, ok := itemMap["status"].(map[string]interface{}); ok {
						if conditions, ok := status["conditions"].([]interface{}); ok {
							for _, cond := range conditions {
								if condMap, ok := cond.(map[string]interface{}); ok {
									if condType, ok := condMap["type"].(string); ok && condType == "Ready" {
										if condStatus, ok := condMap["status"].(string); ok {
											if condStatus == "True" {
												secretsSynced++
											} else {
												secretsFailed++
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	results = append(results, fmt.Sprintf("- ✅ Synced: %d", secretsSynced))
	results = append(results, fmt.Sprintf("- ❌ Failed: %d", secretsFailed))
	if secretsFailed > 0 {
		issues = append(issues, fmt.Sprintf("❌ %d ExternalSecret(s) have sync issues", secretsFailed))
	}

	// Summary
	results = append(results, "\n## Summary")
	if operatorHealthy && storesUnhealthy == 0 && secretsFailed == 0 {
		results = append(results, "✅ **All systems healthy**")
	} else {
		results = append(results, "⚠️ **Issues detected:**")
		for _, issue := range issues {
			results = append(results, fmt.Sprintf("- %s", issue))
		}
		results = append(results, "\nUse `external_secrets_debug` for detailed troubleshooting.")
	}

	return api.NewToolCallResult(strings.Join(results, "\n"), nil), nil
}

func guideHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	topic := getStringArg(params, "topic", "overview")
	provider := getStringArg(params, "provider", "")

	switch topic {
	case "overview":
		return api.NewToolCallResult(guideOverview(), nil), nil
	case "providers":
		return api.NewToolCallResult(guideProviders(provider), nil), nil
	case "secretstore":
		return api.NewToolCallResult(guideSecretStore(), nil), nil
	case "externalsecret":
		return api.NewToolCallResult(guideExternalSecret(), nil), nil
	case "troubleshooting":
		return api.NewToolCallResult(guideTroubleshooting(), nil), nil
	case "security":
		return api.NewToolCallResult(guideSecurity(), nil), nil
	default:
		return api.NewToolCallResult(guideOverview(), nil), nil
	}
}

func guideOverview() string {
	return `# External Secrets Operator Overview

The External Secrets Operator (ESO) synchronizes secrets from external APIs (AWS Secrets Manager, 
HashiCorp Vault, Google Secret Manager, Azure Key Vault, etc.) into Kubernetes Secrets.

## Key Concepts

### SecretStore / ClusterSecretStore
Defines how to connect to an external secret provider. SecretStore is namespaced, 
ClusterSecretStore is cluster-wide.

### ExternalSecret / ClusterExternalSecret
Defines what secrets to fetch and how to create the Kubernetes Secret.

## Quick Start

1. **Install the operator:**
   ` + "`external_secrets_operator_install`" + `

2. **Create a SecretStore:**
   ` + "`external_secrets_store_create`" + ` with your provider configuration

3. **Create an ExternalSecret:**
   ` + "`external_secrets_create`" + ` to sync secrets from the provider

4. **Verify sync status:**
   ` + "`external_secrets_sync_status`" + `

## Available Tools

- ` + "`external_secrets_operator_install/status/uninstall`" + ` - Operator lifecycle
- ` + "`external_secrets_config_get/apply`" + ` - Operator configuration
- ` + "`external_secrets_store_list/get/create/delete/validate`" + ` - SecretStore management
- ` + "`external_secrets_list/get/create/delete/sync_status/refresh`" + ` - ExternalSecret management
- ` + "`external_secrets_debug/events/logs/health`" + ` - Debugging and monitoring
- ` + "`external_secrets_guide`" + ` - Documentation and examples

## References

- [Red Hat Documentation](https://docs.redhat.com/en/documentation/openshift_container_platform/latest/html/security_and_compliance/external-secrets-operator-for-red-hat-openshift)
- [External Secrets Documentation](https://external-secrets.io/latest/)
`
}

func guideProviders(provider string) string {
	switch provider {
	case "aws":
		return guideProviderAWS()
	case "gcp":
		return guideProviderGCP()
	case "azure":
		return guideProviderAzure()
	case "vault":
		return guideProviderVault()
	case "kubernetes":
		return guideProviderKubernetes()
	default:
		return `# Supported Providers

External Secrets Operator supports many secret providers:

## Cloud Providers
- **AWS Secrets Manager** - Use ` + "`external_secrets_guide topic=providers provider=aws`" + `
- **AWS Parameter Store**
- **Google Cloud Secret Manager** - Use ` + "`external_secrets_guide topic=providers provider=gcp`" + `
- **Azure Key Vault** - Use ` + "`external_secrets_guide topic=providers provider=azure`" + `

## Secret Management Tools
- **HashiCorp Vault** - Use ` + "`external_secrets_guide topic=providers provider=vault`" + `
- **CyberArk Conjur**
- **Bitwarden**
- **1Password**
- **Doppler**
- **Infisical**

## Other
- **Kubernetes Secrets** - Use ` + "`external_secrets_guide topic=providers provider=kubernetes`" + `
- **Webhook** - Custom HTTP endpoints

For detailed provider setup, specify the provider parameter.
`
	}
}

func guideProviderAWS() string {
	return `# AWS Secrets Manager Setup

## Prerequisites
- AWS credentials with access to Secrets Manager
- IAM policy allowing secretsmanager:GetSecretValue

## SecretStore Example

` + "```yaml" + `
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
` + "```" + `

## Create AWS credentials secret first:

` + "```yaml" + `
apiVersion: v1
kind: Secret
metadata:
  name: aws-credentials
  namespace: my-namespace
type: Opaque
stringData:
  access-key: AKIAIOSFODNN7EXAMPLE
  secret-access-key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
` + "```" + `

## ExternalSecret Example

` + "```yaml" + `
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
  data:
  - secretKey: password
    remoteRef:
      key: my-aws-secret
      property: password
` + "```" + `

## Using IAM Roles for Service Accounts (IRSA)

For EKS clusters, you can use IRSA instead of static credentials:

` + "```yaml" + `
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: aws-secretsmanager-irsa
spec:
  provider:
    aws:
      service: SecretsManager
      region: us-east-1
      auth:
        jwt:
          serviceAccountRef:
            name: my-service-account
` + "```" + `
`
}

func guideProviderGCP() string {
	return `# Google Cloud Secret Manager Setup

## Prerequisites
- GCP Service Account with Secret Manager Secret Accessor role
- Service account key JSON

## SecretStore Example

` + "```yaml" + `
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: gcp-secretmanager
  namespace: my-namespace
spec:
  provider:
    gcpsm:
      projectID: my-gcp-project
      auth:
        secretRef:
          secretAccessKeySecretRef:
            name: gcp-credentials
            key: secret-access-credentials
` + "```" + `

## Create GCP credentials secret:

` + "```yaml" + `
apiVersion: v1
kind: Secret
metadata:
  name: gcp-credentials
  namespace: my-namespace
type: Opaque
stringData:
  secret-access-credentials: |
    {
      "type": "service_account",
      "project_id": "my-gcp-project",
      ...
    }
` + "```" + `

## ExternalSecret Example

` + "```yaml" + `
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: my-secret
  namespace: my-namespace
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: gcp-secretmanager
    kind: SecretStore
  target:
    name: my-k8s-secret
  data:
  - secretKey: api-key
    remoteRef:
      key: my-gcp-secret
` + "```" + `
`
}

func guideProviderAzure() string {
	return `# Azure Key Vault Setup

## Prerequisites
- Azure Key Vault instance
- Service Principal or Managed Identity with access

## SecretStore Example (Service Principal)

` + "```yaml" + `
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: azure-keyvault
  namespace: my-namespace
spec:
  provider:
    azurekv:
      vaultUrl: https://my-keyvault.vault.azure.net
      authType: ServicePrincipal
      tenantId: "00000000-0000-0000-0000-000000000000"
      authSecretRef:
        clientId:
          name: azure-credentials
          key: client-id
        clientSecret:
          name: azure-credentials
          key: client-secret
` + "```" + `

## ExternalSecret Example

` + "```yaml" + `
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: my-secret
  namespace: my-namespace
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: azure-keyvault
    kind: SecretStore
  target:
    name: my-k8s-secret
  data:
  - secretKey: connection-string
    remoteRef:
      key: database-connection-string
` + "```" + `
`
}

func guideProviderVault() string {
	return `# HashiCorp Vault Setup

## Prerequisites
- HashiCorp Vault instance
- Vault token or Kubernetes auth configured

## SecretStore Example (Token Auth)

` + "```yaml" + `
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: vault
  namespace: my-namespace
spec:
  provider:
    vault:
      server: https://vault.example.com
      path: secret
      version: v2
      auth:
        tokenSecretRef:
          name: vault-token
          key: token
` + "```" + `

## SecretStore Example (Kubernetes Auth)

` + "```yaml" + `
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: vault-k8s
  namespace: my-namespace
spec:
  provider:
    vault:
      server: https://vault.example.com
      path: secret
      version: v2
      auth:
        kubernetes:
          mountPath: kubernetes
          role: my-role
          serviceAccountRef:
            name: vault-auth
` + "```" + `

## ExternalSecret Example

` + "```yaml" + `
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: my-secret
  namespace: my-namespace
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: vault
    kind: SecretStore
  target:
    name: my-k8s-secret
  data:
  - secretKey: password
    remoteRef:
      key: secret/data/myapp
      property: password
` + "```" + `
`
}

func guideProviderKubernetes() string {
	return `# Kubernetes Secrets Provider

The Kubernetes provider allows you to sync secrets from one namespace/cluster to another.

## SecretStore Example

` + "```yaml" + `
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: kubernetes-store
  namespace: my-namespace
spec:
  provider:
    kubernetes:
      remoteNamespace: source-namespace
      auth:
        serviceAccount:
          name: secret-reader
      server:
        caProvider:
          type: ConfigMap
          name: kube-root-ca.crt
          key: ca.crt
` + "```" + `

## ExternalSecret Example

` + "```yaml" + `
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: copied-secret
  namespace: my-namespace
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: kubernetes-store
    kind: SecretStore
  target:
    name: my-copied-secret
  dataFrom:
  - extract:
      key: source-secret-name
` + "```" + `
`
}

func guideSecretStore() string {
	return `# SecretStore Guide

## SecretStore vs ClusterSecretStore

- **SecretStore**: Namespaced, can only be used by ExternalSecrets in the same namespace
- **ClusterSecretStore**: Cluster-scoped, can be used by ExternalSecrets in any namespace

## Best Practices

1. **Use ClusterSecretStore for shared providers**
   When multiple namespaces need access to the same provider

2. **Use SecretStore for namespace isolation**
   When different namespaces should have different access levels

3. **Store credentials in the same namespace**
   SecretStore auth credentials should be in the same namespace

4. **Use conditions to verify store health**
   Always check the Ready condition before using

## Validation

After creating a SecretStore, verify it's valid:

` + "```bash" + `
# Using MCP tool
external_secrets_store_validate

# The store should show:
# - Status: Valid
# - Capabilities: ReadWrite or ReadOnly
` + "```" + `

## Common Issues

1. **Authentication failures**: Check credentials are correct
2. **Network issues**: Ensure the cluster can reach the provider
3. **Permission issues**: Verify IAM/RBAC permissions
`
}

func guideExternalSecret() string {
	return `# ExternalSecret Guide

## Key Fields

- **refreshInterval**: How often to sync (e.g., "1h", "15m")
- **secretStoreRef**: Reference to the SecretStore to use
- **target**: Configuration for the created Kubernetes Secret
- **data**: List of individual secret key mappings
- **dataFrom**: Extract multiple keys at once

## Data vs DataFrom

### data - Individual key mapping
` + "```yaml" + `
data:
- secretKey: my-key        # Key in K8s Secret
  remoteRef:
    key: remote-secret     # Secret name in provider
    property: password     # Specific property (if JSON)
` + "```" + `

### dataFrom - Extract all keys
` + "```yaml" + `
dataFrom:
- extract:
    key: remote-secret     # Extracts all keys from this secret
` + "```" + `

## Creation Policies

- **Owner** (default): Secret is deleted when ExternalSecret is deleted
- **Orphan**: Secret is kept when ExternalSecret is deleted
- **Merge**: Merge with existing secret
- **None**: Don't create, only sync to existing secret

## Template Support

You can transform secret data:

` + "```yaml" + `
target:
  name: my-secret
  template:
    type: kubernetes.io/dockerconfigjson
    data:
      .dockerconfigjson: |
        {"auths":{"registry.example.com":{"auth":"{{ .username }}:{{ .password | b64enc }}"}}}
` + "```" + `
`
}

func guideTroubleshooting() string {
	return `# Troubleshooting Guide

## Common Issues

### 1. SecretStore Not Ready

**Symptoms**: SecretStore shows status "Invalid" or not Ready

**Debug steps**:
` + "```" + `
external_secrets_store_get name=<store-name> namespace=<ns>
external_secrets_events namespace=<ns>
` + "```" + `

**Common causes**:
- Invalid credentials
- Network connectivity issues
- Wrong region/endpoint
- Missing IAM permissions

### 2. ExternalSecret Not Syncing

**Symptoms**: ExternalSecret shows "SecretSyncedError" or stuck

**Debug steps**:
` + "```" + `
external_secrets_sync_status name=<secret-name> namespace=<ns>
external_secrets_debug namespace=<ns>
` + "```" + `

**Common causes**:
- SecretStore not ready
- Secret doesn't exist in provider
- Wrong key/property path
- Permission denied to specific secret

### 3. Operator Not Running

**Symptoms**: No pods in external-secrets-operator namespace

**Debug steps**:
` + "```" + `
external_secrets_operator_status
external_secrets_logs
` + "```" + `

**Common causes**:
- Subscription not approved (Manual approval mode)
- Resource constraints
- Image pull errors

## Quick Health Check

` + "```" + `
external_secrets_health
` + "```" + `

## Comprehensive Debug

` + "```" + `
external_secrets_debug namespace=<ns> include_logs=true
` + "```" + `

## Force Refresh

If a secret should have new data:
` + "```" + `
external_secrets_refresh name=<secret-name> namespace=<ns>
` + "```" + `
`
}

func guideSecurity() string {
	return `# Security Best Practices

## 1. Principle of Least Privilege

- Create separate SecretStores per namespace/team
- Use specific IAM policies that only allow access to needed secrets
- Use ClusterSecretStore sparingly

## 2. Secure Credential Storage

- Store provider credentials in Kubernetes Secrets
- Use workload identity when possible (IRSA, Workload Identity)
- Rotate credentials regularly

## 3. Network Security

- Use VPC endpoints for cloud providers
- Configure network policies for the operator namespace
- Use TLS for all provider connections

## 4. RBAC Configuration

Limit who can create/modify SecretStores:

` + "```yaml" + `
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: external-secrets-user
rules:
- apiGroups: ["external-secrets.io"]
  resources: ["externalsecrets"]
  verbs: ["get", "list", "create", "update", "delete"]
# Note: SecretStore creation should be restricted to admins
` + "```" + `

## 5. Audit and Monitoring

- Enable audit logging for external-secrets resources
- Monitor sync failures with ` + "`external_secrets_health`" + `
- Set up alerts for sync errors

## 6. Secret Rotation

- Set appropriate refreshInterval (not too frequent, not too rare)
- Use provider-side rotation when available
- Test rotation procedures regularly
`
}
