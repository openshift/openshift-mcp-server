package lvms

import (
	"testing"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolsetImplementsInterface(t *testing.T) {
	var _ api.Toolset = (*Toolset)(nil)
}

func TestToolsetGetName(t *testing.T) {
	toolset := &Toolset{}
	assert.Equal(t, "lvms", toolset.GetName())
}

func TestToolsetGetDescription(t *testing.T) {
	toolset := &Toolset{}
	desc := toolset.GetDescription()
	assert.Contains(t, desc, "LVMS")
	assert.Contains(t, desc, "troubleshooting")
}

func TestToolsetGetTools(t *testing.T) {
	toolset := &Toolset{}
	tools := toolset.GetTools(nil)

	// LVMS is a prompts-only toolset - no tools needed.
	// All LVMS operations are covered by:
	// - Core tools (resources_list, resources_get) for LVMS CRD queries
	// - nodes_debug_exec (cluster-diagnostics) for raw LVM commands on nodes
	assert.Nil(t, tools, "LVMS toolset should have no tools - it's a prompts-only toolset")
}

func TestToolsetGetPrompts(t *testing.T) {
	toolset := &Toolset{}
	prompts := toolset.GetPrompts()

	require.NotEmpty(t, prompts, "toolset should return prompts")
	require.Len(t, prompts, 2, "toolset should return 2 prompts")

	// Verify lvms-troubleshoot prompt
	var foundTroubleshoot, foundCapacity bool
	for _, prompt := range prompts {
		switch prompt.Prompt.Name {
		case "lvms-troubleshoot":
			foundTroubleshoot = true
			assert.Equal(t, "LVMS Troubleshoot", prompt.Prompt.Title)
			assert.NotEmpty(t, prompt.Prompt.Description)
			assert.NotNil(t, prompt.Handler)
			require.Len(t, prompt.Prompt.Arguments, 2)
			assert.Equal(t, "namespace", prompt.Prompt.Arguments[0].Name)
			assert.Equal(t, "node", prompt.Prompt.Arguments[1].Name)

		case "lvms-capacity":
			foundCapacity = true
			assert.Equal(t, "LVMS Capacity Check", prompt.Prompt.Title)
			assert.NotEmpty(t, prompt.Prompt.Description)
			assert.NotNil(t, prompt.Handler)
			require.Len(t, prompt.Prompt.Arguments, 1)
			assert.Equal(t, "namespace", prompt.Prompt.Arguments[0].Name)
		}
	}
	assert.True(t, foundTroubleshoot, "lvms-troubleshoot prompt should be registered")
	assert.True(t, foundCapacity, "lvms-capacity prompt should be registered")
}

func TestToolsetGetResources(t *testing.T) {
	toolset := &Toolset{}
	resources := toolset.GetResources()
	assert.Nil(t, resources, "LVMS toolset currently has no resources")
}

func TestToolsetGetResourceTemplates(t *testing.T) {
	toolset := &Toolset{}
	templates := toolset.GetResourceTemplates()
	assert.Nil(t, templates, "LVMS toolset currently has no resource templates")
}
