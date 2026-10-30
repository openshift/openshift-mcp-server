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
  (OpenShift CI does this automatically)

## Configuration

| Variable | Required | Description |
|----------|----------|-------------|
| `KUBECONFIG` | No | Path to kubeconfig (default: `~/.kube/config`) |
| `PROMETHEUS_URL` | No | Skip Thanos/Prometheus route discovery for metrics suite |
| `ALERTMANAGER_URL` | No | Skip Alertmanager route discovery for metrics suite |
| `PROMETHEUS_K8S_TOKEN` | Auto | Bearer for monitoring Route HTTP and metrics tool calls (`auth_mode=header`). Auto-created via `oc`/`kubectl create token prometheus-k8s -n openshift-monitoring` when unset. Needed because CI/ROSA kubeconfigs often use client certs that the monitoring OAuth proxy rejects (401). |
| `LOKI_STORAGE_CLASS` | No | Block StorageClass for LokiStack PVCs during `setup-observability-e2e`. Defaults to the cluster default StorageClass. |

Suite setup also rewrites `KUBECONFIG` once per process when needed:

- Merges `openshift-config-managed/default-ingress-cert` so Route TLS verifies without `insecure=true`.
- When the kubeconfig has only client certificates, mints SA `default/obs-e2e` (`cluster-reader` binding) and switches the context to that bearer token so Tempo/Loki gateway OAuth accepts `auth_mode=kubeconfig`.

## Running

```bash
# Full suite: install stacks, then run all tests
make setup-observability-e2e
make test-observability-e2e
make clean-observability-e2e

# Fast path: routes + metrics against platform monitoring only (no Tempo/Loki)
make test-observability-e2e-metrics

# Subset
go test -tags observability_e2e -v -run TestRouteDiscovery ./pkg/observability/tests/
```

OpenShift CI (`test/openshift/e2e-commands.sh`) runs `make setup-observability-e2e`,
then `make test-observability-e2e`, and tears stacks down on exit.

## Coverage

| Suite | Tools exercised | When |
|-------|-----------------|------|
| Metrics + routes | All `observability/metrics` tools; Thanos/Prometheus/Alertmanager Route discovery (URLs still explicit in toolset config) | CI (full suite) |
| Loki | `loki_list_instances` (Route URLs), `loki_label_names`, `loki_label_values`, `loki_query_range` | CI (after stack setup) |
| Tempo | `tempo_list_instances`, search (single + multitenant), `tempo_search_tags`, `tempo_search_tag_values`, `tempo_get_trace_by_id` | CI (after stack setup) |
| otelcol | `otelcol_get_versions`, `otelcol_list_components`, `otelcol_get_component_schema`, `otelcol_validate_config` | CI (no stack install required; runs with full suite) |

Make targets: `build/openshift/observability.mk`.  
Manifests: `test/e2e/manifests/openshift/observability/{tempo,loki,prometheus}/` (flat OpenShift-only layouts).  
Setup: `hack/e2e/observability/setup-openshift.sh`.
