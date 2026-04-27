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

	// DataProtectionTestGVR is the GroupVersionResource for OADP DataProtectionTest resources
	DataProtectionTestGVR = schema.GroupVersionResource{
		Group:    OADPGroup,
		Version:  OADPVersion,
		Resource: "dataprotectiontests",
	}
)

// NamespaceOrDefault returns the provided namespace if non-empty, otherwise returns DefaultOADPNamespace
func NamespaceOrDefault(namespace string) string {
	if namespace == "" {
		return DefaultOADPNamespace
	}
	return namespace
}
