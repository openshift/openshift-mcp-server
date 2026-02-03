package oadp

import (
	"slices"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets"
)

// Toolset provides OADP (OpenShift API for Data Protection) tools for managing
// Velero backups, restores, and schedules on OpenShift clusters.
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

// GetTools returns all tools provided by this toolset.
// The toolset provides 10 consolidated tools covering all OADP CRDs:
//  1. oadp_backup - Manage backups (list, get, create, delete, logs)
//  2. oadp_restore - Manage restores (list, get, create, delete, logs)
//  3. oadp_schedule - Manage schedules (list, get, create, update, delete, pause)
//  4. oadp_dpa - Manage DataProtectionApplication (list, get, create, update, delete)
//  5. oadp_storage_location - Manage BSL/VSL (list, get, create, update, delete)
//  6. oadp_data_mover - Manage DataUpload/DataDownload (list, get, cancel)
//  7. oadp_repository - Manage BackupRepository (list, get, delete)
//  8. oadp_data_protection_test - Manage DataProtectionTest (list, get, create, delete)
func (t *Toolset) GetTools(_ api.Openshift) []api.ServerTool {
	return slices.Concat(
		initBackupTools(),
		initRestoreTools(),
		initScheduleTools(),
		initDPATools(),
		initStorageTools(),
		initDataMoverTools(),
		initBackupRepositoryTools(),
		initDataProtectionTestTools(),
	)
}

// GetPrompts returns the prompts provided by this toolset.
func (t *Toolset) GetPrompts() []api.ServerPrompt {
	// OADP toolset does not provide prompts
	return nil
}

func init() {
	toolsets.Register(&Toolset{})
}
