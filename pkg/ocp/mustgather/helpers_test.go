package mustgather

import (
	"context"

	authv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	authorizationv1 "k8s.io/client-go/kubernetes/typed/authorization/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	metricsv1beta1 "k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"
)

// mockSelfSubjectAccessReviews implements authorizationv1.SelfSubjectAccessReviewInterface.
// Default behaviour returns allowed: true; use KnownAccessor to deny specific resources.
type mockSelfSubjectAccessReviews struct {
	authorizationv1.SelfSubjectAccessReviewInterface
	KnownAccessor map[string]bool
}

func (m *mockSelfSubjectAccessReviews) Create(ctx context.Context, review *authv1.SelfSubjectAccessReview, opts metav1.CreateOptions) (*authv1.SelfSubjectAccessReview, error) {
	review.Status.Allowed = true

	ra := review.Spec.ResourceAttributes
	keysToCheck := []string{
		ra.Verb + ":" + ra.Group + ":" + ra.Resource + ":" + ra.Namespace + ":" + ra.Name,
		ra.Verb + ":" + ra.Group + ":" + ra.Resource + ":" + ra.Namespace + ":",
		ra.Verb + ":" + ra.Group + ":" + ra.Resource + "::" + ra.Name,
		ra.Verb + ":" + ra.Group + ":" + ra.Resource + "::",
	}

	for _, key := range keysToCheck {
		if allowed, ok := m.KnownAccessor[key]; ok {
			review.Status.Allowed = allowed
			return review, nil
		}
	}

	return review, nil
}

// mockAuthorizationV1Client implements authorizationv1.AuthorizationV1Interface
type mockAuthorizationV1Client struct {
	authorizationv1.AuthorizationV1Interface
	KnownAccessor map[string]bool
}

func (m *mockAuthorizationV1Client) SelfSubjectAccessReviews() authorizationv1.SelfSubjectAccessReviewInterface {
	return &mockSelfSubjectAccessReviews{KnownAccessor: m.KnownAccessor}
}

// resettableRESTMapper wraps a RESTMapper and adds Reset()
type resettableRESTMapper struct {
	meta.RESTMapper
}

func (r *resettableRESTMapper) Reset() {}

// fakeDiscoveryClient implements discovery.CachedDiscoveryInterface
type fakeDiscoveryClient struct {
	discovery.CachedDiscoveryInterface
}

func (f *fakeDiscoveryClient) ServerResourcesForGroupVersion(groupVersion string) (*metav1.APIResourceList, error) {
	return &metav1.APIResourceList{GroupVersion: groupVersion}, nil
}

func (f *fakeDiscoveryClient) Invalidate() {}
func (f *fakeDiscoveryClient) Fresh() bool { return true }

// mockKubernetesClient implements api.KubernetesClient with minimal stubs.
type mockKubernetesClient struct {
	kubernetes.Interface
	knownAccessor map[string]bool
	dynClient     dynamic.Interface
	mapper        *resettableRESTMapper
	discClient    *fakeDiscoveryClient
}

func newMockKubernetesClient(knownAccessor map[string]bool) *mockKubernetesClient {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	return &mockKubernetesClient{
		knownAccessor: knownAccessor,
		dynClient:     fakedynamic.NewSimpleDynamicClient(scheme),
		mapper:        &resettableRESTMapper{RESTMapper: restmapper.NewDiscoveryRESTMapper(nil)},
		discClient:    &fakeDiscoveryClient{},
	}
}

func (m *mockKubernetesClient) NamespaceOrDefault(namespace string) string {
	if namespace != "" {
		return namespace
	}
	return "default"
}

func (m *mockKubernetesClient) RESTConfig() *rest.Config {
	return &rest.Config{Host: "https://fake-server:6443"}
}

func (m *mockKubernetesClient) RESTMapper() meta.ResettableRESTMapper {
	return m.mapper
}

func (m *mockKubernetesClient) DiscoveryClient() discovery.CachedDiscoveryInterface {
	return m.discClient
}

func (m *mockKubernetesClient) DynamicClient() dynamic.Interface {
	return m.dynClient
}

func (m *mockKubernetesClient) MetricsV1beta1Client() *metricsv1beta1.MetricsV1beta1Client {
	return nil
}

func (m *mockKubernetesClient) AuthorizationV1() authorizationv1.AuthorizationV1Interface {
	return &mockAuthorizationV1Client{KnownAccessor: m.knownAccessor}
}

// genericclioptions.RESTClientGetter implementation

func (m *mockKubernetesClient) ToRESTConfig() (*rest.Config, error) {
	return m.RESTConfig(), nil
}

func (m *mockKubernetesClient) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	return m.discClient, nil
}

func (m *mockKubernetesClient) ToRESTMapper() (meta.RESTMapper, error) {
	return m.mapper, nil
}

func (m *mockKubernetesClient) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return clientcmd.NewDefaultClientConfig(*clientcmdapi.NewConfig(), nil)
}
