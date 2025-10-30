# Quickstart Guide: OpenShift AI Capabilities

**Purpose**: Get started with OpenShift AI tools in Kubernetes MCP Server  
**Created**: 2025-10-28  
**Feature**: OpenShift AI Capabilities

## Prerequisites

### Cluster Requirements
- OpenShift 4.10+ with OpenShift AI Operator installed
- GPU nodes with NVIDIA GPU Operator (for GPU workloads)
- Sufficient resource quotas for AI workloads
- Network access to model registries and storage

### User Permissions
- `admin` access to target namespaces
- Permission to create custom resources (DataScienceProject, Notebook, InferenceService)
- GPU resource allocation permissions (if using GPU workloads)

### MCP Server Setup
- Kubernetes MCP Server built with OpenShift AI toolsets enabled
- Proper kubeconfig configuration pointing to OpenShift AI cluster
- Authentication credentials with appropriate RBAC permissions

## Getting Started

### 1. Verify OpenShift AI Availability
```bash
# Check if OpenShift AI tools are available
npx kubernetes-mcp-server --toolsets openshift-ai,ai-resources
```

Expected response: Tools should be listed without errors about missing APIs.

### 2. Create Your First Data Science Project
```json
{
  "tool": "datascience_project_create",
  "arguments": {
    "name": "my-ai-project",
    "displayName": "My AI Project",
    "description": "Project for machine learning experiments",
    "resources": {
      "cpu": "4",
      "memory": "8Gi",
      "requests.nvidia.com/gpu": "1"
    }
  }
}
```

### 3. Launch a Jupyter Notebook
```json
{
  "tool": "jupyter_notebook_create",
  "arguments": {
    "name": "data-science-notebook",
    "namespace": "my-ai-project",
    "image": "jupyter/scipy-notebook:latest",
    "resources": {
      "cpu": "2",
      "memory": "4Gi",
      "requests.nvidia.com/gpu": "1"
    },
    "env": [
      {
        "name": "JUPYTER_ENABLE_LAB",
        "value": "yes"
      }
    ]
  }
}
```

### 4. Deploy a Model for Inference
```json
{
  "tool": "inference_service_create",
  "arguments": {
    "name": "my-model-service",
    "namespace": "my-ai-project",
    "modelFormat": "pytorch",
    "protocol": "v2",
    "storageUri": "s3://my-models/pytorch-model/",
    "resources": {
      "cpu": "2",
      "memory": "4Gi",
      "requests.nvidia.com/gpu": "1"
    },
    "replicas": 2
  }
}
```

### 5. Check GPU Resources
```json
{
  "tool": "gpu_resources_status",
  "arguments": {}
}
```

## Common Workflows

### Data Science Workflow
1. **Create Project**: Set up a Data Science Project with resource quotas
2. **Launch Notebook**: Start Jupyter notebook for model development
3. **Develop Models**: Use notebook to train and evaluate models
4. **Save Models**: Store trained models in accessible storage
5. **Deploy Model**: Create Inference Service for model serving
6. **Monitor**: Check GPU utilization and service performance

### MLOps Workflow
1. **Create Pipeline**: Define ML pipeline with training and evaluation steps
2. **Run Pipeline**: Execute pipeline with different parameters
3. **Monitor Progress**: Track pipeline execution and logs
4. **Access Artifacts**: Retrieve trained models and metrics
5. **Deploy Best Model**: Create Inference Service with best performing model
6. **Scale**: Adjust replicas based on demand

### GPU Resource Management
1. **Check Availability**: List GPU nodes and available resources
2. **Monitor Usage**: Track GPU utilization across workloads
3. **Optimize Allocation**: Adjust resource requests based on usage patterns
4. **Troubleshoot**: Identify GPU bottlenecks and resource conflicts

## Tool Reference

### Data Science Projects
- `datascience_projects_list` - List all projects
- `datascience_project_get` - Get project details
- `datascience_project_create` - Create new project
- `datascience_project_delete` - Delete project

### Jupyter Notebooks
- `jupyter_notebooks_list` - List notebooks
- `jupyter_notebook_get` - Get notebook details
- `jupyter_notebook_create` - Create notebook
- `jupyter_notebook_start` - Start notebook
- `jupyter_notebook_stop` - Stop notebook
- `jupyter_notebook_delete` - Delete notebook

### Model Serving
- `inference_services_list` - List inference services
- `inference_service_get` - Get service details
- `inference_service_create` - Create service
- `inference_service_update` - Update service
- `inference_service_delete` - Delete service
- `llm_inference_service_create` - Create LLM service

### Pipeline Management
- `pipelines_list` - List pipelines
- `pipeline_runs_list` - List pipeline runs
- `pipeline_run_get` - Get run details
- `pipeline_run_create` - Create pipeline run
- `pipeline_run_cancel` - Cancel run
- `pipeline_run_logs` - Get logs
- `pipeline_run_artifacts` - List artifacts

### GPU Resources
- `gpu_nodes_list` - List GPU nodes
- `gpu_resources_status` - Get resource status
- `gpu_workloads_list` - List GPU workloads
- `gpu_metrics` - Get detailed metrics
- `gpu_quotas_check` - Check quotas
- `gpu_device_info` - Get device info

## Best Practices

### Resource Management
- Start with small resource requests and scale up based on usage
- Use GPU resources only when necessary for cost optimization
- Monitor resource utilization to optimize allocation
- Set appropriate resource limits to prevent resource exhaustion

### Security
- Use namespaces to isolate different projects
- Implement proper RBAC for team collaboration
- Store sensitive data in secure storage locations
- Regularly review and clean up unused resources

### Performance
- Use appropriate container images for your workloads
- Optimize model formats for faster inference
- Implement canary deployments for model updates
- Monitor GPU utilization to identify bottlenecks

### Cost Optimization
- Delete unused notebooks and inference services
- Use spot instances for non-critical workloads
- Implement auto-scaling for inference services
- Regular cleanup of completed pipeline runs and artifacts

## Troubleshooting

### Common Issues

**OpenShift AI Not Available**
```
Error: OpenShift AI APIs not found in cluster
```
Solution: Install OpenShift AI Operator and ensure it's properly configured

**GPU Allocation Failed**
```
Error: Insufficient GPU resources
```
Solution: Check GPU availability with `gpu_resources_status` and adjust resource requests

**Notebook Fails to Start**
```
Error: Notebook container image pull failed
```
Solution: Verify image exists and is accessible from cluster

**Pipeline Run Stuck**
```
Error: PipelineRun timeout
```
Solution: Check resource quotas and pipeline configuration

### Getting Help
- Check MCP server logs for detailed error messages
- Use OpenShift console to verify resource status
- Consult OpenShift AI documentation for specific features
- Review GPU metrics for resource-related issues

## Next Steps

1. **Explore Examples**: Try the example workflows in this guide
2. **Customize Resources**: Adjust resource allocations for your specific use cases
3. **Integrate with CI/CD**: Automate model deployment pipelines
4. **Monitor Performance**: Set up monitoring for production workloads
5. **Scale Usage**: Expand to multiple projects and team collaboration

For more advanced usage and configuration options, refer to the complete tool documentation and API contracts.