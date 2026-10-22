package mustgather

import (
	"fmt"
	"strings"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	mg "github.com/containers/kubernetes-mcp-server/pkg/ocp/mustgather"
	"github.com/google/jsonschema-go/jsonschema"
	"k8s.io/utils/ptr"
)

// archiveIDDescription is the shared description for the archive_id
// argument used by every mustgather_* tool.
const archiveIDDescription = "Must-gather archive ID as returned by mustgather_list (format: mg-XXXXYYYYYYYY, e.g. mg-384226d712f0). Call mustgather_list first to discover available archives."

// archiveIDProperty returns the JSON schema property for the required
// archive_id argument.
func archiveIDProperty() *jsonschema.Schema {
	return &jsonschema.Schema{Type: "string", Description: archiveIDDescription}
}

func initList() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        "mustgather_list",
				Description: "List the must-gather archives discovered under the configured directories. Returns each archive's archive_id, which must be passed to the other mustgather_* tools.",
				Annotations: api.ToolAnnotations{
					Title:        "List Must-Gather Archives",
					ReadOnlyHint: ptr.To(true),
				},
				InputSchema: &jsonschema.Schema{
					Type:       "object",
					Properties: map[string]*jsonschema.Schema{},
				},
			},
			RBAC:         api.RBACNone(),
			Handler:      mustgatherList,
			ClusterAware: ptr.To(false),
		},
	}
}

func mustgatherList(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	cfg := configFromParams(params)
	var dirs []string
	if cfg != nil {
		dirs = cfg.MustGatherDirs
	}
	if len(dirs) == 0 {
		return api.NewToolCallResult("", fmt.Errorf("no must-gather directories configured; set mustgather_dirs in the [toolset_configs.\"openshift/mustgather\"] section of the config file to a directory containing must-gather archives")), nil
	}

	result := api.NewBuilderWithBudget(api.ResultLimit(params.Context))
	body := result.Scratch(40)
	if body.Truncated() {
		result.WriteString("Found must-gather archives:\n\n")
		result.WriteString("\n")
		return api.NewToolCallResult(result.String(), nil), nil
	}
	count := 0
	backendLimit, err := kubernetes.BackendLimit(params.Config)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	err = visitArchivesLimited(params.Context, dirs, backendLimit, func(a ArchiveInfo) bool {
		if !body.Fits(0) {
			body.WriteString("\n")
			return false
		}
		count++
		body.WriteString(fmt.Sprintf("- archive_id: %s\n  path: %s%s\n", a.ID, mg.LocalURIPrefix, a.Path))
		if a.Version != "" {
			body.WriteString(fmt.Sprintf("  version: %s\n", a.Version))
		}
		if a.Timestamp != "" {
			body.WriteString(fmt.Sprintf("  timestamp: %s\n", a.Timestamp))
		}
		body.WriteString("\n")
		return !body.Truncated()
	})
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	if count == 0 {
		return api.NewToolCallResult(fmt.Sprintf("No must-gather archives found under the configured directories: %s\n", strings.Join(dirs, ", ")), nil), nil
	}

	result.WriteString(fmt.Sprintf("Found %d must-gather archive(s):\n\n", count))
	text, cut := body.Prefix()
	result.WriteCut(text, cut)
	if !result.Truncated() {
		result.WriteString("Pass the archive_id value to the other mustgather_* tools (e.g. mustgather_resources_list, mustgather_pod_logs_get).\n")
	}
	return api.NewToolCallResult(result.String(), nil), nil
}
