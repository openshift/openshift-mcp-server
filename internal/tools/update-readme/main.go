package main

import (
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	internalk8s "github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets"

	_ "github.com/containers/kubernetes-mcp-server/pkg/toolsets/config"
	_ "github.com/containers/kubernetes-mcp-server/pkg/toolsets/core"
	_ "github.com/containers/kubernetes-mcp-server/pkg/toolsets/helm"
)

type OpenShift struct{}

func (o *OpenShift) IsOpenShift(ctx context.Context) bool {
	return true
}

var _ internalk8s.Openshift = (*OpenShift)(nil)

func main() {
	readmePath, err := resolveReadmePath(os.Args[1:])
	if err != nil {
		panic(err)
	}

	readme, err := os.ReadFile(readmePath)
	if err != nil {
		panic(err)
	}
	// Available Toolsets
	toolsetsList := toolsets.Toolsets()
	maxNameLen, maxDescLen := len("Toolset"), len("Description")
	for _, toolset := range toolsetsList {
		nameLen := len(toolset.GetName())
		descLen := len(toolset.GetDescription())
		if nameLen > maxNameLen {
			maxNameLen = nameLen
		}
		if descLen > maxDescLen {
			maxDescLen = descLen
		}
	}
	availableToolsets := strings.Builder{}
	availableToolsets.WriteString(fmt.Sprintf("| %-*s | %-*s |\n", maxNameLen, "Toolset", maxDescLen, "Description"))
	availableToolsets.WriteString(fmt.Sprintf("|-%s-|-%s-|\n", strings.Repeat("-", maxNameLen), strings.Repeat("-", maxDescLen)))
	for _, toolset := range toolsetsList {
		availableToolsets.WriteString(fmt.Sprintf("| %-*s | %-*s |\n", maxNameLen, toolset.GetName(), maxDescLen, toolset.GetDescription()))
	}
	updated := replaceBetweenMarkers(
		string(readme),
		"<!-- AVAILABLE-TOOLSETS-START -->",
		"<!-- AVAILABLE-TOOLSETS-END -->",
		availableToolsets.String(),
	)

	// Available Toolset Tools
	toolsetTools := strings.Builder{}
	for _, toolset := range toolsetsList {
		toolsetTools.WriteString("<details>\n\n<summary>" + toolset.GetName() + "</summary>\n\n")
		tools := toolset.GetTools(&OpenShift{})
		for _, tool := range tools {
			toolsetTools.WriteString(fmt.Sprintf("- **%s** - %s\n", tool.Tool.Name, tool.Tool.Description))
			for _, propName := range slices.Sorted(maps.Keys(tool.Tool.InputSchema.Properties)) {
				property := tool.Tool.InputSchema.Properties[propName]
				toolsetTools.WriteString(fmt.Sprintf("  - `%s` (`%s`)", propName, property.Type))
				if slices.Contains(tool.Tool.InputSchema.Required, propName) {
					toolsetTools.WriteString(" **(required)**")
				}
				toolsetTools.WriteString(fmt.Sprintf(" - %s\n", property.Description))
			}
			toolsetTools.WriteString("\n")
		}
		toolsetTools.WriteString("</details>\n\n")
	}
	updated = replaceBetweenMarkers(
		updated,
		"<!-- AVAILABLE-TOOLSETS-TOOLS-START -->",
		"<!-- AVAILABLE-TOOLSETS-TOOLS-END -->",
		toolsetTools.String(),
	)

	if err := os.WriteFile(readmePath, []byte(updated), 0o644); err != nil {
		panic(err)
	}
}

func resolveReadmePath(args []string) (string, error) {
	var requested string
	switch len(args) {
	case 0:
		requested = "README.md"
	case 1:
		requested = args[0]
	default:
		return "", fmt.Errorf("Error: Provide at most one README.md argument")
	}

	cleanPath := filepath.Clean(requested)
	if cleanPath != "README.md" {
		return "", fmt.Errorf("Error: This tool can only update the repository root README.md")
	}

	repoRoot, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("determine working directory: %w", err)
	}

	absoluteRepoRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}

	return filepath.Join(absoluteRepoRoot, "README.md"), nil
}

func replaceBetweenMarkers(content, startMarker, endMarker, replacement string) string {
	startIdx := strings.Index(content, startMarker)
	if startIdx == -1 {
		return content
	}
	endIdx := strings.Index(content, endMarker)
	if endIdx == -1 || endIdx <= startIdx {
		return content
	}
	return content[:startIdx+len(startMarker)] + "\n\n" + replacement + "\n" + content[endIdx:]
}
