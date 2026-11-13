# Observability OpenShift e2e tests

OpenShift-only leftovers that shared `TestObs*` (`test/e2e`, `make e2e-ci-test-obs`)
does not cover: monitoring **Route discovery** and an OpenShift-specific **CMO metric**
check via an in-process server pointed at Route URLs.

Tool smoke tests for metrics/logs/traces/otelcol live in Kind-compatible `TestObs*`
and run on OpenShift CI after stack setup.

These tests use the `observability_e2e` build tag and are excluded from
`go test ./...` / `make test`.

## Prerequisites

- OpenShift cluster with platform monitoring (`openshift-monitoring` routes)
- `oc` / `kubectl` and a kubeconfig (`oc login` or `KUBECONFIG`)

## Configuration

| Variable | Required | Description |
|----------|----------|-------------|
| `KUBECONFIG` | No | Path to kubeconfig (default: `~/.kube/config`) |
| `PROMETHEUS_URL` | No | Skip Thanos route discovery for the CMO metric suite |
| `ALERTMANAGER_URL` | No | Skip Alertmanager route discovery for the CMO metric suite |
| `PROMETHEUS_K8S_TOKEN` | Auto | Bearer for monitoring Route HTTP and metrics tool calls (`auth_mode=header`). Auto-created via `oc`/`kubectl create token prometheus-k8s -n openshift-monitoring` when unset. |

## Running

```bash
# Shared tool suites (after Tempo/Loki install when needed)
make setup-observability-e2e
make e2e-ci-test-obs

# OpenShift-only Route discovery + CMO metric
make test-observability-e2e-routes

make clean-observability-e2e
```

OpenShift CI (`test/openshift/e2e-commands.sh`) runs stack setup, `e2e-ci-test-obs`,
then `test-observability-e2e-routes`.

## Coverage

| Suite | What | When |
|-------|------|------|
| `TestObs*` | Shared metrics/logs/traces/otelcol tool e2e | CI via `e2e-ci-test-obs` |
| Route discovery | Thanos/Prometheus/Alertmanager Route resolve + reachability | CI via `test-observability-e2e-routes` |
| CMO metric | `cluster_monitoring_operator_reconcile_attempts_total` via Route URLs | CI via `test-observability-e2e-routes` |

Make targets: `build/openshift/observability.mk`, `build/openshift/e2e.mk`.  
Manifests: `test/e2e/manifests/openshift/observability/{tempo,loki,prometheus}/`.  
LokiStack / log-generator names match Kind `TestObs*` (`obs-loki`, `obs-log-generator`).  
Setup: `hack/e2e/observability/setup-openshift.sh`.
