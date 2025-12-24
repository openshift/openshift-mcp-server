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
	// Operator constants
	externalSecretsOperatorNamespace     = "external-secrets-operator"
	externalSecretsOperatorName          = "external-secrets-operator"
	externalSecretsOperatorChannel       = "stable"
	externalSecretsOperatorCatalogSource = "redhat-operators"
	externalSecretsOperatorSourceNS      = "openshift-marketplace"

	// API Groups
	operatorsAPIGroup             = "operators.coreos.com"
	externalSecretsOperatorAPIGrp = "operator.external-secrets.io"
)

func initOperatorTools() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name: "external_secrets_operator_install",
				Description: `Install the External Secrets Operator for Red Hat OpenShift via OLM (Operator Lifecycle Manager).
This creates the required Namespace, OperatorGroup, and Subscription resources.
The operator will be installed in the 'external-secrets-operator' namespace.
Reference: https://docs.redhat.com/en/documentation/openshift_container_platform/latest/html/security_and_compliance/external-secrets-operator-for-red-hat-openshift`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"channel": {
							Type:        "string",
							Description: "Subscription channel (default: 'stable')",
							Default:     api.ToRawMessage("stable"),
						},
						"approval": {
							Type:        "string",
							Description: "Install plan approval strategy: 'Automatic' or 'Manual' (default: 'Automatic')",
							Default:     api.ToRawMessage("Automatic"),
							Enum:        []interface{}{"Automatic", "Manual"},
						},
					},
				},
				Annotations: api.ToolAnnotations{
					Title:           "External Secrets: Install Operator",
					ReadOnlyHint:    ptr.To(false),
					DestructiveHint: ptr.To(false),
					IdempotentHint:  ptr.To(true),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: operatorInstall,
		},
		{
			Tool: api.Tool{
				Name: "external_secrets_operator_status",
				Description: `Get the status of the External Secrets Operator installation.
Returns information about the Subscription, ClusterServiceVersion (CSV), and operator deployment status.
Use this to verify if the operator is installed and running correctly.`,
				InputSchema: &jsonschema.Schema{
					Type:       "object",
					Properties: map[string]*jsonschema.Schema{},
				},
				Annotations: api.ToolAnnotations{
					Title:           "External Secrets: Operator Status",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: operatorStatus,
		},
		{
			Tool: api.Tool{
				Name: "external_secrets_operator_uninstall",
				Description: `Uninstall the External Secrets Operator for Red Hat OpenShift.
This removes the Subscription and ClusterServiceVersion (CSV) resources.
WARNING: This will remove the operator but NOT the CRDs or existing ExternalSecrets/SecretStores.`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"delete_namespace": {
							Type:        "boolean",
							Description: "Also delete the operator namespace (default: false)",
							Default:     api.ToRawMessage(false),
						},
					},
				},
				Annotations: api.ToolAnnotations{
					Title:           "External Secrets: Uninstall Operator",
					ReadOnlyHint:    ptr.To(false),
					DestructiveHint: ptr.To(true),
					IdempotentHint:  ptr.To(true),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: operatorUninstall,
		},
		{
			Tool: api.Tool{
				Name: "external_secrets_config_get",
				Description: `Get the ExternalSecretsConfig resource which controls the operator configuration.
The ExternalSecretsConfig API allows you to customize operator behavior such as:
- Controller deployment settings
- Webhook configuration
- Cert controller settings
Reference: https://docs.redhat.com/en/documentation/openshift_container_platform/latest/html/security_and_compliance/external-secrets-operator-for-red-hat-openshift#customizing-the-external-secrets-operator-for-red-hat-openshift`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"name": {
							Type:        "string",
							Description: "Name of the ExternalSecretsConfig resource (default: 'cluster')",
							Default:     api.ToRawMessage("cluster"),
						},
					},
				},
				Annotations: api.ToolAnnotations{
					Title:           "External Secrets: Get Operator Config",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: configGet,
		},
		{
			Tool: api.Tool{
				Name: "external_secrets_config_apply",
				Description: `Create or update the ExternalSecretsConfig resource to configure the operator.
The ExternalSecretsConfig controls operator deployment settings, webhook configuration, etc.
Example configuration YAML:
  apiVersion: operator.external-secrets.io/v1alpha1
  kind: ExternalSecretsConfig
  metadata:
    name: cluster
  spec:
    fullnameOverride: my-external-secrets
Reference: https://docs.redhat.com/en/documentation/openshift_container_platform/latest/html/security_and_compliance/external-secrets-operator-for-red-hat-openshift#customizing-the-external-secrets-operator-for-red-hat-openshift`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"config": {
							Type:        "string",
							Description: "YAML or JSON representation of the ExternalSecretsConfig resource",
						},
					},
					Required: []string{"config"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "External Secrets: Apply Operator Config",
					ReadOnlyHint:    ptr.To(false),
					DestructiveHint: ptr.To(false),
					IdempotentHint:  ptr.To(true),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: configApply,
		},
	}
}

func operatorInstall(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	channel := getStringArg(params, "channel", externalSecretsOperatorChannel)
	approval := getStringArg(params, "approval", "Automatic")

	// Create resources in order: Namespace, OperatorGroup, Subscription
	resources := fmt.Sprintf(`---
apiVersion: v1
kind: Namespace
metadata:
  name: %s
  labels:
    openshift.io/cluster-monitoring: "true"
---
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: %s
  namespace: %s
spec: {}
---
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: %s
  namespace: %s
spec:
  channel: %s
  installPlanApproval: %s
  name: %s
  source: %s
  sourceNamespace: %s
`,
		externalSecretsOperatorNamespace,
		externalSecretsOperatorName,
		externalSecretsOperatorNamespace,
		externalSecretsOperatorName,
		externalSecretsOperatorNamespace,
		channel,
		approval,
		externalSecretsOperatorName,
		externalSecretsOperatorCatalogSource,
		externalSecretsOperatorSourceNS,
	)

	result, err := params.ResourcesCreateOrUpdate(params, resources)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to install External Secrets Operator: %w", err)), nil
	}

	marshalledYaml, err := output.MarshalYaml(result)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal result: %w", err)), nil
	}

	return api.NewToolCallResult(
		"# External Secrets Operator installation initiated\n"+
			"The operator is being installed. Use 'external_secrets_operator_status' to monitor the installation progress.\n\n"+
			"# Created resources:\n"+marshalledYaml,
		nil,
	), nil
}

func operatorStatus(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	var statusParts []string
	statusParts = append(statusParts, "# External Secrets Operator Status\n")

	// Check Subscription
	subscriptionGVK := &schema.GroupVersionKind{
		Group:   operatorsAPIGroup,
		Version: "v1alpha1",
		Kind:    "Subscription",
	}
	sub, err := params.ResourcesGet(params, subscriptionGVK, externalSecretsOperatorNamespace, externalSecretsOperatorName)
	if err != nil {
		statusParts = append(statusParts, fmt.Sprintf("## Subscription\nNot found or error: %v\n", err))
	} else {
		subYaml, _ := output.MarshalYaml(sub)
		statusParts = append(statusParts, fmt.Sprintf("## Subscription\n```yaml\n%s```\n", subYaml))
	}

	// Check CSV (ClusterServiceVersion) - list all in namespace
	csvGVK := &schema.GroupVersionKind{
		Group:   operatorsAPIGroup,
		Version: "v1alpha1",
		Kind:    "ClusterServiceVersion",
	}
	csvList, err := params.ResourcesList(params, csvGVK, externalSecretsOperatorNamespace, kubernetes.ResourceListOptions{AsTable: false})
	if err != nil {
		statusParts = append(statusParts, fmt.Sprintf("## ClusterServiceVersion\nNot found or error: %v\n", err))
	} else {
		csvYaml, _ := output.MarshalYaml(csvList)
		statusParts = append(statusParts, fmt.Sprintf("## ClusterServiceVersion(s)\n```yaml\n%s```\n", csvYaml))
	}

	// Check operator deployment
	deploymentGVK := &schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	}
	deployList, err := params.ResourcesList(params, deploymentGVK, externalSecretsOperatorNamespace, kubernetes.ResourceListOptions{AsTable: true})
	if err != nil {
		statusParts = append(statusParts, fmt.Sprintf("## Deployments\nNot found or error: %v\n", err))
	} else {
		deployYaml, _ := params.ListOutput.PrintObj(deployList)
		statusParts = append(statusParts, fmt.Sprintf("## Deployments\n%s\n", deployYaml))
	}

	// Check ExternalSecretsConfig
	configGVK := &schema.GroupVersionKind{
		Group:   externalSecretsOperatorAPIGrp,
		Version: "v1alpha1",
		Kind:    "ExternalSecretsConfig",
	}
	config, err := params.ResourcesGet(params, configGVK, "", "cluster")
	if err != nil {
		statusParts = append(statusParts, "## ExternalSecretsConfig\nNot found (operator may not be fully installed yet)\n")
	} else {
		configYaml, _ := output.MarshalYaml(config)
		statusParts = append(statusParts, fmt.Sprintf("## ExternalSecretsConfig\n```yaml\n%s```\n", configYaml))
	}

	return api.NewToolCallResult(strings.Join(statusParts, "\n"), nil), nil
}

func operatorUninstall(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	deleteNamespace := getBoolArg(params, "delete_namespace", false)

	var results []string
	results = append(results, "# External Secrets Operator Uninstallation\n")

	// Delete Subscription
	subscriptionGVK := &schema.GroupVersionKind{
		Group:   operatorsAPIGroup,
		Version: "v1alpha1",
		Kind:    "Subscription",
	}
	err := params.ResourcesDelete(params, subscriptionGVK, externalSecretsOperatorNamespace, externalSecretsOperatorName)
	if err != nil {
		results = append(results, fmt.Sprintf("- Subscription deletion: %v", err))
	} else {
		results = append(results, "- Subscription deleted successfully")
	}

	// Find and delete CSV
	csvGVK := &schema.GroupVersionKind{
		Group:   operatorsAPIGroup,
		Version: "v1alpha1",
		Kind:    "ClusterServiceVersion",
	}
	csvList, err := params.ResourcesList(params, csvGVK, externalSecretsOperatorNamespace, kubernetes.ResourceListOptions{AsTable: false})
	if err == nil {
		csvListMap := csvList.UnstructuredContent()
		items, ok := csvListMap["items"].([]interface{})
		if ok {
			for _, item := range items {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if metadata, ok := itemMap["metadata"].(map[string]interface{}); ok {
						if name, ok := metadata["name"].(string); ok {
							if strings.Contains(name, "external-secrets") {
								err = params.ResourcesDelete(params, csvGVK, externalSecretsOperatorNamespace, name)
								if err != nil {
									results = append(results, fmt.Sprintf("- CSV %s deletion: %v", name, err))
								} else {
									results = append(results, fmt.Sprintf("- CSV %s deleted successfully", name))
								}
							}
						}
					}
				}
			}
		}
	}

	// Optionally delete namespace
	if deleteNamespace {
		nsGVK := &schema.GroupVersionKind{
			Group:   "",
			Version: "v1",
			Kind:    "Namespace",
		}
		err = params.ResourcesDelete(params, nsGVK, "", externalSecretsOperatorNamespace)
		if err != nil {
			results = append(results, fmt.Sprintf("- Namespace deletion: %v", err))
		} else {
			results = append(results, "- Namespace deleted successfully")
		}
	}

	results = append(results, "\nNote: CRDs and existing ExternalSecrets/SecretStores are not deleted.")

	return api.NewToolCallResult(strings.Join(results, "\n"), nil), nil
}

func configGet(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	name := getStringArg(params, "name", "cluster")

	configGVK := &schema.GroupVersionKind{
		Group:   externalSecretsOperatorAPIGrp,
		Version: "v1alpha1",
		Kind:    "ExternalSecretsConfig",
	}

	config, err := params.ResourcesGet(params, configGVK, "", name)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get ExternalSecretsConfig '%s': %w", name, err)), nil
	}

	configYaml, _ := output.MarshalYaml(config)
	return api.NewToolCallResult(
		"# ExternalSecretsConfig\n```yaml\n"+configYaml+"```",
		nil,
	), nil
}

func configApply(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	config, ok := params.GetArguments()["config"].(string)
	if !ok || config == "" {
		return api.NewToolCallResult("", fmt.Errorf("config argument is required")), nil
	}

	result, err := params.ResourcesCreateOrUpdate(params, config)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to apply ExternalSecretsConfig: %w", err)), nil
	}

	marshalledYaml, err := output.MarshalYaml(result)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal result: %w", err)), nil
	}

	return api.NewToolCallResult(
		"# ExternalSecretsConfig applied successfully\n```yaml\n"+marshalledYaml+"```",
		nil,
	), nil
}
