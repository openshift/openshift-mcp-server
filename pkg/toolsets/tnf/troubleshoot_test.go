package tnf

import (
	"context"
	"testing"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/tnf/fencing"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

type mockPromptCallRequest struct {
	args map[string]string
}

func (m *mockPromptCallRequest) GetArguments() map[string]string {
	return m.args
}

var _ api.PromptCallRequest = (*mockPromptCallRequest)(nil)

type fakeCoreV1 struct {
	corev1client.CoreV1Interface
	nodes   []*corev1.Node
	secrets []*corev1.Secret
}

func (f *fakeCoreV1) Nodes() corev1client.NodeInterface {
	return &fakeNodes{nodes: f.nodes}
}

func (f *fakeCoreV1) Secrets(namespace string) corev1client.SecretInterface {
	return &fakeSecrets{namespace: namespace, secrets: f.secrets}
}

type fakeNodes struct {
	corev1client.NodeInterface
	nodes []*corev1.Node
}

func (f *fakeNodes) List(_ context.Context, _ metav1.ListOptions) (*corev1.NodeList, error) {
	items := make([]corev1.Node, len(f.nodes))
	for i, n := range f.nodes {
		items[i] = *n
	}
	return &corev1.NodeList{Items: items}, nil
}

type fakeSecrets struct {
	corev1client.SecretInterface
	namespace string
	secrets   []*corev1.Secret
}

func (f *fakeSecrets) Get(_ context.Context, name string, _ metav1.GetOptions) (*corev1.Secret, error) {
	for _, s := range f.secrets {
		if s.Name == name && s.Namespace == f.namespace {
			return s, nil
		}
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, name)
}

func newFakeCoreV1(objects ...runtime.Object) corev1client.CoreV1Interface {
	f := &fakeCoreV1{}
	for _, obj := range objects {
		switch o := obj.(type) {
		case *corev1.Node:
			f.nodes = append(f.nodes, o)
		case *corev1.Secret:
			f.secrets = append(f.secrets, o)
		}
	}
	return f
}

type TNFTroubleshootSuite struct {
	suite.Suite
}

func (s *TNFTroubleshootSuite) TestPromptRegistration() {
	prompts := initTNFTroubleshoot()
	s.Require().Len(prompts, 1)
	s.Equal("tnf-troubleshoot", prompts[0].Prompt.Name)
	s.Equal("TNF Fencing Troubleshoot", prompts[0].Prompt.Title)
	s.Len(prompts[0].Prompt.Arguments, 2)
	s.Equal("node", prompts[0].Prompt.Arguments[0].Name)
	s.False(prompts[0].Prompt.Arguments[0].Required)
	s.Equal("namespace", prompts[0].Prompt.Arguments[1].Name)
	s.False(prompts[0].Prompt.Arguments[1].Required)
	s.NotNil(prompts[0].Handler)
}

func (s *TNFTroubleshootSuite) TestToolsetGetPrompts() {
	toolset := &Toolset{}
	prompts := toolset.GetPrompts()
	s.Require().Len(prompts, 1)
	s.Equal("tnf-troubleshoot", prompts[0].Prompt.Name)
}

func (s *TNFTroubleshootSuite) TestFetchClusterTopology() {
	ctx := context.Background()
	gvrToListKind := map[schema.GroupVersionResource]string{
		fencing.InfrastructureGVR: "InfrastructureList",
	}

	s.Run("detects platform from infrastructure CR", func() {
		infra := &unstructured.Unstructured{}
		infra.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "config.openshift.io/v1",
			"kind":       "Infrastructure",
			"metadata": map[string]interface{}{
				"name": "cluster",
			},
			"status": map[string]interface{}{
				"platform":               "BareMetal",
				"infrastructureTopology": "HighlyAvailable",
				"controlPlaneTopology":   "HighlyAvailable",
			},
		})

		node1 := &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "master-0",
				Labels: map[string]string{"node-role.kubernetes.io/master": ""},
			},
		}
		node2 := &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "master-1",
				Labels: map[string]string{"node-role.kubernetes.io/master": ""},
			},
		}

		dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToListKind, infra)
		coreClient := newFakeCoreV1(node1, node2)

		result, isTNF := fetchClusterTopology(ctx, dynamicClient, coreClient)
		s.Contains(result, "BareMetal")
		s.Contains(result, "TNF Profile:** Yes")
		s.Contains(result, "Total Nodes:** 2")
		s.True(isTNF)
	})

	s.Run("handles missing infrastructure CR", func() {
		dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToListKind)
		coreClient := newFakeCoreV1()

		result, isTNF := fetchClusterTopology(ctx, dynamicClient, coreClient)
		s.Contains(result, "not available")
		s.False(isTNF)
	})
}

func (s *TNFTroubleshootSuite) TestFetchNodeHealth() {
	ctx := context.Background()

	s.Run("reports node status table", func() {
		node := &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "master-0",
				Labels: map[string]string{
					"node-role.kubernetes.io/master":        "",
					"node-role.kubernetes.io/control-plane": "",
				},
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: "Ready", Status: "True"},
				},
			},
		}

		coreClient := newFakeCoreV1(node)
		result := fetchNodeHealth(ctx, coreClient)
		s.Contains(result, "master-0")
		s.Contains(result, "Yes")
		s.Contains(result, "master")
	})

	s.Run("reports NotReady node", func() {
		node := &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "master-0",
				Labels: map[string]string{"node-role.kubernetes.io/master": ""},
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: "Ready", Status: "False"},
				},
			},
		}

		coreClient := newFakeCoreV1(node)
		result := fetchNodeHealth(ctx, coreClient)
		s.Contains(result, "**No**")
	})
}

func (s *TNFTroubleshootSuite) TestFetchOperatorHealth() {
	ctx := context.Background()
	gvrToListKind := map[schema.GroupVersionResource]string{
		fencing.ClusterOperatorGVR: "ClusterOperatorList",
	}

	s.Run("reports operator status", func() {
		etcdOp := &unstructured.Unstructured{}
		etcdOp.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "config.openshift.io/v1",
			"kind":       "ClusterOperator",
			"metadata": map[string]interface{}{
				"name": "etcd",
			},
			"status": map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"type":   "Available",
						"status": "True",
					},
					map[string]interface{}{
						"type":   "Degraded",
						"status": "False",
					},
				},
			},
		})

		dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToListKind, etcdOp)
		result := fetchOperatorHealth(ctx, dynamicClient)
		s.Contains(result, "etcd")
		s.Contains(result, "True")
		s.Contains(result, "machine-api")
		s.Contains(result, "NOT FOUND")
	})
}

func (s *TNFTroubleshootSuite) TestFetchBMHStatus() {
	ctx := context.Background()
	gvrToListKind := map[schema.GroupVersionResource]string{
		fencing.BareMetalHostGVR: "BareMetalHostList",
	}

	s.Run("reports BMH details", func() {
		bmh := &unstructured.Unstructured{}
		bmh.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "metal3.io/v1alpha1",
			"kind":       "BareMetalHost",
			"metadata": map[string]interface{}{
				"name":      "master-0",
				"namespace": "openshift-machine-api",
			},
			"spec": map[string]interface{}{
				"bmc": map[string]interface{}{
					"address":         "redfish://192.168.1.10/redfish/v1/Systems/1",
					"credentialsName": "master-0-bmc-secret",
				},
				"online": true,
			},
			"status": map[string]interface{}{
				"provisioning": map[string]interface{}{
					"state": "externally provisioned",
				},
				"operationalStatus": "OK",
				"poweredOn":         true,
			},
		})

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "master-0-bmc-secret",
				Namespace: "openshift-machine-api",
			},
			Data: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte("secret"),
			},
		}

		dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToListKind, bmh)
		coreClient := newFakeCoreV1(secret)

		result := fetchBMHStatus(ctx, dynamicClient, coreClient, "openshift-machine-api")
		s.Contains(result, "master-0")
		s.Contains(result, "redfish://192.168.1.10")
		s.Contains(result, "externally provisioned")
		s.Contains(result, "Valid")
	})

	s.Run("handles no BMH resources", func() {
		dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToListKind)
		coreClient := newFakeCoreV1()

		result := fetchBMHStatus(ctx, dynamicClient, coreClient, "openshift-machine-api")
		s.Contains(result, "No BareMetalHost resources found")
	})
}

func (s *TNFTroubleshootSuite) TestFetchRemediationStatus() {
	ctx := context.Background()
	gvrToListKind := map[schema.GroupVersionResource]string{
		fencing.FenceAgentsRemediationTemplateGVR: "FenceAgentsRemediationTemplateList",
		fencing.FenceAgentsRemediationGVR:         "FenceAgentsRemediationList",
		fencing.NodeHealthCheckGVR:                "NodeHealthCheckList",
	}

	s.Run("reports empty state", func() {
		dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToListKind)
		result := fetchRemediationStatus(ctx, dynamicClient)
		s.Contains(result, "No FenceAgentsRemediationTemplates configured")
		s.Contains(result, "No active fencing remediations")
		s.Contains(result, "No NodeHealthCheck resources configured")
	})
}

func (s *TNFTroubleshootSuite) TestCheckCredentialSecret() {
	ctx := context.Background()

	s.Run("valid credentials", func() {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "bmc-secret",
				Namespace: "ns",
			},
			Data: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte("pass"),
			},
		}

		coreClient := newFakeCoreV1(secret)
		result := checkCredentialSecret(ctx, coreClient, "ns", "bmc-secret")
		s.Contains(result, "Valid")
	})

	s.Run("missing username", func() {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "bmc-secret",
				Namespace: "ns",
			},
			Data: map[string][]byte{
				"password": []byte("pass"),
			},
		}

		coreClient := newFakeCoreV1(secret)
		result := checkCredentialSecret(ctx, coreClient, "ns", "bmc-secret")
		s.Contains(result, "missing or empty 'username'")
	})

	s.Run("missing secret", func() {
		coreClient := newFakeCoreV1()
		result := checkCredentialSecret(ctx, coreClient, "ns", "nonexistent")
		s.Contains(result, "not found")
	})
}

func (s *TNFTroubleshootSuite) TestValueOrNA() {
	s.Equal("N/A", fencing.ValueOrNA(""))
	s.Equal("test", fencing.ValueOrNA("test"))
}

func TestTNFTroubleshoot(t *testing.T) {
	suite.Run(t, new(TNFTroubleshootSuite))
}
