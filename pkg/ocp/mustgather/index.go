package mustgather

import (
	"context"
	"fmt"
	"sort"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ResourceIndex provides fast O(1) lookups for must-gather resources
type ResourceIndex struct {
	// byGVK maps GVK -> key -> resource (key is "namespace/name" or "name")
	byGVK map[schema.GroupVersionKind]map[string]*unstructured.Unstructured
	// byNamespace maps namespace -> GVK -> name -> resource
	byNamespace map[string]map[schema.GroupVersionKind]map[string]*unstructured.Unstructured
	// namespaces is a sorted list of all namespaces
	namespaces []string
	// count is the total number of resources
	count int
}

// NewResourceIndex creates an empty resource index
func NewResourceIndex() *ResourceIndex {
	return &ResourceIndex{
		byGVK:       make(map[schema.GroupVersionKind]map[string]*unstructured.Unstructured),
		byNamespace: make(map[string]map[schema.GroupVersionKind]map[string]*unstructured.Unstructured),
	}
}

// BuildIndex creates a resource index from a list of unstructured resources
func BuildIndex(resources []*unstructured.Unstructured) *ResourceIndex {
	idx := NewResourceIndex()
	for _, r := range resources {
		idx.Add(r)
	}
	idx.buildNamespaceList()
	return idx
}

// Add adds a resource to the index
func (idx *ResourceIndex) Add(obj *unstructured.Unstructured) {
	gvk := obj.GetObjectKind().GroupVersionKind()
	name := obj.GetName()
	namespace := obj.GetNamespace()

	key := name
	if namespace != "" {
		key = namespace + "/" + name
	}

	// Index by GVK
	if _, ok := idx.byGVK[gvk]; !ok {
		idx.byGVK[gvk] = make(map[string]*unstructured.Unstructured)
	}
	idx.byGVK[gvk][key] = obj

	// Index by namespace
	if namespace != "" {
		if _, ok := idx.byNamespace[namespace]; !ok {
			idx.byNamespace[namespace] = make(map[schema.GroupVersionKind]map[string]*unstructured.Unstructured)
		}
		if _, ok := idx.byNamespace[namespace][gvk]; !ok {
			idx.byNamespace[namespace][gvk] = make(map[string]*unstructured.Unstructured)
		}
		idx.byNamespace[namespace][gvk][name] = obj
	}

	idx.count++
}

func (idx *ResourceIndex) buildNamespaceList() {
	nsSet := make(map[string]struct{})
	for ns := range idx.byNamespace {
		nsSet[ns] = struct{}{}
	}
	idx.namespaces = make([]string, 0, len(nsSet))
	for ns := range nsSet {
		idx.namespaces = append(idx.namespaces, ns)
	}
	sort.Strings(idx.namespaces)
}

// Get retrieves a specific resource by GVK, name, and namespace
func (idx *ResourceIndex) Get(gvk schema.GroupVersionKind, name, namespace string) *unstructured.Unstructured {
	gvkMap, ok := idx.byGVK[gvk]
	if !ok {
		return nil
	}

	key := name
	if namespace != "" {
		key = namespace + "/" + name
	}

	obj, ok := gvkMap[key]
	if !ok {
		return nil
	}
	return obj.DeepCopy()
}

// List returns all resources matching the given GVK and optional namespace
func (idx *ResourceIndex) List(_ context.Context, gvk schema.GroupVersionKind, namespace string, opts ListOptions) (*unstructured.UnstructuredList, error) {
	matcher, err := newListMatcher(opts)
	if err != nil {
		return nil, err
	}

	result := &unstructured.UnstructuredList{}

	if namespace != "" {
		// Namespace-scoped query
		nsMap, ok := idx.byNamespace[namespace]
		if !ok {
			return result, nil
		}
		gvkMap, ok := nsMap[gvk]
		if !ok {
			return result, nil
		}
		for _, obj := range gvkMap {
			if matcher.matches(obj) {
				result.Items = append(result.Items, *obj.DeepCopy())
			}
		}
	} else {
		// Cluster-wide query
		gvkMap, ok := idx.byGVK[gvk]
		if !ok {
			return result, nil
		}
		for _, obj := range gvkMap {
			if matcher.matches(obj) {
				result.Items = append(result.Items, *obj.DeepCopy())
			}
		}
	}

	// Apply limit
	if opts.Limit > 0 && len(result.Items) > opts.Limit {
		result.Items = result.Items[:opts.Limit]
	}

	return result, nil
}

// ListNamespaces returns all namespaces in the index
func (idx *ResourceIndex) ListNamespaces() []string {
	ns := make([]string, len(idx.namespaces))
	copy(ns, idx.namespaces)
	return ns
}

// Count returns the total number of indexed resources
func (idx *ResourceIndex) Count() int {
	return idx.count
}

type listMatcher struct {
	labelSel    labels.Selector
	fieldSel    fields.Selector
	hasFieldSel bool
}

func newListMatcher(opts ListOptions) (*listMatcher, error) {
	m := &listMatcher{}

	if opts.LabelSelector != "" {
		sel, err := labels.Parse(opts.LabelSelector)
		if err != nil {
			return nil, fmt.Errorf("invalid label selector %q: %w", opts.LabelSelector, err)
		}
		m.labelSel = sel
	}

	if opts.FieldSelector != "" {
		sel, err := fields.ParseSelector(opts.FieldSelector)
		if err != nil {
			return nil, fmt.Errorf("invalid field selector %q: %w", opts.FieldSelector, err)
		}
		m.fieldSel = sel
		m.hasFieldSel = true
	}

	return m, nil
}

func (m *listMatcher) matches(obj *unstructured.Unstructured) bool {
	if m.labelSel != nil && !m.labelSel.Matches(labels.Set(obj.GetLabels())) {
		return false
	}
	if m.hasFieldSel {
		objFields := fields.Set{
			"metadata.name":      obj.GetName(),
			"metadata.namespace": obj.GetNamespace(),
		}
		if !m.fieldSel.Matches(objFields) {
			return false
		}
	}
	return true
}
