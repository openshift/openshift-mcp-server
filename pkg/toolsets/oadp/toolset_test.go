package oadp

import (
	"testing"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/stretchr/testify/suite"
)

type ToolsetSuite struct {
	suite.Suite
}

func (s *ToolsetSuite) TestGetName() {
	s.Run("returns oadp", func() {
		t := &Toolset{}
		s.Equal("oadp", t.GetName())
	})
}

func (s *ToolsetSuite) TestGetDescription() {
	s.Run("returns non-empty description", func() {
		t := &Toolset{}
		s.NotEmpty(t.GetDescription())
	})
}

func (s *ToolsetSuite) TestGetTools() {
	s.Run("returns nil", func() {
		t := &Toolset{}
		s.Nil(t.GetTools(s.T().Context(), api.ToolsetContext{}))
	})
}

func (s *ToolsetSuite) TestGetPrompts() {
	s.Run("returns one prompt", func() {
		t := &Toolset{}
		prompts := t.GetPrompts(s.T().Context(), api.ToolsetContext{})
		s.Len(prompts, 1)
	})

	s.Run("prompt is named oadp-troubleshoot", func() {
		t := &Toolset{}
		prompts := t.GetPrompts(s.T().Context(), api.ToolsetContext{})
		s.Require().Len(prompts, 1)
		s.Equal("oadp-troubleshoot", prompts[0].Prompt.Name)
	})

	s.Run("prompt has list pagination arguments", func() {
		t := &Toolset{}
		prompts := t.GetPrompts(s.T().Context(), api.ToolsetContext{})
		s.Require().Len(prompts, 1)
		names := make([]string, 0, len(prompts[0].Prompt.Arguments))
		for _, arg := range prompts[0].Prompt.Arguments {
			names = append(names, arg.Name)
		}
		s.Equal([]string{"namespace", "backup", "restore", "limit", "continue"}, names)
	})

	s.Run("limit must be a non-negative integer", func() {
		_, err := promptListPage(map[string]string{"limit": "nope"})
		s.EqualError(err, "limit parameter must be an integer")
		_, err = promptListPage(map[string]string{"limit": "-1"})
		s.EqualError(err, "limit must be greater than or equal to 0")
		page, err := promptListPage(map[string]string{"limit": "2", "continue": "next"})
		s.NoError(err)
		s.Equal(int64(2), page.Limit)
		s.Equal("next", page.Continue)
	})

	s.Run("prompt has a handler", func() {
		t := &Toolset{}
		prompts := t.GetPrompts(s.T().Context(), api.ToolsetContext{})
		s.Require().Len(prompts, 1)
		s.NotNil(prompts[0].Handler)
	})
}

func (s *ToolsetSuite) TestGetResources() {
	s.Run("returns nil", func() {
		t := &Toolset{}
		s.Nil(t.GetResources(s.T().Context(), api.ToolsetContext{}))
	})
}

func (s *ToolsetSuite) TestGetResourceTemplates() {
	s.Run("returns nil", func() {
		t := &Toolset{}
		s.Nil(t.GetResourceTemplates(s.T().Context(), api.ToolsetContext{}))
	})
}

func TestToolsetSuite(t *testing.T) {
	suite.Run(t, new(ToolsetSuite))
}
