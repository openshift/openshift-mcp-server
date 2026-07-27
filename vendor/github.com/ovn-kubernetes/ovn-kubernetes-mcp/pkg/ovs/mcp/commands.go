package mcp

import (
	"fmt"
	"regexp"

	ovstypes "github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/ovs/types"
	"github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/utils"
)

// DefaultMaxLines defines the maximum number of lines to return from command output
const DefaultMaxLines = 100

var (
	// validConntrackParam is the pattern for valid conntrack parameters.
	validConntrackParam = regexp.MustCompile(`^(-[a-zA-Z]|[a-zA-Z0-9_-]+=[a-zA-Z0-9x.:,/_-]+)$`)
)

// validateBridgeName validates that a bridge name is safe and non-empty.
// Bridge names should only contain alphanumeric characters, hyphens, and underscores.
func validateBridgeName(bridge string) error {
	if bridge == "" {
		return fmt.Errorf("bridge name cannot be empty")
	}

	// OVS bridge names typically follow naming conventions: alphanumeric, hyphens, underscores
	validBridgeName := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validBridgeName.MatchString(bridge) {
		return fmt.Errorf("invalid bridge name %q: must contain only alphanumeric characters, hyphens, and underscores", bridge)
	}

	return nil
}

// validateFlowSpec validates that a flow specification is safe and non-empty.
func validateFlowSpec(flow string) error {
	return utils.ValidateSafeString(flow, "flow specification", false, utils.ShellMetaCharactersTypeAllowBrackets)
}

// validateConntrackParams validates that conntrack additional parameters are safe.
// Valid parameters for dpctl/dump-conntrack include: zone=N, mark=0xN, labels=0xN, -m, -s, etc.
func validateConntrackParams(params []string) error {
	for _, param := range params {
		err := utils.ValidateSafeString(param, "conntrack parameter", false, utils.ShellMetaCharactersTypeDefault)
		if err != nil {
			return err
		}

		// Additional validation for common parameter patterns
		// Valid patterns include:
		// - Single-char flags: -m, -s (single hyphen followed by single letter)
		// - Key=value pairs: zone=5, mark=0x1, src=10.0.0.1 (key must contain only alphanumeric, underscore, hyphen)
		if !validConntrackParam.MatchString(param) {
			return fmt.Errorf("invalid conntrack parameter format %q: must be a flag (e.g., '-m') or key=value pair (e.g., 'zone=5')", param)
		}
	}

	return nil
}

// validateVsctlAction validates that the action is a supported ovs-vsctl subcommand.
func validateVsctlAction(action string) error {
	switch ovstypes.VsctlAction(action) {
	case ovstypes.VsctlShow, ovstypes.VsctlListBr, ovstypes.VsctlListPorts, ovstypes.VsctlListIfaces:
		return nil
	default:
		return fmt.Errorf(`invalid action %q: must be one of "show", "list-br", "list-ports", "list-ifaces"`, action)
	}
}

// validateOfctlAction validates that the action is a supported ovs-ofctl subcommand.
func validateOfctlAction(action string) error {
	switch ovstypes.OfctlAction(action) {
	case ovstypes.OfctlDumpFlows:
		return nil
	default:
		return fmt.Errorf(`invalid action %q: must be one of "dump-flows"`, action)
	}
}

// validateAppctlAction validates that the action is a supported ovs-appctl subcommand.
func validateAppctlAction(action string) error {
	switch ovstypes.AppctlAction(action) {
	case ovstypes.AppctlDumpConntrack, ovstypes.AppctlOfprotoTrace:
		return nil
	default:
		return fmt.Errorf(`invalid action %q: must be one of "dpctl/dump-conntrack", "ofproto/trace"`, action)
	}
}
