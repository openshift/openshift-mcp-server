package oadp

import (
	"context"
	"strings"
	"testing"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	fakedynamic "k8s.io/client-go/dynamic/fake"
)

type OADPRetentionSuite struct {
	suite.Suite
}

type retentionClient struct {
	api.KubernetesClient
	dynamic dynamic.Interface
}

func (c retentionClient) DynamicClient() dynamic.Interface {
	return c.dynamic
}

type retentionRequest struct{}

func (retentionRequest) GetArguments() map[string]string {
	return map[string]string{"namespace": "oadp"}
}

func (s *OADPRetentionSuite) TestPromptStopsFetchingAfterDPAFillsBudget() {
	dpaGVR := schema.GroupVersionResource{Group: "oadp.openshift.io", Version: "v1alpha1", Resource: "dataprotectionapplications"}
	listKinds := map[schema.GroupVersionResource]string{
		dpaGVR: "DataProtectionApplicationList",
	}
	client := fakedynamic.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), listKinds)
	dpa := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "oadp.openshift.io/v1alpha1",
			"kind":       "DataProtectionApplication",
			"metadata": map[string]any{
				"name":      "example",
				"namespace": "oadp",
			},
			"status": map[string]any{"message": strings.Repeat("long status ", 300)},
		},
	}

	_, err := client.Resource(dpaGVR).Namespace("oadp").Create(context.Background(), dpa, metav1.CreateOptions{})
	s.Require().NoError(err)

	ctx := api.WithResultBudget(context.Background(), 600)
	result, err := oadpTroubleshootHandler(api.PromptHandlerParams{
		Context:          ctx,
		KubernetesClient: retentionClient{dynamic: client},
		Request:          retentionRequest{},
	})

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Contains(result.Messages[0].Content.Text, api.ResultTruncationNotice)

	lists := 0
	for _, action := range client.Actions() {
		if action.GetVerb() == "list" {
			lists++
			s.Equal("dataprotectionapplications", action.GetResource().Resource)
		}
	}
	s.Equal(1, lists)
}

func (s *OADPRetentionSuite) TestLogFilterKeepsNoticeAfterPrefix() {
	budget := api.NewBuilderWithBudget(20)
	filterLogLines("error first\nerror second\nerror third", []string{"error"}, budget)

	text, cut := budget.Prefix()
	s.True(cut)
	s.Contains(text, "error first")
	s.NotContains(text, "error third")
	s.Equal(text+api.ResultTruncationNotice, budget.String())
}

func TestOADPRetentionSuite(t *testing.T) {
	suite.Run(t, new(OADPRetentionSuite))
}
