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

type DataMoverSuite struct {
	suite.Suite
	ctx    context.Context
	client *fake.FakeDynamicClient
}

func (s *DataMoverSuite) SetupTest() {
	s.ctx = context.Background()
	gvrToListKind := map[schema.GroupVersionResource]string{
		DataUploadGVR:   "DataUploadList",
		DataDownloadGVR: "DataDownloadList",
	}
	s.client = fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToListKind)
}

func (s *DataMoverSuite) createTestDataUpload(name, namespace string) *unstructured.Unstructured {
	du := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": VeleroGroup + "/" + VeleroV2Alpha1Version,
			"kind":       "DataUpload",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]any{
				"snapshotType":          "CSI",
				"csiSnapshot":           map[string]any{"snapshotClass": "test"},
				"backupStorageLocation": "default",
			},
		},
	}
	created, err := s.client.Resource(DataUploadGVR).Namespace(namespace).Create(s.ctx, du, metav1.CreateOptions{})
	s.Require().NoError(err)
	return created
}

func (s *DataMoverSuite) createTestDataDownload(name, namespace string) *unstructured.Unstructured {
	dd := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": VeleroGroup + "/" + VeleroV2Alpha1Version,
			"kind":       "DataDownload",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]any{
				"snapshotID":            "test-snapshot",
				"backupStorageLocation": "default",
			},
		},
	}
	created, err := s.client.Resource(DataDownloadGVR).Namespace(namespace).Create(s.ctx, dd, metav1.CreateOptions{})
	s.Require().NoError(err)
	return created
}

func (s *DataMoverSuite) TestListDataUploads() {
	s.Run("returns empty list when no data uploads exist", func() {
		list, err := ListDataUploads(s.ctx, s.client, DefaultOADPNamespace, metav1.ListOptions{})
		s.NoError(err)
		s.Empty(list.Items)
	})

	s.Run("returns data uploads in namespace", func() {
		s.createTestDataUpload("test-du", DefaultOADPNamespace)

		list, err := ListDataUploads(s.ctx, s.client, DefaultOADPNamespace, metav1.ListOptions{})
		s.NoError(err)
		s.Len(list.Items, 1)
		s.Equal("test-du", list.Items[0].GetName())
	})
}

func (s *DataMoverSuite) TestGetDataUpload() {
	s.Run("returns data upload by name", func() {
		s.createTestDataUpload("get-test", DefaultOADPNamespace)

		du, err := GetDataUpload(s.ctx, s.client, DefaultOADPNamespace, "get-test")
		s.NoError(err)
		s.Equal("get-test", du.GetName())
	})

	s.Run("returns error for non-existent data upload", func() {
		_, err := GetDataUpload(s.ctx, s.client, DefaultOADPNamespace, "non-existent")
		s.Error(err)
	})
}

func (s *DataMoverSuite) TestCancelDataUpload() {
	s.Run("sets cancel flag on data upload", func() {
		s.createTestDataUpload("cancel-test", DefaultOADPNamespace)

		updated, err := CancelDataUpload(s.ctx, s.client, DefaultOADPNamespace, "cancel-test")
		s.NoError(err)

		cancel, _, _ := unstructured.NestedBool(updated.Object, "spec", "cancel")
		s.True(cancel)
	})

	s.Run("returns error for non-existent data upload", func() {
		_, err := CancelDataUpload(s.ctx, s.client, DefaultOADPNamespace, "non-existent")
		s.Error(err)
	})
}

func (s *DataMoverSuite) TestListDataDownloads() {
	s.Run("returns empty list when no data downloads exist", func() {
		list, err := ListDataDownloads(s.ctx, s.client, DefaultOADPNamespace, metav1.ListOptions{})
		s.NoError(err)
		s.Empty(list.Items)
	})

	s.Run("returns data downloads in namespace", func() {
		s.createTestDataDownload("test-dd", DefaultOADPNamespace)

		list, err := ListDataDownloads(s.ctx, s.client, DefaultOADPNamespace, metav1.ListOptions{})
		s.NoError(err)
		s.Len(list.Items, 1)
		s.Equal("test-dd", list.Items[0].GetName())
	})
}

func (s *DataMoverSuite) TestGetDataDownload() {
	s.Run("returns data download by name", func() {
		s.createTestDataDownload("get-test", DefaultOADPNamespace)

		dd, err := GetDataDownload(s.ctx, s.client, DefaultOADPNamespace, "get-test")
		s.NoError(err)
		s.Equal("get-test", dd.GetName())
	})

	s.Run("returns error for non-existent data download", func() {
		_, err := GetDataDownload(s.ctx, s.client, DefaultOADPNamespace, "non-existent")
		s.Error(err)
	})
}

func (s *DataMoverSuite) TestCancelDataDownload() {
	s.Run("sets cancel flag on data download", func() {
		s.createTestDataDownload("cancel-test", DefaultOADPNamespace)

		updated, err := CancelDataDownload(s.ctx, s.client, DefaultOADPNamespace, "cancel-test")
		s.NoError(err)

		cancel, _, _ := unstructured.NestedBool(updated.Object, "spec", "cancel")
		s.True(cancel)
	})

	s.Run("returns error for non-existent data download", func() {
		_, err := CancelDataDownload(s.ctx, s.client, DefaultOADPNamespace, "non-existent")
		s.Error(err)
	})
}

func TestDataMoverSuite(t *testing.T) {
	suite.Run(t, new(DataMoverSuite))
}
