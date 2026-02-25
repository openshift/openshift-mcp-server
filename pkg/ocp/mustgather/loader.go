package mustgather

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// LoadResult contains the results of loading a must-gather archive
type LoadResult struct {
	Metadata  MustGatherMetadata
	Resources []*unstructured.Unstructured
}

// Load reads and parses a must-gather archive from the given path
func Load(path string) (*LoadResult, error) {
	containerDir, err := FindContainerDir(path)
	if err != nil {
		// If no container dir found, use the path directly
		containerDir = path
	}

	metadata := MustGatherMetadata{
		Path:         path,
		ContainerDir: containerDir,
	}

	// Load metadata files
	loadMetadata(containerDir, &metadata)

	// Load all resources
	var resources []*unstructured.Unstructured

	clusterResources, err := loadClusterScopedResources(containerDir)
	if err == nil {
		resources = append(resources, clusterResources...)
	}

	namespacedResources, err := loadNamespacedResources(containerDir)
	if err == nil {
		resources = append(resources, namespacedResources...)
	}

	metadata.ResourceCount = len(resources)

	return &LoadResult{
		Metadata:  metadata,
		Resources: resources,
	}, nil
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

func loadMetadata(containerDir string, metadata *MustGatherMetadata) {
	if data, err := os.ReadFile(filepath.Join(containerDir, "version")); err == nil {
		metadata.Version = strings.TrimSpace(string(data))
	}
	if data, err := os.ReadFile(filepath.Join(containerDir, "timestamp")); err == nil {
		metadata.Timestamp = strings.TrimSpace(string(data))
	}
}

func loadClusterScopedResources(containerDir string) ([]*unstructured.Unstructured, error) {
	clusterDir := filepath.Join(containerDir, "cluster-scoped-resources")
	if _, err := os.Stat(clusterDir); os.IsNotExist(err) {
		return nil, nil
	}

	var resources []*unstructured.Unstructured
	err := filepath.Walk(clusterDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if info.IsDir() || (!strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml")) {
			return nil
		}

		loaded, loadErr := loadYAMLFile(path, "")
		if loadErr != nil {
			return nil // skip files that can't be parsed
		}
		resources = append(resources, loaded...)
		return nil
	})

	return resources, err
}

func loadNamespacedResources(containerDir string) ([]*unstructured.Unstructured, error) {
	namespacesDir := filepath.Join(containerDir, "namespaces")
	if _, err := os.Stat(namespacesDir); os.IsNotExist(err) {
		return nil, nil
	}

	var resources []*unstructured.Unstructured

	nsEntries, err := os.ReadDir(namespacesDir)
	if err != nil {
		return nil, err
	}

	for _, nsEntry := range nsEntries {
		if !nsEntry.IsDir() {
			continue
		}
		namespace := nsEntry.Name()
		nsDir := filepath.Join(namespacesDir, namespace)

		err := filepath.Walk(nsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() || (!strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml")) {
				return nil
			}

			// Skip pod log directories
			rel, _ := filepath.Rel(nsDir, path)
			if strings.HasPrefix(rel, "pods"+string(filepath.Separator)) {
				return nil
			}

			loaded, loadErr := loadYAMLFile(path, namespace)
			if loadErr != nil {
				return nil
			}
			resources = append(resources, loaded...)
			return nil
		})
		if err != nil {
			continue
		}
	}

	return resources, nil
}

func loadYAMLFile(path, namespace string) ([]*unstructured.Unstructured, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

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
