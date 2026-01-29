package mustgather

import (
	"strings"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/ocp/mustgather"
)

// Prompts returns the ServerPrompt definitions for must-gather operations.
func Prompts() []api.ServerPrompt {
	return []api.ServerPrompt{{
		Prompt: api.Prompt{
			Name:        "plan_mustgather",
			Title:       "Plan a must-gather collection",
			Description: "Plan for collecting a must-gather archive from an OpenShift cluster. Must-gather is a tool for collecting cluster data related to debugging and troubleshooting like logs, kubernetes resources, etc.",
			Arguments: []api.PromptArgument{
				{
					Name:        "node_name",
					Description: "Specific node name to run must-gather pod on",
					Required:    false,
				},
				{
					Name:        "node_selector",
					Description: "Node selector in key=value,key2=value2 format to filter nodes for the pod",
					Required:    false,
				},
				{
					Name:        "source_dir",
					Description: "Custom gather directory inside pod (default: /must-gather)",
					Required:    false,
				},
				{
					Name:        "namespace",
					Description: "Privileged namespace to use for must-gather (auto-generated if not specified)",
					Required:    false,
				},
				{
					Name:        "gather_command",
					Description: "Custom gather command eg. /usr/bin/gather_audit_logs (default: /usr/bin/gather)",
					Required:    false,
				},
				{
					Name:        "timeout",
					Description: "Timeout duration for gather command (eg. 30m, 1h)",
					Required:    false,
				},
				{
					Name:        "since",
					Description: "Only gather data newer than this duration (eg. 5s, 2m5s, or 3h6m10s) defaults to all data.",
					Required:    false,
				},
				{
					Name:        "host_network",
					Description: "Use host network for must-gather pod (true/false)",
					Required:    false,
				},
				{
					Name:        "keep_resources",
					Description: "Keep pod resources after collection (true/false, default: false)",
					Required:    false,
				},
				{
					Name:        "all_component_images",
					Description: "Include must-gather images from all installed operators (true/false)",
					Required:    false,
				},
				{
					Name:        "images",
					Description: "Comma-separated list of custom must-gather container images",
					Required:    false,
				},
			},
		},
		Handler: planMustGatherHandler,
	}}
}

// planMustGatherHandler is the handler that parses arguments and calls the core
// PlanMustGather function.
func planMustGatherHandler(params api.PromptHandlerParams) (*api.PromptCallResult, error) {
	args := params.GetArguments()

	mgParams := mustgather.PlanMustGatherParams{
		NodeName:      args["node_name"],
		NodeSelector:  mustgather.ParseNodeSelector(args["node_selector"]),
		SourceDir:     args["source_dir"],
		Namespace:     args["namespace"],
		GatherCommand: args["gather_command"],
		Timeout:       args["timeout"],
		Since:         args["since"],
		HostNetwork:   parseBool(args["host_network"]),
		KeepResources: parseBool(args["keep_resources"]),
		AllImages:     parseBool(args["all_component_images"]),
		Images:        parseImages(args["images"]),
	}

	// params embeds api.KubernetesClient
	result, err := mustgather.PlanMustGather(params.Context, params, mgParams)
	if err != nil {
		return nil, err
	}

	return api.NewPromptCallResult(
		"Must-gather plan generated successfully",
		[]api.PromptMessage{
			{
				Role: "user",
				Content: api.PromptContent{
					Type: "text",
					Text: formatMustGatherPrompt(result),
				},
			},
			{
				Role: "assistant",
				Content: api.PromptContent{
					Type: "text",
					Text: "I'll help you apply this must-gather plan to collect diagnostic data from your OpenShift cluster.",
				},
			},
		},
		nil,
	), nil
}

// parseBool parses a string value to boolean, returns false for empty or invalid values.
func parseBool(value string) bool {
	return strings.ToLower(strings.TrimSpace(value)) == "true"
}

// parseImages parses a comma-separated list of images into a slice.
func parseImages(value string) []string {
	if value == "" {
		return nil
	}
	var images []string
	for _, img := range strings.Split(value, ",") {
		img = strings.TrimSpace(img)
		if img != "" {
			images = append(images, img)
		}
	}
	return images
}

// formatMustGatherPrompt formats the must-gather plan result into a prompt for the LLM.
func formatMustGatherPrompt(planResult string) string {
	var sb strings.Builder

	sb.WriteString("# Must-Gather Collection Plan\n\n")
	sb.WriteString(planResult)
	sb.WriteString("\n---\n\n")
	sb.WriteString("**Please review the plan above and confirm if you want to proceed with applying these resources.**\n")

	return sb.String()
}
