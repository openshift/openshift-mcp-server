package core_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/core"
)

type AppOverridesSuite struct{ suite.Suite }

func (s *AppOverridesSuite) TestCoreSelectsDownstreamPresentations() {
	expected := map[string]string{
		"namespaces_list": "table", "projects_list": "table", "pods_list": "table",
		"pods_list_in_namespace": "table", "resources_list": "table", "events_list": "table",
		"pods_top": "metrics", "nodes_top": "metrics", "pods_get": "resource", "resources_get": "resource",
	}
	tools := (&core.Toolset{}).GetTools(s.T().Context(), api.ToolsetContext{})
	seen := make(map[string]string)
	for _, tool := range tools {
		kind, selected := expected[tool.Tool.Name]
		if !selected {
			s.Nil(tool.App, tool.Tool.Name)
			continue
		}
		s.Require().NotNil(tool.App, tool.Tool.Name)
		content, err := tool.App.Handler(api.ResourceHandlerParams{Context: s.T().Context()})
		s.Require().NoError(err)
		s.Contains(content.Text, `data-app-kind="`+kind+`"`, tool.Tool.Name)
		s.Contains(content.Text, "pf-v6-c-card", tool.Tool.Name)
		s.NotEmpty(tool.App.Description)
		s.NotNil(tool.Handler)
		s.NotNil(tool.RBAC)
		seen[tool.Tool.Name] = kind
		if tool.Tool.Name == "namespaces_list" {
			s.Equal("ui://kubernetes-mcp-server/namespaces-list", tool.App.URI)
			s.Equal(map[string]any{"ui": map[string]any{"prefersBorder": true}}, tool.App.Meta)
		}
	}
	s.Equal(expected, seen)
}

func TestAppOverrides(t *testing.T) {
	suite.Run(t, new(AppOverridesSuite))
}
