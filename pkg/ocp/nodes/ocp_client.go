package nodes

import (
	"context"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"

	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

// ClientsetInterface defines the interface for access-controlled clientset operations.
// This allows code to work with kubernetes client through an interface,
// making it easier to test and decouple from the concrete implementation.
type ClientsetInterface interface {
	Pods(namespace string) (corev1client.PodInterface, error)
}

// OpenshiftClient defines a minimal interface for kubernetes operations commonly needed
// by OCP toolsets. This allows for easier testing and decoupling from the concrete
// kubernetes.Kubernetes type.
type OpenshiftClient interface {
	NamespaceOrDefault(namespace string) string
	AccessControlClientset() ClientsetInterface
	PodsLog(ctx context.Context, namespace, name, container string, previous bool, tail int64) (string, error)
}

// clientsetAdapter adapts api.KubernetesClient to implement ClientsetInterface.
type clientsetAdapter struct {
	client api.KubernetesClient
}

func (a *clientsetAdapter) Pods(namespace string) (corev1client.PodInterface, error) {
	return a.client.CoreV1().Pods(namespace), nil
}

// OpenshiftClientAdapter adapts api.KubernetesClient to implement OpenshiftClient.
// This allows production code to use the api.KubernetesClient interface
// while tests can use a mock implementation.
type OpenshiftClientAdapter struct {
	*kubernetes.Core
	clientset *clientsetAdapter
}

// NewOpenshiftClient creates a new adapter that wraps api.KubernetesClient
// to implement the OpenshiftClient interface.
func NewOpenshiftClient(k api.KubernetesClient) *OpenshiftClientAdapter {
	return &OpenshiftClientAdapter{
		Core:      kubernetes.NewCore(k),
		clientset: &clientsetAdapter{client: k},
	}
}

// AccessControlClientset returns the clientset adapter as an interface.
// This satisfies the OpenshiftClient interface.
func (c *OpenshiftClientAdapter) AccessControlClientset() ClientsetInterface {
	return c.clientset
}
