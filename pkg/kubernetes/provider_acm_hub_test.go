package kubernetes

import (
	"testing"

	"github.com/containers/kubernetes-mcp-server/internal/test"
	"github.com/containers/kubernetes-mcp-server/pkg/config"
	"github.com/stretchr/testify/suite"
	"k8s.io/client-go/rest"
)

type ProviderACMHubTestSuite struct {
	BaseProviderSuite
	mockServer                *test.MockServer
	originalIsInClusterConfig func() (*rest.Config, error)
	provider                  Provider
}

func (s *ProviderACMHubTestSuite) SetupTest() {
	s.BaseProviderSuite.SetupTest()
	s.originalIsInClusterConfig = InClusterConfig
	s.mockServer = test.NewMockServer()
	s.mockServer.Handle(test.NewACMHubHandler(
		test.ManagedCluster{Name: "cluster-a"},
		test.ManagedCluster{Name: "cluster-b"},
		test.ManagedCluster{Name: "cluster-c"},
		test.ManagedCluster{Name: "hub", Labels: map[string]string{"local-cluster": "true"}},
	))

	InClusterConfig = func() (*rest.Config, error) {
		return s.mockServer.Config(), nil
	}

	cfg := test.Must(config.ReadToml([]byte(`
		cluster_provider_strategy = "acm"
		[cluster_provider_configs.acm]
		cluster_proxy_addon_host = "proxy.example.com"
		cluster_proxy_addon_skip_tls_verify = true
	`)))

	provider, err := NewProvider(cfg)
	s.Require().NoError(err, "Expected no error creating ACM provider")
	s.provider = provider
}

func (s *ProviderACMHubTestSuite) TearDownTest() {
	InClusterConfig = s.originalIsInClusterConfig
	if s.provider != nil {
		s.provider.Close()
	}
	if s.mockServer != nil {
		s.mockServer.Close()
	}
	s.BaseProviderSuite.TearDownTest()
}

func (s *ProviderACMHubTestSuite) TestType() {
	s.IsType(&acmHubClusterProvider{}, s.provider)
}

func (s *ProviderACMHubTestSuite) TestWithNonOpenShiftCluster() {
	s.Run("IsOpenShift returns false", func() {
		inOpenShift := s.provider.IsOpenShift(s.T().Context())
		s.False(inOpenShift, "Expected IsOpenShift to return false")
	})
}

func (s *ProviderACMHubTestSuite) TestGetTargets() {
	s.Run("GetTargets returns managed clusters in sorted order", func() {
		targets, err := s.provider.GetTargets(s.T().Context())
		s.Require().NoError(err, "Expected no error from GetTargets")
		s.Len(targets, 4, "Expected 4 targets from GetTargets")
		s.Equal([]string{"cluster-a", "cluster-b", "cluster-c", "hub"}, targets, "Expected sorted cluster names")
	})
}

func (s *ProviderACMHubTestSuite) TestGetDefaultTarget() {
	s.Run("GetDefaultTarget returns hub", func() {
		s.Equal("hub", s.provider.GetDefaultTarget(), "Expected hub as default target")
	})
}

func (s *ProviderACMHubTestSuite) TestGetTargetParameterName() {
	s.Equal("cluster", s.provider.GetTargetParameterName(), "Expected cluster as target parameter name")
}

func (s *ProviderACMHubTestSuite) TestGetDerivedKubernetes() {
	s.Run("GetDerivedKubernetes returns Kubernetes for hub target", func() {
		k8s, err := s.provider.GetDerivedKubernetes(s.T().Context(), "hub")
		s.Require().NoError(err, "Expected no error from GetDerivedKubernetes with hub target")
		s.NotNil(k8s, "Expected Kubernetes from GetDerivedKubernetes with hub target")
	})
	s.Run("GetDerivedKubernetes returns Kubernetes for empty target (defaults to hub)", func() {
		k8s, err := s.provider.GetDerivedKubernetes(s.T().Context(), "")
		s.Require().NoError(err, "Expected no error from GetDerivedKubernetes with empty target")
		s.NotNil(k8s, "Expected Kubernetes from GetDerivedKubernetes with empty target")
	})
	s.Run("GetDerivedKubernetes returns error for unknown cluster", func() {
		k8s, err := s.provider.GetDerivedKubernetes(s.T().Context(), "unknown-cluster")
		s.Require().Error(err, "Expected error from GetDerivedKubernetes with unknown cluster")
		s.ErrorContains(err, "cluster unknown-cluster not found", "Expected cluster not found error")
		s.Nil(k8s, "Expected no Kubernetes from GetDerivedKubernetes with unknown cluster")
	})
}

func (s *ProviderACMHubTestSuite) TestAddCluster() {
	acmProvider := s.provider.(*acmHubClusterProvider)

	s.Run("addCluster adds new cluster", func() {
		acmProvider.addCluster("new-cluster")
		targets, err := s.provider.GetTargets(s.T().Context())
		s.Require().NoError(err)
		s.Contains(targets, "new-cluster", "Expected new-cluster in targets")
	})

	s.Run("addCluster is idempotent", func() {
		acmProvider.addCluster("cluster-a")
		targets, err := s.provider.GetTargets(s.T().Context())
		s.Require().NoError(err)
		count := 0
		for _, t := range targets {
			if t == "cluster-a" {
				count++
			}
		}
		s.Equal(1, count, "Expected cluster-a to appear only once")
	})
}

func (s *ProviderACMHubTestSuite) TestRemoveCluster() {
	acmProvider := s.provider.(*acmHubClusterProvider)

	s.Run("removeCluster removes existing cluster", func() {
		acmProvider.removeCluster("cluster-b")
		targets, err := s.provider.GetTargets(s.T().Context())
		s.Require().NoError(err)
		s.NotContains(targets, "cluster-b", "Expected cluster-b to be removed")
	})

	s.Run("removeCluster is safe for non-existent cluster", func() {
		acmProvider.removeCluster("non-existent")
		targets, err := s.provider.GetTargets(s.T().Context())
		s.Require().NoError(err)
		s.NotEmpty(targets, "Expected targets to still exist")
	})
}

func TestProviderACMHub(t *testing.T) {
	suite.Run(t, new(ProviderACMHubTestSuite))
}
