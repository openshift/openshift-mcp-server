package oadp

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// VeleroGroup is the API group for Velero resources
	VeleroGroup = "velero.io"
	// VeleroVersion is the API version for Velero resources
	VeleroVersion = "v1"
	// VeleroV2Alpha1Version is the API version for Velero v2alpha1 resources
	VeleroV2Alpha1Version = "v2alpha1"
	// OADPGroup is the API group for OADP resources
	OADPGroup = "oadp.openshift.io"
	// OADPVersion is the API version for OADP resources
	OADPVersion = "v1alpha1"
	// DefaultOADPNamespace is the default namespace where OADP is installed
	DefaultOADPNamespace = "openshift-adp"
)

var (
	// BackupGVR is the GroupVersionResource for Velero Backup resources
	BackupGVR = schema.GroupVersionResource{
		Group:    VeleroGroup,
		Version:  VeleroVersion,
		Resource: "backups",
	}

	// RestoreGVR is the GroupVersionResource for Velero Restore resources
	RestoreGVR = schema.GroupVersionResource{
		Group:    VeleroGroup,
		Version:  VeleroVersion,
		Resource: "restores",
	}

	// ScheduleGVR is the GroupVersionResource for Velero Schedule resources
	ScheduleGVR = schema.GroupVersionResource{
		Group:    VeleroGroup,
		Version:  VeleroVersion,
		Resource: "schedules",
	}

	// BackupStorageLocationGVR is the GroupVersionResource for Velero BackupStorageLocation resources
	BackupStorageLocationGVR = schema.GroupVersionResource{
		Group:    VeleroGroup,
		Version:  VeleroVersion,
		Resource: "backupstoragelocations",
	}

	// VolumeSnapshotLocationGVR is the GroupVersionResource for Velero VolumeSnapshotLocation resources
	VolumeSnapshotLocationGVR = schema.GroupVersionResource{
		Group:    VeleroGroup,
		Version:  VeleroVersion,
		Resource: "volumesnapshotlocations",
	}

	// DownloadRequestGVR is the GroupVersionResource for Velero DownloadRequest resources
	DownloadRequestGVR = schema.GroupVersionResource{
		Group:    VeleroGroup,
		Version:  VeleroVersion,
		Resource: "downloadrequests",
	}

	// DeleteBackupRequestGVR is the GroupVersionResource for Velero DeleteBackupRequest resources
	DeleteBackupRequestGVR = schema.GroupVersionResource{
		Group:    VeleroGroup,
		Version:  VeleroVersion,
		Resource: "deletebackuprequests",
	}

	// DataProtectionApplicationGVR is the GroupVersionResource for OADP DataProtectionApplication resources
	DataProtectionApplicationGVR = schema.GroupVersionResource{
		Group:    OADPGroup,
		Version:  OADPVersion,
		Resource: "dataprotectionapplications",
	}

	// BackupRepositoryGVR is the GroupVersionResource for Velero BackupRepository resources
	BackupRepositoryGVR = schema.GroupVersionResource{
		Group:    VeleroGroup,
		Version:  VeleroVersion,
		Resource: "backuprepositories",
	}

	// PodVolumeBackupGVR is the GroupVersionResource for Velero PodVolumeBackup resources
	PodVolumeBackupGVR = schema.GroupVersionResource{
		Group:    VeleroGroup,
		Version:  VeleroVersion,
		Resource: "podvolumebackups",
	}

	// PodVolumeRestoreGVR is the GroupVersionResource for Velero PodVolumeRestore resources
	PodVolumeRestoreGVR = schema.GroupVersionResource{
		Group:    VeleroGroup,
		Version:  VeleroVersion,
		Resource: "podvolumerestores",
	}

	// ServerStatusRequestGVR is the GroupVersionResource for Velero ServerStatusRequest resources
	ServerStatusRequestGVR = schema.GroupVersionResource{
		Group:    VeleroGroup,
		Version:  VeleroVersion,
		Resource: "serverstatusrequests",
	}

	// DataUploadGVR is the GroupVersionResource for Velero DataUpload resources (v2alpha1)
	DataUploadGVR = schema.GroupVersionResource{
		Group:    VeleroGroup,
		Version:  VeleroV2Alpha1Version,
		Resource: "datauploads",
	}

	// DataDownloadGVR is the GroupVersionResource for Velero DataDownload resources (v2alpha1)
	DataDownloadGVR = schema.GroupVersionResource{
		Group:    VeleroGroup,
		Version:  VeleroV2Alpha1Version,
		Resource: "datadownloads",
	}

	// CloudStorageGVR is the GroupVersionResource for OADP CloudStorage resources
	CloudStorageGVR = schema.GroupVersionResource{
		Group:    OADPGroup,
		Version:  OADPVersion,
		Resource: "cloudstorages",
	}

	// DataProtectionTestGVR is the GroupVersionResource for OADP DataProtectionTest resources
	DataProtectionTestGVR = schema.GroupVersionResource{
		Group:    OADPGroup,
		Version:  OADPVersion,
		Resource: "dataprotectiontests",
	}

	// NonAdminBackupGVR is the GroupVersionResource for OADP NonAdminBackup resources
	NonAdminBackupGVR = schema.GroupVersionResource{
		Group:    OADPGroup,
		Version:  OADPVersion,
		Resource: "nonadminbackups",
	}

	// NonAdminRestoreGVR is the GroupVersionResource for OADP NonAdminRestore resources
	NonAdminRestoreGVR = schema.GroupVersionResource{
		Group:    OADPGroup,
		Version:  OADPVersion,
		Resource: "nonadminrestores",
	}

	// NonAdminBackupStorageLocationGVR is the GroupVersionResource for OADP NonAdminBackupStorageLocation resources
	NonAdminBackupStorageLocationGVR = schema.GroupVersionResource{
		Group:    OADPGroup,
		Version:  OADPVersion,
		Resource: "nonadminbackupstoragelocations",
	}

	// NonAdminBackupStorageLocationRequestGVR is the GroupVersionResource for OADP NonAdminBackupStorageLocationRequest resources
	NonAdminBackupStorageLocationRequestGVR = schema.GroupVersionResource{
		Group:    OADPGroup,
		Version:  OADPVersion,
		Resource: "nonadminbackupstoragelocationrequests",
	}

	// NonAdminDownloadRequestGVR is the GroupVersionResource for OADP NonAdminDownloadRequest resources
	NonAdminDownloadRequestGVR = schema.GroupVersionResource{
		Group:    OADPGroup,
		Version:  OADPVersion,
		Resource: "nonadmindownloadrequests",
	}

	// VirtualMachineBackupsDiscoveryGVR is the GroupVersionResource for OADP VirtualMachineBackupsDiscovery resources
	VirtualMachineBackupsDiscoveryGVR = schema.GroupVersionResource{
		Group:    OADPGroup,
		Version:  OADPVersion,
		Resource: "virtualmachinebackupsdiscoveries",
	}

	// VirtualMachineFileRestoreGVR is the GroupVersionResource for OADP VirtualMachineFileRestore resources
	VirtualMachineFileRestoreGVR = schema.GroupVersionResource{
		Group:    OADPGroup,
		Version:  OADPVersion,
		Resource: "virtualmachinefilerestores",
	}
)

// NamespaceOrDefault returns the provided namespace if non-empty, otherwise returns DefaultOADPNamespace
func NamespaceOrDefault(namespace string) string {
	if namespace == "" {
		return DefaultOADPNamespace
	}
	return namespace
}
