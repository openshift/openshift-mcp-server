package mcp

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/containers/kubernetes-mcp-server/internal/test"
	configuration "github.com/containers/kubernetes-mcp-server/pkg/config"
	netedgeToolset "github.com/containers/kubernetes-mcp-server/pkg/toolsets/netedge"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type netEdgeToolGroups struct {
	route         []string
	configMap     []string
	endpointSlice []string
	pod           []string
	probes        []string
	all           []string
}

func netEdgeTools(toolsetName string) netEdgeToolGroups {
	groups := netEdgeToolGroups{
		route:         []string{toolsetName + "_query_prometheus", "inspect_route"},
		configMap:     []string{"get_coredns_config"},
		endpointSlice: []string{"get_service_endpoints"},
		pod:           []string{"exec_dns_in_pod", "get_router_config", "get_router_info", "get_router_sessions"},
		probes:        []string{"probe_dns_local", "probe_http"},
	}
	groups.all = slices.Concat(groups.route, groups.configMap, groups.endpointSlice, groups.pod, groups.probes)
	return groups
}

func (s *ToolsetsSuite) configureNetEdgeCatalog(kubeconfig string, filtering bool) {
	s.Cfg = configuration.BaseDefault()
	s.Cfg.KubeConfig.SetForTest(kubeconfig)
	test.ApplyEnvtestClientLimits(s.Cfg)

	toolsetName := (&netedgeToolset.Toolset{}).GetName()
	enabled := append([]string(nil), s.Cfg.Toolsets.Get()...)
	if !slices.Contains(enabled, toolsetName) {
		enabled = append(enabled, toolsetName)
	}
	s.Cfg.Toolsets.SetForTest(enabled)
	s.Cfg.EnableTargetCompatibilityToolFilters.SetForTest(filtering)
}

func (s *ToolsetsSuite) listNetEdgeToolNames() map[string]struct{} {
	result, err := s.ListTools()
	s.Require().NoError(err)
	s.Require().NotNil(result)

	groups := netEdgeTools((&netedgeToolset.Toolset{}).GetName())
	known := stringSet(groups.all)
	actual := make(map[string]struct{}, len(groups.all))
	for _, tool := range result.Tools {
		if _, isNetEdge := known[tool.Name]; !isNetEdge {
			continue
		}
		_, duplicate := actual[tool.Name]
		s.Require().False(duplicate, "NetEdge tool %s must be registered only once", tool.Name)
		actual[tool.Name] = struct{}{}
	}
	return actual
}

func (s *ToolsetsSuite) requireNetEdgeCatalog(absent []string) {
	groups := netEdgeTools((&netedgeToolset.Toolset{}).GetName())
	expected := stringSet(groups.all)
	for _, name := range absent {
		delete(expected, name)
	}
	s.Equal(expected, s.listNetEdgeToolNames())
	for _, probe := range groups.probes {
		s.Contains(expected, probe, "local probe %s must remain visible", probe)
	}
}

func stringSet(values []string) map[string]struct{} {
	ret := make(map[string]struct{}, len(values))
	for _, value := range values {
		ret[value] = struct{}{}
	}
	return ret
}

func newNetEdgeDiscoveryFixture(coreKinds []string, groupVersions map[string][]string) *test.DiscoveryClientHandler {
	handler := test.NewDiscoveryClientHandler()
	handler.APIResourceLists[0].APIResources = apiResources(coreKinds)
	for groupVersion, kinds := range groupVersions {
		handler.APIResourceLists = append(handler.APIResourceLists, metav1.APIResourceList{
			GroupVersion: groupVersion,
			APIResources: apiResources(kinds),
		})
	}
	return handler
}

func apiResources(kinds []string) []metav1.APIResource {
	resources := make([]metav1.APIResource, 0, len(kinds))
	for _, kind := range kinds {
		resources = append(resources, metav1.APIResource{
			Name: strings.ToLower(kind) + "s",
			Kind: kind,
		})
	}
	return resources
}

func allNetEdgeDiscovery() *test.DiscoveryClientHandler {
	return newNetEdgeDiscoveryFixture(
		[]string{"Pod", "ConfigMap"},
		map[string][]string{
			"route.openshift.io/v1": {"Route"},
			"discovery.k8s.io/v1":   {"EndpointSlice"},
		},
	)
}

func (s *ToolsetsSuite) useNetEdgeFixture(handler http.Handler, filtering bool) {
	s.ResetHandlers()
	s.Handle(handler)
	s.configureNetEdgeCatalog(s.KubeconfigFile(s.T()), filtering)
	s.InitMcpClient()
}

func discoveryStatusWrapper(base http.Handler, path string, status metav1.Status) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != path {
			base.ServeHTTP(w, req)
			return
		}
		w.Header().Set("Content-Type", runtime.ContentTypeJSON)
		w.WriteHeader(int(status.Code))
		test.WriteObject(w, &status)
	})
}

func kubeconfigForNetEdgeMockServers(t *testing.T, currentContext string, servers map[string]*test.MockServer) string {
	t.Helper()
	cfg := clientcmdapi.NewConfig()
	for name, server := range servers {
		cluster := clientcmdapi.NewCluster()
		cluster.Server = server.Config().Host
		cfg.Clusters[name] = cluster
		cfg.AuthInfos[name] = clientcmdapi.NewAuthInfo()
		ctx := clientcmdapi.NewContext()
		ctx.Cluster = name
		ctx.AuthInfo = name
		cfg.Contexts[name] = ctx
	}
	cfg.CurrentContext = currentContext
	return test.KubeconfigFile(t, cfg)
}

func (s *ToolsetsSuite) TestNetEdgeAllRequirementsPresentAndBoundariesRemainUngated() {
	s.Run("all exact top-level GVKs are sufficient", func() {
		s.useNetEdgeFixture(allNetEdgeDiscovery(), true)
		s.requireNetEdgeCatalog(nil)
	})
}

func (s *ToolsetsSuite) TestNetEdgeConfirmedAbsenceHidesOnlyDependentTools() {
	groups := netEdgeTools((&netedgeToolset.Toolset{}).GetName())
	testCases := []struct {
		name          string
		coreKinds     []string
		groupVersions map[string][]string
		absent        []string
	}{
		{
			name:      "Route Kind missing from exact group version",
			coreKinds: []string{"Pod", "ConfigMap"},
			groupVersions: map[string][]string{
				"route.openshift.io/v1": {"Unrelated"},
				"discovery.k8s.io/v1":   {"EndpointSlice"},
			},
			absent: groups.route,
		},
		{
			name:      "ConfigMap missing from core v1",
			coreKinds: []string{"Pod"},
			groupVersions: map[string][]string{
				"route.openshift.io/v1": {"Route"},
				"discovery.k8s.io/v1":   {"EndpointSlice"},
			},
			absent: groups.configMap,
		},
		{
			name:      "EndpointSlice Kind missing from exact group version",
			coreKinds: []string{"Pod", "ConfigMap"},
			groupVersions: map[string][]string{
				"route.openshift.io/v1": {"Route"},
				"discovery.k8s.io/v1":   {"Unrelated"},
			},
			absent: groups.endpointSlice,
		},
		{
			name:      "Pod missing from core v1",
			coreKinds: []string{"ConfigMap"},
			groupVersions: map[string][]string{
				"route.openshift.io/v1": {"Route"},
				"discovery.k8s.io/v1":   {"EndpointSlice"},
			},
			absent: groups.pod,
		},
		{
			name:      "required core Kinds both missing",
			coreKinds: []string{"Node"},
			groupVersions: map[string][]string{
				"route.openshift.io/v1": {"Route"},
				"discovery.k8s.io/v1":   {"EndpointSlice"},
			},
			absent: slices.Concat(groups.configMap, groups.pod),
		},
		{
			name:      "Route exact version missing while alternate is served",
			coreKinds: []string{"Pod", "ConfigMap"},
			groupVersions: map[string][]string{
				"route.openshift.io/v1beta1": {"Route"},
				"discovery.k8s.io/v1":        {"EndpointSlice"},
			},
			absent: groups.route,
		},
		{
			name:      "EndpointSlice exact version missing while alternate is served",
			coreKinds: []string{"Pod", "ConfigMap"},
			groupVersions: map[string][]string{
				"route.openshift.io/v1":    {"Route"},
				"discovery.k8s.io/v1beta1": {"EndpointSlice"},
			},
			absent: groups.endpointSlice,
		},
		{
			name:      "Route API group missing",
			coreKinds: []string{"Pod", "ConfigMap"},
			groupVersions: map[string][]string{
				"discovery.k8s.io/v1": {"EndpointSlice"},
			},
			absent: groups.route,
		},
		{
			name:      "EndpointSlice API group missing",
			coreKinds: []string{"Pod", "ConfigMap"},
			groupVersions: map[string][]string{
				"route.openshift.io/v1": {"Route"},
			},
			absent: groups.endpointSlice,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.useNetEdgeFixture(newNetEdgeDiscoveryFixture(tc.coreKinds, tc.groupVersions), true)
			s.requireNetEdgeCatalog(tc.absent)
		})
	}
}

func (s *ToolsetsSuite) TestNetEdgeMixedTargetsUseAnyTargetSupport() {
	s.Run("support split across targets keeps all tools visible", func() {
		targetA := test.NewMockServer()
		s.T().Cleanup(targetA.Close)
		targetA.Handle(newNetEdgeDiscoveryFixture(
			[]string{"ConfigMap"},
			map[string][]string{"route.openshift.io/v1": {"Route"}},
		))

		targetB := test.NewMockServer()
		s.T().Cleanup(targetB.Close)
		targetB.Handle(newNetEdgeDiscoveryFixture(
			[]string{"Pod"},
			map[string][]string{"discovery.k8s.io/v1": {"EndpointSlice"}},
		))

		kubeconfig := kubeconfigForNetEdgeMockServers(s.T(), "a", map[string]*test.MockServer{
			"a": targetA,
			"b": targetB,
		})
		s.configureNetEdgeCatalog(kubeconfig, true)
		s.InitMcpClient()
		s.requireNetEdgeCatalog(nil)
	})
}

func (s *ToolsetsSuite) TestNetEdgeAllTargetsMissingRequirementHidesOnlyDependentTools() {
	groups := netEdgeTools((&netedgeToolset.Toolset{}).GetName())
	testCases := []struct {
		name          string
		coreKinds     []string
		groupVersions map[string][]string
		absent        []string
	}{
		{
			name:      "Route",
			coreKinds: []string{"Pod", "ConfigMap"},
			groupVersions: map[string][]string{
				"discovery.k8s.io/v1": {"EndpointSlice"},
			},
			absent: groups.route,
		},
		{
			name:      "ConfigMap",
			coreKinds: []string{"Pod"},
			groupVersions: map[string][]string{
				"route.openshift.io/v1": {"Route"},
				"discovery.k8s.io/v1":   {"EndpointSlice"},
			},
			absent: groups.configMap,
		},
		{
			name:      "EndpointSlice",
			coreKinds: []string{"Pod", "ConfigMap"},
			groupVersions: map[string][]string{
				"route.openshift.io/v1": {"Route"},
			},
			absent: groups.endpointSlice,
		},
		{
			name:      "Pod",
			coreKinds: []string{"ConfigMap"},
			groupVersions: map[string][]string{
				"route.openshift.io/v1": {"Route"},
				"discovery.k8s.io/v1":   {"EndpointSlice"},
			},
			absent: groups.pod,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			a := test.NewMockServer()
			s.T().Cleanup(a.Close)
			a.Handle(newNetEdgeDiscoveryFixture(tc.coreKinds, tc.groupVersions))
			b := test.NewMockServer()
			s.T().Cleanup(b.Close)
			b.Handle(newNetEdgeDiscoveryFixture(tc.coreKinds, tc.groupVersions))

			kubeconfig := kubeconfigForNetEdgeMockServers(s.T(), "a", map[string]*test.MockServer{"a": a, "b": b})
			s.configureNetEdgeCatalog(kubeconfig, true)
			s.InitMcpClient()
			s.requireNetEdgeCatalog(tc.absent)
		})
	}
}

func (s *ToolsetsSuite) TestNetEdgeForbiddenDiscoveryFailsOpen() {
	groups := netEdgeTools((&netedgeToolset.Toolset{}).GetName())
	testCases := []struct {
		name         string
		path         string
		groupVersion string
		affected     []string
	}{
		{name: "Route", path: "/apis/route.openshift.io/v1", groupVersion: "route.openshift.io/v1", affected: groups.route},
		{name: "EndpointSlice", path: "/apis/discovery.k8s.io/v1", groupVersion: "discovery.k8s.io/v1", affected: groups.endpointSlice},
		{name: "core", path: "/api/v1", groupVersion: "v1", affected: slices.Concat(groups.configMap, groups.pod)},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			status := metav1.Status{
				TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Status"},
				Status:   metav1.StatusFailure,
				Code:     http.StatusForbidden,
				Reason:   metav1.StatusReasonForbidden,
				Message:  "discovery forbidden",
			}
			handler := discoveryStatusWrapper(allNetEdgeDiscovery(), tc.path, status)
			s.ResetHandlers()
			s.Handle(handler)

			client, err := discovery.NewDiscoveryClientForConfig(s.Config())
			s.Require().NoError(err)
			_, err = client.ServerResourcesForGroupVersion(tc.groupVersion)
			s.Require().Error(err)
			s.True(apierrors.IsForbidden(err), "lower-level discovery must distinguish Forbidden from confirmed absence")

			s.configureNetEdgeCatalog(s.KubeconfigFile(s.T()), true)
			s.InitMcpClient()
			s.requireNetEdgeCatalog(nil)
			actual := s.listNetEdgeToolNames()
			for _, name := range tc.affected {
				s.Contains(actual, name)
			}
		})
	}
}

func (s *ToolsetsSuite) TestNetEdgeFilteringDisabledPreservesFullCatalog() {
	s.Run("all tools remain visible without any required GVK", func() {
		handler := newNetEdgeDiscoveryFixture(nil, nil)
		s.useNetEdgeFixture(handler, false)
		s.requireNetEdgeCatalog(nil)
	})
}
