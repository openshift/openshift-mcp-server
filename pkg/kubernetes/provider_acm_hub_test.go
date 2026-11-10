package kubernetes

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/containers/kubernetes-mcp-server/internal/test"
	"github.com/containers/kubernetes-mcp-server/pkg/config"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
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

	cfg := test.Must(config.ReadToml(s.T().Context(), []byte(`
		cluster_provider_strategy = "acm"
		[cluster_provider_configs.acm]
		cluster_proxy_addon_host = "proxy.example.com"
		cluster_proxy_addon_skip_tls_verify = true
	`)))

	provider, err := NewProvider(s.T().Context(), cfg)
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

// TestReconcileClustersPrunesStale locks in the OCPMCP-538 fix: a relist must prune
// clusters that disappeared (e.g. deleted during a watch gap), not only add new ones.
func (s *ProviderACMHubTestSuite) TestReconcileClustersPrunesStale() {
	p := &acmHubClusterProvider{clusterManagers: map[string]*Manager{"cluster-a": nil, "cluster-b": nil, "hub": nil}}

	s.Run("adds new clusters and prunes removed ones", func() {
		changed := p.reconcileClusters(map[string]bool{"cluster-a": true, "cluster-c": true, "hub": true})
		s.True(changed, "expected reconcile to report a change")
		targets, err := p.GetTargets(s.T().Context())
		s.Require().NoError(err)
		// cluster-b (absent from the list) is pruned, cluster-c (new) is added.
		s.Equal([]string{"cluster-a", "cluster-c", "hub"}, targets)
	})

	s.Run("reports no change for an identical set", func() {
		s.False(p.reconcileClusters(map[string]bool{"cluster-a": true, "cluster-c": true, "hub": true}),
			"expected no change when the set is unchanged")
	})
}

// acm410Handler serves the ManagedCluster list normally but makes every watch request
// fail with an in-stream 410 Gone / Expired ERROR event (HTTP 200 then an ERROR frame),
// reproducing the stale-resourceVersion condition from OCPMCP-538.
type acm410Handler struct {
	inner      *test.ACMHubHandler
	listCount  atomic.Int32
	watchCount atomic.Int32
	// httpError, when set, makes watch requests fail with an HTTP 410 (the Watch()
	// call-error path) instead of an in-stream 410 ERROR event.
	httpError atomic.Bool
}

func (h *acm410Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path == "/apis/cluster.open-cluster-management.io/v1/managedclusters" {
		if req.URL.Query().Get("watch") != "" {
			h.watchCount.Add(1)
			w.Header().Set("Content-Type", "application/json")
			status := &metav1.Status{
				TypeMeta: metav1.TypeMeta{Kind: "Status", APIVersion: "v1"},
				Status:   metav1.StatusFailure,
				Message:  "too old resource version",
				Reason:   metav1.StatusReasonExpired,
				Code:     http.StatusGone,
			}
			if h.httpError.Load() {
				// 410 returned as the watch call result itself, not an in-stream event.
				w.WriteHeader(http.StatusGone)
				_ = json.NewEncoder(w).Encode(status)
				return
			}
			raw, _ := json.Marshal(status)
			event, _ := json.Marshal(metav1.WatchEvent{Type: string(watch.Error), Object: runtime.RawExtension{Raw: raw}})
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(event)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			return
		}
		h.listCount.Add(1)
	}
	h.inner.ServeHTTP(w, req)
}

type ProviderACMHub410TestSuite struct {
	BaseProviderSuite
	mockServer                *test.MockServer
	originalIsInClusterConfig func() (*rest.Config, error)
	handler                   *acm410Handler
	provider                  Provider
}

func (s *ProviderACMHub410TestSuite) SetupTest() {
	s.BaseProviderSuite.SetupTest()
	s.originalIsInClusterConfig = InClusterConfig
	s.mockServer = test.NewMockServer()
	s.handler = &acm410Handler{
		inner: test.NewACMHubHandler(
			test.ManagedCluster{Name: "cluster-a"},
			test.ManagedCluster{Name: "hub", Labels: map[string]string{"local-cluster": "true"}},
		),
	}
	s.mockServer.Handle(s.handler)

	InClusterConfig = func() (*rest.Config, error) {
		return s.mockServer.Config(), nil
	}

	cfg := test.Must(config.ReadToml(s.T().Context(), []byte(`
		cluster_provider_strategy = "acm"
		[cluster_provider_configs.acm]
		cluster_proxy_addon_host = "proxy.example.com"
		cluster_proxy_addon_skip_tls_verify = true
	`)))

	provider, err := NewProvider(s.T().Context(), cfg)
	s.Require().NoError(err, "Expected no error creating ACM provider")
	s.provider = provider
}

func (s *ProviderACMHub410TestSuite) TearDownTest() {
	InClusterConfig = s.originalIsInClusterConfig
	if s.provider != nil {
		s.provider.Close()
	}
	if s.mockServer != nil {
		s.mockServer.Close()
	}
	s.BaseProviderSuite.TearDownTest()
}

// TestWatchRelistsAndBacksOffOn410 is the OCPMCP-538 regression test: a 410 watch
// error must trigger a relist (not an immediate reconnect with the same stale
// resourceVersion) and the reconnects must back off instead of tight-looping.
func (s *ProviderACMHub410TestSuite) TestWatchRelistsAndBacksOffOn410() {
	callback, _ := CallbackWaiter()
	s.provider.WatchTargets(s.T().Context(), McpReloaderFromCallback(callback))

	// The fix responds to the 410 ERROR event by relisting, so at least one relist
	// happens after the startup list. Without it, the loop reconnects with the same
	// stale resourceVersion and never relists (listCount stays at 1).
	s.Require().Eventually(func() bool {
		return s.handler.listCount.Load() >= 2
	}, 10*time.Second, 100*time.Millisecond, "expected a relist after the 410 watch error")

	// And reconnects must back off: without the fix this is a tight loop (hundreds of
	// watch attempts per second); with it, only a handful over a 2s window.
	before := s.handler.watchCount.Load()
	time.Sleep(2 * time.Second)
	s.Less(s.handler.watchCount.Load()-before, int32(10),
		"watch should back off after repeated 410s, not tight-loop")
}

// TestWatchRelistsOnDirectExpiredError covers the other 410 path: the apiserver
// returns ResourceExpired/Gone as the Watch() call result itself. The loop must still
// clear the resourceVersion and relist rather than retrying the stale one forever.
func (s *ProviderACMHub410TestSuite) TestWatchRelistsOnDirectExpiredError() {
	s.handler.httpError.Store(true)
	callback, _ := CallbackWaiter()
	s.provider.WatchTargets(s.T().Context(), McpReloaderFromCallback(callback))

	s.Require().Eventually(func() bool {
		return s.handler.listCount.Load() >= 2
	}, 10*time.Second, 100*time.Millisecond, "expected a relist after a direct 410 watch error")

	before := s.handler.watchCount.Load()
	time.Sleep(2 * time.Second)
	s.Less(s.handler.watchCount.Load()-before, int32(10),
		"watch should back off after repeated 410s, not tight-loop")
}

func TestProviderACMHub410(t *testing.T) {
	suite.Run(t, new(ProviderACMHub410TestSuite))
}
