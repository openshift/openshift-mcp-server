package kubernetes

import (
	"context"
	"errors"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Openshift interface {
	IsOpenShift(context.Context) bool
}

func (m *Manager) IsOpenShift(_ context.Context) bool {
	// This method should be fast and not block (it's called at startup)
	_, err := m.discoveryClient.ServerResourcesForGroupVersion(schema.GroupVersion{
		Group:   "project.openshift.io",
		Version: "v1",
	}.String())
	return err == nil
}

// DiscoverRouteURLForService discovers the external URL exposed by an OpenShift Route
// that targets the given Service name in the provided namespace.
// It returns the base URL including scheme and optional path (if configured on the Route).
func (m *Manager) DiscoverRouteURLForService(ctx context.Context, namespace, serviceName string) (string, error) {
	if m == nil || m.discoveryClient == nil || m.dynamicClient == nil {
		return "", errors.New("kubernetes manager not initialized")
	}
	if _, err := m.discoveryClient.ServerResourcesForGroupVersion("route.openshift.io/v1"); err != nil {
		return "", errors.New("openshift Route API not available")
	}
	routes := m.dynamicClient.Resource(schema.GroupVersionResource{
		Group:    "route.openshift.io",
		Version:  "v1",
		Resource: "routes",
	}).Namespace(namespace)
	list, err := routes.List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", err
	}
	for i := range list.Items {
		r := &list.Items[i]
		to, ok, _ := unstructured.NestedMap(r.Object, "spec", "to")
		if !ok || to == nil {
			continue
		}
		kind, _ := to["kind"].(string)
		name, _ := to["name"].(string)
		if !strings.EqualFold(kind, "Service") || name != serviceName {
			continue
		}
		host, _, _ := unstructured.NestedString(r.Object, "spec", "host")
		if strings.TrimSpace(host) == "" {
			continue
		}
		// Use https if TLS is configured on the Route
		scheme := "http"
		if _, hasTLS, _ := unstructured.NestedFieldNoCopy(r.Object, "spec", "tls"); hasTLS {
			scheme = "https"
		}
		path, _, _ := unstructured.NestedString(r.Object, "spec", "path")
		base := scheme + "://" + host
		if p := strings.TrimSpace(path); p != "" && p != "/" {
			if !strings.HasPrefix(p, "/") {
				p = "/" + p
			}
			base += p
		}
		return base, nil
	}
	return "", errors.New("no Route found for Service")
}
