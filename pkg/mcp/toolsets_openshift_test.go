package mcp

import (
	"net/http"
	"strings"

	"github.com/containers/kubernetes-mcp-server/internal/test"
	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets"
	clusterDiagnosticsToolset "github.com/containers/kubernetes-mcp-server/pkg/toolsets/cluster-diagnostics"
	cniDiagnosticsToolset "github.com/containers/kubernetes-mcp-server/pkg/toolsets/cni-diagnostics"
	mgToolset "github.com/containers/kubernetes-mcp-server/pkg/toolsets/mustgather"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/ovnkubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Downstream-only toolset coverage.
//
// pkg/mcp/toolsets_test.go is kept byte-identical to upstream so upstream
// syncs merge cleanly; every assertion that covers toolsets carried only
// downstream lives in this file instead. Both sides used to append entries
// to the same registration list and test functions to the end of the shared
// file, which conflicted on nearly every sync.

func (s *ToolsetsSuite) TestDownstreamToolsetsTools() {
	s.ResetHandlers()
	s.Handle(test.NewDiscoveryClientHandler(metav1.APIResourceList{
		GroupVersion: "perses.dev/v1alpha2",
		APIResources: []metav1.APIResource{
			{Name: "perses", Kind: "Perses", Namespaced: true, Verbs: metav1.Verbs{"list"}},
		},
	}))
	s.handlePersesList(http.StatusOK, persesListWithInstance)
	testCases := []api.Toolset{
		&ovnkubernetes.Toolset{},
		&clusterDiagnosticsToolset.Toolset{},
		&cniDiagnosticsToolset.Toolset{},
	}
	for _, testCase := range testCases {
		s.Run("Toolset "+testCase.GetName(), func() {
			toolsets.Clear()
			toolsets.Register(testCase)
			s.Cfg.Toolsets.SetForTest([]string{testCase.GetName()})
			s.InitMcpClient()
			tools, err := s.ListTools()
			s.Run("ListTools returns tools", func() {
				s.NotNil(tools, "Expected tools from ListTools")
				s.NoError(err, "Expected no error from ListTools")
			})
			s.Run("ListTools returns correct Tool metadata", func() {
				s.assertJsonSnapshot("toolsets-"+strings.ReplaceAll(testCase.GetName(), "/", "-")+"-tools.json", tools.Tools)
			})
		})
	}
}

func (s *ToolsetsSuite) TestOpenShiftMustGatherToolsetPrompts() {
	s.Run("OpenShift must-gather toolset prompts", func() {
		s.Handle(test.NewInOpenShiftHandler())
		toolsets.Clear()
		toolsets.Register(&mgToolset.Toolset{})
		s.Cfg.Toolsets.SetForTest([]string{"openshift/mustgather"})
		s.InitMcpClient()
		prompts, err := s.ListPrompts()
		s.Run("ListPrompts returns prompts", func() {
			s.NotNil(prompts, "Expected prompts from ListPrompts")
			s.NoError(err, "Expected no error from ListPrompts")
		})
		s.Run("ListPrompts returns correct Prompt metadata", func() {
			s.assertJsonSnapshot("toolsets-openshift-mustgather-prompts.json", prompts.Prompts)
		})
	})
}

func (s *ToolsetsSuite) TestOpenShiftMustGatherToolset() {
	s.Run("OpenShift must-gather toolset", func() {
		toolsets.Clear()
		toolsets.Register(&mgToolset.Toolset{})
		s.Cfg.Toolsets.SetForTest([]string{"openshift/mustgather"})
		s.InitMcpClient()
		tools, err := s.ListTools()
		s.Run("ListTools returns tools", func() {
			s.NotNil(tools, "Expected tools from ListTools")
			s.NoError(err, "Expected no error from ListTools")
		})
		s.Run("ListTools returns correct Tool metadata", func() {
			s.assertJsonSnapshot("toolsets-openshift-mustgather-tools.json", tools.Tools)
		})
	})
}
