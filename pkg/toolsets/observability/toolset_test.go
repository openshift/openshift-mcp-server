package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
)

type ToolsetSuite struct {
	suite.Suite
	toolset *Toolset
}

// mockOpenShift implements api.Openshift for testing
type mockOpenShift struct {
	isOpenShift bool
}

func (m *mockOpenShift) IsOpenShift(_ context.Context) bool {
	return m.isOpenShift
}

var _ api.Openshift = (*mockOpenShift)(nil)

func (s *ToolsetSuite) SetupTest() {
	s.toolset = &Toolset{}
}

func (s *ToolsetSuite) TestGetName() {
	s.Run("returns correct toolset name", func() {
		name := s.toolset.GetName()
		s.Equal("observability", name)
	})
}

func (s *ToolsetSuite) TestGetDescription() {
	s.Run("returns non-empty description", func() {
		desc := s.toolset.GetDescription()
		s.NotEmpty(desc)
		s.Contains(desc, "observability")
	})
}

func (s *ToolsetSuite) TestGetTools() {
	s.Run("returns expected number of tools", func() {
		tools := s.toolset.GetTools(&mockOpenShift{isOpenShift: true})

		// We expect 3 tools: prometheus_query, prometheus_query_range, alertmanager_alerts
		s.Len(tools, 3, "Expected 3 tools in observability toolset")
	})

	s.Run("all tools have required fields", func() {
		tools := s.toolset.GetTools(&mockOpenShift{isOpenShift: true})

		for _, tool := range tools {
			s.NotEmpty(tool.Tool.Name, "Tool name should not be empty")
			s.NotEmpty(tool.Tool.Description, "Tool description should not be empty")
			s.NotNil(tool.Handler, "Tool handler should not be nil")
			s.NotNil(tool.Tool.InputSchema, "Tool input schema should not be nil")
		}
	})

	s.Run("all tools are marked as read-only", func() {
		tools := s.toolset.GetTools(&mockOpenShift{isOpenShift: true})

		for _, tool := range tools {
			s.NotNil(tool.Tool.Annotations.ReadOnlyHint,
				"Tool %s should have ReadOnlyHint set", tool.Tool.Name)
			s.True(*tool.Tool.Annotations.ReadOnlyHint,
				"Tool %s should be marked as read-only", tool.Tool.Name)
		}
	})

	s.Run("all tools are marked as non-destructive", func() {
		tools := s.toolset.GetTools(&mockOpenShift{isOpenShift: true})

		for _, tool := range tools {
			if tool.Tool.Annotations.DestructiveHint != nil {
				s.False(*tool.Tool.Annotations.DestructiveHint,
					"Tool %s should be marked as non-destructive", tool.Tool.Name)
			}
		}
	})

	s.Run("prometheus_query tool exists with correct schema", func() {
		tools := s.toolset.GetTools(&mockOpenShift{isOpenShift: true})

		var found bool
		for _, tool := range tools {
			if tool.Tool.Name == "prometheus_query" {
				found = true
				s.Contains(tool.Tool.InputSchema.Required, "query",
					"prometheus_query should require 'query' parameter")
				s.Contains(tool.Tool.InputSchema.Properties, "query")
				s.Contains(tool.Tool.InputSchema.Properties, "time")
				break
			}
		}
		s.True(found, "prometheus_query tool should exist")
	})

	s.Run("prometheus_query_range tool exists with correct schema", func() {
		tools := s.toolset.GetTools(&mockOpenShift{isOpenShift: true})

		var found bool
		for _, tool := range tools {
			if tool.Tool.Name == "prometheus_query_range" {
				found = true
				s.Contains(tool.Tool.InputSchema.Required, "query")
				s.Contains(tool.Tool.InputSchema.Required, "start")
				s.Contains(tool.Tool.InputSchema.Required, "end")
				s.Contains(tool.Tool.InputSchema.Properties, "step")
				break
			}
		}
		s.True(found, "prometheus_query_range tool should exist")
	})

	s.Run("alertmanager_alerts tool exists with correct schema", func() {
		tools := s.toolset.GetTools(&mockOpenShift{isOpenShift: true})

		var found bool
		for _, tool := range tools {
			if tool.Tool.Name == "alertmanager_alerts" {
				found = true
				s.Contains(tool.Tool.InputSchema.Properties, "active")
				s.Contains(tool.Tool.InputSchema.Properties, "silenced")
				s.Contains(tool.Tool.InputSchema.Properties, "inhibited")
				s.Contains(tool.Tool.InputSchema.Properties, "filter")
				break
			}
		}
		s.True(found, "alertmanager_alerts tool should exist")
	})
}

func (s *ToolsetSuite) TestGetPrompts() {
	s.Run("returns nil (no prompts)", func() {
		prompts := s.toolset.GetPrompts()
		s.Nil(prompts, "Observability toolset should not have prompts")
	})
}

func (s *ToolsetSuite) TestToolsetImplementsInterface() {
	s.Run("implements api.Toolset interface", func() {
		var _ api.Toolset = (*Toolset)(nil)
		// If this compiles, the interface is implemented correctly
	})
}

func TestToolsetSuite(t *testing.T) {
	suite.Run(t, new(ToolsetSuite))
}
