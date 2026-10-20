package mustgather

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type RetentionSuite struct {
	suite.Suite
}

func (s *RetentionSuite) archive(files map[string]string) string {
	root := s.T().TempDir()
	containerDir := filepath.Join(root, "quay-sha256-test", "cluster-scoped-resources")
	for name, content := range files {
		path := filepath.Join(containerDir, name)
		s.Require().NoError(os.MkdirAll(filepath.Dir(path), 0o755))
		s.Require().NoError(os.WriteFile(path, []byte(content), 0o600))
	}

	return root
}

func (s *RetentionSuite) TestListRetainsPrefixWhenFileExceedsReadCap() {
	root := s.archive(map[string]string{
		"a.yaml": "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: first\n",
		"b.yaml": strings.Repeat("x", 512),
		"c.yaml": "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: third\n",
	})

	p, err := NewProvider(root)
	s.Require().NoError(err)
	configMap := schema.GroupVersionKind{Version: "v1", Kind: "ConfigMap"}
	list, err := p.WithReadLimit(128).ListResources(context.Background(), configMap, "", ListOptions{})

	s.Require().NoError(err)
	s.Require().Len(list.Items, 1)
	s.Equal("first", list.Items[0].GetName())
	s.True(ListTruncated(list))
}

func (s *RetentionSuite) TestListAcceptsExactBudgetAndKeepsPrefix() {
	root := s.archive(map[string]string{
		"a.yaml": "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: first\n",
		"b.yaml": "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: second\n",
	})

	p, err := NewProvider(root)
	s.Require().NoError(err)
	configMap := schema.GroupVersionKind{Version: "v1", Kind: "ConfigMap"}
	first := p.GetResource(configMap, "first", "")
	s.Require().NotNil(first)

	data, err := json.Marshal(first.Object)
	s.Require().NoError(err)
	ctx := api.WithResultBudget(context.Background(), int64(len(data)))
	list, err := p.ListResources(ctx, configMap, "", ListOptions{})

	s.Require().NoError(err)
	s.Require().Len(list.Items, 1)
	s.Equal("first", list.Items[0].GetName())
	s.True(ListTruncated(list))
}

func (s *RetentionSuite) TestLoaderRetainsPrefix() {
	root := s.archive(map[string]string{
		"a.yaml": "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: first\n",
		"b.yaml": strings.Repeat("x", 512),
	})

	loaded, err := LoadWithLimits(root, 128, 1024)

	s.Require().NoError(err)
	s.Require().Len(loaded.Resources, 1)
	s.Equal("first", loaded.Resources[0].GetName())
}

func (s *RetentionSuite) TestFilteredEventsRetainMatchingRows() {
	root := s.archive(map[string]string{
		"a.yaml": "apiVersion: v1\nkind: Event\nmetadata:\n  name: ignored\ntype: Normal\n",
		"b.yaml": "apiVersion: v1\nkind: Event\nmetadata:\n  name: warning\ntype: Warning\n",
		"c.yaml": "apiVersion: v1\nkind: Event\nmetadata:\n  name: later\ntype: Warning\n",
	})
	p, err := NewProvider(root)
	s.Require().NoError(err)

	eventGVK := schema.GroupVersionKind{Version: "v1", Kind: "Event"}
	warning := p.GetResource(eventGVK, "warning", "")
	s.Require().NotNil(warning)
	data, err := json.Marshal(warning.Object)
	s.Require().NoError(err)

	ctx := api.WithResultBudget(context.Background(), int64(len(data)))
	list, err := p.ListResourcesFiltered(ctx, eventGVK, "", ListOptions{}, func(event *unstructured.Unstructured) bool {
		return event.Object["type"] == "Warning"
	})

	s.Require().NoError(err)
	s.Require().Len(list.Items, 1)
	s.Equal("warning", list.Items[0].GetName())
	s.True(ListTruncated(list))
}

func (s *RetentionSuite) TestPodAndKubeletTails() {
	root := s.archive(nil)
	containerDir := filepath.Join(root, "quay-sha256-test")
	log := strings.Repeat("older\n", 100) + "second\nthird\n"

	podPath := filepath.Join(containerDir, "namespaces", "sample", "pods", "pod", "app", "app", "logs", "current.log")
	s.Require().NoError(os.MkdirAll(filepath.Dir(podPath), 0o755))
	s.Require().NoError(os.WriteFile(podPath, []byte(log), 0o600))

	nodeDir := filepath.Join(containerDir, "nodes", "node")
	s.Require().NoError(os.MkdirAll(nodeDir, 0o755))
	f, err := os.Create(filepath.Join(nodeDir, "node_logs_kubelet.gz"))
	s.Require().NoError(err)
	gz := gzip.NewWriter(f)
	_, err = gz.Write([]byte(log))
	s.Require().NoError(err)
	s.Require().NoError(gz.Close())
	s.Require().NoError(f.Close())

	p, err := NewProvider(root)
	s.Require().NoError(err)
	p = p.WithReadLimit(1024)
	podLog, err := p.GetPodLog(PodLogOptions{Namespace: "sample", Pod: "pod", Container: "app", TailLines: 2})
	s.Require().NoError(err)
	s.Equal("second\nthird", podLog)

	diagnostics, err := p.GetNodeDiagnosticsSelected("node", func(key string) bool { return key == "kubelet" }, 16, 2)
	s.Require().NoError(err)
	s.Equal("second\nthird", diagnostics.KubeletLog)
}

func TestRetentionSuite(t *testing.T) {
	suite.Run(t, new(RetentionSuite))
}
