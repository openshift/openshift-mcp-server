package mustgather

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Provider gives access to must-gather archive data
type Provider struct {
	metadata  MustGatherMetadata
	readLimit int64
}

// ErrListTruncated means a list retained a prefix and stopped at the result budget.
var ErrListTruncated = errors.New("must-gather list truncated at max_result_bytes")

// NewProvider locates the archive; resources are read on demand rather than
// permanently retaining the parsed contents of every YAML file.
func NewProvider(path string) (*Provider, error) {
	containerDir, err := FindContainerDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load must-gather: %w", err)
	}
	metadata := MustGatherMetadata{Path: path, ContainerDir: containerDir}
	return &Provider{metadata: metadata}, nil
}

// WithReadLimit returns a per-call view of a cached provider. The registry
// never holds a stale limit when configuration is reloaded.
func (p *Provider) WithReadLimit(limit int64) *Provider {
	copy := *p
	copy.readLimit = limit
	return &copy
}

// GetMetadata returns the must-gather metadata
func (p *Provider) GetMetadata() MustGatherMetadata {
	metadata := p.metadata
	metadata.Version, metadata.Timestamp, _ = readMetadata(metadata.ContainerDir, p.readLimit)
	namespaces := make(map[string]struct{})
	_ = walkResources(context.Background(), metadata.ContainerDir, p.readLimit, func(obj *unstructured.Unstructured) bool {
		metadata.ResourceCount++
		if ns := obj.GetNamespace(); ns != "" {
			namespaces[ns] = struct{}{}
		}
		return true
	})
	metadata.NamespaceCount = len(namespaces)
	return metadata
}

// GetResource retrieves a specific resource
func (p *Provider) GetResource(gvk schema.GroupVersionKind, name, namespace string) *unstructured.Unstructured {
	var found *unstructured.Unstructured
	_ = walkResources(context.Background(), p.metadata.ContainerDir, p.readLimit, func(obj *unstructured.Unstructured) bool {
		if obj.GroupVersionKind() == gvk && obj.GetName() == name && obj.GetNamespace() == namespace {
			found = obj
			return false
		}
		return true
	})
	return found
}

// ListResources returns resources matching the given criteria
func (p *Provider) ListResources(ctx context.Context, gvk schema.GroupVersionKind, namespace string, opts ListOptions) (*unstructured.UnstructuredList, error) {
	return p.ListResourcesFiltered(ctx, gvk, namespace, opts, nil)
}

// ListResourcesFiltered applies match during traversal, before consuming the
// result budget. A nil match accepts every resource.
func (p *Provider) ListResourcesFiltered(ctx context.Context, gvk schema.GroupVersionKind, namespace string, opts ListOptions, match func(*unstructured.Unstructured) bool) (*unstructured.UnstructuredList, error) {
	result := &unstructured.UnstructuredList{}
	limit := api.ResultLimit(ctx)
	remaining := limit
	truncated := false
	err := p.VisitResources(ctx, gvk, namespace, opts, func(obj *unstructured.Unstructured) bool {
		if match != nil && !match(obj) {
			return true
		}
		if remaining > 0 {
			data, err := json.Marshal(obj.Object)
			if err != nil || int64(len(data)) >= remaining {
				truncated = true
				return false
			}
			remaining -= int64(len(data))
		} else if limit > 0 {
			truncated = true
			return false
		}
		result.Items = append(result.Items, *obj)
		return true
	})
	if err == nil && truncated {
		err = ErrListTruncated
	}
	return result, err
}

// VisitResources selects and visits resources one by one. It stops reading
// archive files when visit returns false or opts.Limit matches have been seen.
func (p *Provider) VisitResources(ctx context.Context, gvk schema.GroupVersionKind, namespace string, opts ListOptions, visit func(*unstructured.Unstructured) bool) error {
	matcher, err := newListMatcher(opts)
	if err != nil {
		return err
	}
	count := 0
	return walkResources(ctx, p.metadata.ContainerDir, p.readLimit, func(obj *unstructured.Unstructured) bool {
		if obj.GroupVersionKind() != gvk || (namespace != "" && obj.GetNamespace() != namespace) || !matcher.matches(obj) {
			return true
		}
		count++
		return visit(obj) && (opts.Limit <= 0 || count < opts.Limit)
	})
}

// ListNamespaces returns all namespaces found in the archive
func (p *Provider) ListNamespaces() []string {
	seen := make(map[string]struct{})
	_ = walkResources(context.Background(), p.metadata.ContainerDir, p.readLimit, func(obj *unstructured.Unstructured) bool {
		if ns := obj.GetNamespace(); ns != "" {
			seen[ns] = struct{}{}
		}
		return true
	})
	namespaces := make([]string, 0, len(seen))
	for ns := range seen {
		namespaces = append(namespaces, ns)
	}
	sort.Strings(namespaces)
	return namespaces
}

// GetPodLogPath resolves the filesystem path for a pod's container log file
// without reading it. Returns the path or an error if the container cannot be
// determined or the log file does not exist.
func (p *Provider) GetPodLogPath(opts PodLogOptions) (string, error) {
	container := opts.Container

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

	if _, err := os.Stat(logPath); err != nil {
		return "", fmt.Errorf("log file not found: %w", err)
	}

	return logPath, nil
}

// GetPodLog reads pod container logs from the archive
func (p *Provider) GetPodLog(opts PodLogOptions) (string, error) {
	logPath, err := p.GetPodLogPath(opts)
	if err != nil {
		return "", err
	}

	content, err := p.readTextFile(logPath)
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
	return p.GetNodeDiagnosticsSelected(nodeName, nil, 0, 0)
}

// GetNodeDiagnosticsSelected reads only requested sections and stops once the
// retained diagnostic text fills budget. A nil include selects every section.
// kubeletTail selects the final lines before applying the budget (0 keeps all).
func (p *Provider) GetNodeDiagnosticsSelected(nodeName string, include func(string) bool, budget int64, kubeletTail int) (*NodeDiagnostics, error) {
	nodeDir := filepath.Join(p.metadata.ContainerDir, "nodes", nodeName)
	if _, err := os.Stat(nodeDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("node directory not found: %s", nodeName)
	}

	diag := &NodeDiagnostics{NodeName: nodeName}
	selected := func(key string) bool { return include == nil || include(key) }
	remaining := budget
	keep := func(content string) (string, bool) {
		if budget <= 0 {
			return content, true
		}
		if int64(len(content)) >= remaining {
			part := api.NewBuilderWithBudget(remaining)
			part.WriteString(content)
			if !part.Truncated() {
				part.WriteString("\n")
			}
			return part.String(), false
		}
		remaining -= int64(len(content))
		return content, true
	}

	// Read kubelet log (may be gzipped)
	if selected("kubelet") {
		kubeletGz := filepath.Join(nodeDir, nodeName+"_logs_kubelet.gz")
		if content, err := p.readGzipFile(kubeletGz); err == nil {
			if kubeletTail > 0 {
				content = TailLines(content, kubeletTail)
			}
			var more bool
			diag.KubeletLog, more = keep(content)
			if !more {
				return diag, nil
			}
		} else if kubernetes.IsResponseTooLarge(err) {
			return nil, err
		}
	}

	textFiles := []struct {
		key, filename string
		target        *string
	}{
		{"sysinfo", "sysinfo.log", &diag.SysInfo},
		{"lscpu", "lscpu", &diag.Lscpu},
		{"cpu", "cpu_affinities.json", &diag.CPUAffinities},
		{"irq", "irq_affinities.json", &diag.IRQAffinities},
		{"lspci", "lspci", &diag.Lspci},
		{"dmesg", "dmesg", &diag.Dmesg},
		{"cmdline", "proc_cmdline", &diag.ProcCmdline},
		{"pods", "pods_info.json", &diag.PodsInfo},
		{"podresources", "podresources.json", &diag.PodResources},
	}

	for _, file := range textFiles {
		if !selected(file.key) {
			continue
		}
		if content, err := p.readTextFile(filepath.Join(nodeDir, file.filename)); err == nil {
			var more bool
			*file.target, more = keep(content)
			if !more {
				return diag, nil
			}
		} else if kubernetes.IsResponseTooLarge(err) {
			return nil, err
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
func (p *Provider) readTextFile(path string) (string, error) {
	data, err := p.readFile(path)
	return string(data), err
}

func (p *Provider) readFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return kubernetes.ReadCapped(f, p.readLimit)
}

// readGzipFile reads and decompresses a gzipped file.
func (p *Provider) readGzipFile(path string) (string, error) {
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

	data, err := kubernetes.ReadCapped(io.NopCloser(reader), p.readLimit)
	if err != nil {
		return "", fmt.Errorf("failed to decompress %s: %w", path, err)
	}

	return string(data), nil
}

// TailLines returns the last n lines of a string
func TailLines(content string, n int) string {
	lines := strings.Split(content, "\n")
	if len(lines) <= n {
		return content
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}
