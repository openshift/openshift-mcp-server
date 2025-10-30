# ACM Cluster Provider

The ACM (Advanced Cluster Management) cluster provider enables the Kubernetes MCP Server to interact with multiple managed clusters through a Red Hat Advanced Cluster Management (ACM) or Open Cluster Management (OCM) hub cluster.

## Overview

When using the ACM cluster provider, the MCP server connects to an ACM/OCM hub cluster and can then interact with all managed clusters via the [cluster-proxy addon](https://github.com/open-cluster-management-io/cluster-proxy). This allows you to manage multiple clusters from a single MCP server instance without needing separate kubeconfig files for each cluster.

## Features

- **Multi-cluster management**: Interact with all clusters managed by your ACM/OCM hub
- **Automatic discovery**: Automatically discovers managed clusters by listing `ManagedCluster` resources
- **Dynamic updates**: Watches for changes to managed clusters and updates available targets in real-time
- **Flexible authentication**: Supports both local kubeconfig-based and in-cluster authentication strategies
- **TLS configuration**: Optional TLS verification with custom CA certificate support

## Architecture

```
┌─────────────────┐
│  MCP Server     │
└────────┬────────┘
         │
         ├─────────────────────────────────┐
         │                                 │
         ▼                                 ▼
┌─────────────────┐              ┌─────────────────┐
│   ACM Hub       │              │  Managed        │
│   Cluster       │─────────────▶│  Clusters       │
│  (Discovery)    │  Cluster     │  (via proxy)    │
└─────────────────┘  Proxy       └─────────────────┘
```

The MCP server:
1. Connects to the ACM hub cluster to discover managed clusters
2. Routes requests to managed clusters through the cluster-proxy addon
3. Uses the hub cluster's credentials to authenticate with managed clusters

## Configuration

The ACM cluster provider supports two authentication strategies:

### 1. `acm-kubeconfig` - Remote Hub Cluster

Use this strategy when running the MCP server locally and connecting to a remote ACM hub cluster.

#### Configuration File

Create or update your `config.toml` file:

```toml
# Specify the ACM kubeconfig strategy
cluster_provider_strategy = "acm-kubeconfig"

[cluster_provider_configs.acm-kubeconfig]
# Name of the kubeconfig context pointing to your ACM hub cluster
context_name = "my-acm-hub"

# The route hostname for the cluster proxy addon
# Format: <route-name>.<hub-cluster-domain>
cluster_proxy_addon_host = "cluster-proxy-addon-user.apps.my-hub.example.com"

# Path to the CA certificate file for the cluster proxy (relative to config directory)
# Required if TLS verification is enabled
cluster_proxy_addon_ca_file = "ca-bundle.crt"

# Optional: Skip TLS verification (not recommended for production)
# cluster_proxy_addon_skip_tls_verify = false
```

#### Setup Steps

1. **Get the cluster proxy route**:
   ```bash
   oc get route -n open-cluster-management-addon cluster-proxy-addon-user \
     -o jsonpath='{.spec.host}'
   ```

2. **Extract the CA certificate**:
   ```bash
   # Get the CA certificate from the route
   oc get route -n open-cluster-management-addon cluster-proxy-addon-user \
     -o jsonpath='{.spec.tls.caCertificate}' > ca-bundle.crt

   # If the route doesn't have a CA, get it from the cluster
   oc get configmap -n openshift-config-managed default-ingress-cert \
     -o jsonpath='{.data.ca-bundle\.crt}' > ca-bundle.crt
   ```

3. **Place the CA file** in your config directory (e.g., `~/.config/kubernetes-mcp-server/ca-bundle.crt`)

4. **Ensure your kubeconfig** contains a context named as specified in `context_name`

#### Example Configuration

For a development environment where you want to skip TLS verification:

```toml
cluster_provider_strategy = "acm-kubeconfig"

[cluster_provider_configs.acm-kubeconfig]
context_name = "local-hub"
cluster_proxy_addon_host = "cluster-proxy-addon-user.apps.hub.example.com"
cluster_proxy_addon_skip_tls_verify = true
```

For production with proper TLS verification:

```toml
cluster_provider_strategy = "acm-kubeconfig"

[cluster_provider_configs.acm-kubeconfig]
context_name = "prod-hub"
cluster_proxy_addon_host = "cluster-proxy-addon-user.apps.hub.prod.example.com"
cluster_proxy_addon_ca_file = "ca-bundle.crt"
```

### 2. `acm` - In-Cluster Deployment

Use this strategy when deploying the MCP server as a pod within the ACM hub cluster itself.

#### Configuration File

```toml
# Specify the ACM in-cluster strategy
cluster_provider_strategy = "acm"

[cluster_provider_configs.acm]
# The service hostname for the cluster proxy addon
# Format: <service-name>.<namespace>.svc
cluster_proxy_addon_host = "cluster-proxy-addon-user.open-cluster-management-addon.svc"

# Path to the CA certificate file (relative to config directory)
cluster_proxy_addon_ca_file = "service-ca.crt"

# Optional: Skip TLS verification
# cluster_proxy_addon_skip_tls_verify = false
```

#### Setup Steps

1. **Get the cluster proxy service**:
   ```bash
   oc get service -n open-cluster-management-addon cluster-proxy-addon-user
   ```

2. **Extract the CA certificate**:
   ```bash
   # Get the service CA certificate
   oc get configmap -n open-cluster-management-addon cluster-proxy-addon-user-ca \
     -o jsonpath='{.data.ca\.crt}' > service-ca.crt
   ```

3. **Create a ConfigMap** with the CA certificate:
   ```bash
   oc create configmap mcp-server-config \
     --from-file=service-ca.crt=service-ca.crt \
     --from-file=config.toml=config.toml \
     -n your-namespace
   ```

4. **Deploy the MCP server** with the ConfigMap mounted

## Usage

Once configured, the ACM cluster provider will:

1. **Automatically discover managed clusters** by listing `ManagedCluster` resources from the hub
2. **Expose each cluster as a target** that can be used in MCP tool calls
3. **Watch for changes** and update the list of available clusters dynamically

### Listing Available Clusters

When using Claude Code or other MCP clients, you can query available clusters:

```
List all available Kubernetes contexts
```

The response will include:
- `hub` - The ACM hub cluster itself
- All managed cluster names as discovered from `ManagedCluster` resources

### Targeting Specific Clusters

MCP tools will include a `cluster` parameter to specify which cluster to target:

```
List all pods in the production cluster
```

Claude Code will automatically use the appropriate cluster based on your request.

## Prerequisites

### ACM Hub Cluster

Your ACM hub cluster must have:

1. **ACM or OCM installed** with `ManagedCluster` CRDs available
2. **Cluster proxy addon** installed and configured
   - Check installation: `oc get deploy -n open-cluster-management-addon cluster-proxy-addon-user`
3. **Managed clusters** registered with the hub
   - Verify: `oc get managedclusters`

### Managed Clusters

Each managed cluster must have:

1. **Cluster proxy addon** installed
   - Verify: `oc get managedclusteraddon -A | grep cluster-proxy`
2. **Connection to the hub** established
   - Check status: `oc get managedcluster <cluster-name> -o jsonpath='{.status.conditions}'`

### Permissions

The service account or user running the MCP server needs:

**On the hub cluster:**
- Read access to `ManagedCluster` resources
- Read access to cluster configuration
- Token review permissions (for authentication)

**Example RBAC:**
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: mcp-server-acm
rules:
- apiGroups: ["cluster.open-cluster-management.io"]
  resources: ["managedclusters"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["serviceaccounts/token"]
  verbs: ["create"]
- apiGroups: ["authentication.k8s.io"]
  resources: ["tokenreviews"]
  verbs: ["create"]
```

**On managed clusters:**
- The hub cluster's service account token is used for authentication
- Permissions are determined by what the proxy allows

## Troubleshooting

### Issue: "not deployed in an ACM hub cluster"

**Cause**: The MCP server cannot detect ACM/OCM installation.

**Solution**:
1. Verify ACM is installed: `oc get crd managedclusters.cluster.open-cluster-management.io`
2. Check API access: `oc get managedclusters`
3. Ensure your kubeconfig context has proper permissions

### Issue: "failed to list cluster managers"

**Cause**: Cannot list `ManagedCluster` resources.

**Solution**:
1. Check permissions: `oc auth can-i list managedclusters.cluster.open-cluster-management.io`
2. Verify RBAC: Review the service account's role bindings
3. Check API availability: `oc api-resources | grep managedcluster`

### Issue: TLS certificate errors when connecting to managed clusters

**Cause**: CA certificate mismatch or not provided.

**Solution**:
1. Verify the CA file path in your configuration
2. Ensure the CA file contains the correct certificate
3. For testing, temporarily enable `cluster_proxy_addon_skip_tls_verify = true`
4. Check the route/service certificate:
   ```bash
   # For routes
   oc get route cluster-proxy-addon-user -n open-cluster-management-addon -o yaml
   ```

### Issue: Cannot connect to specific managed cluster

**Cause**: Cluster proxy addon may not be installed or configured correctly.

**Solution**:
1. Verify addon installation on managed cluster:
   ```bash
   oc get managedclusteraddon -A | grep cluster-proxy
   ```
2. Check addon status:
   ```bash
   oc get managedclusteraddon -n <cluster-name> cluster-proxy -o yaml
   ```
3. Review proxy addon logs:
   ```bash
   oc logs -n open-cluster-management-addon -l app=cluster-proxy-addon
   ```

### Debug Mode

Enable verbose logging to troubleshoot issues:

```bash
kubernetes-mcp-server --log-level 5
```

Log levels:
- `2`: Info level - basic operation logs
- `3`: Debug level - detailed operation logs
- `5`: Trace level - very verbose, includes API calls

## Security Considerations

1. **Use TLS verification in production**: Always provide a valid CA certificate file and avoid `skip_tls_verify`
2. **Least privilege**: Grant only necessary RBAC permissions to the service account
3. **Secure credential storage**: Store kubeconfig files and CA certificates securely
4. **Network policies**: Consider implementing network policies to restrict access to the cluster proxy
5. **Audit logging**: Enable audit logging on both hub and managed clusters
6. **Regular rotation**: Rotate service account tokens and certificates regularly

## Advanced Configuration

### Custom Timeout Settings

The ACM provider uses exponential backoff for watch operations. Default settings:
- Initial delay: 1 second
- Maximum delay: 5 minutes
- Backoff rate: 2.0x

These are currently hardcoded but can be modified in the source if needed.

### Multiple Hub Clusters

To manage multiple ACM hubs, you can run separate MCP server instances with different configurations:

```bash
# Instance 1: Hub A
kubernetes-mcp-server --config /path/to/hub-a-config.toml --port 3000

# Instance 2: Hub B
kubernetes-mcp-server --config /path/to/hub-b-config.toml --port 3001
```

## Examples

### Example 1: Local Development Setup

```toml
cluster_provider_strategy = "acm-kubeconfig"

[cluster_provider_configs.acm-kubeconfig]
context_name = "crc-acm"
cluster_proxy_addon_host = "cluster-proxy-addon-user.apps-crc.testing"
cluster_proxy_addon_skip_tls_verify = true
```

### Example 2: Production In-Cluster Deployment

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kubernetes-mcp-server
  namespace: mcp-server
spec:
  replicas: 1
  selector:
    matchLabels:
      app: kubernetes-mcp-server
  template:
    metadata:
      labels:
        app: kubernetes-mcp-server
    spec:
      serviceAccountName: mcp-server-sa
      containers:
      - name: server
        image: quay.io/manusa/kubernetes_mcp_server:latest
        args:
        - --port=3000
        - --config=/config/config.toml
        volumeMounts:
        - name: config
          mountPath: /config
          readOnly: true
      volumes:
      - name: config
        configMap:
          name: mcp-server-config
---
# config.toml in ConfigMap
cluster_provider_strategy = "acm"

[cluster_provider_configs.acm]
cluster_proxy_addon_host = "cluster-proxy-addon-user.open-cluster-management-addon.svc"
cluster_proxy_addon_ca_file = "service-ca.crt"
```

## Related Documentation

- [ACM Documentation](https://access.redhat.com/documentation/en-us/red_hat_advanced_cluster_management_for_kubernetes/)
- [Open Cluster Management](https://open-cluster-management.io/)
- [Cluster Proxy Addon](https://github.com/open-cluster-management-io/cluster-proxy)
- [Main README](../README.md)
- [Getting Started with Kubernetes](GETTING_STARTED_KUBERNETES.md)
