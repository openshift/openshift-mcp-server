# 🚀 OpenShift AI MCP Server - Practical Usage Guide

## 📥 Installation

### **Option 1: Main Package (Recommended)**
```bash
npm install -g kubernetes-mcp-server-openshift-ai
```

### **Option 2: Platform-Specific**
```bash
# Linux AMD64
npm install -g kubernetes-mcp-server-openshift-ai-linux-amd64

# macOS ARM64 (Apple Silicon)  
npm install -g kubernetes-mcp-server-openshift-ai-darwin-arm64
```

### **Option 3: Direct Download**
```bash
curl -sSL https://raw.githubusercontent.com/macayaven/openshift-mcp-server/main/install-openshift-ai.sh | bash
```

## 🔧 Configuration

### **Basic Setup**
```bash
# Start with all toolsets (recommended)
kubernetes-mcp-server --toolsets core,config,helm,openshift-ai

# Start with specific toolsets
kubernetes-mcp-server --toolsets core,openshift-ai

# Check available toolsets
kubernetes-mcp-server --help
```

### **Kubernetes Configuration**
```bash
# Use specific kubeconfig
kubernetes-mcp-server --kubeconfig ~/.kube/config

# Use current context
kubernetes-mcp-server --toolsets openshift-ai

# Read-only mode (safe for production)
kubernetes-mcp-server --read-only --toolsets openshift-ai
```

## 🎯 Core Usage Scenarios

### **Scenario 1: Data Science Project Management**
```bash
# Start server with OpenShift AI tools
kubernetes-mcp-server --toolsets core,config,helm,openshift-ai

# Now in your AI assistant (Claude, Cursor, etc.), you can:
```

**Available Commands:**
- `create_datascience_project` - Create new DS project
- `list_datascience_projects` - List all projects  
- `get_datascience_project` - Get project details
- `update_datascience_project` - Modify existing project
- `delete_datascience_project` - Remove project

**Example Workflow:**
```
1. "Create a data science project called 'ml-experiments'"
2. "List all data science projects" 
3. "Get details of the ml-experiments project"
4. "Add a description to the ml-experiments project"
```

### **Scenario 2: Model Management**
```bash
# Start server (same as above)
kubernetes-mcp-server --toolsets core,openshift-ai

# Available Model Commands:
- `list_models` - List all models in project
- `get_model` - Get model details
- `create_model` - Deploy new model
- `update_model` - Update model configuration
- `delete_model` - Remove model
```

**Example Workflow:**
```
1. "List all models in the ml-experiments project"
2. "Create a new PyTorch model with GPU support"
3. "Update the model to use 2 GPU replicas"
4. "Get current status of the PyTorch model"
```

### **Scenario 3: Application Deployment**
```bash
# Start server
kubernetes-mcp-server --toolsets core,openshift-ai

# Application Commands:
- `deploy_application` - Deploy new application
- `list_applications` - List applications
- `get_application` - Get app details
- `delete_application` - Remove application
```

**Example Workflow:**
```
1. "Deploy a Streamlit application with 3 replicas"
2. "List all applications in the project"
3. "Get details of the Streamlit app"
4. "Scale the application to 5 replicas"
5. "Delete the application when done"
```

### **Scenario 4: Experiment Management**
```bash
# Start server
kubernetes-mcp-server --toolsets core,openshift-ai

# Experiment Commands:
- `run_experiment` - Execute new experiment
- `list_experiments` - List all experiments
- `get_experiment` - Get experiment details
- `delete_experiment` - Remove experiment
```

**Example Workflow:**
```
1. "Run a training experiment with a PyTorch model"
2. "List all experiments in the project"
3. "Get results and logs of the training experiment"
4. "Delete the experiment after analyzing results"
```

### **Scenario 5: Pipeline Management**
```bash
# Start server
kubernetes-mcp-server --toolsets core,openshift-ai

# Pipeline Commands:
- `run_pipeline` - Execute new pipeline
- `list_pipelines` - List all pipelines
- `get_pipeline` - Get pipeline details
- `create_pipeline` - Create new pipeline
- `delete_pipeline` - Remove pipeline
```

**Example Workflow:**
```
1. "Create a new ML pipeline for data preprocessing"
2. "Run the pipeline with the latest dataset"
3. "List all pipelines and their status"
4. "Get the execution logs of the preprocessing pipeline"
5. "Delete the pipeline after completion"
```

## 🛠️ Advanced Usage

### **Multi-Cluster Management**
```bash
# Work with multiple Kubernetes clusters
kubernetes-mcp-server --toolsets core,config,openshift-ai

# Switch between clusters using context tools
```

### **Helm Integration**
```bash
# Include Helm tools
kubernetes-mcp-server --toolsets core,helm,openshift-ai

# Helm Commands Available:
- `list_helm_releases`
- `get_helm_release`  
- `install_helm_chart`
- `upgrade_helm_release`
- `uninstall_helm_release`
```

### **Production Safety**
```bash
# Read-only mode (no destructive operations)
kubernetes-mcp-server --read-only --toolsets openshift-ai

# Disable destructive tools
kubernetes-mcp-server --disable-destructive --toolsets core,openshift-ai
```

## 🔍 Integration with AI Assistants

### **Claude Desktop Integration**
1. Install Claude Desktop
2. Go to Settings → MCP Servers
3. Add new server:
   - **Name**: OpenShift AI Server
   - **Command**: `kubernetes-mcp-server --toolsets core,config,helm,openshift-ai`
4. Enable the server
5. Start chatting with Claude about your OpenShift resources

### **Cursor Integration**
1. Open Cursor
2. Go to Settings → AI Tools
3. Add MCP Server:
   - **Name**: OpenShift AI
   - **Path**: `/usr/local/bin/kubernetes-mcp-server`
   - **Args**: `--toolsets core,config,helm,openshift-ai`
4. Restart Cursor

### **GitHub Copilot Integration**
1. Install GitHub Copilot extension
2. Configure MCP server in settings
3. Use same command as above

## 💡 Real-World Examples

### **Example 1: ML Project Setup**
```
You: "I want to create a new ML project for fraud detection"

AI: "I'll help you create a data science project for fraud detection.
[Creates project with appropriate resources]

Your fraud detection project is now ready! Here's what I created:
- Project: fraud-detection-ml
- Namespace: fraud-detection-ml
- Storage: 100Gi PVC
- Service Account: fraud-detection-sa

Next steps:
1. Deploy your training data
2. Set up model serving
3. Configure monitoring
Would you like me to help with any of these?"
```

### **Example 2: Model Deployment**
```
You: "Deploy a BERT model for text classification"

AI: "I'll deploy a BERT model for text classification in your project.
[Deploys model with appropriate resources]

BERT model deployed successfully! Details:
- Model: bert-text-classifier
- Framework: PyTorch
- Replicas: 2
- Memory: 4Gi each
- Endpoint: https://bert-text-classifier.apps.cluster.com

The model is ready for inference requests!"
```

### **Example 3: Batch Experiment**
```
You: "Run a hyperparameter tuning experiment"

AI: "I'll set up a hyperparameter tuning experiment for your model.
[Creates experiment with multiple parameter combinations]

Hyperparameter experiment started! Details:
- Experiment: bert-hp-tuning-v2
- Parameters: learning_rate, batch_size, epochs
- Trials: 27 combinations
- Max parallel: 3 trials

The experiment will run all combinations and report the best configuration."
```

## 🔧 Troubleshooting

### **Common Issues**
```bash
# Check if server is working
kubernetes-mcp-server --version

# Test specific toolset
kubernetes-mcp-server --toolsets openshift-ai --help

# Check connectivity
kubectl cluster-info

# Verify OpenShift AI access
oc get datascienceprojects
```

### **Debug Mode**
```bash
# Enable verbose logging
kubernetes-mcp-server --log-level 9 --toolsets openshift-ai

# Test with dry-run
kubernetes-mcp-server --toolsets core,openshift-ai --help
```

## 📚 Next Steps

### **Learning Resources**
- OpenShift AI Documentation: https://docs.redhat.com/en-us/openshift_ai/
- Kubernetes Documentation: https://kubernetes.io/docs/
- MCP Documentation: https://modelcontextprotocol.io/

### **Community**
- GitHub Repository: https://github.com/macayaven/openshift-mcp-server
- Issues: Report bugs or request features
- Discussions: Ask questions and share workflows

---

**🎉 You now have a complete OpenShift AI MCP server with 28 tools for full ML lifecycle management!**