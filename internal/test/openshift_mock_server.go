package test

import (
	"net/http"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ManagedCluster represents a managed cluster with its name and optional labels.
type ManagedCluster struct {
	Name   string
	Labels map[string]string
}

// ACMHubHandler handles mock ACM hub cluster API requests.
// It embeds DiscoveryClientHandler for API discovery endpoints and adds
// ACM-specific endpoints for managed clusters.
type ACMHubHandler struct {
	*DiscoveryClientHandler
	ManagedClusters []ManagedCluster
}

var _ http.Handler = (*ACMHubHandler)(nil)

// NewACMHubHandler creates an ACMHubHandler configured for ACM hub clusters.
// It includes the ACM cluster.open-cluster-management.io API group with ManagedCluster resources.
func NewACMHubHandler(clusters ...ManagedCluster) *ACMHubHandler {
	acmResources := []metav1.APIResourceList{
		{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				{Name: "services", Kind: "Service", Namespaced: true, Verbs: metav1.Verbs{"get", "list", "watch", "create", "update", "patch", "delete"}},
			},
		},
		{
			GroupVersion: "cluster.open-cluster-management.io/v1",
			APIResources: []metav1.APIResource{
				{
					Name:         "managedclusters",
					SingularName: "managedcluster",
					Kind:         "ManagedCluster",
					Namespaced:   false,
					Verbs:        metav1.Verbs{"get", "list", "watch", "create", "update", "patch", "delete"},
				},
			},
		},
	}
	return &ACMHubHandler{
		DiscoveryClientHandler: &DiscoveryClientHandler{APIResourceLists: acmResources},
		ManagedClusters:        clusters,
	}
}

func (h *ACMHubHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Handle ACM-specific endpoints first
	if req.URL.Path == "/apis/cluster.open-cluster-management.io/v1/managedclusters" {
		items := make([]unstructured.Unstructured, 0, len(h.ManagedClusters))
		for i, cluster := range h.ManagedClusters {
			metadata := map[string]interface{}{
				"name":            cluster.Name,
				"resourceVersion": strconv.Itoa(i + 1),
			}
			if len(cluster.Labels) > 0 {
				labels := make(map[string]interface{}, len(cluster.Labels))
				for k, v := range cluster.Labels {
					labels[k] = v
				}
				metadata["labels"] = labels
			}
			items = append(items, unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "cluster.open-cluster-management.io/v1",
					"kind":       "ManagedCluster",
					"metadata":   metadata,
				},
			})
		}
		WriteObject(w, &unstructured.UnstructuredList{
			Object: map[string]interface{}{
				"apiVersion": "cluster.open-cluster-management.io/v1",
				"kind":       "ManagedClusterList",
				"metadata": map[string]interface{}{
					"resourceVersion": "100",
				},
			},
			Items: items,
		})
		return
	}

	if req.URL.Path == "/api/v1/namespaces/multicluster-engine/services/cluster-proxy-addon-user" {
		WriteObject(w, &corev1.Service{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Service",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cluster-proxy-addon-user",
				Namespace: "multicluster-engine",
			},
		})
		return
	}

	// Delegate to embedded DiscoveryClientHandler for API discovery
	h.DiscoveryClientHandler.ServeHTTP(w, req)
}
