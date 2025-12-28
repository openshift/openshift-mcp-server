# External Secrets Operator for Red Hat OpenShift

This document provides guidance on using the External Secrets toolset with the Kubernetes MCP Server.

## Overview

The External Secrets Operator for Red Hat OpenShift synchronizes secrets from external secret management systems (AWS Secrets Manager, HashiCorp Vault, Google Cloud Secret Manager, Azure Key Vault, etc.) into Kubernetes Secrets.

The `external-secrets` toolset provides comprehensive tools for:
- **Operator lifecycle management** - Install, configure, and uninstall the operator
- **SecretStore management** - Create and manage connections to secret providers
- **ExternalSecret management** - Define and manage secret synchronization
- **Debugging and monitoring** - Health checks, logs, events, and diagnostics

## Prerequisites

- OpenShift Container Platform 4.x cluster
- Cluster administrator access (for operator installation)
- Access to an external secrets provider (AWS, GCP, Azure, Vault, etc.)

## Enabling the Toolset

The `external-secrets` toolset is not enabled by default. Enable it using the `--toolsets` flag:

```bash
kubernetes-mcp-server --toolsets core,config,helm,external-secrets
```

Or in your MCP client configuration:

```json
{
  "mcpServers": {
    "kubernetes": {
      "command": "npx",
      "args": [
        "-y",
        "kubernetes-mcp-server@latest",
        "--toolsets", "core,config,helm,external-secrets"
      ]
    }
  }
}
```

## Available Tools

### Operator Management

| Tool | Description |
|------|-------------|
| `external_secrets_operator_install` | Install the External Secrets Operator via OLM |
| `external_secrets_operator_status` | Get operator installation status |
| `external_secrets_operator_uninstall` | Uninstall the operator |
| `external_secrets_config_get` | Get ExternalSecretsConfig resource |
| `external_secrets_config_apply` | Apply/update ExternalSecretsConfig |

### SecretStore Management

| Tool | Description |
|------|-------------|
| `external_secrets_store_list` | List SecretStores/ClusterSecretStores |
| `external_secrets_store_get` | Get details of a specific store |
| `external_secrets_store_create` | Create/update a SecretStore |
| `external_secrets_store_delete` | Delete a SecretStore |
| `external_secrets_store_validate` | Validate store health status |

### ExternalSecret Management

| Tool | Description |
|------|-------------|
| `external_secrets_list` | List ExternalSecrets/ClusterExternalSecrets |
| `external_secrets_get` | Get details of a specific ExternalSecret |
| `external_secrets_create` | Create/update an ExternalSecret |
| `external_secrets_delete` | Delete an ExternalSecret |
| `external_secrets_sync_status` | Check synchronization status |
| `external_secrets_refresh` | Trigger immediate secret refresh |

### Debugging and Monitoring

| Tool | Description |
|------|-------------|
| `external_secrets_debug` | Comprehensive debugging information |
| `external_secrets_events` | View related Kubernetes events |
| `external_secrets_logs` | Get operator pod logs |
| `external_secrets_health` | Quick health check summary |
| `external_secrets_guide` | Get documentation and examples |

## Quick Start

### 1. Install the Operator

```
Use tool: external_secrets_operator_install
```

This creates:
- `external-secrets-operator` namespace
- OperatorGroup
- Subscription to the Red Hat operator catalog

### 2. Verify Installation

```
Use tool: external_secrets_operator_status
```

Wait until the operator pods are running.

### 3. Create a SecretStore

Example for AWS Secrets Manager:

```
Use tool: external_secrets_store_create
Argument store:
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: aws-secretsmanager
  namespace: my-namespace
spec:
  provider:
    aws:
      service: SecretsManager
      region: us-east-1
      auth:
        secretRef:
          accessKeyIDSecretRef:
            name: aws-credentials
            key: access-key
          secretAccessKeySecretRef:
            name: aws-credentials
            key: secret-access-key
```

### 4. Create an ExternalSecret

```
Use tool: external_secrets_create
Argument secret:
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: my-secret
  namespace: my-namespace
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: aws-secretsmanager
    kind: SecretStore
  target:
    name: my-k8s-secret
  data:
  - secretKey: password
    remoteRef:
      key: my-aws-secret
      property: password
```

### 5. Verify Sync Status

```
Use tool: external_secrets_sync_status
Argument namespace: my-namespace
```

## Supported Providers

The External Secrets Operator supports many providers:

- **Cloud Providers**: AWS Secrets Manager, AWS Parameter Store, GCP Secret Manager, Azure Key Vault
- **Secret Management**: HashiCorp Vault, CyberArk Conjur, Bitwarden, 1Password, Doppler
- **Other**: Kubernetes Secrets, Webhook (custom HTTP endpoints)

Use the guide tool for provider-specific examples:

```
Use tool: external_secrets_guide
Argument topic: providers
Argument provider: aws
```

## Troubleshooting

### Quick Health Check

```
Use tool: external_secrets_health
```

### Comprehensive Debug

```
Use tool: external_secrets_debug
Argument include_logs: true
```

### Common Issues

1. **SecretStore not valid**: Check credentials and network connectivity
2. **ExternalSecret not syncing**: Verify SecretStore is ready and secret exists in provider
3. **Operator not running**: Check subscription approval mode and resource availability

For detailed troubleshooting guidance:

```
Use tool: external_secrets_guide
Argument topic: troubleshooting
```

## Security Best Practices

- Use separate SecretStores per namespace/team for isolation
- Prefer workload identity (IRSA, Workload Identity) over static credentials
- Set appropriate RBAC to restrict SecretStore creation
- Enable audit logging for external-secrets resources

For more security guidance:

```
Use tool: external_secrets_guide
Argument topic: security
```

## References

- [External Secrets Operator for Red Hat OpenShift Documentation](https://docs.redhat.com/en/documentation/openshift_container_platform/latest/html/security_and_compliance/external-secrets-operator-for-red-hat-openshift)
- [External Secrets Documentation](https://external-secrets.io/latest/)
- [OpenShift External Secrets Operator Repository](https://github.com/openshift/external-secrets-operator)
- [External Secrets Repository](https://github.com/openshift/external-secrets)

