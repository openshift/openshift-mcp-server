package mcp

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/errgroup"

	oadpToolset "github.com/containers/kubernetes-mcp-server/pkg/toolsets/oadp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

var oadpApis = []schema.GroupVersionResource{
	{Group: "velero.io", Version: "v1", Resource: "backups"},
	{Group: "velero.io", Version: "v1", Resource: "restores"},
	{Group: "velero.io", Version: "v1", Resource: "backupstoragelocations"},
	{Group: "oadp.openshift.io", Version: "v1alpha1", Resource: "dataprotectionapplications"},
}

type OADPSuite struct {
	BaseMcpSuite
}

func (s *OADPSuite) SetupSuite() {
	ctx := s.T().Context()
	tasks, _ := errgroup.WithContext(ctx)
	for _, api := range oadpApis {
		gvr := api
		tasks.Go(func() error { return EnvTestEnableCRD(ctx, gvr.Group, gvr.Version, gvr.Resource) })
	}
	s.Require().NoError(tasks.Wait())

	_, err := kubernetes.NewForConfigOrDie(envTestRestConfig).CoreV1().Namespaces().
		Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "openshift-adp"}}, metav1.CreateOptions{})
	s.Require().NoError(err)
}

func (s *OADPSuite) TearDownSuite() {
	tasks, _ := errgroup.WithContext(s.T().Context())
	for _, api := range oadpApis {
		gvr := api
		tasks.Go(func() error { return EnvTestDisableCRD(s.T().Context(), gvr.Group, gvr.Version, gvr.Resource) })
	}
	s.Require().NoError(tasks.Wait())
}

func (s *OADPSuite) SetupTest() {
	s.BaseMcpSuite.SetupTest()
	s.Cfg.Toolsets = append(s.Cfg.Toolsets, (&oadpToolset.Toolset{}).GetName())
	s.InitMcpClient()
}

func (s *OADPSuite) TestToolsetRegistration() {
	s.Run("toolset has no tools", func() {
		tools, err := s.ListTools()
		s.Require().NoError(err)
		for _, tool := range tools.Tools {
			s.Falsef(tool.Name == "oadp_backup" || tool.Name == "oadp_restore",
				"OADP toolset should not expose dedicated tools, found: %s", tool.Name)
		}
	})

	s.Run("toolset exposes oadp-troubleshoot prompt", func() {
		prompts, err := s.ListPrompts()
		s.Require().NoError(err)
		var found bool
		for _, prompt := range prompts.Prompts {
			if prompt.Name == "oadp-troubleshoot" {
				found = true
				s.Equal("Generate a step-by-step troubleshooting guide for diagnosing OADP backup and restore issues", prompt.Description)
				s.Len(prompt.Arguments, 3)
				break
			}
		}
		s.True(found, "expected oadp-troubleshoot prompt to be listed")
	})
}

func (s *OADPSuite) TestTroubleshootPromptDefaultNamespace() {
	result, err := s.GetPrompt("oadp-troubleshoot", map[string]string{})

	s.Run("returns successfully", func() {
		s.Require().NoError(err)
		s.Require().NotNil(result)
	})

	s.Run("contains diagnostic sections", func() {
		s.Require().NotNil(result)
		s.Require().NotEmpty(result.Messages)
		text := result.Messages[0].Content.(*mcp.TextContent).Text
		s.Contains(text, "Namespace: openshift-adp")
		s.Contains(text, "DataProtectionApplication")
		s.Contains(text, "BackupStorageLocations")
		s.Contains(text, "Recent Backups")
		s.Contains(text, "Recent Restores")
		s.Contains(text, "Velero Pods")
		s.Contains(text, "Events")
	})

	s.Run("includes assistant analysis message", func() {
		s.Require().NotNil(result)
		s.Require().Len(result.Messages, 2)
		s.Equal(mcp.Role("assistant"), result.Messages[1].Role)
	})
}

func (s *OADPSuite) TestTroubleshootPromptCustomNamespace() {
	result, err := s.GetPrompt("oadp-troubleshoot", map[string]string{
		"namespace": "custom-ns",
	})

	s.Run("uses custom namespace", func() {
		s.Require().NoError(err)
		s.Require().NotNil(result)
		s.Require().NotEmpty(result.Messages)
		text := result.Messages[0].Content.(*mcp.TextContent).Text
		s.Contains(text, "Namespace: custom-ns")
	})
}

func (s *OADPSuite) TestTroubleshootPromptWithBackup() {
	s.Run("with existing backup", func() {
		dynamicClient := dynamic.NewForConfigOrDie(envTestRestConfig)
		backup := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "velero.io/v1",
				"kind":       "Backup",
				"metadata": map[string]any{
					"name":      "test-backup",
					"namespace": "openshift-adp",
				},
				"spec": map[string]any{
					"includedNamespaces": []any{"default"},
				},
				"status": map[string]any{
					"phase": "Completed",
				},
			},
		}
		_, err := dynamicClient.Resource(schema.GroupVersionResource{
			Group: "velero.io", Version: "v1", Resource: "backups",
		}).Namespace("openshift-adp").Create(s.T().Context(), backup, metav1.CreateOptions{})
		s.Require().NoError(err)

		result, err := s.GetPrompt("oadp-troubleshoot", map[string]string{
			"backup": "test-backup",
		})
		s.Require().NoError(err)
		s.Require().NotNil(result)
		text := result.Messages[0].Content.(*mcp.TextContent).Text
		s.Contains(text, "Backup: test-backup")
	})

	s.Run("with non-existent backup", func() {
		result, err := s.GetPrompt("oadp-troubleshoot", map[string]string{
			"backup": "non-existent",
		})
		s.Require().NoError(err)
		s.Require().NotNil(result)
		text := result.Messages[0].Content.(*mcp.TextContent).Text
		s.Contains(text, "Error fetching backup")
	})
}

func (s *OADPSuite) TestTroubleshootPromptWithRestore() {
	s.Run("with non-existent restore", func() {
		result, err := s.GetPrompt("oadp-troubleshoot", map[string]string{
			"restore": "non-existent",
		})
		s.Require().NoError(err)
		s.Require().NotNil(result)
		text := result.Messages[0].Content.(*mcp.TextContent).Text
		s.Contains(text, "Error fetching restore")
	})
}

func TestOADPSuite(t *testing.T) {
	suite.Run(t, new(OADPSuite))
}
