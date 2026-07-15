package oadp

import (
	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets"
)

// Toolset provides OADP (OpenShift API for Data Protection) prompts for managing
// Velero backups, restores, and schedules on OpenShift clusters.
// OADP resources are managed via the core toolset's generic resource tools.
// This toolset provides a troubleshooting prompt for diagnosing backup/restore issues.
type Toolset struct{}

var _ api.Toolset = (*Toolset)(nil)

// GetName returns the name of the toolset.
func (t *Toolset) GetName() string {
	return "oadp"
}

// GetDescription returns a human-readable description of the toolset.
func (t *Toolset) GetDescription() string {
	return "OADP (OpenShift API for Data Protection) tools for managing Velero backups, restores, and schedules"
}

// GetTools returns nil — OADP resources are managed via the core toolset's generic resource tools.
func (t *Toolset) GetTools(_ api.FilteringProvider) []api.ServerTool {
	return nil
}

// GetPrompts returns the prompts provided by this toolset.
func (t *Toolset) GetPrompts() []api.ServerPrompt {
	return initOADPTroubleshoot()
}

// GetResources returns the resources provided by this toolset.
func (t *Toolset) GetResources() []api.ServerResource {
	return nil
}

// GetResourceTemplates returns the resource templates provided by this toolset.
func (t *Toolset) GetResourceTemplates() []api.ServerResourceTemplate {
	return nil
}

func init() {
	toolsets.Register(&Toolset{})
}
