package openshiftai

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/klog/v2"
)

// AvailabilityStatus represents the availability status of OpenShift AI components
type AvailabilityStatus struct {
	Available   bool     `json:"available"`
	Version     string   `json:"version,omitempty"`
	Components  []string `json:"components,omitempty"`
	MissingCRDs []string `json:"missingCRDs,omitempty"`
	Warnings    []string `json:"warnings,omitempty"`
}

// Detector handles OpenShift AI detection and availability checking
type Detector struct {
	client          *Client
	discoveryClient discovery.DiscoveryInterface
}

// NewDetector creates a new OpenShift AI detector
func NewDetector(client *Client) *Detector {
	return &Detector{
		client:          client,
		discoveryClient: client.GetDiscoveryClient(),
	}
}

// CheckAvailability performs comprehensive OpenShift AI availability check
func (d *Detector) CheckAvailability(ctx context.Context) *AvailabilityStatus {
	status := &AvailabilityStatus{
		Available:   false,
		Components:  []string{},
		MissingCRDs: []string{},
		Warnings:    []string{},
	}

	// Define required OpenShift AI CRD groups
	requiredCRDs := map[string]string{
		"datascience.opendatahub.io": "Data Science Projects",
		"kubeflow.org":               "Jupyter Notebooks",
		"serving.kserve.io":          "Model Serving",
		"tekton.dev":                 "AI Pipelines",
	}

	// Check each required CRD group
	for group, component := range requiredCRDs {
		available, version := d.checkCRDGroup(ctx, group)
		if available {
			status.Components = append(status.Components, fmt.Sprintf("%s (%s)", component, version))
			if group == "datascience.opendatahub.io" {
				status.Version = version
			}
		} else {
			status.MissingCRDs = append(status.MissingCRDs, fmt.Sprintf("%s (%s)", group, component))
		}
	}

	// Check for optional GPU monitoring components
	gpuAvailable := d.checkGPUSupport(ctx)
	if gpuAvailable {
		status.Components = append(status.Components, "GPU Monitoring")
	}

	// Determine overall availability
	// Core requirement: at least Data Science Projects should be available
	coreAvailable, _ := d.checkCRDGroup(ctx, "datascience.opendatahub.io")
	if coreAvailable {
		status.Available = true
	}

	// Add warnings for missing optional components
	if len(status.MissingCRDs) > 0 {
		status.Warnings = append(status.Warnings, fmt.Sprintf("Some OpenShift AI components are not available: %v", status.MissingCRDs))
	}

	if status.Available {
		klog.V(2).InfoS("OpenShift AI is available", "components", status.Components)
	} else {
		klog.V(2).InfoS("OpenShift AI is not available in this cluster")
	}

	return status
}

// checkCRDGroup checks if a CRD group is available and returns its version
func (d *Detector) checkCRDGroup(ctx context.Context, group string) (bool, string) {
	// Try common versions
	versions := []string{"v1", "v1beta1", "v1alpha1"}

	for _, version := range versions {
		gv := schema.GroupVersion{Group: group, Version: version}
		_, err := d.discoveryClient.ServerResourcesForGroupVersion(gv.String())
		if err == nil {
			klog.V(3).InfoS("Found OpenShift AI CRD group", "group", group, "version", version)
			return true, version
		}
	}

	klog.V(3).InfoS("OpenShift AI CRD group not found", "group", group)
	return false, ""
}

// checkGPUSupport checks if GPU monitoring is available
func (d *Detector) checkGPUSupport(ctx context.Context) bool {
	// Check for GPU-related resources
	gpuIndicators := []string{
		"nvidia.com/gpu",
		"amd.com/gpu",
		"intel.com/gpu",
	}

	// Check for GPU nodes
	gvr := schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "nodes",
	}

	nodes, err := d.client.GetDynamicClient().Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		klog.V(3).InfoS("Failed to list nodes for GPU detection", "error", err)
		return false
	}

	for _, node := range nodes.Items {
		if capacity, found, _ := unstructured.NestedStringMap(node.Object, "status", "capacity"); found {
			for _, indicator := range gpuIndicators {
				if _, exists := capacity[indicator]; exists {
					klog.V(3).InfoS("Found GPU indicator", "indicator", indicator, "node", node.GetName())
					return true
				}
			}
		}
	}

	return false
}

// IsOpenShiftAICluster quickly checks if this is an OpenShift AI cluster
func (d *Detector) IsOpenShiftAICluster(ctx context.Context) bool {
	// Quick check for the core DataScienceProject CRD
	available, _ := d.checkCRDGroup(ctx, "datascience.opendatahub.io")
	return available
}

// GetOpenShiftAIVersion returns the detected OpenShift AI version
func (d *Detector) GetOpenShiftAIVersion(ctx context.Context) (string, error) {
	available, version := d.checkCRDGroup(ctx, "datascience.opendatahub.io")
	if !available {
		return "", fmt.Errorf("OpenShift AI is not available")
	}
	return version, nil
}

// WaitForAvailability waits for OpenShift AI to become available
func (d *Detector) WaitForAvailability(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for OpenShift AI to become available")
		case <-ticker.C:
			if d.IsOpenShiftAICluster(ctx) {
				klog.V(2).InfoS("OpenShift AI is now available")
				return nil
			}
		}
	}
}
