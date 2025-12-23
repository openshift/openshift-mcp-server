package certmanager

import (
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/ptr"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/certmanager"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
)

func initLogs() []api.ServerTool {
	return []api.ServerTool{
		// controller_logs
		{
			Tool: api.Tool{
				Name: "certmanager_controller_logs",
				Description: `Get logs from the cert-manager controller.

The controller is responsible for:
- Watching Certificate resources
- Creating CertificateRequests
- Coordinating with Issuers
- Managing certificate lifecycle

Use this tool when certificates are not being created or updated.`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"tail": {
							Type:        "integer",
							Description: "Number of log lines to retrieve (default: 100)",
							Default:     api.ToRawMessage(100),
						},
					},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Cert-Manager: Controller Logs",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: controllerLogs,
		},
		// webhook_logs
		{
			Tool: api.Tool{
				Name: "certmanager_webhook_logs",
				Description: `Get logs from the cert-manager webhook.

The webhook is responsible for:
- Validating cert-manager resources
- Converting between API versions
- Mutating resources with defaults

Use this tool when you see admission webhook errors or validation failures.`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"tail": {
							Type:        "integer",
							Description: "Number of log lines to retrieve (default: 100)",
							Default:     api.ToRawMessage(100),
						},
					},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Cert-Manager: Webhook Logs",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: webhookLogs,
		},
		// cainjector_logs
		{
			Tool: api.Tool{
				Name: "certmanager_cainjector_logs",
				Description: `Get logs from the cert-manager CA injector.

The CA injector is responsible for:
- Injecting CA certificates into webhooks
- Injecting CA certificates into API services
- Managing caBundle annotations

Use this tool when webhook certificates are not being injected properly.`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"tail": {
							Type:        "integer",
							Description: "Number of log lines to retrieve (default: 100)",
							Default:     api.ToRawMessage(100),
						},
					},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Cert-Manager: CAInjector Logs",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: cainjectorLogs,
		},
	}
}

func controllerLogs(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	return getComponentLogs(params, certmanager.ControllerDeploymentName, "app=cert-manager")
}

func webhookLogs(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	return getComponentLogs(params, certmanager.WebhookDeploymentName, "app=cert-manager-webhook")
}

func cainjectorLogs(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	return getComponentLogs(params, certmanager.CAInjectorDeploymentName, "app=cert-manager-cainjector")
}

func getComponentLogs(params api.ToolHandlerParams, deploymentName, labelSelector string) (*api.ToolCallResult, error) {
	// List pods with the app label
	listOptions := kubernetes.ResourceListOptions{
		AsTable: false,
	}
	listOptions.LabelSelector = labelSelector

	pods, err := params.ResourcesList(params.Context, &certmanager.PodGVK, certmanager.CertManagerNamespace, listOptions)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to list pods for %s: %v", deploymentName, err)), nil
	}

	items, found, _ := unstructured.NestedSlice(pods.UnstructuredContent(), "items")
	if !found || len(items) == 0 {
		return api.NewToolCallResult(fmt.Sprintf("No pods found for deployment %s in namespace %s", deploymentName, certmanager.CertManagerNamespace), nil), nil
	}

	// Get the first pod name
	firstPod, ok := items[0].(map[string]interface{})
	if !ok {
		return api.NewToolCallResult("", fmt.Errorf("invalid pod structure")), nil
	}
	podName, _, _ := unstructured.NestedString(firstPod, "metadata", "name")

	// Build log options
	tail := int64(100)
	if t := params.GetArguments()["tail"]; t != nil {
		tail = int64(t.(float64))
	}

	logs, err := params.PodsLog(params.Context, certmanager.CertManagerNamespace, podName, "", false, tail)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get logs for pod %s: %v", podName, err)), nil
	}

	header := fmt.Sprintf("# Logs from %s (pod: %s)\n\n```\n", deploymentName, podName)
	footer := "\n```"

	return api.NewToolCallResult(header+logs+footer, nil), nil
}
