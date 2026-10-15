package kubernetes

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/containers/kubernetes-mcp-server/internal/test"
	"github.com/containers/kubernetes-mcp-server/pkg/config"
)

type staticManagerProvider struct {
	managers []*Manager
}

func (p staticManagerProvider) GetTargetManagers(context.Context) ([]*Manager, error) {
	return p.managers, nil
}

type ProviderGVKFilterTestSuite struct {
	suite.Suite
}

func (s *ProviderGVKFilterTestSuite) TestClientDeadlineIsBoundedAndFailsOpen() {
	server := test.NewMockServer()
	s.T().Cleanup(server.Close)

	base := test.NewDiscoveryClientHandler(metav1.APIResourceList{
		GroupVersion: "route.openshift.io/v1",
		APIResources: []metav1.APIResource{{Name: "routes", Kind: "Route"}},
	})
	requestStarted := make(chan struct{})
	requestCanceled := make(chan struct{})
	var startedOnce sync.Once
	var canceledOnce sync.Once
	server.Handle(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/apis/route.openshift.io/v1" {
			base.ServeHTTP(w, req)
			return
		}
		startedOnce.Do(func() { close(requestStarted) })
		<-req.Context().Done()
		canceledOnce.Do(func() { close(requestCanceled) })
	}))

	restConfig := rest.CopyConfig(server.Config())
	restConfig.Timeout = 75 * time.Millisecond
	cfg := config.BaseDefault()
	test.ApplyEnvtestClientLimits(cfg)
	manager, err := NewManager(
		s.T().Context(),
		cfg,
		restConfig,
		clientcmd.NewDefaultClientConfig(*server.Kubeconfig(), nil),
	)
	s.Require().NoError(err)
	s.T().Cleanup(manager.Close)

	filter := NewProviderGVKFilter(staticManagerProvider{managers: []*Manager{manager}})
	start := time.Now()
	hasGVK := filter.AnyTargetHasGVKs(s.T().Context(), []schema.GroupVersionKind{{
		Group:   "route.openshift.io",
		Version: "v1",
		Kind:    "Route",
	}})
	elapsed := time.Since(start)

	s.True(hasGVK, "client deadline errors must fail open")
	select {
	case <-requestStarted:
	case <-time.After(time.Second):
		s.FailNow("discovery request did not reach the blocking endpoint")
	}
	select {
	case <-requestCanceled:
	case <-time.After(time.Second):
		s.FailNow("HTTP client deadline did not cancel the discovery request context")
	}
	s.Less(elapsed, 2*time.Second, "client-side discovery deadline must bound filter evaluation")
}

func TestProviderGVKFilter(t *testing.T) {
	suite.Run(t, new(ProviderGVKFilterTestSuite))
}
