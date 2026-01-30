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
func (t *Toolset) GetTools(_ api.Openshift) []api.ServerTool {
	return slices.Concat(
		initBackupTools(),
		initRestoreTools(),
		initScheduleTools(),
		initStorageTools(),
		initDPATools(),
		initBackupRepositoryTools(),
		initPodVolumeTools(),
		initServerStatusRequestTools(),
		initDataMoverTools(),
		initDownloadRequestTools(),
		initDeleteBackupRequestTools(),
		initCloudStorageTools(),
		initDataProtectionTestTools(),
		initNonAdminTools(),
		initVMRestoreTools(),
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
