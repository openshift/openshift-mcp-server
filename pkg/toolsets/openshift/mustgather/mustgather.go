package mustgather

import (
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

		Handler: mustgather.PlanMustGather,
	}}
}
