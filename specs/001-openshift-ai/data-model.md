# Data Model: OpenShift AI Capabilities

**Purpose**: Define data entities and relationships for OpenShift AI implementation  
**Created**: 2025-10-28  
**Feature**: OpenShift AI Capabilities

## Core Entities

### DataScienceProject
**Description**: OpenShift AI project container with resource quotas and user permissions

**Fields**:
- `metadata.name` (string) - Unique project identifier
- `spec.displayName` (string) - Human-readable project name
- `spec.description` (string) - Project description and purpose
- `spec.resources.limits.cpu` (string) - CPU resource limit
- `spec.resources.limits.memory` (string) - Memory resource limit
- `spec.resources.limits.requests.nvidia.com/gpu` (string) - GPU resource limit
- `status.phase` (string) - Project lifecycle phase (Creating, Ready, Failed)
- `status.message` (string) - Status message for errors or information

**Validation Rules**:
- Name must be valid Kubernetes resource name
- CPU and memory limits must follow Kubernetes resource format
- GPU limits must be non-negative integers
- Display name required, description optional

**State Transitions**:
```
Creating → Ready (successful creation)
Creating → Failed (creation error)
Ready → Deleting (deletion initiated)
Deleting → Deleted (successful deletion)
```

### Notebook
**Description**: Jupyter notebook instance with compute resources and persistent storage

**Fields**:
- `metadata.name` (string) - Unique notebook identifier
- `metadata.namespace` (string) - Target namespace/project
- `spec.template.spec.containers[0].name` (string) - Container name (typically "notebook")
- `spec.template.spec.containers[0].image` (string) - Notebook container image
- `spec.template.spec.containers[0].resources.limits.cpu` (string) - CPU limit
- `spec.template.spec.containers[0].resources.limits.memory` (string) - Memory limit
- `spec.template.spec.containers[0].resources.limits.requests.nvidia.com/gpu` (string) - GPU allocation
- `spec.template.spec.containers[0].env` (array) - Environment variables
- `status.phase` (string) - Notebook lifecycle phase
- `status.conditions` (array) - Status conditions with reasons
- `status.url` (string) - Access URL when notebook is ready

**Validation Rules**:
- Name and namespace required
- Image must be valid container image reference
- Resource limits must follow Kubernetes format
- GPU allocation must be available in cluster
- Environment variables must have valid key-value pairs

**State Transitions**:
```
Creating → Pending → Ready (successful start)
Creating → Failed (creation error)
Ready → Stopping (stop initiated)
Stopping → Stopped (successful stop)
Stopped → Starting (start initiated)
Starting → Ready (successful start)
Ready → Deleting (deletion initiated)
```

### InferenceService
**Description**: KServe model serving deployment with scaling and versioning

**Fields**:
- `metadata.name` (string) - Unique service identifier
- `metadata.namespace` (string) - Target namespace/project
- `spec.predictor.model.modelFormat.name` (string) - Model framework (pytorch, tensorflow, etc.)
- `spec.predictor.model.protocol` (string) - Inference protocol (v1, v2, grpc-v1)
- `spec.predictor.model.storageUri` (string) - Model storage location
- `spec.predictor.model.resources.limits.cpu` (string) - CPU limit
- `spec.predictor.model.resources.limits.memory` (string) - Memory limit
- `spec.predictor.model.resources.limits.requests.nvidia.com/gpu` (string) - GPU allocation
- `spec.predictor.replicas` (integer) - Number of service replicas
- `status.url` (string) - Inference endpoint URL
- `status.conditions` (array) - Deployment status conditions
- `status.predictorStatus` (object) - Detailed predictor status

**Validation Rules**:
- Name and namespace required
- Model format must be supported framework
- Protocol must be valid KServe protocol
- Storage URI must be accessible
- Resource limits must follow Kubernetes format
- Replicas must be positive integer

**State Transitions**:
```
Creating → Ready (successful deployment)
Creating → Failed (deployment error)
Ready → Updating (model update initiated)
Updating → Ready (update successful)
Ready → Deleting (deletion initiated)
```

### PipelineRun
**Description**: OpenShift Pipelines execution with tasks, steps, and artifacts

**Fields**:
- `metadata.name` (string) - Unique pipeline run identifier
- `metadata.namespace` (string) - Target namespace/project
- `spec.pipelineRef.name` (string) - Reference to Pipeline definition
- `spec.params` (array) - Pipeline parameters
- `spec.workspaces` (array) - Pipeline workspace configurations
- `status.startTime` (timestamp) - Execution start time
- `status.completionTime` (timestamp) - Execution completion time
- `status.conditions` (array) - Execution status conditions
- `status.taskRuns` (array) - Task execution details
- `status.artifacts` (array) - Produced artifacts

**Validation Rules**:
- Name and namespace required
- Pipeline reference must exist
- Parameters must match Pipeline definition
- Workspaces must be properly configured

**State Transitions**:
```
Pending -> Running -> Succeeded (successful completion)
Pending -> Running -> Failed (execution error)
Running -> Cancelled (user cancellation)
Running -> PipelineRunStopping (graceful stop)
```

### GPU Resource
**Description**: Compute resource with GPU allocation and utilization metrics

**Fields**:
- `nodeName` (string) - Kubernetes node name
- `gpuType` (string) - GPU model (e.g., "NVIDIA A100", "NVIDIA V100")
- `totalGPUs` (integer) - Total GPU count on node
- `allocatedGPUs` (integer) - Currently allocated GPUs
- `availableGPUs` (integer) - Available GPUs for allocation
- `utilizationPercent` (float) - Current GPU utilization (0-100)
- `memoryUsedMB` (integer) - GPU memory usage in MB
- `memoryTotalMB` (integer) - Total GPU memory in MB
- `temperatureCelsius` (float) - Current GPU temperature
- `powerUsageWatts` (float) - Current power consumption

**Validation Rules**:
- Node name must be valid Kubernetes node
- GPU type must be recognized model
- All numeric values must be non-negative
- Utilization must be 0-100 range

## Entity Relationships

### Project Containment
```
DataScienceProject (1) ── contains ── (N) Notebook
DataScienceProject (1) ── contains ── (N) InferenceService
DataScienceProject (1) ── contains ── (N) PipelineRun
```

### Resource Allocation
```
Notebook (1) ── allocates ── (N) GPU Resource
InferenceService (1) ── allocates ── (N) GPU Resource
PipelineRun (1) ── may allocate ── (N) GPU Resource
```

### Namespace Mapping
```
DataScienceProject (1) ── maps to ── (1) Kubernetes Namespace
Notebook (1) ── deployed in ── (1) Namespace
InferenceService (1) ── deployed in ── (1) Namespace
PipelineRun (1) ── executes in ── (1) Namespace
```

## Data Access Patterns

### Read Operations
- List operations support filtering by namespace, labels, and status
- Get operations require name and optional namespace
- GPU monitoring supports node-level and cluster-level aggregation

### Write Operations
- Create operations validate resource availability before allocation
- Update operations support canary deployments for InferenceServices
- Delete operations include cleanup of dependent resources

### Monitoring Operations
- Status polling uses Kubernetes watch for real-time updates
- GPU metrics collected from Prometheus endpoints
- Pipeline logs streamed from container logs

## Error Handling

### Validation Errors
- Resource quota exceeded
- Invalid resource specifications
- Missing required fields
- Unsupported model formats

### Runtime Errors
- GPU allocation failures
- Network connectivity issues
- Container image pull failures
- Pipeline execution timeouts

### Recovery Patterns
- Automatic retry for transient failures
- Graceful degradation for missing OpenShift AI
- Resource cleanup on failed operations
- Status synchronization with cluster state