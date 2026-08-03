package types

// GetResourceParams is a type that contains the group, version, kind, name and namespace of a resource.
type GetResourceParams struct {
	GroupVersionKind
	GetParams
}

// GetResourceResult is a type that contains the resource data.
type GetResourceResult struct {
	Resource Resource `json:"resource"`
}

// ListParams is a type that contains the namespace, label selector and output type of a resource.
type ListParams struct {
	Namespace     string `json:"namespace,omitempty"`
	LabelSelector string `json:"label_selector,omitempty"`
	OutputParams
}

// ListResourcesParams is a type that contains the group, version, kind, namespace and output type of a resource.
type ListResourcesParams struct {
	GroupVersionKind
	ListParams
}

// ListResourcesResult is a type that contains the resource data.
type ListResourcesResult struct {
	Resources []Resource `json:"resources"`
}
