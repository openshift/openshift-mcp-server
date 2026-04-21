package netedge

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

var (
	podGVR = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "pods",
	}
)

func initRouter() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        "get_router_config",
				Description: `Retrieve the current router's HAProxy configuration from the cluster.`,
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
					Title:           "Get Router Config",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: getRouterConfig,
		},
		{
			Tool: api.Tool{
				Name:        "get_router_info",
				Description: `Retrieve HAProxy runtime information from the router.`,
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
					Title:           "Get Router Info",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: getRouterInfo,
		},
		{
			Tool: api.Tool{
				Name:        "get_router_sessions",
				Description: `Retrieve all active sessions from the router.`,
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
					Title:           "Get Router Sessions",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: getRouterSessions,
		},
	}
}

// getRouterConfig requires a live cluster as it reads the HAProxy configuration
// from a running router pod via exec. It cannot work against offline data (must-gather).
func getRouterConfig(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
        var results []string

        pod, ok := params.GetArguments()["pod"].(string)
        if !ok || pod == "" {
                p, err := getAnyRouterPod(params, defaultIngressControllerName)
                if err != nil {
                        results = append(results, "# Router configuration")
                        results = append(results, fmt.Sprintf("Error getting router pod: %v", err))
                        return api.NewToolCallResult(strings.Join(results, "\n"), err), nil
                }
                pod = p
        }

        out, err := kubernetes.NewCore(params).PodsExec(params.Context, ingressNamespace, pod, routerContainerName, []string{"cat", "/var/lib/haproxy/conf/haproxy.config"})
        if err != nil {
                results = append(results, fmt.Sprintf("# Router configuration (pod: %s)", pod))
                results = append(results, fmt.Sprintf("Error showing router configuration from pod %q: %v", pod, err))
                return api.NewToolCallResult(strings.Join(results, "\n"), err), nil
        }

        results = append(results, fmt.Sprintf("# Router configuration (pod: %s)", pod))
        results = append(results, "```")
        results = append(results, out)
        results = append(results, "```")

        return api.NewToolCallResult(strings.Join(results, "\n"), nil), nil
}
// getRouterInfo requires a live cluster as it queries the HAProxy admin socket
// via exec on a running router pod. It cannot work against offline data (must-gather).
func getRouterInfo(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
        var results []string

        pod, ok := params.GetArguments()["pod"].(string)
        if !ok || pod == "" {
                p, err := getAnyRouterPod(params, defaultIngressControllerName)
                if err != nil {
                        results = append(results, "# Router HAProxy info")
                        results = append(results, fmt.Sprintf("Error getting router pod: %v", err))
                        return api.NewToolCallResult(strings.Join(results, "\n"), err), nil
                }
                pod = p
        }

        out, err := kubernetes.NewCore(params).PodsExec(params.Context, ingressNamespace, pod, routerContainerName, []string{"sh", "-c", "echo 'show info' | socat stdio /var/lib/haproxy/run/haproxy.sock"})
        if err != nil {
                results = append(results, fmt.Sprintf("# Router HAProxy info (pod: %s)", pod))
                results = append(results, fmt.Sprintf("Error getting HAProxy info from pod %q: %v", pod, err))
                return api.NewToolCallResult(strings.Join(results, "\n"), err), nil
        }

        results = append(results, fmt.Sprintf("# Router HAProxy info (pod: %s)", pod))
        results = append(results, "```")
        results = append(results, out)
        results = append(results, "```")

        return api.NewToolCallResult(strings.Join(results, "\n"), nil), nil
}
// getRouterSessions requires a live cluster as it queries the HAProxy admin socket
// via exec on a running router pod. It cannot work against offline data (must-gather).
func getRouterSessions(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
        var results []string

        pod, ok := params.GetArguments()["pod"].(string)
        if !ok || pod == "" {
                p, err := getAnyRouterPod(params, defaultIngressControllerName)
                if err != nil {
                        results = append(results, "# Router active sessions")
                        results = append(results, fmt.Sprintf("Error getting router pod: %v", err))
                        return api.NewToolCallResult(strings.Join(results, "\n"), err), nil
                }
                pod = p
        }

        out, err := kubernetes.NewCore(params).PodsExec(params.Context, ingressNamespace, pod, routerContainerName, []string{"sh", "-c", "echo 'show sess all' | socat stdio /var/lib/haproxy/run/haproxy.sock"})
        if err != nil {
                results = append(results, fmt.Sprintf("# Router active sessions (pod: %s)", pod))
                results = append(results, fmt.Sprintf("Error getting active sessions from pod %q: %v", pod, err))
                return api.NewToolCallResult(strings.Join(results, "\n"), err), nil
        }

        results = append(results, fmt.Sprintf("# Router active sessions (pod: %s)", pod))
        results = append(results, "```")
        results = append(results, out)
        results = append(results, "```")

        return api.NewToolCallResult(strings.Join(results, "\n"), nil), nil
}
func getAnyRouterPod(params api.ToolHandlerParams, icName string) (string, error) {
	pods, err := params.DynamicClient().Resource(podGVR).Namespace(ingressNamespace).List(params.Context, metav1.ListOptions{
		LabelSelector: "ingresscontroller.operator.openshift.io/deployment-ingresscontroller=" + icName,
		FieldSelector: "status.phase=Running",
	})
	if err != nil {
		return "", fmt.Errorf("failed to list router pods: %v", err)
	}
	for _, pod := range pods.Items {
		return pod.GetName(), nil
	}
	return "", errors.New("no running router pod found")
}
