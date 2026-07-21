package lvms

import (
	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets"
)

// Toolset provides LVMS (Logical Volume Manager Storage) prompts for troubleshooting
// storage issues on OpenShift clusters. LVMS resources are managed via the core
// toolset's generic resource tools. This toolset provides a troubleshooting prompt
// that gathers diagnostic data and explains LVMS-specific fields like vg_attr flags.
type Toolset struct{}

var _ api.Toolset = (*Toolset)(nil)

// GetName returns the name of the toolset.
func (t *Toolset) GetName() string {
	return "lvms"
}

// GetDescription returns a human-readable description of the toolset.
func (t *Toolset) GetDescription() string {
	return "LVMS (Logical Volume Manager Storage) troubleshooting prompts for diagnosing storage issues"
}

// GetTools returns nil — LVMS resources are managed via the core toolset's generic resource tools.
func (t *Toolset) GetTools(_ api.FilteringProvider) []api.ServerTool {
	return nil
}

// GetPrompts returns the prompts provided by this toolset.
func (t *Toolset) GetPrompts() []api.ServerPrompt {
	return append(initLVMSTroubleshoot(), initLVMSCapacity()...)
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
