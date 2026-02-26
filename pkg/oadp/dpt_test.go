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

type DataProtectionTestSuite struct {
	suite.Suite
	ctx    context.Context
	client *fake.FakeDynamicClient
}

func (s *DataProtectionTestSuite) SetupTest() {
	s.ctx = context.Background()
	gvrToListKind := map[schema.GroupVersionResource]string{
		DataProtectionTestGVR: "DataProtectionTestList",
	}
	s.client = fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToListKind)
}

func (s *DataProtectionTestSuite) createTestDataProtectionTest(name, namespace string) *unstructured.Unstructured {
	dpt := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": OADPGroup + "/" + OADPVersion,
			"kind":       "DataProtectionTest",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]any{
				"backupLocationName": "default",
			},
		},
	}
	created, err := s.client.Resource(DataProtectionTestGVR).Namespace(namespace).Create(s.ctx, dpt, metav1.CreateOptions{})
	s.Require().NoError(err)
	return created
}

func (s *DataProtectionTestSuite) TestListDataProtectionTests() {
	s.Run("returns empty list when no data protection tests exist", func() {
		list, err := ListDataProtectionTests(s.ctx, s.client, DefaultOADPNamespace, metav1.ListOptions{})
		s.NoError(err)
		s.Empty(list.Items)
	})

	s.Run("returns data protection tests in namespace", func() {
		s.createTestDataProtectionTest("test-dpt", DefaultOADPNamespace)

		list, err := ListDataProtectionTests(s.ctx, s.client, DefaultOADPNamespace, metav1.ListOptions{})
		s.NoError(err)
		s.Len(list.Items, 1)
		s.Equal("test-dpt", list.Items[0].GetName())
	})
}

func (s *DataProtectionTestSuite) TestGetDataProtectionTest() {
	s.Run("returns data protection test by name", func() {
		s.createTestDataProtectionTest("get-test", DefaultOADPNamespace)

		dpt, err := GetDataProtectionTest(s.ctx, s.client, DefaultOADPNamespace, "get-test")
		s.NoError(err)
		s.Equal("get-test", dpt.GetName())
	})

	s.Run("returns error for non-existent data protection test", func() {
		_, err := GetDataProtectionTest(s.ctx, s.client, DefaultOADPNamespace, "non-existent")
		s.Error(err)
	})
}

func (s *DataProtectionTestSuite) TestCreateDataProtectionTest() {
	s.Run("creates data protection test", func() {
		dpt := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": OADPGroup + "/" + OADPVersion,
				"kind":       "DataProtectionTest",
				"metadata": map[string]any{
					"name":      "create-test",
					"namespace": DefaultOADPNamespace,
				},
				"spec": map[string]any{
					"backupLocationName": "default",
				},
			},
		}

		created, err := CreateDataProtectionTest(s.ctx, s.client, dpt)
		s.NoError(err)
		s.Equal("create-test", created.GetName())

		// Verify it was created
		fetched, err := GetDataProtectionTest(s.ctx, s.client, DefaultOADPNamespace, "create-test")
		s.NoError(err)
		s.Equal("create-test", fetched.GetName())
	})
}

func (s *DataProtectionTestSuite) TestDeleteDataProtectionTest() {
	s.Run("deletes existing data protection test", func() {
		s.createTestDataProtectionTest("delete-test", DefaultOADPNamespace)

		err := DeleteDataProtectionTest(s.ctx, s.client, DefaultOADPNamespace, "delete-test")
		s.NoError(err)

		// Verify it's deleted
		_, err = GetDataProtectionTest(s.ctx, s.client, DefaultOADPNamespace, "delete-test")
		s.Error(err)
	})

	s.Run("returns error for non-existent data protection test", func() {
		err := DeleteDataProtectionTest(s.ctx, s.client, DefaultOADPNamespace, "non-existent")
		s.Error(err)
	})
}

func TestDataProtectionTestSuite(t *testing.T) {
	suite.Run(t, new(DataProtectionTestSuite))
}
