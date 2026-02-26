package oadp

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
		s.Equal("oadp", name)
	})
}

func (s *ToolsetSuite) TestGetDescription() {
	s.Run("returns non-empty description", func() {
		desc := s.toolset.GetDescription()
		s.NotEmpty(desc)
		s.Contains(desc, "OADP")
	})
}

func (s *ToolsetSuite) TestGetTools() {
	s.Run("returns expected number of tools", func() {
		tools := s.toolset.GetTools(&mockOpenShift{isOpenShift: true})

		// We expect 8 consolidated tools:
		// 1. oadp_backup - Backup operations
		// 2. oadp_restore - Restore operations
		// 3. oadp_schedule - Schedule operations
		// 4. oadp_dpa - DataProtectionApplication operations
		// 5. oadp_storage_location - BSL/VSL operations
		// 6. oadp_data_mover - DataUpload/DataDownload operations
		// 7. oadp_repository - BackupRepository operations
		// 8. oadp_data_protection_test - DataProtectionTest operations
		s.Len(tools, 8, "Expected 8 consolidated tools in OADP toolset")
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

	s.Run("all tools have action parameter in schema", func() {
		tools := s.toolset.GetTools(&mockOpenShift{isOpenShift: true})

		for _, tool := range tools {
			s.Contains(tool.Tool.InputSchema.Required, "action",
				"Tool %s should require 'action' parameter", tool.Tool.Name)
			s.Contains(tool.Tool.InputSchema.Properties, "action",
				"Tool %s should have 'action' in properties", tool.Tool.Name)
		}
	})

	s.Run("expected tools exist", func() {
		tools := s.toolset.GetTools(&mockOpenShift{isOpenShift: true})
		toolNames := make(map[string]bool)
		for _, tool := range tools {
			toolNames[tool.Tool.Name] = true
		}

		expectedTools := []string{
			"oadp_backup",
			"oadp_restore",
			"oadp_schedule",
			"oadp_dpa",
			"oadp_storage_location",
			"oadp_data_mover",
			"oadp_repository",
			"oadp_data_protection_test",
		}

		for _, expected := range expectedTools {
			s.True(toolNames[expected], "%s should exist", expected)
		}
	})

	s.Run("oadp_backup has correct action enum", func() {
		tools := s.toolset.GetTools(&mockOpenShift{isOpenShift: true})

		for _, tool := range tools {
			if tool.Tool.Name == "oadp_backup" {
				actionProp := tool.Tool.InputSchema.Properties["action"]
				s.NotNil(actionProp.Enum, "oadp_backup action should have enum values")
				s.Contains(actionProp.Enum, "list", "oadp_backup should support 'list' action")
				s.Contains(actionProp.Enum, "get", "oadp_backup should support 'get' action")
				s.Contains(actionProp.Enum, "create", "oadp_backup should support 'create' action")
				s.Contains(actionProp.Enum, "delete", "oadp_backup should support 'delete' action")
				s.Contains(actionProp.Enum, "status", "oadp_backup should support 'status' action")
				break
			}
		}
	})

	s.Run("oadp_restore has correct action enum", func() {
		tools := s.toolset.GetTools(&mockOpenShift{isOpenShift: true})

		for _, tool := range tools {
			if tool.Tool.Name == "oadp_restore" {
				actionProp := tool.Tool.InputSchema.Properties["action"]
				s.NotNil(actionProp.Enum, "oadp_restore action should have enum values")
				s.Contains(actionProp.Enum, "list", "oadp_restore should support 'list' action")
				s.Contains(actionProp.Enum, "get", "oadp_restore should support 'get' action")
				s.Contains(actionProp.Enum, "create", "oadp_restore should support 'create' action")
				s.Contains(actionProp.Enum, "delete", "oadp_restore should support 'delete' action")
				s.Contains(actionProp.Enum, "status", "oadp_restore should support 'status' action")
				break
			}
		}
	})

	s.Run("oadp_schedule has correct action enum", func() {
		tools := s.toolset.GetTools(&mockOpenShift{isOpenShift: true})

		for _, tool := range tools {
			if tool.Tool.Name == "oadp_schedule" {
				actionProp := tool.Tool.InputSchema.Properties["action"]
				s.NotNil(actionProp.Enum, "oadp_schedule action should have enum values")
				s.Contains(actionProp.Enum, "list", "oadp_schedule should support 'list' action")
				s.Contains(actionProp.Enum, "get", "oadp_schedule should support 'get' action")
				s.Contains(actionProp.Enum, "create", "oadp_schedule should support 'create' action")
				s.Contains(actionProp.Enum, "update", "oadp_schedule should support 'update' action")
				s.Contains(actionProp.Enum, "delete", "oadp_schedule should support 'delete' action")
				s.Contains(actionProp.Enum, "pause", "oadp_schedule should support 'pause' action")
				break
			}
		}
	})

	s.Run("oadp_dpa has correct action enum", func() {
		tools := s.toolset.GetTools(&mockOpenShift{isOpenShift: true})

		for _, tool := range tools {
			if tool.Tool.Name == "oadp_dpa" {
				actionProp := tool.Tool.InputSchema.Properties["action"]
				s.NotNil(actionProp.Enum, "oadp_dpa action should have enum values")
				s.Contains(actionProp.Enum, "list", "oadp_dpa should support 'list' action")
				s.Contains(actionProp.Enum, "get", "oadp_dpa should support 'get' action")
				s.Contains(actionProp.Enum, "create", "oadp_dpa should support 'create' action")
				s.Contains(actionProp.Enum, "update", "oadp_dpa should support 'update' action")
				s.Contains(actionProp.Enum, "delete", "oadp_dpa should support 'delete' action")
				break
			}
		}
	})

	s.Run("oadp_storage_location has type parameter", func() {
		tools := s.toolset.GetTools(&mockOpenShift{isOpenShift: true})

		for _, tool := range tools {
			if tool.Tool.Name == "oadp_storage_location" {
				s.Contains(tool.Tool.InputSchema.Required, "type",
					"oadp_storage_location should require 'type' parameter")
				typeProp := tool.Tool.InputSchema.Properties["type"]
				s.NotNil(typeProp.Enum, "oadp_storage_location type should have enum values")
				s.Contains(typeProp.Enum, "bsl", "oadp_storage_location should support 'bsl' type")
				s.Contains(typeProp.Enum, "vsl", "oadp_storage_location should support 'vsl' type")
				break
			}
		}
	})

	s.Run("oadp_data_mover has type parameter", func() {
		tools := s.toolset.GetTools(&mockOpenShift{isOpenShift: true})

		for _, tool := range tools {
			if tool.Tool.Name == "oadp_data_mover" {
				s.Contains(tool.Tool.InputSchema.Required, "type",
					"oadp_data_mover should require 'type' parameter")
				typeProp := tool.Tool.InputSchema.Properties["type"]
				s.NotNil(typeProp.Enum, "oadp_data_mover type should have enum values")
				s.Contains(typeProp.Enum, "upload", "oadp_data_mover should support 'upload' type")
				s.Contains(typeProp.Enum, "download", "oadp_data_mover should support 'download' type")
				break
			}
		}
	})
}

func (s *ToolsetSuite) TestGetPrompts() {
	s.Run("returns nil (no prompts)", func() {
		prompts := s.toolset.GetPrompts()
		s.Nil(prompts, "OADP toolset should not have prompts")
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
