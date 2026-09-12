package mustgather

import (
	"fmt"
	"strings"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/google/jsonschema-go/jsonschema"
	"k8s.io/utils/ptr"
)

// archiveIDDescription is the shared description for the archive_id
// argument used by every mustgather_* tool.
const archiveIDDescription = "Must-gather archive ID as returned by mustgather_list (format: mg-XXXX-YYYYYYYY, e.g. mg-3842-26d712f0). Call mustgather_list first to discover available archives."

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
			Handler:      mustgatherList,
			ClusterAware: ptr.To(false),
		},
	}
}

func mustgatherList(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	dirs := params.GetMustGatherDirs()
	if len(dirs) == 0 {
		return api.NewToolCallResult("", fmt.Errorf("no must-gather directories configured; set --mustgather-dirs (or mustgather_dirs in config) to a directory containing must-gather archives")), nil
	}

	archives := discoverArchives(dirs)
	if len(archives) == 0 {
		return api.NewToolCallResult(fmt.Sprintf("No must-gather archives found under the configured directories: %s\n", strings.Join(dirs, ", ")), nil), nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Found %d must-gather archive(s):\n\n", len(archives))
	for _, a := range archives {
		fmt.Fprintf(&b, "- archive_id: %s\n", a.ID)
		fmt.Fprintf(&b, "  path: local://%s\n", a.Path)
		if a.Version != "" {
			fmt.Fprintf(&b, "  version: %s\n", a.Version)
		}
		if a.Timestamp != "" {
			fmt.Fprintf(&b, "  timestamp: %s\n", a.Timestamp)
		}
		b.WriteString("\n")
	}
	b.WriteString("Pass the archive_id value to the other mustgather_* tools (e.g. mustgather_resources_list, mustgather_pod_logs_get).\n")

	return api.NewToolCallResult(b.String(), nil), nil
}
