package mcp

import (
	"fmt"
	"regexp"

	"github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/utils"
)

const defaultMaxLines = 100

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
