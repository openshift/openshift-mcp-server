package mcp

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/utils/commandbuilder"
)

const (
	// DefaultMaxOutputLines defines the maximum number of lines to return from command output
	DefaultMaxOutputLines = 100
)

// ConntrackSummaryPattern matches conntrack informational stderr printed on successful -L/--dump.
// Format: "conntrack v<version> (conntrack-tools): <count> flow entries have been shown."
var ConntrackSummaryPattern = regexp.MustCompile(`^conntrack v\d+\.\d+\.\d+ \(conntrack-tools\): \d+ flow entries have been shown\.?$`)

// utilityExists checks if a utility/command exists in the container
func (s *MCPServer) utilityExists(ctx context.Context, namespace, node, utility string) error {
	cmd := commandbuilder.NewCommand(utility, "-V")
	_, stderr, err := s.runDebugNodeCommand(ctx, namespace, node, s.cfg.Image, cmd.Build(), "", "", 0)
	if err != nil {
		return fmt.Errorf("error while checking availability of the utility %s: %w", utility, err)
	}
	stderr = strings.TrimSpace(filterWarnings(stderr))
	if stderr != "" {
		return fmt.Errorf("utility %s is unavailable in configured image: %s", utility, stderr)
	}
	return nil
}

// filterWarnings filters out lines starting with "Warning" from the output
func filterWarnings(output string) string {
	if output == "" {
		return output
	}

	lines := strings.Split(output, "\n")
	var filteredLines []string

	for _, line := range lines {
		if !strings.Contains(line, "Warning") && !strings.Contains(line, "warning") && !strings.Contains(line, "WARNING") {
			filteredLines = append(filteredLines, line)
		}
	}

	return strings.Join(filteredLines, "\n")
}

// splitConntrackSummary separates conntrack informational stderr from other stderr.
// The conntrack informational stderr is printed when using conntrack CLI -L/--dump command.
// Format: "conntrack v<version> (conntrack-tools): <count> flow entries have been shown."
func splitConntrackSummary(output string) (summary, remaining string) {
	if output == "" {
		return "", ""
	}

	lines := strings.Split(output, "\n")
	var summaryLines []string
	var remainingLines []string

	for _, line := range lines {
		if ConntrackSummaryPattern.MatchString(line) {
			summaryLines = append(summaryLines, line)
		} else {
			remainingLines = append(remainingLines, line)
		}
	}

	return strings.Join(summaryLines, "\n"), strings.Join(remainingLines, "\n")
}
