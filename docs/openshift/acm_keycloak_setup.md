# Advanced Cluster Management (ACM) with Keycloak Authentication

This guide shows you how to set up Red Hat Advanced Cluster Management (ACM) with Keycloak-based OIDC authentication for secure multi-cluster access with the OpenShift MCP Server.

## Overview

This setup combines ACM's multi-cluster management capabilities with Keycloak's OIDC authentication and token exchange features. This enables:

- **Centralized Authentication**: Single Keycloak instance for all clusters
- **Token Exchange**: Seamless cross-cluster authentication using V1 token exchange
- **Federated Users**: Create users once, access all clusters
- **Secure Access**: OIDC-based authentication for the MCP server

The architecture uses:
- **Hub Cluster**: Runs ACM and Keycloak, manages other clusters
- **Managed Clusters**: Configured with OIDC to authenticate against Keycloak
- **MCP Server**: Connects using OIDC tokens exchanged via Keycloak

## Prerequisites

- OpenShift 4.19+ or 4.20+ clusters
- ACM installed on hub cluster (see [acm_setup.md](acm_setup.md))
- Cluster-admin access to all clusters
- `kubectl` or `oc` CLI tools

## Step 1: Set Up Keycloak on Hub Cluster

Deploy Keycloak with V1 token exchange features enabled on your hub cluster.

### Complete Hub Setup

Install and configure Keycloak with a single command:

```bash
make keycloak-acm-setup-hub
```

This command performs the following:

1. Enables `TechPreviewNoUpgrade` feature gate (required for OIDC)
2. Deploys Keycloak with V1 token exchange capabilities
3. Creates hub realm with MCP user and clients
4. Configures same-realm token exchange
5. Fixes CA trust for cross-realm token exchange
6. Creates RBAC for the MCP user
7. Saves configuration to `.keycloak-config/hub-config.env`

**⚠️ Important**: On OpenShift 4.19 and earlier, OIDC authentication requires enabling the `TechPreviewNoUpgrade` feature gate. **This flag breaks upgrades** - clusters with this feature gate enabled cannot be upgraded through standard procedures.

**Note**: OIDC support (via the `ExternalOIDC` feature) is GA in OpenShift 4.20+ and does not require the `TechPreviewNoUpgrade` feature gate. If you're running OpenShift 4.20 or later, the setup script will automatically detect this and skip enabling `TechPreviewNoUpgrade`.

### Verify Hub Setup

Check Keycloak deployment status:

```bash
make keycloak-acm-status
```

Expected output:
```
===========================================
Keycloak ACM Configuration Status
===========================================

Pod: keycloak-xxxxxxxx-xxxxx (Running)

Route: https://keycloak-keycloak.apps.cluster.example.com

Admin Console:
  URL: https://keycloak-keycloak.apps.cluster.example.com/admin
  Username: admin
  Password: admin

Hub Realm: hub
  MCP User: mcp
  Client ID: openshift

OIDC Endpoints (hub realm):
  Discovery: https://keycloak-.../realms/hub/.well-known/openid-configuration
  Token:     https://keycloak-.../realms/hub/protocol/openid-connect/token
  Authorize: https://keycloak-.../realms/hub/protocol/openid-connect/auth
```

## Step 2: Register Managed Clusters

Import additional OpenShift clusters with OIDC authentication configured.

### Register a Managed Cluster

```bash
make keycloak-acm-register-managed-cluster \
  CLUSTER_NAME=production-east \
  MANAGED_KUBECONFIG=/path/to/production-east-kubeconfig
```

**Parameters**:
- `CLUSTER_NAME`: Unique name for the managed cluster
- `MANAGED_KUBECONFIG`: Path to the kubeconfig file for the managed cluster
- `HUB_KUBECONFIG`: (Optional) Path to hub kubeconfig, defaults to `$KUBECONFIG`

### What Happens During Registration

The registration process performs the following steps (~25-30 minutes):

1. **ACM Integration**:
   - Creates `ManagedCluster` resource on hub
   - Applies ACM import manifests to managed cluster
   - Installs klusterlet agent and cluster-proxy

2. **Keycloak Configuration**:
   - Creates dedicated realm for the managed cluster
   - Configures cross-realm token exchange (hub ↔ managed)
   - Sets up OIDC clients

3. **Managed Cluster OIDC Setup**:
   - Enables `TechPreviewNoUpgrade` feature gate (on OpenShift 4.19 and earlier)
   - Configures `OAuth` resource to use Keycloak
   - Creates `ClusterRoleBinding` for MCP user

**⚠️ Important**: On OpenShift 4.19 and earlier, OIDC authentication requires enabling the `TechPreviewNoUpgrade` feature gate on the managed cluster. **This flag breaks upgrades** - clusters with this feature gate enabled cannot be upgraded through standard procedures. On OpenShift 4.20+, the `ExternalOIDC` feature is GA and does not require this feature gate.

4. **Waits for Rollouts**:
   - API server pods (for OIDC configuration)
   - OAuth server pods
   - Cluster stabilization

**Note**: Rollouts happen in the background. The script waits for stability before completing.

### Verify Managed Cluster Registration

Check ACM cluster status:

```bash
oc get managedclusters
```

Expected output:
```
NAME               HUB ACCEPTED   MANAGED CLUSTER URLS                      JOINED   AVAILABLE   AGE
local-cluster      true           https://api.hub.example.com:6443         True     True        30m
production-east    true           https://api.prod-east.example.com:6443   True     True        15m
```

Check Keycloak configuration:

```bash
make keycloak-acm-status
```

You should see the managed cluster listed with its realm:
```
Managed Clusters:
  - production-east (realm: production-east)

Configured clusters in TOML:
  - hub
  - production-east
```

## Step 3: Generate MCP Server Configuration

After registering your managed clusters, generate the TOML configuration file for the MCP server:

```bash
make keycloak-acm-generate-toml
```

This creates `_output/acm-kubeconfig.toml` with:
- Hub cluster connection details
- Keycloak OIDC configuration for all registered clusters
- Token exchange settings
- CA certificates

Example generated configuration:

```toml
cluster_provider_strategy = "acm-kubeconfig"

[cluster_provider_configs.acm-kubeconfig]
kubeconfig = "/tmp/hub-kubeconfig.yaml"
context_name = "hub-context"
cluster_proxy_addon_ca_file = "_output/hub-ca.crt"

[cluster_provider_configs.acm-kubeconfig.clusters.hub]
issuer_url = "https://keycloak-.../realms/hub"
client_id = "openshift"
client_secret = "..."
ca_file = "_output/hub-keycloak-ca.crt"

[cluster_provider_configs.acm-kubeconfig.clusters.production-east]
issuer_url = "https://keycloak-.../realms/production-east"
client_id = "openshift"
client_secret = "..."
ca_file = "_output/production-east-keycloak-ca.crt"
```

**Note**: Run this command each time you add or remove managed clusters to update the configuration file.

## Step 4: Run the MCP Server

Start the MCP server with the generated configuration:

```bash
./kubernetes-mcp-server --config _output/acm-kubeconfig.toml --port 8080
```

The MCP server will:
1. Authenticate to Keycloak using OIDC
2. Connect to the hub cluster
3. Discover all ACM-managed clusters
4. Use token exchange to authenticate to managed clusters
5. Provide tools to interact with all clusters

## Step 5: Test Multi-Cluster Access with OIDC

Test authentication and multi-cluster access:

### Test with MCP Inspector

```bash
npx @modelcontextprotocol/inspector@latest \
  ./kubernetes-mcp-server --config _output/acm-kubeconfig.toml
```

In the inspector:
- Select a tool (e.g., `namespaces_list`)
- Choose the target cluster from the dropdown
- Execute the tool
- Verify that authentication and data retrieval work

<a href="images/mcp-inspector-acm-managed-cluster.png">
  <img src="images/mcp-inspector-acm-managed-cluster.png" alt="MCP Inspector with ACM cluster selection" width="600" />
</a>

## Step 6: Add Additional Users

Create federated users that can access all clusters:

### Add a Federated User

```bash
make keycloak-acm-add-user \
  KEYCLOAK_USER=alice \
  KEYCLOAK_PASS=secret123
```

This creates a user in the hub realm with:
- Username: `alice`
- Password: `secret123`
- Role: `cluster-admin` (default)
- Access: All clusters (hub + managed)

### Add a Cluster-Specific User

For standalone access to a specific cluster:

```bash
make keycloak-acm-add-user \
  KEYCLOAK_USER=bob \
  KEYCLOAK_PASS=secret456 \
  CLUSTER_NAME=production-east
```

This creates a user only in the `production-east` realm.

### Specify Custom Role

```bash
make keycloak-acm-add-user \
  KEYCLOAK_USER=viewer \
  KEYCLOAK_PASS=readonly \
  KEYCLOAK_USER_ROLE=view
```
