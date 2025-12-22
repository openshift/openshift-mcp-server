package test

import (
	"fmt"
	"net/http"
)

type ACMHubHandler struct {
	ManagedClusters []string
}

var _ http.Handler = (*ACMHubHandler)(nil)

func NewACMHubHandler(clusters ...string) *ACMHubHandler {
	return &ACMHubHandler{ManagedClusters: clusters}
}

func (h *ACMHubHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if req.URL.Path == "/api" {
		_, _ = w.Write([]byte(`{"kind":"APIVersions","versions":["v1"],"serverAddressByClientCIDRs":[{"clientCIDR":"0.0.0.0/0"}]}`))
		return
	}

	if req.URL.Path == "/apis" {
		_, _ = w.Write([]byte(`{
			"kind":"APIGroupList",
			"groups":[
				{"name":"cluster.open-cluster-management.io","versions":[{"groupVersion":"cluster.open-cluster-management.io/v1","version":"v1"}],"preferredVersion":{"groupVersion":"cluster.open-cluster-management.io/v1","version":"v1"}}
			]}`))
		return
	}

	if req.URL.Path == "/apis/cluster.open-cluster-management.io/v1" {
		_, _ = w.Write([]byte(`{
			"kind":"APIResourceList",
			"apiVersion":"v1",
			"groupVersion":"cluster.open-cluster-management.io/v1",
			"resources":[
				{"name":"managedclusters","singularName":"managedcluster","namespaced":false,"kind":"ManagedCluster","verbs":["get","list","watch","create","update","patch","delete"]}
			]}`))
		return
	}

	if req.URL.Path == "/apis/cluster.open-cluster-management.io/v1/managedclusters" {
		items := ""
		for i, cluster := range h.ManagedClusters {
			if i > 0 {
				items += ","
			}
			items += fmt.Sprintf(`{"apiVersion":"cluster.open-cluster-management.io/v1","kind":"ManagedCluster","metadata":{"name":"%s","resourceVersion":"%d"}}`, cluster, i+1)
		}
		_, _ = fmt.Fprintf(w, `{"apiVersion":"cluster.open-cluster-management.io/v1","kind":"ManagedClusterList","metadata":{"resourceVersion":"100"},"items":[%s]}`, items)
		return
	}

	if req.URL.Path == "/api/v1" {
		_, _ = w.Write([]byte(`{"kind":"APIResourceList","apiVersion":"v1","resources":[
			{"name":"services","singularName":"","namespaced":true,"kind":"Service","verbs":["get","list","watch","create","update","patch","delete"]}
		]}`))
		return
	}

	if req.URL.Path == "/api/v1/namespaces/multicluster-engine/services/cluster-proxy-addon-user" {
		_, _ = w.Write([]byte(`{
			"apiVersion":"v1",
			"kind":"Service",
			"metadata":{"name":"cluster-proxy-addon-user","namespace":"multicluster-engine"}
		}`))
		return
	}
}
