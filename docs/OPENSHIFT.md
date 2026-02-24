# OpenShift Toolset

This toolset provides OpenShift-specific prompts for cluster management and troubleshooting.

## Prompts

### plan_mustgather

Plan for collecting a must-gather archive from an OpenShift cluster. Must-gather is a tool for collecting cluster data related to debugging and troubleshooting like logs, Kubernetes resources, and more.

This prompt generates YAML manifests for the must-gather resources that can be applied to the cluster.

**Arguments:**
- `node_name` (optional) - Specific node name to run must-gather pod on
- `node_selector` (optional) - Node selector in `key=value,key2=value2` format to filter nodes for the pod
- `source_dir` (optional) - Custom gather directory inside pod (default: `/must-gather`)
- `namespace` (optional) - Privileged namespace to use for must-gather (auto-generated if not specified)
- `gather_command` (optional) - Custom gather command e.g. `/usr/bin/gather_audit_logs` (default: `/usr/bin/gather`)
- `timeout` (optional) - Timeout duration for gather command (e.g., `30m`, `1h`)
- `since` (optional) - Only gather data newer than this duration (e.g., `5s`, `2m5s`, or `3h6m10s`), defaults to all data
- `host_network` (optional) - Use host network for must-gather pod (`true`/`false`)
- `keep_resources` (optional) - Keep pod resources after collection (`true`/`false`, default: `false`)
- `all_component_images` (optional) - Include must-gather images from all installed operators (`true`/`false`)
- `images` (optional) - Comma-separated list of custom must-gather container images

**Example:**
```
# Basic must-gather collection
{}

# Collect with custom timeout and since
{
  "timeout": "30m",
  "since": "1h"
}

# Collect from all component images
{
  "all_component_images": "true"
}

# Collect from specific operator image
{
  "images": "registry.redhat.io/openshift-logging/cluster-logging-rhel9-operator@sha256:..."
}
```

## Enable the OpenShift Toolset

### Option 1: Command Line

```bash
kubernetes-mcp-server --toolsets core,config,helm,openshift
```

### Option 2: Configuration File

```toml
toolsets = ["core", "config", "helm", "openshift"]
```

### Option 3: MCP Client Configuration

```json
{
  "mcpServers": {
    "kubernetes": {
      "command": "npx",
      "args": ["-y", "kubernetes-mcp-server@latest", "--toolsets", "core,config,helm,openshift"]
    }
  }
}
```

## Prerequisites

The OpenShift toolset requires:

1. **OpenShift cluster** - These prompts are designed for OpenShift and automatically detect the cluster type
2. **Proper RBAC** - The user/service account must have permissions to:
   - Create namespaces
   - Create service accounts
   - Create cluster role bindings
   - Create pods with privileged access
   - List ClusterOperators and ClusterServiceVersions (for `all_component_images`)

## How It Works

### Must-Gather Plan Generation

The `plan_mustgather` prompt generates YAML manifests for collecting diagnostic data from an OpenShift cluster:

1. **Namespace** - A temporary namespace (e.g., `openshift-must-gather-xyz`) is created unless an existing namespace is specified
2. **ServiceAccount** - A service account with cluster-admin permissions is created for the must-gather pod
3. **ClusterRoleBinding** - Binds the service account to the cluster-admin role
4. **Pod** - Runs the must-gather container(s) with the specified configuration

### Component Image Discovery

When `all_component_images` is enabled, the prompt discovers must-gather images from:
- **ClusterOperators** - Looks for the `operators.openshift.io/must-gather-image` annotation
- **ClusterServiceVersions** - Checks OLM-installed operators for the same annotation

### Multiple Images Support

Up to 8 gather images can be run concurrently. Each image runs in a separate container within the same pod, sharing the output volume.

## Common Use Cases

### Basic Cluster Diagnostics

Collect general cluster diagnostics:
```json
{}
```

### Audit Logs Collection

Collect audit logs with a custom gather command:
```json
{
  "gather_command": "/usr/bin/gather_audit_logs",
  "timeout": "2h"
}
```

### Recent Logs Only

Collect logs from the last 30 minutes:
```json
{
  "since": "30m"
}
```

### Specific Operator Diagnostics

Collect diagnostics for a specific operator:
```json
{
  "images": "registry.redhat.io/openshift-logging/cluster-logging-rhel9-operator@sha256:..."
}
```

### Host Network Access

For gather scripts that need host-level network access:
```json
{
  "host_network": "true"
}
```

### All Component Diagnostics

Collect diagnostics from all operators with must-gather images:
```json
{
  "all_component_images": "true",
  "timeout": "1h"
}
```

## Troubleshooting

### Permission Errors

If you see permission warnings, ensure your user has the required RBAC permissions:
```bash
oc auth can-i create namespaces
oc auth can-i create clusterrolebindings
oc auth can-i create pods --as=system:serviceaccount:openshift-must-gather-xxx:must-gather-collector
```

### Pod Not Starting

Check if the node has enough resources and can pull the must-gather image:
```bash
oc get pods -n openshift-must-gather-xxx
oc describe pod <pod-name> -n openshift-must-gather-xxx
```

### Timeout Issues

For large clusters or audit log collection, increase the timeout:
```json
{
  "timeout": "2h"
}
```

### Image Pull Errors

Ensure the must-gather image is accessible:
```bash
oc get secret -n openshift-config pull-secret
```

## Security Considerations

### Privileged Access

The must-gather pods run with:
- `cluster-admin` ClusterRoleBinding
- `system-cluster-critical` priority class
- Tolerations for all taints
- Optional host network access

### Temporary Resources

By default, all created resources (namespace, service account, cluster role binding) should be cleaned up after the must-gather collection is complete. Use `"keep_resources": "true"` to retain them for debugging.

### Image Sources

The prompt uses these default images:
- **Must-gather**: `registry.redhat.io/openshift4/ose-must-gather:latest`
- **Wait container**: `registry.redhat.io/ubi9/ubi-minimal`

Custom images should be from trusted sources.
