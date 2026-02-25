package mustgather

import (
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Provider gives access to must-gather archive data
type Provider struct {
	metadata MustGatherMetadata
	index    *ResourceIndex
}

// NewProvider loads a must-gather archive and builds an in-memory index
func NewProvider(path string) (*Provider, error) {
	result, err := Load(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load must-gather: %w", err)
	}

	index := BuildIndex(result.Resources)
	result.Metadata.NamespaceCount = len(index.ListNamespaces())

	return &Provider{
		metadata: result.Metadata,
		index:    index,
	}, nil
}

// GetMetadata returns the must-gather metadata
func (p *Provider) GetMetadata() MustGatherMetadata {
	return p.metadata
}

// GetResource retrieves a specific resource
func (p *Provider) GetResource(gvk schema.GroupVersionKind, name, namespace string) *unstructured.Unstructured {
	return p.index.Get(gvk, name, namespace)
}

// ListResources returns resources matching the given criteria
func (p *Provider) ListResources(ctx context.Context, gvk schema.GroupVersionKind, namespace string, opts ListOptions) *unstructured.UnstructuredList {
	return p.index.List(ctx, gvk, namespace, opts)
}

// ListNamespaces returns all namespaces found in the archive
func (p *Provider) ListNamespaces() []string {
	return p.index.ListNamespaces()
}

// GetPodLog reads pod container logs from the archive
func (p *Provider) GetPodLog(opts PodLogOptions) (string, error) {
	container := opts.Container

	// If container not specified, try to find first available
	if container == "" {
		containers, err := p.ListPodContainers(opts.Namespace, opts.Pod)
		if err != nil {
			return "", err
		}
		if len(containers) == 0 {
			return "", fmt.Errorf("no containers found for pod %s/%s", opts.Namespace, opts.Pod)
		}
		container = containers[0]
	}

	logType := string(opts.LogType)
	if logType == "" {
		logType = string(LogTypeCurrent)
	}

	logPath := filepath.Join(
		p.metadata.ContainerDir, "namespaces", opts.Namespace,
		"pods", opts.Pod, container, container, "logs", logType+".log",
	)

	content, err := readTextFile(logPath)
	if err != nil {
		return "", fmt.Errorf("failed to read log file: %w", err)
	}

	if opts.TailLines > 0 {
		content = TailLines(content, opts.TailLines)
	}

	return content, nil
}

// ListPodContainers returns container names that have logs available
func (p *Provider) ListPodContainers(namespace, pod string) ([]string, error) {
	podDir := filepath.Join(p.metadata.ContainerDir, "namespaces", namespace, "pods", pod)
	entries, err := os.ReadDir(podDir)
	if err != nil {
		return nil, fmt.Errorf("pod directory not found: %w", err)
	}

	var containers []string
	for _, entry := range entries {
		if entry.IsDir() {
			containers = append(containers, entry.Name())
		}
	}
	return containers, nil
}

// GetNodeDiagnostics reads all diagnostic data for a node
func (p *Provider) GetNodeDiagnostics(nodeName string) (*NodeDiagnostics, error) {
	nodeDir := filepath.Join(p.metadata.ContainerDir, "nodes", nodeName)
	if _, err := os.Stat(nodeDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("node directory not found: %s", nodeName)
	}

	diag := &NodeDiagnostics{NodeName: nodeName}

	// Read kubelet log (may be gzipped)
	kubeletGz := filepath.Join(nodeDir, nodeName+"_logs_kubelet.gz")
	if content, err := readGzipFile(kubeletGz); err == nil {
		diag.KubeletLog = content
	}

	// Read text diagnostic files
	textFiles := map[string]*string{
		"sysinfo.log":         &diag.SysInfo,
		"cpu_affinities.json": &diag.CPUAffinities,
		"irq_affinities.json": &diag.IRQAffinities,
		"pods_info.json":      &diag.PodsInfo,
		"podresources.json":   &diag.PodResources,
		"lscpu":               &diag.Lscpu,
		"lspci":               &diag.Lspci,
		"dmesg":               &diag.Dmesg,
		"proc_cmdline":        &diag.ProcCmdline,
	}

	for filename, target := range textFiles {
		if content, err := readTextFile(filepath.Join(nodeDir, filename)); err == nil {
			*target = content
		}
	}

	return diag, nil
}

// ListNodes returns all node names with diagnostic data
func (p *Provider) ListNodes() ([]string, error) {
	nodesDir := filepath.Join(p.metadata.ContainerDir, "nodes")
	entries, err := os.ReadDir(nodesDir)
	if err != nil {
		return nil, nil // no nodes directory is not an error
	}

	var nodes []string
	for _, entry := range entries {
		if entry.IsDir() {
			nodes = append(nodes, entry.Name())
		}
	}
	return nodes, nil
}

// readTextFile reads a text file and returns its content
func readTextFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// readGzipFile reads and decompresses a gzipped file
func readGzipFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	reader, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer func() { _ = reader.Close() }()

	var sb strings.Builder
	buf := make([]byte, 32*1024)
	for {
		n, readErr := reader.Read(buf)
		if n > 0 {
			sb.Write(buf[:n])
		}
		if readErr != nil {
			break
		}
	}

	return sb.String(), nil
}

// TailLines returns the last n lines of a string
func TailLines(content string, n int) string {
	lines := strings.Split(content, "\n")
	if len(lines) <= n {
		return content
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}
