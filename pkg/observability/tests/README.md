# Observability OpenShift e2e tests

Cluster-backed tests for the `observability/metrics`, `observability/logs`,
`observability/traces`, and `observability/otelcol` toolsets against a live
OpenShift cluster. They exercise OpenShift Route discovery and MCP tool calls
through an in-process server.

These tests use the `observability_e2e` build tag and are excluded from
`go test ./...` / `make test`.

The in-process server forces `cluster_provider_strategy = "disabled"` so only
the current kubeconfig context is used (avoids discovery noise from stale
contexts in a shared `~/.kube/config`).

## Prerequisites

- OpenShift cluster with platform monitoring (`openshift-monitoring` routes)
- `oc` / `kubectl` and a kubeconfig (`oc login` or `KUBECONFIG`)
- For Tempo/Loki cases: stacks installed via `make setup-observability-e2e`

## Configuration

| Variable | Required | Description |
|----------|----------|-------------|
| `KUBECONFIG` | No | Path to kubeconfig (default: `~/.kube/config`) |
| `PROMETHEUS_URL` | No | Skip Thanos/Prometheus route discovery for metrics suite |
| `ALERTMANAGER_URL` | No | Skip Alertmanager route discovery for metrics suite |
| `PROMETHEUS_K8S_TOKEN` | Auto | Bearer for monitoring Route HTTP and metrics tool calls (`auth_mode=header`). Auto-created via `oc`/`kubectl create token prometheus-k8s -n openshift-monitoring` when unset. Needed because CI/ROSA kubeconfigs often use client certs that the monitoring OAuth proxy rejects (401). Not used by Tempo/Loki stack tests (those use kubeconfig). |
| `LOKI_STORAGE_CLASS` | No | Block StorageClass for LokiStack PVCs during `setup-observability-e2e`. Defaults to the cluster default StorageClass. |

## Running

```bash
# Full suite (recommended locally): install stacks, then run all tests
make setup-observability-e2e
make test-observability-e2e
make clean-observability-e2e

# Fast path: routes + metrics against platform monitoring (OpenShift CI default)
make test-observability-e2e-metrics

# Subset
go test -tags observability_e2e -v -run TestRouteDiscovery ./pkg/observability/tests/
```

OpenShift CI (`test/openshift/e2e-commands.sh`) always runs
`make test-observability-e2e-metrics`. Full stacks remain optional/local.

Make targets: `build/openshift/observability.mk`.  
Manifests: `test/e2e/manifests/openshift/observability/{tempo,loki,prometheus}/` (flat OpenShift-only layouts).  
Setup: `hack/e2e/observability/setup-openshift.sh`.
