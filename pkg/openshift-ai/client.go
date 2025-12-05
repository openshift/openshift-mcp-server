package openshiftai

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

// Client manages OpenShift AI operations
type Client struct {
	config          *rest.Config
	dynamicClient   dynamic.Interface
	discoveryClient discovery.DiscoveryInterface
	restMapper      meta.RESTMapper
}

// Config holds OpenShift AI client configuration
type Config struct {
	// Timeout for API operations
	Timeout time.Duration
	// Enable debug logging
	Debug bool
}

// DefaultConfig returns default configuration for OpenShift AI client
func DefaultConfig() *Config {
	return &Config{
		Timeout: 30 * time.Second,
		Debug:   false,
	}
}

// NewClient creates a new OpenShift AI client
func NewClient(cfg *rest.Config, clientConfig *Config) (*Client, error) {
	if clientConfig == nil {
		clientConfig = DefaultConfig()
	}

	dynamicClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery client: %w", err)
	}

	restMapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{})

	client := &Client{
		config:          cfg,
		dynamicClient:   dynamicClient,
		discoveryClient: discoveryClient,
		restMapper:      restMapper,
	}

	if clientConfig.Debug {
		klog.V(2).InfoS("OpenShift AI client initialized with debug logging")
	}

	return client, nil
}

// IsAvailable checks if OpenShift AI is available in the cluster
func (c *Client) IsAvailable(ctx context.Context) bool {
	// Check for key OpenShift AI CRDs
	crdGroups := []string{
		"datasciencepipelinesapplications.opendatahub.io",
		"kserve.io",
		"tekton.dev",
	}

	for _, group := range crdGroups {
		_, err := c.discoveryClient.ServerResourcesForGroupVersion(group + "/v1")
		if err != nil {
			klog.V(3).InfoS("OpenShift AI CRD group not available", "group", group, "error", err)
			continue
		}
		klog.V(2).InfoS("Found OpenShift AI CRD group", "group", group)
		return true
	}

	return false
}

// GetDynamicClient returns the dynamic client for CRD operations
func (c *Client) GetDynamicClient() dynamic.Interface {
	return c.dynamicClient
}

// GetDiscoveryClient returns the discovery client for API discovery
func (c *Client) GetDiscoveryClient() discovery.DiscoveryInterface {
	return c.discoveryClient
}

// GetRESTMapper returns the REST mapper for resource mapping
func (c *Client) GetRESTMapper() meta.RESTMapper {
	return c.restMapper
}

// GetGVR returns GroupVersionResource for a given resource name
func (c *Client) GetGVR(resource string) (schema.GroupVersionResource, error) {
	switch resource {
	case "datascienceprojects":
		return schema.GroupVersionResource{
			Group:    "datasciencepipelinesapplications.opendatahub.io",
			Version:  "v1",
			Resource: "datasciencepipelinesapplications",
		}, nil
	case "applications":
		return schema.GroupVersionResource{
			Group:    "app.opendatahub.io",
			Version:  "v1",
			Resource: "applications",
		}, nil
	case "models":
		return schema.GroupVersionResource{
			Group:    "model.opendatahub.io",
			Version:  "v1",
			Resource: "models",
		}, nil
	case "experiments":
		return schema.GroupVersionResource{
			Group:    "datasciencepipelines.opendatahub.io",
			Version:  "v1",
			Resource: "experiments",
		}, nil
	case "notebooks":
		return schema.GroupVersionResource{
			Group:    "kubeflow.org",
			Version:  "v1",
			Resource: "notebooks",
		}, nil
	case "inferenceservices":
		return schema.GroupVersionResource{
			Group:    "serving.kserve.io",
			Version:  "v1beta1",
			Resource: "inferenceservices",
		}, nil
	case "pipelines":
		return schema.GroupVersionResource{
			Group:    "datasciencepipelines.opendatahub.io",
			Version:  "v1alpha1",
			Resource: "pipelines",
		}, nil
	case "pipelineruns":
		return schema.GroupVersionResource{
			Group:    "tekton.dev",
			Version:  "v1beta1",
			Resource: "pipelineruns",
		}, nil
	default:
		return schema.GroupVersionResource{}, fmt.Errorf("unknown resource type: %s", resource)
	}
}

// ListNamespaces returns all namespaces that have OpenShift AI resources
func (c *Client) ListNamespaces(ctx context.Context) ([]string, error) {
	gvr, err := c.GetGVR("datascienceprojects")
	if err != nil {
		return nil, err
	}

	resourceInterface := c.dynamicClient.Resource(gvr)
	list, err := resourceInterface.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list DataScienceProjects: %w", err)
	}

	namespaceSet := make(map[string]bool)
	for _, item := range list.Items {
		if ns := item.GetNamespace(); ns != "" {
			namespaceSet[ns] = true
		}
	}

	var namespaces []string
	for ns := range namespaceSet {
		namespaces = append(namespaces, ns)
	}

	return namespaces, nil
}
