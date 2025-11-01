package mcp

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"testing"

	"github.com/containers/kubernetes-mcp-server/internal/test"
	configuration "github.com/containers/kubernetes-mcp-server/pkg/config"
	"github.com/mark3labs/mcp-go/mcp"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func TestGenerateSnapshots(t *testing.T) {
	if os.Getenv("GENERATE_SNAPSHOTS") != "true" {
		t.Skip("Skipping snapshot generation test. Set GENERATE_SNAPSHOTS=true to run")
	}

	// Test 1: Default toolsets
	t.Run("Generate toolsets-full-tools.json", func(t *testing.T) {
		mockServer := test.NewMockServer()
		defer mockServer.Close()
		cfg := configuration.Default()
		cfg.KubeConfig = mockServer.KubeconfigFile(t)
		mcpServer, err := NewServer(Configuration{StaticConfig: cfg})
		if err != nil {
			t.Fatal(err)
		}
		defer mcpServer.Close()
		client := test.NewMcpClient(t, mcpServer.ServeHTTP(nil))
		defer client.Close()
		tools, err := client.ListTools(context.Background(), mcp.ListToolsRequest{})
		if err != nil {
			t.Fatal(err)
		}
		writeJSON(t, "testdata/toolsets-full-tools.json", tools.Tools)
	})

	// Test 2: OpenShift
	t.Run("Generate toolsets-full-tools-openshift.json", func(t *testing.T) {
		mockServer := test.NewMockServer()
		mockServer.Handle(&test.InOpenShiftHandler{})
		defer mockServer.Close()
		cfg := configuration.Default()
		cfg.KubeConfig = mockServer.KubeconfigFile(t)
		mcpServer, err := NewServer(Configuration{StaticConfig: cfg})
		if err != nil {
			t.Fatal(err)
		}
		defer mcpServer.Close()
		client := test.NewMcpClient(t, mcpServer.ServeHTTP(nil))
		defer client.Close()
		tools, err := client.ListTools(context.Background(), mcp.ListToolsRequest{})
		if err != nil {
			t.Fatal(err)
		}
		writeJSON(t, "testdata/toolsets-full-tools-openshift.json", tools.Tools)
	})

	// Test 3: Multi-cluster (11 clusters)
	t.Run("Generate toolsets-full-tools-multicluster.json", func(t *testing.T) {
		mockServer := test.NewMockServer()
		defer mockServer.Close()
		kubeconfig := mockServer.Kubeconfig()
		for i := 0; i < 10; i++ {
			kubeconfig.Contexts[strconv.Itoa(i)] = clientcmdapi.NewContext()
		}
		cfg := configuration.Default()
		cfg.KubeConfig = test.KubeconfigFile(t, kubeconfig)
		mcpServer, err := NewServer(Configuration{StaticConfig: cfg})
		if err != nil {
			t.Fatal(err)
		}
		defer mcpServer.Close()
		client := test.NewMcpClient(t, mcpServer.ServeHTTP(nil))
		defer client.Close()
		tools, err := client.ListTools(context.Background(), mcp.ListToolsRequest{})
		if err != nil {
			t.Fatal(err)
		}
		writeJSON(t, "testdata/toolsets-full-tools-multicluster.json", tools.Tools)
	})

	// Test 4: Multi-cluster enum (2 clusters)
	t.Run("Generate toolsets-full-tools-multicluster-enum.json", func(t *testing.T) {
		mockServer := test.NewMockServer()
		defer mockServer.Close()
		kubeconfig := mockServer.Kubeconfig()
		kubeconfig.Contexts["extra-cluster"] = clientcmdapi.NewContext()
		cfg := configuration.Default()
		cfg.KubeConfig = test.KubeconfigFile(t, kubeconfig)
		mcpServer, err := NewServer(Configuration{StaticConfig: cfg})
		if err != nil {
			t.Fatal(err)
		}
		defer mcpServer.Close()
		client := test.NewMcpClient(t, mcpServer.ServeHTTP(nil))
		defer client.Close()
		tools, err := client.ListTools(context.Background(), mcp.ListToolsRequest{})
		if err != nil {
			t.Fatal(err)
		}
		writeJSON(t, "testdata/toolsets-full-tools-multicluster-enum.json", tools.Tools)
	})
}

func writeJSON(t *testing.T, filename string, data interface{}) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filename, jsonData, 0644)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Written %s", filename)
}
