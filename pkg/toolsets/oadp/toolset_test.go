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

		// We expect 90 tools total covering all OADP/Velero CRDs:
		// 5 backup, 5 restore, 6 schedule, 10 storage (BSL/VSL), 5 DPA,
		// 3 backup repository, 4 pod volume, 4 server status, 6 data mover,
		// 4 download request, 2 delete backup request, 4 cloud storage,
		// 4 data protection test, 20 non-admin, 8 VM restore
		s.Len(tools, 90, "Expected 90 tools in OADP toolset")
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

	s.Run("read-only tools are marked correctly", func() {
		tools := s.toolset.GetTools(&mockOpenShift{isOpenShift: true})

		readOnlyTools := map[string]bool{
			// Backup tools
			"oadp_backup_list": true,
			"oadp_backup_get":  true,
			"oadp_backup_logs": true,
			// Restore tools
			"oadp_restore_list": true,
			"oadp_restore_get":  true,
			"oadp_restore_logs": true,
			// Schedule tools
			"oadp_schedule_list": true,
			"oadp_schedule_get":  true,
			// Storage location tools
			"oadp_backup_storage_location_list":  true,
			"oadp_backup_storage_location_get":   true,
			"oadp_volume_snapshot_location_list": true,
			"oadp_volume_snapshot_location_get":  true,
			// DPA tools
			"oadp_dpa_list": true,
			"oadp_dpa_get":  true,
			// Backup repository tools
			"oadp_backup_repository_list": true,
			"oadp_backup_repository_get":  true,
			// Pod volume tools
			"oadp_pod_volume_backup_list":  true,
			"oadp_pod_volume_backup_get":   true,
			"oadp_pod_volume_restore_list": true,
			"oadp_pod_volume_restore_get":  true,
			// Server status tools
			"oadp_server_status_request_list": true,
			"oadp_server_status_request_get":  true,
			// Data mover tools
			"oadp_data_upload_list":   true,
			"oadp_data_upload_get":    true,
			"oadp_data_download_list": true,
			"oadp_data_download_get":  true,
			// Download request tools
			"oadp_download_request_list": true,
			"oadp_download_request_get":  true,
			// Delete backup request tools
			"oadp_delete_backup_request_list": true,
			"oadp_delete_backup_request_get":  true,
			// Cloud storage tools
			"oadp_cloud_storage_list": true,
			"oadp_cloud_storage_get":  true,
			// Data protection test tools
			"oadp_data_protection_test_list": true,
			"oadp_data_protection_test_get":  true,
			// Non-admin tools
			"oadp_non_admin_backup_list":           true,
			"oadp_non_admin_backup_get":            true,
			"oadp_non_admin_restore_list":          true,
			"oadp_non_admin_restore_get":           true,
			"oadp_non_admin_bsl_list":              true,
			"oadp_non_admin_bsl_get":               true,
			"oadp_non_admin_bsl_request_list":      true,
			"oadp_non_admin_bsl_request_get":       true,
			"oadp_non_admin_download_request_list": true,
			"oadp_non_admin_download_request_get":  true,
			// VM restore tools
			"oadp_vm_backup_discovery_list": true,
			"oadp_vm_backup_discovery_get":  true,
			"oadp_vm_file_restore_list":     true,
			"oadp_vm_file_restore_get":      true,
		}

		for _, tool := range tools {
			if readOnlyTools[tool.Tool.Name] {
				s.NotNil(tool.Tool.Annotations.ReadOnlyHint,
					"Tool %s should have ReadOnlyHint set", tool.Tool.Name)
				s.True(*tool.Tool.Annotations.ReadOnlyHint,
					"Tool %s should be marked as read-only", tool.Tool.Name)
			}
		}
	})

	s.Run("destructive tools are marked correctly", func() {
		tools := s.toolset.GetTools(&mockOpenShift{isOpenShift: true})

		destructiveTools := map[string]bool{
			// Backup/restore tools
			"oadp_backup_delete":  true,
			"oadp_restore_create": true,
			"oadp_restore_delete": true,
			// Schedule tools
			"oadp_schedule_delete": true,
			// Storage location tools
			"oadp_backup_storage_location_delete":  true,
			"oadp_volume_snapshot_location_delete": true,
			// DPA tools
			"oadp_dpa_delete": true,
			// Backup repository tools
			"oadp_backup_repository_delete": true,
			// Server status tools
			"oadp_server_status_request_delete": true,
			// Data mover tools
			"oadp_data_upload_cancel":   true,
			"oadp_data_download_cancel": true,
			// Download request tools
			"oadp_download_request_delete": true,
			// Cloud storage tools
			"oadp_cloud_storage_delete": true,
			// Data protection test tools
			"oadp_data_protection_test_delete": true,
			// Non-admin tools
			"oadp_non_admin_backup_delete":           true,
			"oadp_non_admin_restore_delete":          true,
			"oadp_non_admin_bsl_delete":              true,
			"oadp_non_admin_download_request_delete": true,
			// VM restore tools
			"oadp_vm_backup_discovery_delete": true,
			"oadp_vm_file_restore_delete":     true,
		}

		for _, tool := range tools {
			if destructiveTools[tool.Tool.Name] {
				s.NotNil(tool.Tool.Annotations.DestructiveHint,
					"Tool %s should have DestructiveHint set", tool.Tool.Name)
				s.True(*tool.Tool.Annotations.DestructiveHint,
					"Tool %s should be marked as destructive", tool.Tool.Name)
			}
		}
	})

	s.Run("backup tools exist", func() {
		tools := s.toolset.GetTools(&mockOpenShift{isOpenShift: true})
		toolNames := make(map[string]bool)
		for _, tool := range tools {
			toolNames[tool.Tool.Name] = true
		}

		s.True(toolNames["oadp_backup_list"], "oadp_backup_list should exist")
		s.True(toolNames["oadp_backup_get"], "oadp_backup_get should exist")
		s.True(toolNames["oadp_backup_create"], "oadp_backup_create should exist")
		s.True(toolNames["oadp_backup_delete"], "oadp_backup_delete should exist")
		s.True(toolNames["oadp_backup_logs"], "oadp_backup_logs should exist")
	})

	s.Run("restore tools exist", func() {
		tools := s.toolset.GetTools(&mockOpenShift{isOpenShift: true})
		toolNames := make(map[string]bool)
		for _, tool := range tools {
			toolNames[tool.Tool.Name] = true
		}

		s.True(toolNames["oadp_restore_list"], "oadp_restore_list should exist")
		s.True(toolNames["oadp_restore_get"], "oadp_restore_get should exist")
		s.True(toolNames["oadp_restore_create"], "oadp_restore_create should exist")
		s.True(toolNames["oadp_restore_delete"], "oadp_restore_delete should exist")
		s.True(toolNames["oadp_restore_logs"], "oadp_restore_logs should exist")
	})

	s.Run("schedule tools exist", func() {
		tools := s.toolset.GetTools(&mockOpenShift{isOpenShift: true})
		toolNames := make(map[string]bool)
		for _, tool := range tools {
			toolNames[tool.Tool.Name] = true
		}

		s.True(toolNames["oadp_schedule_list"], "oadp_schedule_list should exist")
		s.True(toolNames["oadp_schedule_get"], "oadp_schedule_get should exist")
		s.True(toolNames["oadp_schedule_create"], "oadp_schedule_create should exist")
		s.True(toolNames["oadp_schedule_delete"], "oadp_schedule_delete should exist")
		s.True(toolNames["oadp_schedule_pause"], "oadp_schedule_pause should exist")
	})

	s.Run("storage location tools exist", func() {
		tools := s.toolset.GetTools(&mockOpenShift{isOpenShift: true})
		toolNames := make(map[string]bool)
		for _, tool := range tools {
			toolNames[tool.Tool.Name] = true
		}

		s.True(toolNames["oadp_backup_storage_location_list"], "oadp_backup_storage_location_list should exist")
		s.True(toolNames["oadp_backup_storage_location_get"], "oadp_backup_storage_location_get should exist")
		s.True(toolNames["oadp_volume_snapshot_location_list"], "oadp_volume_snapshot_location_list should exist")
		s.True(toolNames["oadp_volume_snapshot_location_get"], "oadp_volume_snapshot_location_get should exist")
	})

	s.Run("DPA tools exist", func() {
		tools := s.toolset.GetTools(&mockOpenShift{isOpenShift: true})
		toolNames := make(map[string]bool)
		for _, tool := range tools {
			toolNames[tool.Tool.Name] = true
		}

		s.True(toolNames["oadp_dpa_list"], "oadp_dpa_list should exist")
		s.True(toolNames["oadp_dpa_get"], "oadp_dpa_get should exist")
	})

	s.Run("oadp_backup_create has correct required fields", func() {
		tools := s.toolset.GetTools(&mockOpenShift{isOpenShift: true})

		var found bool
		for _, tool := range tools {
			if tool.Tool.Name == "oadp_backup_create" {
				found = true
				s.Contains(tool.Tool.InputSchema.Required, "name",
					"oadp_backup_create should require 'name' parameter")
				break
			}
		}
		s.True(found, "oadp_backup_create tool should exist")
	})

	s.Run("oadp_restore_create has correct required fields", func() {
		tools := s.toolset.GetTools(&mockOpenShift{isOpenShift: true})

		var found bool
		for _, tool := range tools {
			if tool.Tool.Name == "oadp_restore_create" {
				found = true
				s.Contains(tool.Tool.InputSchema.Required, "name",
					"oadp_restore_create should require 'name' parameter")
				s.Contains(tool.Tool.InputSchema.Required, "backupName",
					"oadp_restore_create should require 'backupName' parameter")
				break
			}
		}
		s.True(found, "oadp_restore_create tool should exist")
	})

	s.Run("oadp_schedule_create has correct required fields", func() {
		tools := s.toolset.GetTools(&mockOpenShift{isOpenShift: true})

		var found bool
		for _, tool := range tools {
			if tool.Tool.Name == "oadp_schedule_create" {
				found = true
				s.Contains(tool.Tool.InputSchema.Required, "name",
					"oadp_schedule_create should require 'name' parameter")
				s.Contains(tool.Tool.InputSchema.Required, "schedule",
					"oadp_schedule_create should require 'schedule' parameter")
				break
			}
		}
		s.True(found, "oadp_schedule_create tool should exist")
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
