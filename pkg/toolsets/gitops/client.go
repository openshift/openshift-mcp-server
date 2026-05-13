package gitops

import (
	"context"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	applicationGVR = schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "applications",
	}
	appProjectGVR = schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "appprojects",
	}
)

// ApplicationGVR returns the GroupVersionResource for ArgoCD Applications.
func ApplicationGVR() schema.GroupVersionResource {
	return applicationGVR
}

// AppProjectGVR returns the GroupVersionResource for ArgoCD AppProjects.
func AppProjectGVR() schema.GroupVersionResource {
	return appProjectGVR
}

type gitOpsClient struct {
	api.KubernetesClient
}

func newGitOpsClient(client api.KubernetesClient) *gitOpsClient {
	return &gitOpsClient{KubernetesClient: client}
}

func (g *gitOpsClient) applicationsList(ctx context.Context, namespace string, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	ns := g.resolveNamespace(ctx, namespace)
	return g.DynamicClient().Resource(applicationGVR).Namespace(ns).List(ctx, opts)
}

func (g *gitOpsClient) applicationGet(ctx context.Context, namespace, name string) (*unstructured.Unstructured, error) {
	ns := g.resolveNamespace(ctx, namespace)
	return g.DynamicClient().Resource(applicationGVR).Namespace(ns).Get(ctx, name, metav1.GetOptions{})
}

func (g *gitOpsClient) appProjectsList(ctx context.Context, namespace string, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	ns := g.resolveNamespace(ctx, namespace)
	return g.DynamicClient().Resource(appProjectGVR).Namespace(ns).List(ctx, opts)
}

func (g *gitOpsClient) appProjectGet(ctx context.Context, namespace, name string) (*unstructured.Unstructured, error) {
	ns := g.resolveNamespace(ctx, namespace)
	return g.DynamicClient().Resource(appProjectGVR).Namespace(ns).Get(ctx, name, metav1.GetOptions{})
}

// resolveNamespace returns the provided namespace, or auto-detects the ArgoCD namespace.
// Checks openshift-gitops first, then argocd, then falls back to the configured default.
func (g *gitOpsClient) resolveNamespace(ctx context.Context, namespace string) string {
	if namespace != "" {
		return namespace
	}
	for _, candidate := range []string{"openshift-gitops", "argocd"} {
		if g.namespaceExists(ctx, candidate) {
			return candidate
		}
	}
	return g.NamespaceOrDefault("")
}

func (g *gitOpsClient) namespaceExists(ctx context.Context, name string) bool {
	_, err := g.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	return err == nil
}
