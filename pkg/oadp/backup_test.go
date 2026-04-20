package oadp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
)

type BackupSuite struct {
	suite.Suite
	ctx    context.Context
	client *fake.FakeDynamicClient
}

func (s *BackupSuite) SetupTest() {
	s.ctx = context.Background()
	gvrToListKind := map[schema.GroupVersionResource]string{
		BackupGVR:              "BackupList",
		DeleteBackupRequestGVR: "DeleteBackupRequestList",
	}
	s.client = fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToListKind)
}

func (s *BackupSuite) createTestBackup(name, namespace string) *unstructured.Unstructured {
	backup := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": VeleroGroup + "/" + VeleroVersion,
			"kind":       "Backup",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]any{
				"includedNamespaces": []any{"default"},
				"ttl":                "24h",
			},
		},
	}
	created, err := s.client.Resource(BackupGVR).Namespace(namespace).Create(s.ctx, backup, metav1.CreateOptions{})
	s.Require().NoError(err)
	return created
}

func (s *BackupSuite) TestListBackupsEmpty() {
	s.Run("returns empty list when no backups exist", func() {
		list, err := ListBackups(s.ctx, s.client, DefaultOADPNamespace, metav1.ListOptions{})
		s.NoError(err)
		s.Empty(list.Items)
	})
}

func (s *BackupSuite) TestListBackups() {
	s.Run("returns backups in namespace", func() {
		s.createTestBackup("test-backup", DefaultOADPNamespace)

		list, err := ListBackups(s.ctx, s.client, DefaultOADPNamespace, metav1.ListOptions{})
		s.NoError(err)
		s.Len(list.Items, 1)
		s.Equal("test-backup", list.Items[0].GetName())
	})
}

func (s *BackupSuite) TestGetBackup() {
	s.Run("returns backup by name", func() {
		s.createTestBackup("get-test", DefaultOADPNamespace)

		backup, err := GetBackup(s.ctx, s.client, DefaultOADPNamespace, "get-test")
		s.NoError(err)
		s.Equal("get-test", backup.GetName())
	})
}

func (s *BackupSuite) TestGetBackupNotFound() {
	s.Run("returns error for non-existent backup", func() {
		_, err := GetBackup(s.ctx, s.client, DefaultOADPNamespace, "non-existent")
		s.Error(err)
	})
}

func (s *BackupSuite) TestCreateBackup() {
	s.Run("creates backup in specified namespace", func() {
		backup := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": VeleroGroup + "/" + VeleroVersion,
				"kind":       "Backup",
				"metadata": map[string]any{
					"name":      "new-backup",
					"namespace": DefaultOADPNamespace,
				},
				"spec": map[string]any{
					"includedNamespaces": []any{"my-app"},
				},
			},
		}

		created, err := CreateBackup(s.ctx, s.client, backup)
		s.NoError(err)
		s.Equal("new-backup", created.GetName())

		fetched, err := GetBackup(s.ctx, s.client, DefaultOADPNamespace, "new-backup")
		s.NoError(err)
		s.Equal("new-backup", fetched.GetName())
	})
}

func (s *BackupSuite) TestCreateBackupDefaultNamespace() {
	s.Run("defaults to openshift-adp namespace when not specified", func() {
		backup := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": VeleroGroup + "/" + VeleroVersion,
				"kind":       "Backup",
				"metadata": map[string]any{
					"name": "default-ns-backup",
				},
				"spec": map[string]any{},
			},
		}

		created, err := CreateBackup(s.ctx, s.client, backup)
		s.NoError(err)
		s.Equal(DefaultOADPNamespace, created.GetNamespace())
	})
}

func (s *BackupSuite) TestDeleteBackupCreatesDeleteBackupRequest() {
	s.Run("creates a DeleteBackupRequest instead of directly deleting the backup", func() {
		// Create a backup
		s.createTestBackup("delete-test", DefaultOADPNamespace)

		// Delete the backup via the OADP DeleteBackup function
		err := DeleteBackup(s.ctx, s.client, DefaultOADPNamespace, "delete-test")
		s.NoError(err)

		// The backup itself should still exist — DeleteBackup does NOT directly
		// delete the Backup CR. It creates a DeleteBackupRequest which triggers
		// the Velero controller to properly clean up backup data from object
		// storage and then remove the Backup CR.
		backup, err := GetBackup(s.ctx, s.client, DefaultOADPNamespace, "delete-test")
		s.NoError(err, "Backup should still exist after DeleteBackup — only a DeleteBackupRequest should be created")
		s.Equal("delete-test", backup.GetName())

		// Verify a DeleteBackupRequest was created with the correct spec
		dbrList, err := s.client.Resource(DeleteBackupRequestGVR).Namespace(DefaultOADPNamespace).List(
			s.ctx, metav1.ListOptions{})
		s.NoError(err, "should be able to list DeleteBackupRequests")
		s.Len(dbrList.Items, 1, "exactly one DeleteBackupRequest should have been created")

		dbr := dbrList.Items[0]
		s.Contains(dbr.GetName(), "delete-test-delete-", "DeleteBackupRequest name should start with backup name")

		// Verify the DeleteBackupRequest references the correct backup
		backupName, found, err := unstructured.NestedString(dbr.Object, "spec", "backupName")
		s.NoError(err)
		s.True(found, "DeleteBackupRequest should have spec.backupName")
		s.Equal("delete-test", backupName, "DeleteBackupRequest should reference the correct backup")
	})
}

func (s *BackupSuite) TestGetBackupStatus() {
	s.Run("returns formatted status for completed backup", func() {
		backup := s.createTestBackup("status-test", DefaultOADPNamespace)

		// Set status fields
		s.Require().NoError(unstructured.SetNestedField(backup.Object, "Completed", "status", "phase"))
		s.Require().NoError(unstructured.SetNestedField(backup.Object, int64(0), "status", "errors"))
		s.Require().NoError(unstructured.SetNestedField(backup.Object, int64(2), "status", "warnings"))
		_, err := s.client.Resource(BackupGVR).Namespace(DefaultOADPNamespace).Update(s.ctx, backup, metav1.UpdateOptions{})
		s.Require().NoError(err)

		status, err := GetBackupStatus(s.ctx, s.client, DefaultOADPNamespace, "status-test")
		s.NoError(err)
		s.Contains(status, "Completed")
		s.Contains(status, "Warnings: 2")
	})
}

func TestBackupSuite(t *testing.T) {
	suite.Run(t, new(BackupSuite))
}
