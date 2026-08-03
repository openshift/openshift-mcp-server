package oadp

import (
	"testing"

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
		s.Nil(t.GetTools(nil))
	})
}

func (s *ToolsetSuite) TestGetPrompts() {
	s.Run("returns one prompt", func() {
		t := &Toolset{}
		prompts := t.GetPrompts()
		s.Len(prompts, 1)
	})

	s.Run("prompt is named oadp-troubleshoot", func() {
		t := &Toolset{}
		prompts := t.GetPrompts()
		s.Require().Len(prompts, 1)
		s.Equal("oadp-troubleshoot", prompts[0].Prompt.Name)
	})

	s.Run("prompt has three arguments", func() {
		t := &Toolset{}
		prompts := t.GetPrompts()
		s.Require().Len(prompts, 1)
		s.Len(prompts[0].Prompt.Arguments, 3)
	})

	s.Run("all prompt arguments are optional", func() {
		t := &Toolset{}
		prompts := t.GetPrompts()
		s.Require().Len(prompts, 1)
		for _, arg := range prompts[0].Prompt.Arguments {
			s.False(arg.Required)
		}
	})

	s.Run("prompt has a handler", func() {
		t := &Toolset{}
		prompts := t.GetPrompts()
		s.Require().Len(prompts, 1)
		s.NotNil(prompts[0].Handler)
	})
}

func (s *ToolsetSuite) TestGetResources() {
	s.Run("returns nil", func() {
		t := &Toolset{}
		s.Nil(t.GetResources())
	})
}

func (s *ToolsetSuite) TestGetResourceTemplates() {
	s.Run("returns nil", func() {
		t := &Toolset{}
		s.Nil(t.GetResourceTemplates())
	})
}

func TestToolsetSuite(t *testing.T) {
	suite.Run(t, new(ToolsetSuite))
}
