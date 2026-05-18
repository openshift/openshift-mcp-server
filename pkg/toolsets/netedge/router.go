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

	defaultConfigTailLines int64 = 200
	defaultSessionLimit    int64 = 50
)

var podGVR = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "pods",
}

var haproxySectionKeywords = []string{"global", "defaults", "frontend", "backend", "listen"}

func initRouter() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        "get_router_config",
				Description: `Retrieve the current router's HAProxy configuration from the cluster. Supports filtering by section type (global/defaults/frontend/backend), substring filter on section headers, and line-count limiting via tail_lines.`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"pod": {
							Type:        "string",
							Description: "Router pod name (optional, chooses any existing if not provided)",
						},
						"tail_lines": {
							Type:        "integer",
							Description: "Maximum number of lines to return from the end of the config output (default: 200)",
							Default:     api.ToRawMessage(defaultConfigTailLines),
							Minimum:     ptr.To(float64(1)),
						},
						"section": {
							Type:        "string",
							Description: "Filter to a specific HAProxy config section type",
							Enum:        []any{"global", "defaults", "frontend", "backend", "listen"},
						},
						"filter": {
							Type:        "string",
							Description: "Substring filter applied to section headers (e.g. a route or backend name). Only sections whose header contains this string are returned.",
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
				Description: `Retrieve active sessions from the router. Supports limiting the number of sessions returned and filtering by substring (e.g. backend name or source IP).`,
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"pod": {
							Type:        "string",
							Description: "Router pod name (optional, chooses any existing if not provided)",
						},
						"limit": {
							Type:        "integer",
							Description: "Maximum number of session blocks to return (default: 50)",
							Default:     api.ToRawMessage(defaultSessionLimit),
							Minimum:     ptr.To(float64(1)),
						},
						"filter": {
							Type:        "string",
							Description: "Substring filter applied to each session block. Only sessions containing this string are returned (e.g. a backend name or source IP).",
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

func getRouterConfig(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p := api.WrapParams(params)
	pod := p.OptionalString("pod", "")
	tailLines := p.OptionalInt64("tail_lines", defaultConfigTailLines)
	section := p.OptionalString("section", "")
	filter := p.OptionalString("filter", "")
	if err := p.Err(); err != nil {
		return api.NewToolCallResult("", fmt.Errorf("invalid parameters: %w", err)), nil
	}

	var results []string

	if pod == "" {
		resolved, err := getAnyRouterPod(params, defaultIngressControllerName)
		if err != nil {
			results = append(results, "# Router configuration")
			results = append(results, fmt.Sprintf("Error getting router pod: %v", err))
			return api.NewToolCallResult(strings.Join(results, "\n"), err), nil
		}
		pod = resolved
	}

	out, err := kubernetes.NewCore(params).PodsExec(params.Context, ingressNamespace, pod, routerContainerName, []string{"cat", "/var/lib/haproxy/conf/haproxy.config"})
	if err != nil {
		results = append(results, fmt.Sprintf("# Router configuration (pod: %s)", pod))
		results = append(results, fmt.Sprintf("Error showing router configuration from pod %q: %v", pod, err))
		return api.NewToolCallResult(strings.Join(results, "\n"), err), nil
	}

	truncated, totalLines, shownLines, wasTruncated := truncateConfigOutput(out, tailLines, section, filter)

	results = append(results, fmt.Sprintf("# Router configuration (pod: %s)", pod))
	if wasTruncated {
		results = append(results, fmt.Sprintf("**Output truncated**: showing %d of %d total lines. Use `section`, `filter`, or increase `tail_lines` to refine.", shownLines, totalLines))
	} else if section != "" || filter != "" {
		results = append(results, fmt.Sprintf("Showing %d lines (filtered from %d total).", shownLines, totalLines))
	}
	results = append(results, "```")
	results = append(results, truncated)
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

func getRouterSessions(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p := api.WrapParams(params)
	pod := p.OptionalString("pod", "")
	limit := p.OptionalInt64("limit", defaultSessionLimit)
	filter := p.OptionalString("filter", "")
	if err := p.Err(); err != nil {
		return api.NewToolCallResult("", fmt.Errorf("invalid parameters: %w", err)), nil
	}

	var results []string

	if pod == "" {
		resolved, err := getAnyRouterPod(params, defaultIngressControllerName)
		if err != nil {
			results = append(results, "# Router active sessions")
			results = append(results, fmt.Sprintf("Error getting router pod: %v", err))
			return api.NewToolCallResult(strings.Join(results, "\n"), err), nil
		}
		pod = resolved
	}

	out, err := kubernetes.NewCore(params).PodsExec(params.Context, ingressNamespace, pod, routerContainerName, []string{"sh", "-c", "echo 'show sess all' | socat stdio /var/lib/haproxy/run/haproxy.sock"})
	if err != nil {
		results = append(results, fmt.Sprintf("# Router active sessions (pod: %s)", pod))
		results = append(results, fmt.Sprintf("Error getting active sessions from pod %q: %v", pod, err))
		return api.NewToolCallResult(strings.Join(results, "\n"), err), nil
	}

	truncated, totalSessions, shownSessions, wasTruncated := truncateSessionsOutput(out, limit, filter)

	results = append(results, fmt.Sprintf("# Router active sessions (pod: %s)", pod))
	if wasTruncated {
		results = append(results, fmt.Sprintf("**Output truncated**: showing %d of %d total sessions. Use `filter` to narrow results or increase `limit` to see more.", shownSessions, totalSessions))
	} else if filter != "" {
		results = append(results, fmt.Sprintf("Showing %d sessions (filtered from %d total).", shownSessions, totalSessions))
	}
	results = append(results, "```")
	results = append(results, truncated)
	results = append(results, "```")

	return api.NewToolCallResult(strings.Join(results, "\n"), nil), nil
}

func configSectionType(line string) string {
	for _, kw := range haproxySectionKeywords {
		if strings.HasPrefix(line, kw) && (len(line) == len(kw) || line[len(kw)] == ' ' || line[len(kw)] == '\t') {
			return kw
		}
	}
	return ""
}

func truncateConfigOutput(output string, tailLines int64, section, filter string) (string, int, int, bool) {
	if output == "" {
		return "", 0, 0, false
	}

	allLines := strings.Split(output, "\n")
	totalLines := len(allLines)

	if section == "" && filter == "" {
		if int64(totalLines) <= tailLines {
			return output, totalLines, totalLines, false
		}
		result := allLines[totalLines-int(tailLines):]
		return strings.Join(result, "\n"), totalLines, len(result), true
	}

	filterLower := strings.ToLower(filter)
	var resultLines []string
	include := false
	for _, line := range allLines {
		if st := configSectionType(line); st != "" {
			include = (section == "" || st == section) && (filter == "" || strings.Contains(strings.ToLower(line), filterLower))
		}
		if include {
			resultLines = append(resultLines, line)
		}
	}
	shownLines := len(resultLines)

	if int64(shownLines) <= tailLines {
		return strings.Join(resultLines, "\n"), totalLines, shownLines, false
	}
	result := resultLines[shownLines-int(tailLines):]
	return strings.Join(result, "\n"), totalLines, len(result), true
}

func truncateSessionsOutput(output string, limit int64, filter string) (string, int, int, bool) {
	if output == "" {
		return "", 0, 0, false
	}

	lines := strings.Split(output, "\n")
	var blocks [][]string
	var current []string

	for _, line := range lines {
		if strings.HasPrefix(line, "0x") {
			if len(current) > 0 {
				blocks = append(blocks, current)
			}
			current = []string{line}
		} else if len(current) > 0 {
			current = append(current, line)
		}
	}
	if len(current) > 0 {
		blocks = append(blocks, current)
	}

	totalSessions := len(blocks)

	if filter != "" {
		filterLower := strings.ToLower(filter)
		var filtered [][]string
		for _, block := range blocks {
			if blockContainsSubstring(block, filterLower) {
				filtered = append(filtered, block)
			}
		}
		blocks = filtered
	}

	truncated := int64(len(blocks)) > limit
	if truncated {
		blocks = blocks[:limit]
	}

	var resultLines []string
	for _, block := range blocks {
		resultLines = append(resultLines, block...)
	}
	return strings.Join(resultLines, "\n"), totalSessions, len(blocks), truncated
}

func blockContainsSubstring(block []string, substr string) bool {
	for _, line := range block {
		if strings.Contains(strings.ToLower(line), substr) {
			return true
		}
	}
	return false
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
