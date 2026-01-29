package mustgather

import (
	"context"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/ocp/mustgather"
	"github.com/google/jsonschema-go/jsonschema"
	"k8s.io/utils/ptr"
)

func Tools() []api.ServerTool {
	return []api.ServerTool{{
		Tool: api.Tool{
			Name:        "plan_mustgather",
			Description: "Plan for collecting a must-gather archive from an OpenShift cluster, must-gather is a tool for collecting cluster data related to debugging and troubleshooting like logs, kubernetes resources, etc.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"node_name": {
						Type:        "string",
						Description: "Optional node to run the mustgather pod. If not provided, a random control-plane node will be selected automatically",
					},
					"node_selector": {
						Type:        "string",
						Description: "Optional node label selector to use, only relevant when specifying a command and image which needs to capture data on a set of cluster nodes simultaneously",
					},
					"host_network": {
						Type:        "boolean",
						Description: "Optionally run the must-gather pods in the host network of the node. This is only relevant if a specific gather image needs to capture host-level data",
					},
					"gather_command": {
						Type:        "string",
						Description: "Optionally specify a custom gather command to run a specialized script, eg. /usr/bin/gather_audit_logs",
						Default:     api.ToRawMessage("/usr/bin/gather"),
					},
					"all_component_images": {
						Type:        "boolean",
						Description: "Optional when enabled, collects and runs multiple must gathers for all operators and components on the cluster that have an annotated must-gather image available",
					},
					"images": {
						Type:        "array",
						Description: "Optional list of images to use for gathering custom information about specific operators or cluster components. If not specified, OpenShift's default must-gather image will be used by default",
						Items: &jsonschema.Schema{
							Type: "string",
						},
					},
					"source_dir": {
						Type:        "string",
						Description: "Optional to set a specific directory where the pod will copy gathered data from",
						Default:     api.ToRawMessage("/must-gather"),
					},
					"timeout": {
						Type:        "string",
						Description: "Timeout of the gather process eg. 30s, 6m20s, or 2h10m30s",
					},
					"namespace": {
						Type:        "string",
						Description: "Optional to specify an existing privileged namespace where must-gather pods should run. If not provided, a temporary namespace will be created",
					},
					"keep_resources": {
						Type:        "boolean",
						Description: "Optional to retain all temporary resources when the mustgather completes, otherwise temporary resources created will be advised to be cleaned up",
					},
					"since": {
						Type:        "string",
						Description: "Optional to collect logs newer than a relative duration like 5s, 2m5s, or 3h6m10s. If unspecified, all available logs will be collected",
					},
				},
			},
			Annotations: api.ToolAnnotations{
				Title:           "MustGather: Plan",
				ReadOnlyHint:    ptr.To(true),
				DestructiveHint: ptr.To(false),
				IdempotentHint:  ptr.To(false),
				OpenWorldHint:   ptr.To(true),
			},
		},

		Handler: planMustGather,
	}}
}

// planMustGather is the handler that parses arguments and calls the core
// PlanMustGather tool.
func planMustGather(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()
	ctx := context.Background()

	mgParams := mustgather.PlanMustGatherParams{}

	if args["node_name"] != nil {
		mgParams.NodeName = args["node_name"].(string)
	}

	if args["node_selector"] != nil {
		mgParams.NodeSelector = mustgather.ParseNodeSelector(args["node_selector"].(string))
	}

	if args["host_network"] != nil {
		mgParams.HostNetwork = args["host_network"].(bool)
	}

	if args["source_dir"] != nil {
		mgParams.SourceDir = args["source_dir"].(string)
	}

	if args["namespace"] != nil {
		mgParams.Namespace = args["namespace"].(string)
	}

	if args["keep_resources"] != nil {
		mgParams.KeepResources = args["keep_resources"].(bool)
	}

	if args["gather_command"] != nil {
		mgParams.GatherCommand = args["gather_command"].(string)
	}

	if args["all_component_images"] != nil {
		mgParams.AllImages = args["all_component_images"].(bool)
	}

	if args["images"] != nil {
		if imagesArg, ok := args["images"].([]interface{}); ok {
			for _, img := range imagesArg {
				if imgStr, ok := img.(string); ok {
					mgParams.Images = append(mgParams.Images, imgStr)
				}
			}
		}
	}

	if args["timeout"] != nil {
		mgParams.Timeout = args["timeout"].(string)
	}

	if args["since"] != nil {
		mgParams.Since = args["since"].(string)
	}

	// params implements api.KubernetesClient
	result, err := mustgather.PlanMustGather(ctx, params, mgParams)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	return api.NewToolCallResult(result, nil), nil
}
