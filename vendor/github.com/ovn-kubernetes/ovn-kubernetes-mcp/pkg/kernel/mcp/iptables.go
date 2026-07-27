package mcp

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/kernel/types"
	"github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/utils"
	"github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/utils/commandbuilder"
)

// GetIptables MCP handler for iptables operations.
// GetIptables retrieves iptables/ip6tables rules from a Kubernetes node.
// Automatically detects IPv6 and uses ip6tables when needed.
func (s *MCPServer) GetIptables(ctx context.Context, req *mcp.CallToolRequest, in types.ListIPTablesParams) (*mcp.CallToolResult, types.Result, error) {
	// If timeout is specified, create a new context with timeout
	var cancel context.CancelFunc
	ctx, cancel = in.TimeoutParams.WithTimeout(ctx)
	if cancel != nil {
		defer cancel()
	}

	err := s.utilityExists(ctx, in.Namespace, in.Node, "iptables")
	if err != nil {
		return nil, types.Result{}, fmt.Errorf("error while getting list of iptables rules: failed to verify iptables utility availability in configured image: %w", err)
	}

	if err := validateTableName(in.Table); err != nil {
		return nil, types.Result{}, fmt.Errorf("error while getting list of iptables rules: %w", err)
	}
	if err := validateIptablesCommand(in.Command); err != nil {
		return nil, types.Result{}, fmt.Errorf("error while getting list of iptables rules: %w", err)
	}

	if err := utils.ValidateSafeString(in.FilterParameters, "filter parameters", true, utils.ShellMetaCharactersTypeDefault); err != nil {
		return nil, types.Result{}, fmt.Errorf("error while getting list of iptables rules: %w", err)
	}

	table := strings.TrimSpace(in.Table)
	command := strings.TrimSpace(in.Command)
	cmd := commandbuilder.NewCommand(iptablesCommand(in.FilterParameters))
	// Defaults to 'filter' table when not specified
	cmd.AddIf(table == "", "-t", "filter")
	cmd.AddIfNotEmpty(table, "-t", table)
	// Defaults to -L (list) when command not specified
	cmd.AddIf(command == "", "-L")
	cmd.AddIfNotEmpty(command, command)
	// FilterParameters are invalid with -S/--list-rules command
	cmd.AddIf(in.FilterParameters != "" && command != "-S" && command != "--list-rules", strings.Fields(in.FilterParameters)...)

	stdout, stderr, err := s.executeCommand(ctx, in.Namespace, in.Node, cmd.Build())
	if err != nil {
		return nil, types.Result{}, fmt.Errorf("error while getting list of iptables rules: %w", err)
	}

	if stderr != "" {
		return nil, types.Result{}, fmt.Errorf("error while running command: %s", stderr)
	}

	// Strip empty lines from the output
	lines := utils.StripEmptyLines(strings.Split(stdout, "\n"))
	// Apply the head and tail parameters to the lines
	lines = in.HeadTailParams.Apply(lines, DefaultMaxOutputLines)
	// Join the lines back into a single string
	stdout = strings.Join(lines, "\n")
	return nil, types.Result{Data: stdout}, nil
}

// iptablesCommand determines whether to use iptables or ip6tables.
func iptablesCommand(filterParameters string) string {
	for _, item := range strings.Split(filterParameters, " ") {
		// Use of ip6tables CLI is required while either --ipv6 or -6 flag is mentioned
		if item == "--ipv6" || (strings.Contains(item, "-") && strings.Contains(item, "6")) {
			return "ip6tables"
		}
	}
	return "iptables"
}

// validateTableName validates iptables table name
func validateTableName(table string) error {
	if table == "" {
		return nil
	}

	if _, err := strconv.Atoi(table); err == nil {
		return fmt.Errorf("invalid table name: %s", table)
	}
	validTables := map[string]bool{
		"filter":   true,
		"nat":      true,
		"mangle":   true,
		"raw":      true,
		"security": true,
	}

	if !validTables[strings.TrimSpace(table)] {
		return fmt.Errorf("invalid table name: %s", table)
	}
	return nil
}

// validateIptablesCommand only allow list operation.
func validateIptablesCommand(command string) error {
	if command == "" {
		return nil
	}

	if _, err := strconv.Atoi(command); err == nil {
		return fmt.Errorf("invalid iptables command: %s", command)
	}
	validCommand := map[string]bool{
		"-L":           true,
		"-S":           true,
		"--list":       true,
		"--list-rules": true,
	}

	if !validCommand[strings.TrimSpace(command)] {
		return fmt.Errorf("invalid iptables command: %s", command)
	}
	return nil
}
