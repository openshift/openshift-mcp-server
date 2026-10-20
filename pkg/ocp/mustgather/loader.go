package mustgather

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/containers/kubernetes-mcp-server/pkg/config"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// LoadResult contains the results of loading a must-gather archive
type LoadResult struct {
	Metadata  MustGatherMetadata
	Resources []*unstructured.Unstructured
}

// Load reads and parses a must-gather archive from the given path. The path
// must be an archive root, i.e. it must contain a recognizable container
// directory (see FindContainerDir); a path that is itself the container
// directory is not accepted, matching IsArchive.
func Load(path string) (*LoadResult, error) {
	return LoadWithLimits(path, config.DefaultMaxBackendResponseBytes, config.DefaultMaxResultBytes)
}

// LoadWithLimits retains a prefix of the archive within the result budget.
// Callers with a live configuration pass its current limits here.
func LoadWithLimits(path string, backendLimit, resultLimit int64) (*LoadResult, error) {
	containerDir, err := FindContainerDir(path)
	if err != nil {
		return nil, err
	}

	metadata := MustGatherMetadata{
		Path:         path,
		ContainerDir: containerDir,
	}

	// Load metadata files
	metadata.Version, metadata.Timestamp, err = readMetadata(containerDir, backendLimit)
	if err != nil {
		return nil, err
	}

	var resources []*unstructured.Unstructured
	var retained int64
	err = walkResources(context.Background(), containerDir, backendLimit, func(obj *unstructured.Unstructured) bool {
		if resultLimit > 0 {
			data, err := json.Marshal(obj.Object)
			if err != nil || retained+int64(len(data)) > resultLimit {
				return false
			}
			retained += int64(len(data))
		}
		resources = append(resources, obj)
		return true
	})
	if err != nil && !(len(resources) > 0 && kubernetes.IsResponseTooLarge(err)) {
		return nil, err
	}

	metadata.ResourceCount = len(resources)

	return &LoadResult{
		Metadata:  metadata,
		Resources: resources,
	}, nil
}

// walkResources visits archive resources without retaining the entire archive.
// Returning false from visit stops the walk, including subsequent file reads.
func walkResources(ctx context.Context, containerDir string, limit int64, visit func(*unstructured.Unstructured) bool) error {
	errStop := errors.New("resource walk complete")
	for _, root := range []string{filepath.Join(containerDir, "cluster-scoped-resources"), filepath.Join(containerDir, "namespaces")} {
		if _, err := os.Stat(root); os.IsNotExist(err) {
			continue
		}
		err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return nil // keep the same best-effort behavior as Load
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if info.IsDir() || (!strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml")) {
				return nil
			}
			namespace := ""
			if root == filepath.Join(containerDir, "namespaces") {
				rel, err := filepath.Rel(root, path)
				if err != nil {
					return nil
				}
				parts := strings.Split(rel, string(filepath.Separator))
				if len(parts) < 2 || (len(parts) > 2 && parts[1] == "pods") {
					return nil
				}
				namespace = parts[0]
			}
			f, err := os.Open(path)
			if err != nil {
				return nil
			}
			data, readErr := kubernetes.ReadCapped(f, limit)
			_ = f.Close()
			if readErr != nil {
				return readErr
			}
			resources, err := decodeYAMLResources(data, namespace)
			if err != nil {
				return nil // malformed files were skipped by Load
			}
			for _, resource := range resources {
				if !visit(resource) {
					return errStop
				}
			}
			return nil
		})
		if errors.Is(err, errStop) {
			return nil
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// FindContainerDir locates the must-gather container directory
// (e.g., quay-io-okd-scos-content-sha256-...)
func FindContainerDir(basePath string) (string, error) {
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			name := entry.Name()
			if strings.HasPrefix(name, "quay") || strings.Contains(name, "sha256") {
				return filepath.Join(basePath, name), nil
			}
		}
	}

	return "", fmt.Errorf("container directory not found in %s", basePath)
}

// ReadArchiveMetadata cheaply reads the version and timestamp of the archive at
// path without indexing its resources. It returns an error when the archive's
// container directory cannot be located (matching Load and IsArchive); the
// version and timestamp are best-effort and may be empty even on success. Used
// by discovery to list archives without paying the full load cost.
func ReadArchiveMetadata(path string) (version, timestamp string, err error) {
	return ReadArchiveMetadataWithLimit(path, config.DefaultMaxBackendResponseBytes)
}

// ReadArchiveMetadataWithLimit caps each metadata file read at limit bytes.
func ReadArchiveMetadataWithLimit(path string, limit int64) (version, timestamp string, err error) {
	containerDir, err := FindContainerDir(path)
	if err != nil {
		return "", "", err
	}
	return readMetadata(containerDir, limit)
}

// IsArchive reports whether path looks like a must-gather archive, i.e. it
// contains a recognizable container directory.
func IsArchive(path string) bool {
	_, err := FindContainerDir(path)
	return err == nil
}

func readMetadata(containerDir string, limit int64) (version, timestamp string, err error) {
	read := func(name string) ([]byte, error) {
		f, err := os.Open(filepath.Join(containerDir, name))
		if err != nil {
			return nil, err
		}
		defer func() { _ = f.Close() }()
		return kubernetes.ReadCapped(f, limit)
	}
	if data, err := read("version"); err == nil {
		version = strings.TrimSpace(string(data))
	} else if kubernetes.IsResponseTooLarge(err) {
		return "", "", err
	}
	if data, err := read("timestamp"); err == nil {
		// The timestamp file accumulates one line per gather step; the first
		// non-empty line marks when the must-gather started, which is the
		// timestamp we report.
		for line := range strings.SplitSeq(string(data), "\n") {
			if line = strings.TrimSpace(line); line != "" {
				timestamp = line
				break
			}
		}
	} else if kubernetes.IsResponseTooLarge(err) {
		return "", "", err
	}
	return version, timestamp, nil
}

func decodeYAMLResources(data []byte, namespace string) ([]*unstructured.Unstructured, error) {
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, nil
	}

	normalizeYAMLTypes(raw)

	// Check if it's a List
	kind, _ := raw["kind"].(string)
	if strings.HasSuffix(kind, "List") {
		return loadList(raw, namespace)
	}

	obj := &unstructured.Unstructured{Object: raw}
	if namespace != "" && obj.GetNamespace() == "" {
		obj.SetNamespace(namespace)
	}

	// Set GVK from apiVersion + kind if not already set
	if obj.GetObjectKind().GroupVersionKind().Kind == "" {
		apiVersion, _ := raw["apiVersion"].(string)
		kind, _ := raw["kind"].(string)
		if apiVersion != "" && kind != "" {
			gv, _ := schema.ParseGroupVersion(apiVersion)
			obj.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   gv.Group,
				Version: gv.Version,
				Kind:    kind,
			})
		}
	}

	return []*unstructured.Unstructured{obj}, nil
}

func loadList(raw map[string]interface{}, namespace string) ([]*unstructured.Unstructured, error) {
	items, ok := raw["items"].([]interface{})
	if !ok {
		return nil, nil
	}

	var resources []*unstructured.Unstructured
	for _, item := range items {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		normalizeYAMLTypes(itemMap)
		obj := &unstructured.Unstructured{Object: itemMap}
		if namespace != "" && obj.GetNamespace() == "" {
			obj.SetNamespace(namespace)
		}
		// Set GVK
		apiVersion, _ := itemMap["apiVersion"].(string)
		kind, _ := itemMap["kind"].(string)
		if apiVersion != "" && kind != "" {
			gv, _ := schema.ParseGroupVersion(apiVersion)
			obj.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   gv.Group,
				Version: gv.Version,
				Kind:    kind,
			})
		}
		resources = append(resources, obj)
	}
	return resources, nil
}

// normalizeYAMLTypes converts YAML int types to int64 for JSON compatibility.
// YAML v3 unmarshals integers as int, but JSON expects int64/float64.
func normalizeYAMLTypes(m map[string]interface{}) {
	for k, v := range m {
		switch val := v.(type) {
		case int:
			m[k] = int64(val)
		case map[string]interface{}:
			normalizeYAMLTypes(val)
		case []interface{}:
			normalizeSlice(val)
		}
	}
}

func normalizeSlice(s []interface{}) {
	for i, v := range s {
		switch val := v.(type) {
		case int:
			s[i] = int64(val)
		case map[string]interface{}:
			normalizeYAMLTypes(val)
		case []interface{}:
			normalizeSlice(val)
		}
	}
}
