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

// GetIPCommandOutput MCP handler for ip utility operations.
// GetIPCommandOutput executes 'ip' utility commands on a node.
// Requires ip utility in the debug container image.
func (s *MCPServer) GetIPCommandOutput(ctx context.Context, req *mcp.CallToolRequest, in types.ListIPParams) (*mcp.CallToolResult, types.Result, error) {
	// If timeout is specified, create a new context with timeout
	var cancel context.CancelFunc
	ctx, cancel = in.TimeoutParams.WithTimeout(ctx)
	if cancel != nil {
		defer cancel()
	}

	err := s.utilityExists(ctx, in.Namespace, in.Node, "ip")
	if err != nil {
		return nil, types.Result{}, fmt.Errorf("error while getting ip data: failed to verify ip utility availability in configured image: %w", err)
	}

	if err := validateIPCommand(in.Command); err != nil {
		return nil, types.Result{}, fmt.Errorf("error while getting ip data: %w", err)
	}
	if err := utils.ValidateSafeString(in.FilterParameters, "filter parameters", true, utils.ShellMetaCharactersTypeDefault); err != nil {
		return nil, types.Result{}, fmt.Errorf("error while getting ip data: %w", err)
	}
	if err := utils.ValidateSafeString(in.Options, "options", true, utils.ShellMetaCharactersTypeDefault); err != nil {
		return nil, types.Result{}, fmt.Errorf("error while getting ip data: %w", err)
	}

	cmd := commandbuilder.NewCommand("ip")
	cmd.AddIfNotEmpty(in.Options, strings.Fields(in.Options)...)
	cmd.Add(strings.Fields(in.Command)...)
	cmd.AddIfNotEmpty(in.FilterParameters, strings.Fields(in.FilterParameters)...)

	stdout, stderr, err := s.executeCommand(ctx, in.Namespace, in.Node, cmd.Build())
	if err != nil {
		return nil, types.Result{}, fmt.Errorf("error while getting ip data: %w", err)
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

// validateIPCommand validates that the IP command is allowed.
func validateIPCommand(ipCommand string) error {
	if _, err := strconv.Atoi(ipCommand); err == nil {
		return fmt.Errorf("invalid ip command: %s", ipCommand)
	}
	splitCommand := strings.Fields(strings.TrimSpace(ipCommand))
	if len(splitCommand) < 2 {
		return fmt.Errorf("invalid ip command: %s", ipCommand)
	}
	validIPCommand := map[string]bool{
		"address":     true,
		"link":        true,
		"neighbour":   true,
		"netns":       true,
		"route":       true,
		"rule":        true,
		"vrf":         true,
		"xfrm state":  true,
		"xfrm policy": true,
	}

	// "ip l s" can be used to set a link up or down. This should be considered as an invalid command.
	if strings.HasPrefix(splitCommand[0], "l") && splitCommand[1] == "s" {
		return fmt.Errorf("invalid ip command: %s", ipCommand)
	}

	var valid bool
	for command := range validIPCommand {
		commandFields := strings.Fields(command)
		// Check if the input command matches the expected command pattern
		// For single-word commands like "address", we check: splitCommand[0] matches "address" and splitCommand[1] matches "show"
		// For multi-word xfrm commands, we check: splitCommand[0:2] matches ["xfrm", "state"/"policy"] and splitCommand[2] matches "list"

		if len(commandFields) == 1 {
			// Single-word command (e.g., "address show", "link show")
			if strings.HasPrefix(command, splitCommand[0]) && strings.HasPrefix("show", splitCommand[1]) {
				valid = true
				break
			}
		} else {
			// Multi-word command (e.g., "xfrm state list", "xfrm policy list")
			// Need at least len(commandFields) + 1 words (command words + subcommand)
			if len(splitCommand) >= len(commandFields)+1 {
				allMatch := true
				for i, cmdField := range commandFields {
					if !strings.HasPrefix(cmdField, splitCommand[i]) {
						allMatch = false
						break
					}
				}
				// For xfrm state and xfrm policy, only accept "list"
				if allMatch && (command == "xfrm state" || command == "xfrm policy") {
					if strings.HasPrefix("list", splitCommand[len(commandFields)]) {
						valid = true
						break
					}
				}
			}
		}
	}
	if !valid {
		return fmt.Errorf("invalid ip command: %s", ipCommand)
	}

	return nil
}
