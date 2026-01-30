# Kubernetes MCP Server Documentation

Welcome to the Kubernetes MCP Server documentation! This directory contains guides to help you set up and use the Kubernetes MCP Server with your Kubernetes cluster and Claude Code CLI.

## Getting Started Guides

Choose the guide that matches your needs:

| Guide | Description | Best For |
|-------|-------------|----------|
| **[Getting Started with Kubernetes](GETTING_STARTED_KUBERNETES.md)** | Base setup: Create ServiceAccount, token, and kubeconfig | Everyone - **start here first** |
| **[Using with Claude Code CLI](GETTING_STARTED_CLAUDE_CODE.md)** | Configure MCP server with Claude Code CLI | Claude Code CLI users |

## Recommended Workflow

1. **Complete the base setup**: Start with [Getting Started with Kubernetes](GETTING_STARTED_KUBERNETES.md) to create a ServiceAccount and kubeconfig file
2. **Configure Claude Code**: Then follow the [Claude Code CLI guide](GETTING_STARTED_CLAUDE_CODE.md)

## Other toolsets

- **[OADP](OADP.md)** - Tools for OpenShift API for Data Protection (Velero backups, restores, schedules)
- **[Kiali](KIALI.md)** - Tools for Kiali ServiceMesh with Istio
- **[Observability](OBSERVABILITY.md)** - Tools for Prometheus metrics and Alertmanager alerts

## Additional Documentation

- **[Keycloak OIDC Setup](KEYCLOAK_OIDC_SETUP.md)** - Developer guide for local Keycloak environment and testing with MCP Inspector
- **[Main README](../README.md)** - Project overview and general information



