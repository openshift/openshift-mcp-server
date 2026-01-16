package externalsecrets

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
)

const (
	ingressNamespace             = "openshift-ingress"
	defaultIngressControllerName = "default"
	routerContainerName          = "router"
)

func initShowTools() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        "router_show_config",
				Description: `Tool to show router's configuration.`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"pod": {
							Type:        "string",
							Description: "Router pod name (optional, chooses any existing if not provided)",
						},
					},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Router: show config",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(false),
				},
			},
			Handler: showConfigHandler,
		},
		{
			Tool: api.Tool{
				Name:        "router_show_info",
				Description: `Tool to get HAProxy runtime information from the router.`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"pod": {
							Type:        "string",
							Description: "Router pod name (optional, chooses any existing if not provided)",
						},
					},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Router: show info",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(false),
				},
			},
			Handler: showInfoHandler,
		},
		{
			Tool: api.Tool{
				Name:        "router_show_sessions",
				Description: `Tool to view all active sessions in the router.`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"pod": {
							Type:        "string",
							Description: "Router pod name (optional, chooses any existing if not provided)",
						},
					},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Router: show sessions",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(false),
				},
			},
			Handler: showSessionsHandler,
		},
	}
}

func showConfigHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	var results []string

	pod, ok := params.GetArguments()["pod"].(string)
	if !ok || pod == "" {
		p, err := getAnyRouterPod(params, defaultIngressControllerName)
		if err != nil {
			results = append(results, "# Router configuration")
			results = append(results, fmt.Sprintf("Error getting router pod: %v", err))
			return api.NewToolCallResult(strings.Join(results, "\n"), nil), nil
		}
		pod = p
	}

	out, err := kubernetes.NewCore(params).PodsExec(params.Context, ingressNamespace, pod, routerContainerName, []string{"cat", "/var/lib/haproxy/conf/haproxy.config"})
	if err != nil {
		results = append(results, fmt.Sprintf("# Router configuration (pod: %s)", pod))
		results = append(results, fmt.Sprintf("Error showing router configuration from pod %q: %v", pod, err))
	} else {
		results = append(results, fmt.Sprintf("# Router configuration (pod: %s)", pod))
		results = append(results, "```")
		results = append(results, out)
		results = append(results, "```")
	}

	return api.NewToolCallResult(strings.Join(results, "\n"), nil), nil
}

func showInfoHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	var results []string

	pod, ok := params.GetArguments()["pod"].(string)
	if !ok || pod == "" {
		p, err := getAnyRouterPod(params, defaultIngressControllerName)
		if err != nil {
			results = append(results, "# Router HAProxy info")
			results = append(results, fmt.Sprintf("Error getting router pod: %v", err))
			return api.NewToolCallResult(strings.Join(results, "\n"), nil), nil
		}
		pod = p
	}

	out, err := kubernetes.NewCore(params).PodsExec(params.Context, ingressNamespace, pod, routerContainerName, []string{"sh", "-c", "echo 'show info' | socat stdio /var/lib/haproxy/run/haproxy.sock"})
	if err != nil {
		results = append(results, fmt.Sprintf("# Router HAProxy info (pod: %s)", pod))
		results = append(results, fmt.Sprintf("Error getting HAProxy info from pod %q: %v", pod, err))
	} else {
		results = append(results, fmt.Sprintf("# Router HAProxy info (pod: %s)", pod))
		results = append(results, "```")
		results = append(results, out)
		results = append(results, "```")
	}

	return api.NewToolCallResult(strings.Join(results, "\n"), nil), nil
}

func showSessionsHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	var results []string

	pod, ok := params.GetArguments()["pod"].(string)
	if !ok || pod == "" {
		p, err := getAnyRouterPod(params, defaultIngressControllerName)
		if err != nil {
			results = append(results, "# Router active sessions")
			results = append(results, fmt.Sprintf("Error getting router pod: %v", err))
			return api.NewToolCallResult(strings.Join(results, "\n"), nil), nil
		}
		pod = p
	}

	out, err := kubernetes.NewCore(params).PodsExec(params.Context, ingressNamespace, pod, routerContainerName, []string{"sh", "-c", "echo 'show sess all' | socat stdio /var/lib/haproxy/run/haproxy.sock"})
	if err != nil {
		results = append(results, fmt.Sprintf("# Router active sessions (pod: %s)", pod))
		results = append(results, fmt.Sprintf("Error getting active sessions from pod %q: %v", pod, err))
	} else {
		results = append(results, fmt.Sprintf("# Router active sessions (pod: %s)", pod))
		results = append(results, "```")
		results = append(results, out)
		results = append(results, "```")
	}

	return api.NewToolCallResult(strings.Join(results, "\n"), nil), nil
}

func getAnyRouterPod(params api.ToolHandlerParams, icName string) (string, error) {
	podGVK := &schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Pod",
	}
	pods, err := kubernetes.NewCore(params).ResourcesList(params, podGVK, ingressNamespace, api.ListOptions{
		ListOptions: metav1.ListOptions{
			LabelSelector: "ingresscontroller.operator.openshift.io/deployment-ingresscontroller=" + icName,
		},
		AsTable: false,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list router pods: %v", err)
	}
	podsMap := pods.UnstructuredContent()
	if items, ok := podsMap["items"].([]interface{}); ok {
		for _, item := range items {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if metadata, ok := itemMap["metadata"].(map[string]interface{}); ok {
					if podName, ok := metadata["name"].(string); ok {
						return podName, nil
					}
				}
			}
		}
	}
	return "", errors.New("no router pod found")
}
