# Keycloak ACM Multi-Cluster Setup

This guide shows you how to set up Keycloak-based OIDC authentication for ACM multi-cluster environments with cross-realm token exchange.

## Overview

This setup enables:
- Single Keycloak instance on the hub cluster
- OIDC authentication for both hub (local-cluster) and managed clusters
- Cross-realm token exchange for seamless multi-cluster access
- Single MCP server instance accessing all clusters with user authentication

## Prerequisites

- ACM installed on hub cluster (see [acm.md](acm.md))
- At least one managed cluster imported
- Hub cluster kubeconfig available

## Step 1: Setup Keycloak and Hub Realm

Deploy Keycloak on the hub cluster and configure the hub realm:

```bash
make keycloak-acm-setup-hub
```

This performs:
1. Deploys Keycloak and PostgreSQL on hub cluster
2. Creates hub realm with OIDC configuration
3. Creates `mcp-sts` client for token exchange
4. Creates `mcp-server` scope with required mappers
5. Creates test user `mcp` (username: `mcp`, password: `mcp`)
6. Saves configuration to `.keycloak-config/hub-config.env`

**Wait for Keycloak to be ready** (2-3 minutes):

```bash
make keycloak-status
```

Expected output:
```
Keycloak Status:
  Pod: keycloak-6c94fb478b-gxqzf (Running)
  Route: https://keycloak-keycloak.apps.example.com
  Admin Console: https://keycloak-keycloak.apps.example.com/admin
```

## Step 2: Register Managed Cluster

Register each managed cluster with ACM and configure OIDC authentication:

```bash
make keycloak-acm-register-managed-cluster \
  CLUSTER_NAME=production-east \
  MANAGED_KUBECONFIG=/path/to/production-east-kubeconfig
```

**Parameters**:
- `CLUSTER_NAME`: Name of the managed cluster (must match ACM ManagedCluster name)
- `MANAGED_KUBECONFIG`: Path to the managed cluster's kubeconfig

### What Happens During Registration

1. Creates `ManagedCluster` resource (if not exists)
2. Applies ACM import manifests to managed cluster
3. Creates dedicated Keycloak realm for the cluster
4. Configures `mcp-server` client with protocol mappers
5. Creates identity provider linking to hub realm
6. Enables TechPreviewNoUpgrade feature gate on managed cluster
7. Configures OIDC authentication pointing to Keycloak
8. Creates RBAC for authenticated users
9. Saves configuration to `.keycloak-config/clusters/$CLUSTER_NAME.env`

### Important: kube-apiserver Rollout

**The kube-apiserver will restart to apply OIDC configuration** (10-15 minutes). Monitor the rollout:

```bash
kubectl --kubeconfig=/path/to/managed-kubeconfig get co kube-apiserver -w
```

Wait for:
```
NAME              VERSION   AVAILABLE   PROGRESSING   DEGRADED   SINCE   MESSAGE
kube-apiserver    4.17.0    True        False         False      2m      ...
```

When `PROGRESSING` is `False`, the cluster is ready.

### Quick Status Check

```bash
make keycloak-acm-status
```

Shows all configured clusters and their Keycloak realms.

## Step 3: Generate MCP Server Configuration

Generate the MCP server configuration file with all cluster credentials:

```bash
make keycloak-acm-generate-toml
```

This creates `_output/acm-kubeconfig.toml` with:
- Hub cluster OAuth configuration
- Token exchange settings for local-cluster (hub)
- Token exchange settings for each managed cluster
- Keycloak CA certificate

**Example generated configuration**:

```toml
cluster_provider_strategy = "acm-kubeconfig"
kubeconfig = "/path/to/hub-kubeconfig"

# Hub OAuth Configuration
require_oauth = true
oauth_audience = "mcp-server"
authorization_url = "https://keycloak-keycloak.apps.example.com/realms/hub"
token_url = "https://keycloak-keycloak.apps.example.com/realms/hub/protocol/openid-connect/token"
sts_client_id = "mcp-sts"
sts_client_secret = "..."
certificate_authority = "_output/keycloak-ca.crt"

# Local cluster (hub) - same-realm token exchange
[cluster_provider_configs.acm-kubeconfig.clusters."local-cluster"]
token_url = "https://keycloak-keycloak.apps.example.com/realms/hub/protocol/openid-connect/token"
client_id = "mcp-sts"
client_secret = "..."
audience = "mcp-server"
subject_token_type = "urn:ietf:params:oauth:token-type:access_token"

# Managed cluster - cross-realm token exchange
[cluster_provider_configs.acm-kubeconfig.clusters."production-east"]
token_url = "https://keycloak-keycloak.apps.example.com/realms/production-east/protocol/openid-connect/token"
client_id = "mcp-server"
client_secret = "..."
subject_issuer = "hub-realm"
audience = "mcp-server"
subject_token_type = "urn:ietf:params:oauth:token-type:jwt"
```

## Step 4: Run the MCP Server

Start the MCP Server with OAuth authentication:

```bash
./kubernetes-mcp-server --config _output/acm-kubeconfig.toml --port 8080
```

The server will:
1. Listen on `http://localhost:8080/mcp`
2. Require OAuth authentication via Keycloak
3. Support token exchange for all configured clusters
4. Provide tools for multi-cluster operations

## Step 5: Test with MCP Inspector

Test the setup using the MCP Inspector:

```bash
npx @modelcontextprotocol/inspector@latest http://localhost:8080/mcp
```

**Authentication flow**:
1. Inspector opens browser for OAuth login
2. You authenticate with Keycloak (user: `mcp`, password: `mcp`)
3. Inspector receives OAuth token
4. MCP server performs token exchange for each cluster access
