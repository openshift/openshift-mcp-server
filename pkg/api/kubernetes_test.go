package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/containers/kubernetes-mcp-server/internal/test"
	"github.com/stretchr/testify/suite"
	apidiscovery "k8s.io/api/apidiscovery/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
)

type HasGVKsTestSuite struct {
	suite.Suite
	mockServer *test.MockServer
}

func TestHasGVKs(t *testing.T) {
	suite.Run(t, new(HasGVKsTestSuite))
}

func (s *HasGVKsTestSuite) SetupTest() {
	s.mockServer = test.NewMockServer()
}

func (s *HasGVKsTestSuite) TearDownTest() {
	if s.mockServer != nil {
		s.mockServer.Close()
	}
}

func (s *HasGVKsTestSuite) discoveryClient() discovery.DiscoveryInterface {
	return discovery.NewDiscoveryClientForConfigOrDie(s.mockServer.Config())
}

func (s *HasGVKsTestSuite) TestAnyServedVersion() {
	for _, tc := range []struct {
		name      string
		resources []metav1.APIResourceList
		expected  bool
	}{
		{name: "absent group"},
		{name: "absent kind", resources: []metav1.APIResourceList{{
			GroupVersion: "flows.netobserv.io/v1beta2",
			APIResources: []metav1.APIResource{{Name: "other", Kind: "OtherKind"}},
		}}},
		{name: "arbitrary served version", expected: true, resources: []metav1.APIResourceList{{
			GroupVersion: "flows.netobserv.io/v99",
			APIResources: []metav1.APIResource{{Name: "flowcollectors", Kind: "FlowCollector"}},
		}}},
		{name: "kind present only in later version", expected: true, resources: []metav1.APIResourceList{
			{GroupVersion: "flows.netobserv.io/v1beta1", APIResources: []metav1.APIResource{{Name: "other", Kind: "OtherKind"}}},
			{GroupVersion: "flows.netobserv.io/v1beta2", APIResources: []metav1.APIResource{{Name: "flowcollectors", Kind: "FlowCollector"}}},
		}},
	} {
		s.Run(tc.name, func() {
			s.mockServer.ResetHandlers()
			s.mockServer.Handle(test.NewDiscoveryClientHandler(tc.resources...))
			for _, cached := range []bool{false, true} {
				s.Run(fmt.Sprintf("cached=%t", cached), func() {
					dc := s.discoveryClient()
					if cached {
						dc = memory.NewMemCacheClient(dc)
					}
					has, err := HasGVKs(dc, []schema.GroupVersionKind{{Group: "flows.netobserv.io", Kind: "FlowCollector"}})
					s.Require().NoError(err)
					s.Equal(tc.expected, has)
				})
			}
		})
	}
}

func (s *HasGVKsTestSuite) TestAnyServedVersionAggregatedDiscovery() {
	for _, tc := range []struct {
		name       string
		staleGroup string
		freshKind  string
		want       bool
		wantError  bool
	}{
		{name: "all relevant versions stale", staleGroup: "flows.netobserv.io", wantError: true},
		{name: "stale version and healthy version without kind", staleGroup: "flows.netobserv.io", freshKind: "OtherKind", wantError: true},
		{name: "healthy version confirms kind despite stale version", staleGroup: "flows.netobserv.io", freshKind: "FlowCollector", want: true},
		{name: "unrelated stale group is not uncertainty", staleGroup: "unrelated.example"},
	} {
		for _, cached := range []bool{false, true} {
			s.Run(fmt.Sprintf("%s/cached=%t", tc.name, cached), func() {
				s.mockServer.ResetHandlers()
				groups := []apidiscovery.APIGroupDiscovery{{
					ObjectMeta: metav1.ObjectMeta{Name: tc.staleGroup},
					Versions:   []apidiscovery.APIVersionDiscovery{{Version: "v1beta1", Freshness: apidiscovery.DiscoveryFreshnessStale}},
				}}
				if tc.freshKind != "" {
					groups[0].Versions = append(groups[0].Versions, apidiscovery.APIVersionDiscovery{
						Version: "v1beta2", Freshness: apidiscovery.DiscoveryFreshnessCurrent,
						Resources: []apidiscovery.APIResourceDiscovery{{
							Resource: "flowcollectors", Scope: apidiscovery.ScopeCluster,
							ResponseKind: &metav1.GroupVersionKind{Group: "flows.netobserv.io", Version: "v1beta2", Kind: tc.freshKind},
						}},
					})
				}
				s.mockServer.Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					list := apidiscovery.APIGroupDiscoveryList{
						TypeMeta: metav1.TypeMeta{APIVersion: "apidiscovery.k8s.io/v2", Kind: "APIGroupDiscoveryList"},
					}
					if r.URL.Path == "/apis" {
						list.Items = groups
					} else if r.URL.Path != "/api" {
						http.NotFound(w, r)
						return
					}
					w.Header().Set("Content-Type", "application/json;g=apidiscovery.k8s.io;v=v2;as=APIGroupDiscoveryList")
					if err := json.NewEncoder(w).Encode(list); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
					}
				}))
				dc := s.discoveryClient()
				if cached {
					dc = memory.NewMemCacheClient(dc)
				}
				has, err := HasGVKs(dc, []schema.GroupVersionKind{{Group: "flows.netobserv.io", Kind: "FlowCollector"}})
				if tc.wantError {
					s.Error(err)
				} else {
					s.NoError(err)
				}
				s.Equal(tc.want, has)
			})
		}
	}
}

func (s *HasGVKsTestSuite) TestAnyServedVersionResourceDiscoveryError() {
	for _, available := range []bool{false, true} {
		s.Run(fmt.Sprintf("another version available=%t", available), func() {
			s.mockServer.ResetHandlers()
			resources := []metav1.APIResourceList{{GroupVersion: "flows.netobserv.io/v1beta1"}}
			if available {
				resources = append(resources, metav1.APIResourceList{
					GroupVersion: "flows.netobserv.io/v1beta2",
					APIResources: []metav1.APIResource{{Name: "flowcollectors", Kind: "FlowCollector"}},
				})
			}
			handler := test.NewDiscoveryClientHandler(resources...)
			s.mockServer.Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/apis/flows.netobserv.io/v1beta1" {
					http.Error(w, "discovery unavailable", http.StatusServiceUnavailable)
					return
				}
				handler.ServeHTTP(w, r)
			}))
			has, err := HasGVKs(s.discoveryClient(), []schema.GroupVersionKind{{Group: "flows.netobserv.io", Kind: "FlowCollector"}})
			if available {
				s.Require().NoError(err)
			} else {
				s.Error(err)
			}
			s.Equal(available, has)
		})
	}
}

func (s *HasGVKsTestSuite) TestAnyServedVersionCoreGroup() {
	s.mockServer.Handle(test.NewDiscoveryClientHandler())
	has, err := HasGVKs(s.discoveryClient(), []schema.GroupVersionKind{{Kind: "Pod"}, {Group: "apps", Version: "v1", Kind: "Deployment"}})
	s.Require().NoError(err)
	s.True(has)
}

func (s *HasGVKsTestSuite) TestAnyServedVersionDiscoveryError() {
	s.mockServer.Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "discovery unavailable", http.StatusServiceUnavailable)
	}))
	has, err := HasGVKs(s.discoveryClient(), []schema.GroupVersionKind{{Group: "flows.netobserv.io", Kind: "FlowCollector"}})
	s.Error(err)
	s.False(has)
}

func (s *HasGVKsTestSuite) TestAllGVKsExist() {
	s.Run("returns true when all GVKs exist", func() {
		handler := test.NewDiscoveryClientHandler(
			metav1.APIResourceList{
				GroupVersion: "project.openshift.io/v1",
				APIResources: []metav1.APIResource{
					{Name: "projects", Kind: "Project"},
				},
			},
		)
		s.mockServer.Handle(handler)

		gvks := []schema.GroupVersionKind{
			{Group: "", Version: "v1", Kind: "Pod"},                         // From default handler
			{Group: "project.openshift.io", Version: "v1", Kind: "Project"}, // From added handler
		}

		hasGVKs, err := HasGVKs(s.discoveryClient(), gvks)
		s.NoError(err)
		s.True(hasGVKs)
	})
}

func (s *HasGVKsTestSuite) TestGroupVersionDoesNotExist() {
	s.Run("returns false with no error when GroupVersion returns 404", func() {
		// Default handler doesn't include project.openshift.io
		handler := test.NewDiscoveryClientHandler()
		s.mockServer.Handle(handler)

		gvks := []schema.GroupVersionKind{
			{Group: "project.openshift.io", Version: "v1", Kind: "Project"},
		}

		hasGVKs, err := HasGVKs(s.discoveryClient(), gvks)
		s.NoError(err, "404 should not be returned as an error")
		s.False(hasGVKs, "should return false when GV doesn't exist")
	})
}

func (s *HasGVKsTestSuite) TestMemCacheGroupVersionDoesNotExist() {
	s.Run("returns false with no error for missing GroupVersion via memcache client", func() {
		// Production derives discovery from a memory-cached client, which returns
		// memory.ErrCacheNotFound (a plain error, not a StatusError with IsNotFound)
		// for an absent GroupVersion. This guards the errors.Is(err, ErrCacheNotFound)
		// branch in HasGVKs, which the raw-discovery cases above do not exercise.
		handler := test.NewDiscoveryClientHandler()
		s.mockServer.Handle(handler)

		cached := memory.NewMemCacheClient(s.discoveryClient())

		gvks := []schema.GroupVersionKind{
			{Group: "project.openshift.io", Version: "v1", Kind: "Project"},
		}

		hasGVKs, err := HasGVKs(cached, gvks)
		s.NoError(err, "missing GroupVersion via memcache should map to (false, nil)")
		s.False(hasGVKs, "should return false when the GroupVersion is absent from the memcache")
	})
}

func (s *HasGVKsTestSuite) TestKindDoesNotExist() {
	s.Run("returns false with no error when Kind is not in the resource list", func() {
		handler := test.NewDiscoveryClientHandler()
		s.mockServer.Handle(handler)

		gvks := []schema.GroupVersionKind{
			{Group: "", Version: "v1", Kind: "ConfigMap"}, // v1 exists but ConfigMap is not in default handler
		}

		hasGVKs, err := HasGVKs(s.discoveryClient(), gvks)
		s.NoError(err)
		s.False(hasGVKs, "should return false when Kind doesn't exist in GV")
	})
}

func (s *HasGVKsTestSuite) TestProperSubsetExists() {
	s.Run("returns false when only some GVKs exist", func() {
		handler := test.NewDiscoveryClientHandler()
		s.mockServer.Handle(handler)

		gvks := []schema.GroupVersionKind{
			{Group: "", Version: "v1", Kind: "Pod"},                         // Exists in default handler
			{Group: "project.openshift.io", Version: "v1", Kind: "Project"}, // Doesn't exist
		}

		hasGVKs, err := HasGVKs(s.discoveryClient(), gvks)
		s.NoError(err)
		s.False(hasGVKs, "should return false when not all GVKs exist")
	})
}

func (s *HasGVKsTestSuite) TestDiscoveryError() {
	s.Run("returns error when discovery fails with non-404 error", func() {
		// Don't set up any handler - server will return empty response causing JSON parse error
		s.mockServer.ResetHandlers()

		gvks := []schema.GroupVersionKind{
			{Group: "", Version: "v1", Kind: "Pod"},
		}

		hasGVKs, err := HasGVKs(s.discoveryClient(), gvks)
		s.Error(err, "should return error for non-404 discovery failures")
		s.False(hasGVKs)
	})
}

func (s *HasGVKsTestSuite) TestEmptyGVKList() {
	s.Run("returns true for empty GVK list", func() {
		handler := test.NewDiscoveryClientHandler()
		s.mockServer.Handle(handler)

		hasGVKs, err := HasGVKs(s.discoveryClient(), []schema.GroupVersionKind{})
		s.NoError(err)
		s.True(hasGVKs, "should return true for empty GVK list")
	})
}

func (s *HasGVKsTestSuite) TestMultipleGVKsInSameGroupVersion() {
	s.Run("returns true when multiple GVKs in same GV all exist", func() {
		handler := test.NewDiscoveryClientHandler()
		s.mockServer.Handle(handler)

		gvks := []schema.GroupVersionKind{
			{Group: "", Version: "v1", Kind: "Pod"}, // Both exist in default handler
			{Group: "", Version: "v1", Kind: "Node"},
		}

		hasGVKs, err := HasGVKs(s.discoveryClient(), gvks)
		s.NoError(err)
		s.True(hasGVKs)
	})

	s.Run("returns false when one of multiple GVKs in same GV doesn't exist", func() {
		// Reset handlers and create fresh discovery client to avoid cache issues
		s.mockServer.ResetHandlers()
		handler := test.NewDiscoveryClientHandler()
		s.mockServer.Handle(handler)

		gvks := []schema.GroupVersionKind{
			{Group: "", Version: "v1", Kind: "Pod"},       // Exists
			{Group: "", Version: "v1", Kind: "ConfigMap"}, // Doesn't exist
		}

		hasGVKs, err := HasGVKs(s.discoveryClient(), gvks)
		s.NoError(err)
		s.False(hasGVKs)
	})
}

func (s *HasGVKsTestSuite) TestRealWorldOpenShiftScenario() {
	s.Run("OpenShift Project GVK detection", func() {
		// Simulate non-OpenShift cluster (no project.openshift.io)
		handler := test.NewDiscoveryClientHandler()
		s.mockServer.Handle(handler)

		gvks := []schema.GroupVersionKind{
			{Group: "project.openshift.io", Version: "v1", Kind: "Project"},
		}

		hasGVKs, err := HasGVKs(s.discoveryClient(), gvks)
		s.NoError(err, "missing GV should return false, not error")
		s.False(hasGVKs, "non-OpenShift cluster should not have Project GVK")

		// Simulate OpenShift cluster
		s.mockServer.ResetHandlers()
		openshiftHandler := test.NewInOpenShiftHandler()
		s.mockServer.Handle(openshiftHandler)

		hasGVKs, err = HasGVKs(s.discoveryClient(), gvks)
		s.NoError(err)
		s.True(hasGVKs, "OpenShift cluster should have Project GVK")
	})
}
