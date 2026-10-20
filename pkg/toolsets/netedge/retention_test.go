package netedge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	fakedynamic "k8s.io/client-go/dynamic/fake"
)

func (s *NetEdgeTestSuite) TestDiagnosticsKeepsValidPrefixAtBudget() {
	var queries atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		queries.Add(1)
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[]}}`))
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	s.Require().NoError(err)
	route := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "route.openshift.io/v1",
			"kind":       "Route",
			"metadata": map[string]any{
				"name":      thanosQuerierRoute,
				"namespace": defaultMonitoringNamespace,
			},
			"spec": map[string]any{"host": u.Host},
		},
	}

	s.SetDynamicClient(fakedynamic.NewSimpleDynamicClient(runtime.NewScheme(), route))
	s.mockClient.restConfig.Insecure = true
	s.SetArgs(map[string]any{"diagnostic_target": "ingress"})
	s.params.Context = api.WithResultBudget(s.params.Context, 280)

	result, err := queryPrometheusHandler(s.params)

	s.Require().NoError(err)
	s.Require().NoError(result.Error)
	s.Contains(result.Content, api.ResultTruncationNotice)

	var items []json.RawMessage
	jsonText := strings.TrimSuffix(result.Content, api.ResultTruncationNotice)
	s.Require().NoError(json.Unmarshal([]byte(jsonText), &items))
	s.NotEmpty(items)
	s.Less(queries.Load(), int32(4))
}
