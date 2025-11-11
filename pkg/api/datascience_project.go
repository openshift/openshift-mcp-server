package api

import (
	"github.com/google/jsonschema-go/jsonschema"
	"k8s.io/utils/ptr"
)

// DataScienceProjectListRequest represents a request to list DataScienceProjects
type DataScienceProjectListRequest struct {
	// Namespace to filter projects (optional, defaults to all namespaces)
	Namespace *string `json:"namespace,omitempty"`
}

// DataScienceProjectGetRequest represents a request to get a specific DataScienceProject
type DataScienceProjectGetRequest struct {
	// Name of the DataScienceProject
	Name string `json:"name"`
	// Namespace of the DataScienceProject
	Namespace string `json:"namespace"`
}

// DataScienceProjectCreateRequest represents a request to create a DataScienceProject
type DataScienceProjectCreateRequest struct {
	// Name of the DataScienceProject
	Name string `json:"name"`
	// Namespace where to create the DataScienceProject
	Namespace string `json:"namespace"`
	// Display name for the project (optional)
	DisplayName *string `json:"display_name,omitempty"`
	// Description for the project (optional)
	Description *string `json:"description,omitempty"`
	// Labels to apply to the project (optional)
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations to apply to the project (optional)
	Annotations map[string]string `json:"annotations,omitempty"`
}

// DataScienceProjectDeleteRequest represents a request to delete a DataScienceProject
type DataScienceProjectDeleteRequest struct {
	// Name of the DataScienceProject
	Name string `json:"name"`
	// Namespace of the DataScienceProject
	Namespace string `json:"namespace"`
}

// DataScienceProject represents a DataScienceProject resource
type DataScienceProject struct {
	// Name of the DataScienceProject
	Name string `json:"name"`
	// Namespace of the DataScienceProject
	Namespace string `json:"namespace"`
	// Display name of the project
	DisplayName *string `json:"display_name,omitempty"`
	// Description of the project
	Description *string `json:"description,omitempty"`
	// Labels applied to the project
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations applied to the project
	Annotations map[string]string `json:"annotations,omitempty"`
	// Creation timestamp
	CreatedAt *string `json:"created_at,omitempty"`
	// Status of the project
	Status DataScienceProjectStatus `json:"status"`
}

// DataScienceProjectStatus represents the status of a DataScienceProject
type DataScienceProjectStatus struct {
	// Current phase of the project
	Phase string `json:"phase"`
	// Status message
	Message *string `json:"message,omitempty"`
	// Conditions describing the project status
	Conditions []DataScienceProjectCondition `json:"conditions,omitempty"`
}

// DataScienceProjectCondition represents a condition of a DataScienceProject
type DataScienceProjectCondition struct {
	// Type of the condition
	Type string `json:"type"`
	// Status of the condition (True, False, Unknown)
	Status string `json:"status"`
	// Reason for the condition's last transition
	Reason *string `json:"reason,omitempty"`
	// Human-readable message indicating details about the transition
	Message *string `json:"message,omitempty"`
	// Last time the condition transitioned from one status to another
	LastTransitionTime *string `json:"last_transition_time,omitempty"`
}

// GetDataScienceProjectListTool returns the tool definition for listing DataScienceProjects
func GetDataScienceProjectListTool() Tool {
	return Tool{
		Name:        "datascience_projects_list",
		Description: "List all Data Science Projects in the current OpenShift AI cluster",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"namespace": {
					Type:        "string",
					Description: "The namespace to search for Data Science Projects (optional, defaults to all namespaces)",
				},
			},
		},
		Annotations: ToolAnnotations{
			Title:           "Data Science Projects: List",
			ReadOnlyHint:    ptr.To(true),
			DestructiveHint: ptr.To(false),
			IdempotentHint:  ptr.To(false),
			OpenWorldHint:   ptr.To(true),
		},
	}
}

// GetDataScienceProjectGetTool returns the tool definition for getting a specific DataScienceProject
func GetDataScienceProjectGetTool() Tool {
	return Tool{
		Name:        "datascience_project_get",
		Description: "Get details of a specific Data Science Project",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"name": {
					Type:        "string",
					Description: "The name of the Data Science Project",
				},
				"namespace": {
					Type:        "string",
					Description: "The namespace of the Data Science Project",
				},
			},
			Required: []string{"name", "namespace"},
		},
		Annotations: ToolAnnotations{
			Title:           "Data Science Project: Get",
			ReadOnlyHint:    ptr.To(true),
			DestructiveHint: ptr.To(false),
			IdempotentHint:  ptr.To(false),
			OpenWorldHint:   ptr.To(false),
		},
	}
}

// GetDataScienceProjectCreateTool returns the tool definition for creating a DataScienceProject
func GetDataScienceProjectCreateTool() Tool {
	return Tool{
		Name:        "datascience_project_create",
		Description: "Create a new Data Science Project",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"name": {
					Type:        "string",
					Description: "The name of the Data Science Project",
				},
				"namespace": {
					Type:        "string",
					Description: "The namespace where to create the Data Science Project",
				},
				"display_name": {
					Type:        "string",
					Description: "A display name for the Data Science Project (optional)",
				},
				"description": {
					Type:        "string",
					Description: "A description for the Data Science Project (optional)",
				},
				"labels": {
					Type:        "object",
					Description: "Labels to apply to the Data Science Project (optional)",
					AdditionalProperties: &jsonschema.Schema{
						Type: "string",
					},
				},
				"annotations": {
					Type:        "object",
					Description: "Annotations to apply to the Data Science Project (optional)",
					AdditionalProperties: &jsonschema.Schema{
						Type: "string",
					},
				},
			},
			Required: []string{"name", "namespace"},
		},
		Annotations: ToolAnnotations{
			Title:           "Data Science Project: Create",
			ReadOnlyHint:    ptr.To(false),
			DestructiveHint: ptr.To(false),
			IdempotentHint:  ptr.To(false),
			OpenWorldHint:   ptr.To(false),
		},
	}
}

// GetDataScienceProjectDeleteTool returns the tool definition for deleting a DataScienceProject
func GetDataScienceProjectDeleteTool() Tool {
	return Tool{
		Name:        "datascience_project_delete",
		Description: "Delete a Data Science Project",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"name": {
					Type:        "string",
					Description: "The name of the Data Science Project",
				},
				"namespace": {
					Type:        "string",
					Description: "The namespace of the Data Science Project",
				},
			},
			Required: []string{"name", "namespace"},
		},
		Annotations: ToolAnnotations{
			Title:           "Data Science Project: Delete",
			ReadOnlyHint:    ptr.To(false),
			DestructiveHint: ptr.To(true),
			IdempotentHint:  ptr.To(false),
			OpenWorldHint:   ptr.To(false),
		},
	}
}

// Model represents a machine learning model in OpenShift AI
type Model struct {
	// Name of the model
	Name string `json:"name"`
	// Namespace of the model
	Namespace string `json:"namespace"`
	// Display name of the model
	DisplayName *string `json:"display_name,omitempty"`
	// Description of the model
	Description *string `json:"description,omitempty"`
	// Model type (e.g., "pytorch", "tensorflow", "sklearn")
	ModelType *string `json:"model_type,omitempty"`
	// Model framework version
	FrameworkVersion *string `json:"framework_version,omitempty"`
	// Model format (e.g., "pickle", "onnx", "savedmodel")
	Format *string `json:"format,omitempty"`
	// Model size in bytes
	Size *int64 `json:"size,omitempty"`
	// Model version
	Version *string `json:"version,omitempty"`
	// Creation timestamp
	CreatedAt *string `json:"created_at,omitempty"`
	// Last updated timestamp
	UpdatedAt *string `json:"updated_at,omitempty"`
	// Model status
	Status ModelStatus `json:"status"`
	// Labels applied to the model
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations applied to the model
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ModelStatus represents the status of a model
type ModelStatus struct {
	// Current phase of the model
	Phase string `json:"phase"`
	// Status message
	Message *string `json:"message,omitempty"`
	// Ready state
	Ready bool `json:"ready"`
	// Deployment status
	DeploymentStatus *string `json:"deployment_status,omitempty"`
}

// ModelListRequest represents a request to list models
type ModelListRequest struct {
	// Namespace to filter models (optional, defaults to all namespaces)
	Namespace *string `json:"namespace,omitempty"`
	// Model type filter (optional)
	ModelType *string `json:"model_type,omitempty"`
	// Status filter (optional)
	Status *string `json:"status,omitempty"`
}

// ModelGetRequest represents a request to get a specific model
type ModelGetRequest struct {
	// Name of the model
	Name string `json:"name"`
	// Namespace of the model
	Namespace string `json:"namespace"`
}

// ModelCreateRequest represents a request to create a model
type ModelCreateRequest struct {
	// Name of the model
	Name string `json:"name"`
	// Namespace where to create the model
	Namespace string `json:"namespace"`
	// Display name for the model (optional)
	DisplayName *string `json:"display_name,omitempty"`
	// Description for the model (optional)
	Description *string `json:"description,omitempty"`
	// Model type (e.g., "pytorch", "tensorflow", "sklearn")
	ModelType string `json:"model_type"`
	// Model framework version (optional)
	FrameworkVersion *string `json:"framework_version,omitempty"`
	// Model format (e.g., "pickle", "onnx", "savedmodel")
	Format string `json:"format"`
	// Model version (optional)
	Version *string `json:"version,omitempty"`
	// Labels to apply to the model (optional)
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations to apply to the model (optional)
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ModelUpdateRequest represents a request to update a model
type ModelUpdateRequest struct {
	// Name of the model
	Name string `json:"name"`
	// Namespace of the model
	Namespace string `json:"namespace"`
	// Display name for the model (optional)
	DisplayName *string `json:"display_name,omitempty"`
	// Description for the model (optional)
	Description *string `json:"description,omitempty"`
	// Labels to apply to the model (optional)
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations to apply to the model (optional)
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ModelDeleteRequest represents a request to delete a model
type ModelDeleteRequest struct {
	// Name of the model
	Name string `json:"name"`
	// Namespace of the model
	Namespace string `json:"namespace"`
}

// GetModelListTool returns the tool definition for listing models
func GetModelListTool() Tool {
	return Tool{
		Name:        "models_list",
		Description: "List all machine learning models in the current OpenShift AI cluster",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"namespace": {
					Type:        "string",
					Description: "The namespace to search for models (optional, defaults to all namespaces)",
				},
				"model_type": {
					Type:        "string",
					Description: "Filter models by type (e.g., pytorch, tensorflow, sklearn)",
				},
				"status": {
					Type:        "string",
					Description: "Filter models by status (e.g., Ready, Pending, Failed)",
				},
			},
		},
		Annotations: ToolAnnotations{
			Title:           "Models: List",
			ReadOnlyHint:    ptr.To(true),
			DestructiveHint: ptr.To(false),
			IdempotentHint:  ptr.To(false),
			OpenWorldHint:   ptr.To(true),
		},
	}
}

// GetModelGetTool returns the tool definition for getting a specific model
func GetModelGetTool() Tool {
	return Tool{
		Name:        "model_get",
		Description: "Get details of a specific machine learning model",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"name": {
					Type:        "string",
					Description: "The name of the model",
				},
				"namespace": {
					Type:        "string",
					Description: "The namespace of the model",
				},
			},
			Required: []string{"name", "namespace"},
		},
		Annotations: ToolAnnotations{
			Title:           "Model: Get",
			ReadOnlyHint:    ptr.To(true),
			DestructiveHint: ptr.To(false),
			IdempotentHint:  ptr.To(false),
			OpenWorldHint:   ptr.To(false),
		},
	}
}

// GetModelCreateTool returns the tool definition for creating a model
func GetModelCreateTool() Tool {
	return Tool{
		Name:        "model_create",
		Description: "Create a new machine learning model entry in OpenShift AI",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"name": {
					Type:        "string",
					Description: "The name of the model",
				},
				"namespace": {
					Type:        "string",
					Description: "The namespace where to create the model",
				},
				"display_name": {
					Type:        "string",
					Description: "A display name for the model (optional)",
				},
				"description": {
					Type:        "string",
					Description: "A description for the model (optional)",
				},
				"model_type": {
					Type:        "string",
					Description: "The model type (e.g., pytorch, tensorflow, sklearn)",
				},
				"framework_version": {
					Type:        "string",
					Description: "The framework version (optional)",
				},
				"format": {
					Type:        "string",
					Description: "The model format (e.g., pickle, onnx, savedmodel)",
				},
				"version": {
					Type:        "string",
					Description: "The model version (optional)",
				},
				"labels": {
					Type:        "object",
					Description: "Labels to apply to the model (optional)",
					AdditionalProperties: &jsonschema.Schema{
						Type: "string",
					},
				},
				"annotations": {
					Type:        "object",
					Description: "Annotations to apply to the model (optional)",
					AdditionalProperties: &jsonschema.Schema{
						Type: "string",
					},
				},
			},
			Required: []string{"name", "namespace", "model_type", "format"},
		},
		Annotations: ToolAnnotations{
			Title:           "Model: Create",
			ReadOnlyHint:    ptr.To(false),
			DestructiveHint: ptr.To(false),
			IdempotentHint:  ptr.To(false),
			OpenWorldHint:   ptr.To(false),
		},
	}
}

// GetModelUpdateTool returns the tool definition for updating a model
func GetModelUpdateTool() Tool {
	return Tool{
		Name:        "model_update",
		Description: "Update an existing machine learning model in OpenShift AI",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"name": {
					Type:        "string",
					Description: "The name of the model",
				},
				"namespace": {
					Type:        "string",
					Description: "The namespace of the model",
				},
				"display_name": {
					Type:        "string",
					Description: "A display name for the model (optional)",
				},
				"description": {
					Type:        "string",
					Description: "A description for the model (optional)",
				},
				"labels": {
					Type:        "object",
					Description: "Labels to apply to the model (optional)",
					AdditionalProperties: &jsonschema.Schema{
						Type: "string",
					},
				},
				"annotations": {
					Type:        "object",
					Description: "Annotations to apply to the model (optional)",
					AdditionalProperties: &jsonschema.Schema{
						Type: "string",
					},
				},
			},
			Required: []string{"name", "namespace"},
		},
		Annotations: ToolAnnotations{
			Title:           "Model: Update",
			ReadOnlyHint:    ptr.To(false),
			DestructiveHint: ptr.To(false),
			IdempotentHint:  ptr.To(false),
			OpenWorldHint:   ptr.To(false),
		},
	}
}

// GetModelDeleteTool returns the tool definition for deleting a model
func GetModelDeleteTool() Tool {
	return Tool{
		Name:        "model_delete",
		Description: "Delete a machine learning model from OpenShift AI",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"name": {
					Type:        "string",
					Description: "The name of the model",
				},
				"namespace": {
					Type:        "string",
					Description: "The namespace of the model",
				},
			},
			Required: []string{"name", "namespace"},
		},
		Annotations: ToolAnnotations{
			Title:           "Model: Delete",
			ReadOnlyHint:    ptr.To(false),
			DestructiveHint: ptr.To(true),
			IdempotentHint:  ptr.To(false),
			OpenWorldHint:   ptr.To(false),
		},
	}
}

// Experiment represents an OpenShift AI Experiment for ML experiment tracking
type Experiment struct {
	// Name of experiment
	Name string `json:"name"`
	// Namespace of experiment
	Namespace string `json:"namespace"`
	// Display name (optional)
	DisplayName *string `json:"display_name,omitempty"`
	// Description (optional)
	Description *string `json:"description,omitempty"`
	// Labels associated with experiment
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations associated with experiment
	Annotations map[string]string `json:"annotations,omitempty"`
	// Experiment status
	Status ExperimentStatus `json:"status"`
}

// ExperimentStatus represents the status of an experiment
type ExperimentStatus struct {
	// Phase of the experiment (e.g., "Created", "Running", "Completed", "Failed")
	Phase string `json:"phase"`
	// Optional message about experiment status
	Message *string `json:"message,omitempty"`
	// Whether the experiment is ready
	Ready bool `json:"ready"`
	// Number of runs in this experiment
	RunCount int `json:"run_count"`
	// Last time the experiment was updated
	LastUpdated *string `json:"last_updated,omitempty"`
}

// ExperimentListRequest represents a request to list experiments
type ExperimentListRequest struct {
	// Namespace to filter experiments (optional, defaults to all namespaces)
	Namespace *string `json:"namespace,omitempty"`
	// Filter by status (optional)
	Status *string `json:"status,omitempty"`
}

// ExperimentGetRequest represents a request to get a specific experiment
type ExperimentGetRequest struct {
	// Name of the experiment
	Name string `json:"name"`
	// Namespace of the experiment
	Namespace string `json:"namespace"`
}

// ExperimentCreateRequest represents a request to create an experiment
type ExperimentCreateRequest struct {
	// Name of the experiment
	Name string `json:"name"`
	// Namespace where to create the experiment
	Namespace string `json:"namespace"`
	// Display name for the experiment (optional)
	DisplayName *string `json:"display_name,omitempty"`
	// Description for the experiment (optional)
	Description *string `json:"description,omitempty"`
	// Labels to apply to the experiment (optional)
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations to apply to the experiment (optional)
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ExperimentDeleteRequest represents a request to delete an experiment
type ExperimentDeleteRequest struct {
	// Name of the experiment
	Name string `json:"name"`
	// Namespace of the experiment
	Namespace string `json:"namespace"`
}

// GetExperimentsListTool returns the tool definition for listing experiments
func GetExperimentsListTool() Tool {
	return Tool{
		Name:        "experiments_list",
		Description: "List all OpenShift AI machine learning experiments",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"namespace": {
					Type:        "string",
					Description: "Namespace to filter experiments (optional, defaults to all namespaces)",
				},
				"status": {
					Type:        "string",
					Description: "Filter by experiment status (optional)",
				},
			},
		},
		Annotations: ToolAnnotations{
			Title:           "Experiments: List",
			ReadOnlyHint:    ptr.To(true),
			DestructiveHint: ptr.To(false),
			IdempotentHint:  ptr.To(true),
			OpenWorldHint:   ptr.To(false),
		},
	}
}

// GetExperimentGetTool returns the tool definition for getting a specific experiment
func GetExperimentGetTool() Tool {
	return Tool{
		Name:        "experiment_get",
		Description: "Get a specific OpenShift AI machine learning experiment",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"name": {
					Type:        "string",
					Description: "The name of the experiment",
				},
				"namespace": {
					Type:        "string",
					Description: "The namespace of the experiment",
				},
			},
			Required: []string{"name", "namespace"},
		},
		Annotations: ToolAnnotations{
			Title:           "Experiment: Get",
			ReadOnlyHint:    ptr.To(true),
			DestructiveHint: ptr.To(false),
			IdempotentHint:  ptr.To(false),
			OpenWorldHint:   ptr.To(false),
		},
	}
}

// GetExperimentCreateTool returns the tool definition for creating an experiment
func GetExperimentCreateTool() Tool {
	return Tool{
		Name:        "experiment_create",
		Description: "Create a new OpenShift AI machine learning experiment",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"name": {
					Type:        "string",
					Description: "The name of the experiment",
				},
				"namespace": {
					Type:        "string",
					Description: "The namespace where to create the experiment",
				},
				"display_name": {
					Type:        "string",
					Description: "Display name for the experiment (optional)",
				},
				"description": {
					Type:        "string",
					Description: "Description for the experiment (optional)",
				},
				"labels": {
					Type:        "object",
					Description: "Labels to apply to the experiment (optional)",
					AdditionalProperties: &jsonschema.Schema{
						Type: "string",
					},
				},
				"annotations": {
					Type:        "object",
					Description: "Annotations to apply to the experiment (optional)",
					AdditionalProperties: &jsonschema.Schema{
						Type: "string",
					},
				},
			},
			Required: []string{"name", "namespace"},
		},
		Annotations: ToolAnnotations{
			Title:           "Experiment: Create",
			ReadOnlyHint:    ptr.To(false),
			DestructiveHint: ptr.To(false),
			IdempotentHint:  ptr.To(false),
			OpenWorldHint:   ptr.To(false),
		},
	}
}

// GetExperimentDeleteTool returns the tool definition for deleting an experiment
func GetExperimentDeleteTool() Tool {
	return Tool{
		Name:        "experiment_delete",
		Description: "Delete an OpenShift AI machine learning experiment",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"name": {
					Type:        "string",
					Description: "The name of experiment",
				},
				"namespace": {
					Type:        "string",
					Description: "The namespace of the experiment",
				},
			},
			Required: []string{"name", "namespace"},
		},
		Annotations: ToolAnnotations{
			Title:           "Experiment: Delete",
			ReadOnlyHint:    ptr.To(false),
			DestructiveHint: ptr.To(true),
			IdempotentHint:  ptr.To(false),
			OpenWorldHint:   ptr.To(false),
		},
	}
}

// Application represents an OpenShift AI Application (e.g., Jupyter notebook)
type Application struct {
	// Name of application
	Name string `json:"name"`
	// Namespace of application
	Namespace string `json:"namespace"`
	// Display name (optional)
	DisplayName *string `json:"display_name,omitempty"`
	// Description (optional)
	Description *string `json:"description,omitempty"`
	// Labels associated with the application
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations associated with the application
	Annotations map[string]string `json:"annotations,omitempty"`
	// Application type (e.g., "Jupyter", "CodeServer")
	AppType string `json:"app_type,omitempty"`
	// Application status
	Status ApplicationStatus `json:"status"`
}

// ApplicationStatus represents the status of an application
type ApplicationStatus struct {
	// Phase of the application (e.g., "Creating", "Ready", "Failed", "Stopped")
	Phase string `json:"phase"`
	// Optional message about the application status
	Message *string `json:"message,omitempty"`
	// Whether the application is ready
	Ready bool `json:"ready"`
	// Application type (e.g., "Jupyter", "CodeServer")
	AppType *string `json:"app_type,omitempty"`
	// URL to access the application
	URL *string `json:"url,omitempty"`
	// Last time the application was updated
	LastUpdated *string `json:"last_updated,omitempty"`
}

// ApplicationListRequest represents a request to list applications
type ApplicationListRequest struct {
	// Namespace to filter applications (optional, defaults to all namespaces)
	Namespace *string `json:"namespace,omitempty"`
	// Filter by status (optional)
	Status *string `json:"status,omitempty"`
	// Filter by application type (optional)
	AppType *string `json:"app_type,omitempty"`
}

// ApplicationGetRequest represents a request to get a specific application
type ApplicationGetRequest struct {
	// Name of the application
	Name string `json:"name"`
	// Namespace of the application
	Namespace string `json:"namespace"`
}

// ApplicationCreateRequest represents a request to create an application
type ApplicationCreateRequest struct {
	// Name of the application
	Name string `json:"name"`
	// Namespace where to create the application
	Namespace string `json:"namespace"`
	// Display name for the application (optional)
	DisplayName *string `json:"display_name,omitempty"`
	// Description for the application (optional)
	Description *string `json:"description,omitempty"`
	// Application type (e.g., "Jupyter", "CodeServer")
	AppType string `json:"app_type"`
	// Labels to apply to the application (optional)
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations to apply to the application (optional)
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ApplicationDeleteRequest represents a request to delete an application
type ApplicationDeleteRequest struct {
	// Name of the application
	Name string `json:"name"`
	// Namespace of the application
	Namespace string `json:"namespace"`
}

// GetApplicationsListTool returns the tool definition for listing applications
func GetApplicationsListTool() Tool {
	return Tool{
		Name:        "applications_list",
		Description: "List all OpenShift AI applications (Jupyter notebooks, code servers, etc.)",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"namespace": {
					Type:        "string",
					Description: "Namespace to filter applications (optional, defaults to all namespaces)",
				},
				"status": {
					Type:        "string",
					Description: "Filter by application status (optional)",
				},
				"app_type": {
					Type:        "string",
					Description: "Filter by application type (optional, e.g., 'Jupyter', 'CodeServer')",
				},
			},
		},
		Annotations: ToolAnnotations{
			Title:           "Applications: List",
			ReadOnlyHint:    ptr.To(true),
			DestructiveHint: ptr.To(false),
			IdempotentHint:  ptr.To(true),
			OpenWorldHint:   ptr.To(false),
		},
	}
}

// GetApplicationGetTool returns the tool definition for getting a specific application
func GetApplicationGetTool() Tool {
	return Tool{
		Name:        "application_get",
		Description: "Get a specific OpenShift AI application",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"name": {
					Type:        "string",
					Description: "The name of the application",
				},
				"namespace": {
					Type:        "string",
					Description: "The namespace of the application",
				},
			},
			Required: []string{"name", "namespace"},
		},
		Annotations: ToolAnnotations{
			Title:           "Application: Get",
			ReadOnlyHint:    ptr.To(true),
			DestructiveHint: ptr.To(false),
			IdempotentHint:  ptr.To(false),
			OpenWorldHint:   ptr.To(false),
		},
	}
}

// GetApplicationCreateTool returns the tool definition for creating an application
func GetApplicationCreateTool() Tool {
	return Tool{
		Name:        "application_create",
		Description: "Create a new OpenShift AI application (Jupyter notebook, code server, etc.)",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"name": {
					Type:        "string",
					Description: "The name of the application",
				},
				"namespace": {
					Type:        "string",
					Description: "The namespace where to create the application",
				},
				"display_name": {
					Type:        "string",
					Description: "Display name for the application (optional)",
				},
				"description": {
					Type:        "string",
					Description: "Description for the application (optional)",
				},
				"app_type": {
					Type:        "string",
					Description: "Application type (e.g., 'Jupyter', 'CodeServer')",
				},
				"labels": {
					Type:        "object",
					Description: "Labels to apply to the application (optional)",
					AdditionalProperties: &jsonschema.Schema{
						Type: "string",
					},
				},
				"annotations": {
					Type:        "object",
					Description: "Annotations to apply to the application (optional)",
					AdditionalProperties: &jsonschema.Schema{
						Type: "string",
					},
				},
			},
			Required: []string{"name", "namespace", "app_type"},
		},
		Annotations: ToolAnnotations{
			Title:           "Application: Create",
			ReadOnlyHint:    ptr.To(false),
			DestructiveHint: ptr.To(false),
			IdempotentHint:  ptr.To(false),
			OpenWorldHint:   ptr.To(false),
		},
	}
}

// GetApplicationDeleteTool returns the tool definition for deleting an application
func GetApplicationDeleteTool() Tool {
	return Tool{
		Name:        "application_delete",
		Description: "Delete an OpenShift AI application",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"name": {
					Type:        "string",
					Description: "The name of the application",
				},
				"namespace": {
					Type:        "string",
					Description: "The namespace of the application",
				},
			},
			Required: []string{"name", "namespace"},
		},
		Annotations: ToolAnnotations{
			Title:           "Application: Delete",
			ReadOnlyHint:    ptr.To(false),
			DestructiveHint: ptr.To(true),
			IdempotentHint:  ptr.To(false),
			OpenWorldHint:   ptr.To(false),
		},
	}
}

// Pipeline represents an OpenShift AI Data Science Pipeline
type Pipeline struct {
	// Name of pipeline
	Name string `json:"name"`
	// Namespace of pipeline
	Namespace string `json:"namespace"`
	// Display name (optional)
	DisplayName *string `json:"display_name,omitempty"`
	// Description (optional)
	Description *string `json:"description,omitempty"`
	// Labels associated with pipeline
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations associated with pipeline
	Annotations map[string]string `json:"annotations,omitempty"`
	// Pipeline status
	Status PipelineStatus `json:"status"`
}

// PipelineStatus represents the status of a pipeline
type PipelineStatus struct {
	// Phase of pipeline (e.g., "Created", "Running", "Succeeded", "Failed")
	Phase string `json:"phase"`
	// Optional message about the pipeline status
	Message *string `json:"message,omitempty"`
	// Whether the pipeline is ready
	Ready bool `json:"ready"`
	// Number of runs in this pipeline
	RunCount int `json:"run_count"`
	// Last time the pipeline was updated
	LastUpdated *string `json:"last_updated,omitempty"`
}

// PipelineRun represents a run of a pipeline
type PipelineRun struct {
	// Name of pipeline run
	Name string `json:"name"`
	// Pipeline name that this run belongs to
	PipelineName string `json:"pipeline_name"`
	// Namespace of pipeline run
	Namespace string `json:"namespace"`
	// Display name (optional)
	DisplayName *string `json:"display_name,omitempty"`
	// Description (optional)
	Description *string `json:"description,omitempty"`
	// Labels associated with pipeline run
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations associated with pipeline run
	Annotations map[string]string `json:"annotations,omitempty"`
	// Pipeline run status
	Status PipelineRunStatus `json:"status"`
}

// PipelineRunStatus represents the status of a pipeline run
type PipelineRunStatus struct {
	// Phase of pipeline run (e.g., "Created", "Running", "Succeeded", "Failed")
	Phase string `json:"phase"`
	// Optional message about the pipeline run status
	Message *string `json:"message,omitempty"`
	// Whether the pipeline run is ready
	Ready bool `json:"ready"`
	// Start time of the pipeline run
	StartedAt *string `json:"started_at,omitempty"`
	// End time of the pipeline run
	FinishedAt *string `json:"finished_at,omitempty"`
	// Last time the pipeline run was updated
	LastUpdated *string `json:"last_updated,omitempty"`
}

// PipelineListRequest represents a request to list pipelines
type PipelineListRequest struct {
	// Namespace to filter pipelines (optional, defaults to all namespaces)
	Namespace *string `json:"namespace,omitempty"`
	// Filter by status (optional)
	Status *string `json:"status,omitempty"`
}

// PipelineGetRequest represents a request to get a specific pipeline
type PipelineGetRequest struct {
	// Name of the pipeline
	Name string `json:"name"`
	// Namespace of the pipeline
	Namespace string `json:"namespace"`
}

// PipelineCreateRequest represents a request to create a pipeline
type PipelineCreateRequest struct {
	// Name of the pipeline
	Name string `json:"name"`
	// Namespace where to create the pipeline
	Namespace string `json:"namespace"`
	// Display name for the pipeline (optional)
	DisplayName *string `json:"display_name,omitempty"`
	// Description for the pipeline (optional)
	Description *string `json:"description,omitempty"`
	// Labels to apply to the pipeline (optional)
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations to apply to the pipeline (optional)
	Annotations map[string]string `json:"annotations,omitempty"`
}

// PipelineDeleteRequest represents a request to delete a pipeline
type PipelineDeleteRequest struct {
	// Name of the pipeline
	Name string `json:"name"`
	// Namespace of the pipeline
	Namespace string `json:"namespace"`
}

// PipelineRunListRequest represents a request to list pipeline runs
type PipelineRunListRequest struct {
	// Namespace to filter pipeline runs (optional, defaults to all namespaces)
	Namespace *string `json:"namespace,omitempty"`
	// Filter by pipeline name (optional)
	PipelineName *string `json:"pipeline_name,omitempty"`
	// Filter by status (optional)
	Status *string `json:"status,omitempty"`
}

// PipelineRunGetRequest represents a request to get a specific pipeline run
type PipelineRunGetRequest struct {
	// Name of the pipeline run
	Name string `json:"name"`
	// Namespace of the pipeline run
	Namespace string `json:"namespace"`
}

// GetPipelinesListTool returns the tool definition for listing pipelines
func GetPipelinesListTool() Tool {
	return Tool{
		Name:        "pipelines_list",
		Description: "List all OpenShift AI data science pipelines",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"namespace": {
					Type:        "string",
					Description: "Namespace to filter pipelines (optional, defaults to all namespaces)",
				},
				"status": {
					Type:        "string",
					Description: "Filter by pipeline status (optional)",
				},
			},
		},
		Annotations: ToolAnnotations{
			Title:           "Pipelines: List",
			ReadOnlyHint:    ptr.To(true),
			DestructiveHint: ptr.To(false),
			IdempotentHint:  ptr.To(true),
			OpenWorldHint:   ptr.To(false),
		},
	}
}

// GetPipelineGetTool returns the tool definition for getting a specific pipeline
func GetPipelineGetTool() Tool {
	return Tool{
		Name:        "pipeline_get",
		Description: "Get a specific OpenShift AI data science pipeline",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"name": {
					Type:        "string",
					Description: "The name of the pipeline",
				},
				"namespace": {
					Type:        "string",
					Description: "The namespace of the pipeline",
				},
			},
			Required: []string{"name", "namespace"},
		},
		Annotations: ToolAnnotations{
			Title:           "Pipeline: Get",
			ReadOnlyHint:    ptr.To(true),
			DestructiveHint: ptr.To(false),
			IdempotentHint:  ptr.To(false),
			OpenWorldHint:   ptr.To(false),
		},
	}
}

// GetPipelineCreateTool returns the tool definition for creating a pipeline
func GetPipelineCreateTool() Tool {
	return Tool{
		Name:        "pipeline_create",
		Description: "Create a new OpenShift AI data science pipeline",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"name": {
					Type:        "string",
					Description: "The name of the pipeline",
				},
				"namespace": {
					Type:        "string",
					Description: "The namespace where to create the pipeline",
				},
				"display_name": {
					Type:        "string",
					Description: "Display name for the pipeline (optional)",
				},
				"description": {
					Type:        "string",
					Description: "Description for the pipeline (optional)",
				},
				"labels": {
					Type:        "object",
					Description: "Labels to apply to the pipeline (optional)",
					AdditionalProperties: &jsonschema.Schema{
						Type: "string",
					},
				},
				"annotations": {
					Type:        "object",
					Description: "Annotations to apply to the pipeline (optional)",
					AdditionalProperties: &jsonschema.Schema{
						Type: "string",
					},
				},
			},
			Required: []string{"name", "namespace"},
		},
		Annotations: ToolAnnotations{
			Title:           "Pipeline: Create",
			ReadOnlyHint:    ptr.To(false),
			DestructiveHint: ptr.To(false),
			IdempotentHint:  ptr.To(false),
			OpenWorldHint:   ptr.To(false),
		},
	}
}

// GetPipelineDeleteTool returns the tool definition for deleting a pipeline
func GetPipelineDeleteTool() Tool {
	return Tool{
		Name:        "pipeline_delete",
		Description: "Delete an OpenShift AI data science pipeline",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"name": {
					Type:        "string",
					Description: "The name of the pipeline",
				},
				"namespace": {
					Type:        "string",
					Description: "The namespace of the pipeline",
				},
			},
			Required: []string{"name", "namespace"},
		},
		Annotations: ToolAnnotations{
			Title:           "Pipeline: Delete",
			ReadOnlyHint:    ptr.To(false),
			DestructiveHint: ptr.To(true),
			IdempotentHint:  ptr.To(false),
			OpenWorldHint:   ptr.To(false),
		},
	}
}

// GetPipelineRunsListTool returns the tool definition for listing pipeline runs
func GetPipelineRunsListTool() Tool {
	return Tool{
		Name:        "pipeline_runs_list",
		Description: "List all OpenShift AI data science pipeline runs",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"namespace": {
					Type:        "string",
					Description: "Namespace to filter pipeline runs (optional, defaults to all namespaces)",
				},
				"pipeline_name": {
					Type:        "string",
					Description: "Filter by pipeline name (optional)",
				},
				"status": {
					Type:        "string",
					Description: "Filter by pipeline run status (optional)",
				},
			},
		},
		Annotations: ToolAnnotations{
			Title:           "Pipeline Runs: List",
			ReadOnlyHint:    ptr.To(true),
			DestructiveHint: ptr.To(false),
			IdempotentHint:  ptr.To(true),
			OpenWorldHint:   ptr.To(false),
		},
	}
}

// GetPipelineRunGetTool returns the tool definition for getting a specific pipeline run
func GetPipelineRunGetTool() Tool {
	return Tool{
		Name:        "pipeline_run_get",
		Description: "Get a specific OpenShift AI data science pipeline run",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"name": {
					Type:        "string",
					Description: "The name of the pipeline run",
				},
				"namespace": {
					Type:        "string",
					Description: "The namespace of the pipeline run",
				},
			},
			Required: []string{"name", "namespace"},
		},
		Annotations: ToolAnnotations{
			Title:           "Pipeline Run: Get",
			ReadOnlyHint:    ptr.To(true),
			DestructiveHint: ptr.To(false),
			IdempotentHint:  ptr.To(false),
			OpenWorldHint:   ptr.To(false),
		},
	}
}
