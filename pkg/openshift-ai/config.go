package openshiftai

import (
	"time"

	"github.com/containers/kubernetes-mcp-server/pkg/config"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

// OpenShiftAIConfig holds OpenShift AI specific configuration
type OpenShiftAIConfig struct {
	// Enable OpenShift AI toolset
	Enabled bool `toml:"enabled"`

	// Timeout for API operations
	Timeout time.Duration `toml:"timeout"`

	// Enable debug logging
	Debug bool `toml:"debug"`

	// Default namespace for operations
	DefaultNamespace string `toml:"default_namespace"`

	// Component-specific configuration
	DataScienceProjects DataScienceProjectsConfig `toml:"datascience_projects"`
	JupyterNotebooks    JupyterNotebooksConfig    `toml:"jupyter_notebooks"`
	ModelServing        ModelServingConfig        `toml:"model_serving"`
	Pipelines           PipelinesConfig           `toml:"pipelines"`
	GPUMonitoring       GPUMonitoringConfig       `toml:"gpu_monitoring"`
}

// DataScienceProjectsConfig holds configuration for Data Science Projects
type DataScienceProjectsConfig struct {
	Enabled bool `toml:"enabled"`

	// Auto-create namespaces if they don't exist
	AutoCreateNamespaces bool `toml:"auto_create_namespaces"`

	// Default project settings
	DefaultDisplayName string `toml:"default_display_name"`
	DefaultDescription string `toml:"default_description"`
}

// JupyterNotebooksConfig holds configuration for Jupyter Notebooks
type JupyterNotebooksConfig struct {
	Enabled bool `toml:"enabled"`

	// Default notebook image
	DefaultImage string `toml:"default_image"`

	// Default resource requests
	DefaultCPURequest    string `toml:"default_cpu_request"`
	DefaultMemoryRequest string `toml:"default_memory_request"`

	// Default resource limits
	DefaultCPULimit    string `toml:"default_cpu_limit"`
	DefaultMemoryLimit string `toml:"default_memory_limit"`

	// Auto-stop idle notebooks
	AutoStopIdle bool          `toml:"auto_stop_idle"`
	IdleTimeout  time.Duration `toml:"idle_timeout"`
}

// ModelServingConfig holds configuration for Model Serving
type ModelServingConfig struct {
	Enabled bool `toml:"enabled"`

	// Default runtime for model serving
	DefaultRuntime string `toml:"default_runtime"`

	// Auto-scaling configuration
	EnableAutoScaling bool `toml:"enable_auto_scaling"`
	MinReplicas       int  `toml:"min_replicas"`
	MaxReplicas       int  `toml:"max_replicas"`

	// Resource defaults
	DefaultCPURequest    string `toml:"default_cpu_request"`
	DefaultMemoryRequest string `toml:"default_memory_request"`
}

// PipelinesConfig holds configuration for AI Pipelines
type PipelinesConfig struct {
	Enabled bool `toml:"enabled"`

	// Default service account for pipelines
	DefaultServiceAccount string `toml:"default_service_account"`

	// Pipeline timeout
	DefaultTimeout time.Duration `toml:"default_timeout"`

	// Enable pipeline artifacts
	EnableArtifacts bool `toml:"enable_artifacts"`

	// Artifact storage configuration
	ArtifactStorage ArtifactStorageConfig `toml:"artifact_storage"`
}

// ArtifactStorageConfig holds configuration for pipeline artifact storage
type ArtifactStorageConfig struct {
	Type     string `toml:"type"` // "s3", "gcs", "azure", "pvc"
	Bucket   string `toml:"bucket"`
	Endpoint string `toml:"endpoint"`
	Path     string `toml:"path"`
}

// GPUMonitoringConfig holds configuration for GPU monitoring
type GPUMonitoringConfig struct {
	Enabled bool `toml:"enabled"`

	// GPU vendors to monitor
	Vendors []string `toml:"vendors"`

	// Metrics collection interval
	MetricsInterval time.Duration `toml:"metrics_interval"`

	// Enable detailed GPU metrics
	DetailedMetrics bool `toml:"detailed_metrics"`
}

// DefaultOpenShiftAIConfig returns default configuration for OpenShift AI
func DefaultOpenShiftAIConfig() *OpenShiftAIConfig {
	return &OpenShiftAIConfig{
		Enabled:          true,
		Timeout:          30 * time.Second,
		Debug:            false,
		DefaultNamespace: "default",
		DataScienceProjects: DataScienceProjectsConfig{
			Enabled:              true,
			AutoCreateNamespaces: false,
			DefaultDisplayName:   "",
			DefaultDescription:   "",
		},
		JupyterNotebooks: JupyterNotebooksConfig{
			Enabled:              true,
			DefaultImage:         "quay.io/opendatahub/workbench-notebook:latest",
			DefaultCPURequest:    "100m",
			DefaultMemoryRequest: "1Gi",
			DefaultCPULimit:      "2",
			DefaultMemoryLimit:   "4Gi",
			AutoStopIdle:         false,
			IdleTimeout:          30 * time.Minute,
		},
		ModelServing: ModelServingConfig{
			Enabled:              true,
			DefaultRuntime:       "kserve",
			EnableAutoScaling:    false,
			MinReplicas:          1,
			MaxReplicas:          3,
			DefaultCPURequest:    "100m",
			DefaultMemoryRequest: "512Mi",
		},
		Pipelines: PipelinesConfig{
			Enabled:               true,
			DefaultServiceAccount: "pipeline",
			DefaultTimeout:        60 * time.Minute,
			EnableArtifacts:       true,
			ArtifactStorage: ArtifactStorageConfig{
				Type:     "pvc",
				Bucket:   "",
				Endpoint: "",
				Path:     "/mnt/artifacts",
			},
		},
		GPUMonitoring: GPUMonitoringConfig{
			Enabled:         true,
			Vendors:         []string{"nvidia", "amd", "intel"},
			MetricsInterval: 30 * time.Second,
			DetailedMetrics: false,
		},
	}
}

// ConfigManager manages OpenShift AI configuration
type ConfigManager struct {
	config *OpenShiftAIConfig
}

// NewConfigManager creates a new configuration manager
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		config: DefaultOpenShiftAIConfig(),
	}
}

// LoadFromStatic loads configuration from static config
func (cm *ConfigManager) LoadFromStatic(staticConfig *config.StaticConfig) error {
	if staticConfig.Toolsets == nil {
		klog.V(2).InfoS("No toolsets configuration found, using defaults")
		return nil
	}

	// Look for openshift-ai toolset configuration
	for _, toolsetName := range staticConfig.Toolsets {
		if toolsetName == "openshift-ai" {
			klog.V(1).InfoS("Loading OpenShift AI configuration from static config")
			// In a real implementation, we would parse the toolset configuration
			// For now, we'll use defaults
			return nil
		}
	}

	klog.V(2).InfoS("OpenShift AI toolset configuration not found, using defaults")
	return nil
}

// LoadFromKubeConfig loads configuration from kubeconfig
func (cm *ConfigManager) LoadFromKubeConfig(kubeConfig *rest.Config) error {
	// Load configuration from kubeconfig context or cluster info
	// This is a placeholder for future implementation
	klog.V(2).InfoS("Loading OpenShift AI configuration from kubeconfig")
	return nil
}

// GetConfig returns the current configuration
func (cm *ConfigManager) GetConfig() *OpenShiftAIConfig {
	return cm.config
}

// UpdateConfig updates the configuration
func (cm *ConfigManager) UpdateConfig(newConfig *OpenShiftAIConfig) error {
	if newConfig == nil {
		return InvalidArgumentError("config cannot be nil")
	}

	cm.config = newConfig
	klog.V(1).InfoS("OpenShift AI configuration updated")
	return nil
}

// Validate validates the current configuration
func (cm *ConfigManager) Validate() error {
	cfg := cm.config

	if cfg.Timeout <= 0 {
		return InvalidArgumentError("timeout must be positive")
	}

	if cfg.DefaultNamespace == "" {
		return InvalidArgumentError("default_namespace cannot be empty")
	}

	if cfg.JupyterNotebooks.DefaultImage == "" {
		return InvalidArgumentError("default_image cannot be empty")
	}

	if cfg.ModelServing.DefaultRuntime == "" {
		return InvalidArgumentError("default_runtime cannot be empty")
	}

	if cfg.Pipelines.DefaultServiceAccount == "" {
		return InvalidArgumentError("default_service_account cannot be empty")
	}

	if len(cfg.GPUMonitoring.Vendors) == 0 {
		return InvalidArgumentError("at least one GPU vendor must be specified")
	}

	klog.V(1).InfoS("OpenShift AI configuration validation passed")
	return nil
}

// IsEnabled checks if OpenShift AI is enabled
func (cm *ConfigManager) IsEnabled() bool {
	return cm.config.Enabled
}

// IsComponentEnabled checks if a specific component is enabled
func (cm *ConfigManager) IsComponentEnabled(component string) bool {
	switch component {
	case "datascience_projects":
		return cm.config.DataScienceProjects.Enabled
	case "jupyter_notebooks":
		return cm.config.JupyterNotebooks.Enabled
	case "model_serving":
		return cm.config.ModelServing.Enabled
	case "pipelines":
		return cm.config.Pipelines.Enabled
	case "gpu_monitoring":
		return cm.config.GPUMonitoring.Enabled
	default:
		return false
	}
}

// GetTimeout returns the configured timeout
func (cm *ConfigManager) GetTimeout() time.Duration {
	return cm.config.Timeout
}

// GetDefaultNamespace returns the configured default namespace
func (cm *ConfigManager) GetDefaultNamespace() string {
	return cm.config.DefaultNamespace
}

// GetDebugMode returns whether debug mode is enabled
func (cm *ConfigManager) GetDebugMode() bool {
	return cm.config.Debug
}

// ApplyDefaults applies default values to the configuration
func (cm *ConfigManager) ApplyDefaults() {
	defaults := DefaultOpenShiftAIConfig()

	if cm.config.Timeout == 0 {
		cm.config.Timeout = defaults.Timeout
	}

	if cm.config.DefaultNamespace == "" {
		cm.config.DefaultNamespace = defaults.DefaultNamespace
	}

	if cm.config.JupyterNotebooks.DefaultImage == "" {
		cm.config.JupyterNotebooks.DefaultImage = defaults.JupyterNotebooks.DefaultImage
	}

	if cm.config.ModelServing.DefaultRuntime == "" {
		cm.config.ModelServing.DefaultRuntime = defaults.ModelServing.DefaultRuntime
	}

	if cm.config.Pipelines.DefaultServiceAccount == "" {
		cm.config.Pipelines.DefaultServiceAccount = defaults.Pipelines.DefaultServiceAccount
	}

	if len(cm.config.GPUMonitoring.Vendors) == 0 {
		cm.config.GPUMonitoring.Vendors = defaults.GPUMonitoring.Vendors
	}

	klog.V(2).InfoS("Applied default configuration values")
}
