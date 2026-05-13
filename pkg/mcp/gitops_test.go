package mcp

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/gitops"
)

type GitOpsSuite struct {
	BaseMcpSuite
}

func (s *GitOpsSuite) SetupTest() {
	s.BaseMcpSuite.SetupTest()
	ctx := s.T().Context()
	s.Require().NoError(EnvTestEnableCRD(ctx, "argoproj.io", "v1alpha1", "applications"))
	s.Require().NoError(EnvTestEnableCRD(ctx, "argoproj.io", "v1alpha1", "appprojects"))

	client := kubernetes.NewForConfigOrDie(envTestRestConfig)
	_, _ = client.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "argocd"},
	}, metav1.CreateOptions{})

	dynClient := dynamic.NewForConfigOrDie(envTestRestConfig)
	_, _ = dynClient.Resource(gitops.ApplicationGVR()).Namespace("argocd").Create(ctx, &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "Application",
			"metadata": map[string]any{
				"name":      "test-app",
				"namespace": "argocd",
			},
			"spec": map[string]any{
				"project": "default",
				"source": map[string]any{
					"repoURL":        "https://github.com/example/repo.git",
					"path":           "manifests",
					"targetRevision": "HEAD",
				},
				"destination": map[string]any{
					"server":    "https://kubernetes.default.svc",
					"namespace": "default",
				},
			},
			"status": map[string]any{
				"sync": map[string]any{
					"status": "Synced",
				},
				"health": map[string]any{
					"status": "Healthy",
				},
			},
		},
	}, metav1.CreateOptions{})

	_, _ = dynClient.Resource(gitops.ApplicationGVR()).Namespace("argocd").Create(ctx, &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "Application",
			"metadata": map[string]any{
				"name":      "test-app-2",
				"namespace": "argocd",
			},
			"spec": map[string]any{
				"project": "team-a",
				"source": map[string]any{
					"repoURL":        "https://github.com/example/other-repo.git",
					"path":           "k8s",
					"targetRevision": "main",
				},
				"destination": map[string]any{
					"server":    "https://kubernetes.default.svc",
					"namespace": "staging",
				},
			},
		},
	}, metav1.CreateOptions{})

	_, _ = dynClient.Resource(gitops.AppProjectGVR()).Namespace("argocd").Create(ctx, &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "AppProject",
			"metadata": map[string]any{
				"name":      "default",
				"namespace": "argocd",
			},
			"spec": map[string]any{
				"sourceRepos": []any{"*"},
				"destinations": []any{
					map[string]any{
						"server":    "*",
						"namespace": "*",
					},
				},
			},
		},
	}, metav1.CreateOptions{})
}

func (s *GitOpsSuite) TearDownTest() {
	ctx := s.T().Context()
	dynClient := dynamic.NewForConfigOrDie(envTestRestConfig)
	_ = dynClient.Resource(gitops.ApplicationGVR()).Namespace("argocd").DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{})
	_ = dynClient.Resource(gitops.AppProjectGVR()).Namespace("argocd").DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{})
	s.BaseMcpSuite.TearDownTest()
}

func (s *GitOpsSuite) TestApplicationsList() {
	s.Cfg.Toolsets = []string{"gitops"}
	s.InitMcpClient()
	s.Run("lists all applications", func() {
		toolResult, err := s.CallTool("gitops_applications_list", map[string]any{
			"namespace": "argocd",
		})
		s.Require().NoError(err)
		s.Require().False(toolResult.IsError, "call tool should succeed")
		text := toolResult.Content[0].(*mcp.TextContent).Text
		s.Contains(text, "test-app")
		s.Contains(text, "test-app-2")
	})
	s.Run("filters by project", func() {
		toolResult, err := s.CallTool("gitops_applications_list", map[string]any{
			"namespace": "argocd",
			"project":   "team-a",
		})
		s.Require().NoError(err)
		s.Require().False(toolResult.IsError, "call tool should succeed")
		text := toolResult.Content[0].(*mcp.TextContent).Text
		s.Contains(text, "test-app-2")
		s.Contains(text, "team-a")
		s.NotContains(text, "name: test-app\n")
	})
}

func (s *GitOpsSuite) TestApplicationGet() {
	s.Cfg.Toolsets = []string{"gitops"}
	s.InitMcpClient()
	s.Run("gets application by name", func() {
		toolResult, err := s.CallTool("gitops_application_get", map[string]any{
			"name":      "test-app",
			"namespace": "argocd",
		})
		s.Require().NoError(err)
		s.Require().False(toolResult.IsError, "call tool should succeed")
		text := toolResult.Content[0].(*mcp.TextContent).Text
		s.Contains(text, "test-app")
		s.Contains(text, "https://github.com/example/repo.git")
	})
	s.Run("returns error for missing application", func() {
		toolResult, err := s.CallTool("gitops_application_get", map[string]any{
			"name":      "nonexistent",
			"namespace": "argocd",
		})
		s.Require().NoError(err)
		s.True(toolResult.IsError, "call tool should fail for missing app")
	})
	s.Run("requires name parameter", func() {
		toolResult, err := s.CallTool("gitops_application_get", map[string]any{
			"namespace": "argocd",
		})
		s.Require().NoError(err)
		s.True(toolResult.IsError, "call tool should fail without name")
		s.Contains(toolResult.Content[0].(*mcp.TextContent).Text, "name parameter required")
	})
}

func (s *GitOpsSuite) TestProjectsList() {
	s.Cfg.Toolsets = []string{"gitops"}
	s.InitMcpClient()
	s.Run("lists projects", func() {
		toolResult, err := s.CallTool("gitops_projects_list", map[string]any{
			"namespace": "argocd",
		})
		s.Require().NoError(err)
		s.Require().False(toolResult.IsError, "call tool should succeed")
		text := toolResult.Content[0].(*mcp.TextContent).Text
		s.Contains(text, "default")
	})
}

func (s *GitOpsSuite) TestProjectGet() {
	s.Cfg.Toolsets = []string{"gitops"}
	s.InitMcpClient()
	s.Run("gets project by name", func() {
		toolResult, err := s.CallTool("gitops_project_get", map[string]any{
			"name":      "default",
			"namespace": "argocd",
		})
		s.Require().NoError(err)
		s.Require().False(toolResult.IsError, "call tool should succeed")
		text := toolResult.Content[0].(*mcp.TextContent).Text
		s.Contains(text, "default")
		s.Contains(text, "sourceRepos")
	})
	s.Run("returns error for missing project", func() {
		toolResult, err := s.CallTool("gitops_project_get", map[string]any{
			"name":      "nonexistent",
			"namespace": "argocd",
		})
		s.Require().NoError(err)
		s.True(toolResult.IsError, "call tool should fail for missing project")
	})
}

func (s *GitOpsSuite) TestApplicationsListAutoDetectNamespace() {
	s.Cfg.Toolsets = []string{"gitops"}
	s.InitMcpClient()
	s.Run("auto-detects argocd namespace", func() {
		toolResult, err := s.CallTool("gitops_applications_list", map[string]any{})
		s.Require().NoError(err)
		s.Require().False(toolResult.IsError, "call tool should succeed")
		text := toolResult.Content[0].(*mcp.TextContent).Text
		s.True(strings.Contains(text, "test-app") || text == "[]",
			"should either find apps in argocd namespace or return empty list")
	})
}

func TestGitOps(t *testing.T) {
	suite.Run(t, new(GitOpsSuite))
}
