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

type NonAdminSuite struct {
	suite.Suite
	ctx    context.Context
	client *fake.FakeDynamicClient
}

func (s *NonAdminSuite) SetupTest() {
	s.ctx = context.Background()
	gvrToListKind := map[schema.GroupVersionResource]string{
		NonAdminBackupGVR:                       "NonAdminBackupList",
		NonAdminRestoreGVR:                      "NonAdminRestoreList",
		NonAdminBackupStorageLocationGVR:        "NonAdminBackupStorageLocationList",
		NonAdminBackupStorageLocationRequestGVR: "NonAdminBackupStorageLocationRequestList",
		NonAdminDownloadRequestGVR:              "NonAdminDownloadRequestList",
	}
	s.client = fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToListKind)
}

// NonAdminBackup tests

func (s *NonAdminSuite) createTestNonAdminBackup(name, namespace string) *unstructured.Unstructured {
	nab := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": OADPGroup + "/" + OADPVersion,
			"kind":       "NonAdminBackup",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]any{
				"backupSpec": map[string]any{},
			},
		},
	}
	created, err := s.client.Resource(NonAdminBackupGVR).Namespace(namespace).Create(s.ctx, nab, metav1.CreateOptions{})
	s.Require().NoError(err)
	return created
}

func (s *NonAdminSuite) TestListNonAdminBackups() {
	s.Run("returns empty list when no non-admin backups exist", func() {
		list, err := ListNonAdminBackups(s.ctx, s.client, "test-ns", metav1.ListOptions{})
		s.NoError(err)
		s.Empty(list.Items)
	})

	s.Run("returns non-admin backups in namespace", func() {
		s.createTestNonAdminBackup("test-nab", "test-ns")

		list, err := ListNonAdminBackups(s.ctx, s.client, "test-ns", metav1.ListOptions{})
		s.NoError(err)
		s.Len(list.Items, 1)
		s.Equal("test-nab", list.Items[0].GetName())
	})
}

func (s *NonAdminSuite) TestGetNonAdminBackup() {
	s.Run("returns non-admin backup by name", func() {
		s.createTestNonAdminBackup("get-test", "test-ns")

		nab, err := GetNonAdminBackup(s.ctx, s.client, "test-ns", "get-test")
		s.NoError(err)
		s.Equal("get-test", nab.GetName())
	})

	s.Run("returns error for non-existent non-admin backup", func() {
		_, err := GetNonAdminBackup(s.ctx, s.client, "test-ns", "non-existent")
		s.Error(err)
	})
}

func (s *NonAdminSuite) TestCreateNonAdminBackup() {
	s.Run("creates non-admin backup", func() {
		nab := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": OADPGroup + "/" + OADPVersion,
				"kind":       "NonAdminBackup",
				"metadata": map[string]any{
					"name":      "create-test",
					"namespace": "test-ns",
				},
				"spec": map[string]any{
					"backupSpec": map[string]any{},
				},
			},
		}

		created, err := CreateNonAdminBackup(s.ctx, s.client, nab)
		s.NoError(err)
		s.Equal("create-test", created.GetName())
	})
}

func (s *NonAdminSuite) TestDeleteNonAdminBackup() {
	s.Run("deletes existing non-admin backup", func() {
		s.createTestNonAdminBackup("delete-test", "test-ns")

		err := DeleteNonAdminBackup(s.ctx, s.client, "test-ns", "delete-test")
		s.NoError(err)

		_, err = GetNonAdminBackup(s.ctx, s.client, "test-ns", "delete-test")
		s.Error(err)
	})
}

// NonAdminRestore tests

func (s *NonAdminSuite) createTestNonAdminRestore(name, namespace string) *unstructured.Unstructured {
	nar := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": OADPGroup + "/" + OADPVersion,
			"kind":       "NonAdminRestore",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]any{
				"restoreSpec": map[string]any{
					"backupName": "test-backup",
				},
			},
		},
	}
	created, err := s.client.Resource(NonAdminRestoreGVR).Namespace(namespace).Create(s.ctx, nar, metav1.CreateOptions{})
	s.Require().NoError(err)
	return created
}

func (s *NonAdminSuite) TestListNonAdminRestores() {
	s.Run("returns non-admin restores in namespace", func() {
		s.createTestNonAdminRestore("test-nar", "test-ns")

		list, err := ListNonAdminRestores(s.ctx, s.client, "test-ns", metav1.ListOptions{})
		s.NoError(err)
		s.Len(list.Items, 1)
		s.Equal("test-nar", list.Items[0].GetName())
	})
}

func (s *NonAdminSuite) TestGetNonAdminRestore() {
	s.Run("returns non-admin restore by name", func() {
		s.createTestNonAdminRestore("get-test", "test-ns")

		nar, err := GetNonAdminRestore(s.ctx, s.client, "test-ns", "get-test")
		s.NoError(err)
		s.Equal("get-test", nar.GetName())
	})
}

func (s *NonAdminSuite) TestCreateNonAdminRestore() {
	s.Run("creates non-admin restore", func() {
		nar := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": OADPGroup + "/" + OADPVersion,
				"kind":       "NonAdminRestore",
				"metadata": map[string]any{
					"name":      "create-test",
					"namespace": "test-ns",
				},
				"spec": map[string]any{
					"restoreSpec": map[string]any{
						"backupName": "test-backup",
					},
				},
			},
		}

		created, err := CreateNonAdminRestore(s.ctx, s.client, nar)
		s.NoError(err)
		s.Equal("create-test", created.GetName())
	})
}

func (s *NonAdminSuite) TestDeleteNonAdminRestore() {
	s.Run("deletes existing non-admin restore", func() {
		s.createTestNonAdminRestore("delete-test", "test-ns")

		err := DeleteNonAdminRestore(s.ctx, s.client, "test-ns", "delete-test")
		s.NoError(err)

		_, err = GetNonAdminRestore(s.ctx, s.client, "test-ns", "delete-test")
		s.Error(err)
	})
}

// NonAdminBackupStorageLocation tests

func (s *NonAdminSuite) createTestNonAdminBSL(name, namespace string) *unstructured.Unstructured {
	nabsl := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": OADPGroup + "/" + OADPVersion,
			"kind":       "NonAdminBackupStorageLocation",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]any{
				"backupStorageLocationSpec": map[string]any{
					"provider": "aws",
					"objectStorage": map[string]any{
						"bucket": "test-bucket",
					},
				},
			},
		},
	}
	created, err := s.client.Resource(NonAdminBackupStorageLocationGVR).Namespace(namespace).Create(s.ctx, nabsl, metav1.CreateOptions{})
	s.Require().NoError(err)
	return created
}

func (s *NonAdminSuite) TestListNonAdminBSLs() {
	s.Run("returns non-admin BSLs in namespace", func() {
		s.createTestNonAdminBSL("test-nabsl", "test-ns")

		list, err := ListNonAdminBackupStorageLocations(s.ctx, s.client, "test-ns", metav1.ListOptions{})
		s.NoError(err)
		s.Len(list.Items, 1)
		s.Equal("test-nabsl", list.Items[0].GetName())
	})
}

func (s *NonAdminSuite) TestGetNonAdminBSL() {
	s.Run("returns non-admin BSL by name", func() {
		s.createTestNonAdminBSL("get-test", "test-ns")

		nabsl, err := GetNonAdminBackupStorageLocation(s.ctx, s.client, "test-ns", "get-test")
		s.NoError(err)
		s.Equal("get-test", nabsl.GetName())
	})
}

func (s *NonAdminSuite) TestCreateNonAdminBSL() {
	s.Run("creates non-admin BSL", func() {
		nabsl := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": OADPGroup + "/" + OADPVersion,
				"kind":       "NonAdminBackupStorageLocation",
				"metadata": map[string]any{
					"name":      "create-test",
					"namespace": "test-ns",
				},
				"spec": map[string]any{
					"backupStorageLocationSpec": map[string]any{
						"provider": "aws",
					},
				},
			},
		}

		created, err := CreateNonAdminBackupStorageLocation(s.ctx, s.client, nabsl)
		s.NoError(err)
		s.Equal("create-test", created.GetName())
	})
}

func (s *NonAdminSuite) TestUpdateNonAdminBSL() {
	s.Run("updates non-admin BSL", func() {
		nabsl := s.createTestNonAdminBSL("update-test", "test-ns")

		// Modify the spec
		err := unstructured.SetNestedField(nabsl.Object, "gcp", "spec", "backupStorageLocationSpec", "provider")
		s.Require().NoError(err)

		updated, err := UpdateNonAdminBackupStorageLocation(s.ctx, s.client, nabsl)
		s.NoError(err)

		provider, _, _ := unstructured.NestedString(updated.Object, "spec", "backupStorageLocationSpec", "provider")
		s.Equal("gcp", provider)
	})
}

func (s *NonAdminSuite) TestDeleteNonAdminBSL() {
	s.Run("deletes existing non-admin BSL", func() {
		s.createTestNonAdminBSL("delete-test", "test-ns")

		err := DeleteNonAdminBackupStorageLocation(s.ctx, s.client, "test-ns", "delete-test")
		s.NoError(err)

		_, err = GetNonAdminBackupStorageLocation(s.ctx, s.client, "test-ns", "delete-test")
		s.Error(err)
	})
}

// NonAdminBackupStorageLocationRequest tests

func (s *NonAdminSuite) createTestNonAdminBSLRequest(name, namespace string) *unstructured.Unstructured {
	nabslr := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": OADPGroup + "/" + OADPVersion,
			"kind":       "NonAdminBackupStorageLocationRequest",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]any{
				"nonAdminBackupStorageLocationName": "test-nabsl",
			},
		},
	}
	created, err := s.client.Resource(NonAdminBackupStorageLocationRequestGVR).Namespace(namespace).Create(s.ctx, nabslr, metav1.CreateOptions{})
	s.Require().NoError(err)
	return created
}

func (s *NonAdminSuite) TestListNonAdminBSLRequests() {
	s.Run("returns non-admin BSL requests in namespace", func() {
		s.createTestNonAdminBSLRequest("test-nabslr", DefaultOADPNamespace)

		list, err := ListNonAdminBackupStorageLocationRequests(s.ctx, s.client, DefaultOADPNamespace, metav1.ListOptions{})
		s.NoError(err)
		s.Len(list.Items, 1)
		s.Equal("test-nabslr", list.Items[0].GetName())
	})
}

func (s *NonAdminSuite) TestGetNonAdminBSLRequest() {
	s.Run("returns non-admin BSL request by name", func() {
		s.createTestNonAdminBSLRequest("get-test", DefaultOADPNamespace)

		nabslr, err := GetNonAdminBackupStorageLocationRequest(s.ctx, s.client, DefaultOADPNamespace, "get-test")
		s.NoError(err)
		s.Equal("get-test", nabslr.GetName())
	})
}

func (s *NonAdminSuite) TestApproveNonAdminBSLRequest() {
	s.Run("approves non-admin BSL request", func() {
		s.createTestNonAdminBSLRequest("approve-test", DefaultOADPNamespace)

		updated, err := ApproveNonAdminBackupStorageLocationRequest(s.ctx, s.client, DefaultOADPNamespace, "approve-test", "Approved")
		s.NoError(err)

		decision, _, _ := unstructured.NestedString(updated.Object, "spec", "approvalDecision")
		s.Equal("Approved", decision)
	})
}

// NonAdminDownloadRequest tests

func (s *NonAdminSuite) createTestNonAdminDownloadRequest(name, namespace string) *unstructured.Unstructured {
	nadr := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": OADPGroup + "/" + OADPVersion,
			"kind":       "NonAdminDownloadRequest",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]any{
				"target": map[string]any{
					"kind": "BackupLog",
					"name": "test-backup",
				},
			},
		},
	}
	created, err := s.client.Resource(NonAdminDownloadRequestGVR).Namespace(namespace).Create(s.ctx, nadr, metav1.CreateOptions{})
	s.Require().NoError(err)
	return created
}

func (s *NonAdminSuite) TestListNonAdminDownloadRequests() {
	s.Run("returns non-admin download requests in namespace", func() {
		s.createTestNonAdminDownloadRequest("test-nadr", "test-ns")

		list, err := ListNonAdminDownloadRequests(s.ctx, s.client, "test-ns", metav1.ListOptions{})
		s.NoError(err)
		s.Len(list.Items, 1)
		s.Equal("test-nadr", list.Items[0].GetName())
	})
}

func (s *NonAdminSuite) TestGetNonAdminDownloadRequest() {
	s.Run("returns non-admin download request by name", func() {
		s.createTestNonAdminDownloadRequest("get-test", "test-ns")

		nadr, err := GetNonAdminDownloadRequest(s.ctx, s.client, "test-ns", "get-test")
		s.NoError(err)
		s.Equal("get-test", nadr.GetName())
	})
}

func (s *NonAdminSuite) TestCreateNonAdminDownloadRequest() {
	s.Run("creates non-admin download request", func() {
		nadr := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": OADPGroup + "/" + OADPVersion,
				"kind":       "NonAdminDownloadRequest",
				"metadata": map[string]any{
					"name":      "create-test",
					"namespace": "test-ns",
				},
				"spec": map[string]any{
					"target": map[string]any{
						"kind": "BackupLog",
						"name": "test-backup",
					},
				},
			},
		}

		created, err := CreateNonAdminDownloadRequest(s.ctx, s.client, nadr)
		s.NoError(err)
		s.Equal("create-test", created.GetName())
	})
}

func (s *NonAdminSuite) TestDeleteNonAdminDownloadRequest() {
	s.Run("deletes existing non-admin download request", func() {
		s.createTestNonAdminDownloadRequest("delete-test", "test-ns")

		err := DeleteNonAdminDownloadRequest(s.ctx, s.client, "test-ns", "delete-test")
		s.NoError(err)

		_, err = GetNonAdminDownloadRequest(s.ctx, s.client, "test-ns", "delete-test")
		s.Error(err)
	})
}

func TestNonAdminSuite(t *testing.T) {
	suite.Run(t, new(NonAdminSuite))
}
