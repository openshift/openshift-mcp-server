package ocp

import (
	"context"

	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"

	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

// ClientsetInterface defines the interface for access-controlled clientset operations.
// This allows code to work with kubernetes.AccessControlClientset through an interface,
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

// OpenshiftClientAdapter adapts kubernetes.Kubernetes to implement OpenshiftClient.
// This allows production code to use the concrete *kubernetes.Kubernetes type
// while tests can use a mock implementation.
type OpenshiftClientAdapter struct {
	*kubernetes.Kubernetes
}

// NewOpenshiftClient creates a new adapter that wraps kubernetes.Kubernetes
// to implement the OpenshiftClient interface.
func NewOpenshiftClient(k *kubernetes.Kubernetes) *OpenshiftClientAdapter {
	return &OpenshiftClientAdapter{Kubernetes: k}
}

// AccessControlClientset returns the access control clientset as an interface.
// This satisfies the OpenshiftClient interface.
func (c *OpenshiftClientAdapter) AccessControlClientset() ClientsetInterface {
	return c.Kubernetes.AccessControlClientset()
}
