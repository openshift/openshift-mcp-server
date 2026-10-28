#!/usr/bin/env bash
# Install OpenShift observability stacks (Tempo / Loki) for observability_e2e tests.
# Platform Prometheus/Alertmanager in openshift-monitoring are assumed present.
#
# Usage:
#   ./hack/e2e/observability/setup-openshift.sh              # tempo + loki
#   ./hack/e2e/observability/setup-openshift.sh tempo        # tempo only
#   ./hack/e2e/observability/setup-openshift.sh loki         # loki only
#   ./hack/e2e/observability/setup-openshift.sh clean        # best-effort teardown
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
MANIFESTS="${ROOT_DIR}/test/e2e/manifests/openshift/observability"
KUBECTL="${KUBECTL:-kubectl}"
STACKS="${*:-tempo loki}"
UWM_BACKUP="${MANIFESTS}/.cluster-monitoring-config.backup.json"
UWM_MARKER="${MANIFESTS}/.cluster-monitoring-config.created"

_wait_rollout() {
  local ns="$1" resource="$2" timeout="${3:-10m}"
  $KUBECTL -n "${ns}" wait --for=create "${resource}" --timeout="${timeout}"
  $KUBECTL -n "${ns}" rollout status "${resource}" --timeout="${timeout}"
}

# Resolve Loki Operator channel from redhat-operators PackageManifest.
# Pinning stable-6.5 fails with ResolutionFailed when that channel is missing from
# the cluster catalog (common on OpenShift CI release:latest). Override with
# LOKI_OPERATOR_CHANNEL when needed.
_loki_operator_channel() {
  if [[ -n "${LOKI_OPERATOR_CHANNEL:-}" ]]; then
    echo "${LOKI_OPERATOR_CHANNEL}"
    return
  fi
  local ch
  ch="$($KUBECTL get packagemanifest -n openshift-marketplace -o json 2>/dev/null | python3 -c '
import json, sys
data = json.load(sys.stdin)
for item in data.get("items") or []:
    if item.get("metadata", {}).get("name") != "loki-operator":
        continue
    status = item.get("status") or {}
    if status.get("catalogSource") != "redhat-operators":
        continue
    ch = status.get("defaultChannel") or ""
    if not ch:
        names = [c.get("name") or "" for c in (status.get("channels") or []) if c.get("name")]
        # Prefer highest stable-6.x when defaultChannel is unset.
        stable = sorted((n for n in names if n.startswith("stable-6.")), reverse=True)
        ch = stable[0] if stable else (names[0] if names else "")
    print(ch)
    break
' 2>/dev/null || true)"
  if [[ -z "${ch}" ]]; then
    echo "warning: could not resolve loki-operator channel from PackageManifest; falling back to stable-6.5" >&2
    ch="stable-6.5"
  fi
  echo "${ch}"
}

_dump_loki_operator() {
  echo "==> Loki operator install diagnostics" >&2
  $KUBECTL -n openshift-operators-redhat get subscription,csv,installplan,deploy,pods -o wide >&2 || true
  $KUBECTL -n openshift-operators-redhat describe subscription loki-operator >&2 || true
  $KUBECTL get packagemanifest -n openshift-marketplace -l catalog=redhat-operators 2>/dev/null \
    | awk 'NR==1 || /loki-operator/' >&2 || true
  $KUBECTL get packagemanifest -n openshift-marketplace -o json 2>/dev/null | python3 -c '
import json, sys
data = json.load(sys.stdin)
for item in data.get("items") or []:
    if item.get("metadata", {}).get("name") != "loki-operator":
        continue
    status = item.get("status") or {}
    if status.get("catalogSource") != "redhat-operators":
        continue
    chans = [c.get("name") for c in (status.get("channels") or [])]
    print("redhat-operators loki-operator defaultChannel=%s channels=%s" % (
        status.get("defaultChannel"), chans), file=sys.stderr)
    break
' 2>/dev/null || true
}

# Wait until Subscription resolves an installedCSV, then CSV reaches Succeeded.
_wait_subscription_csv() {
  local ns="$1" sub="$2" timeout="${3:-10m}"
  local timeout_sec reason
  timeout_sec="$(_duration_to_seconds "${timeout}")"
  local deadline=$((SECONDS + timeout_sec))
  local csv="" phase=""
  while (( SECONDS < deadline )); do
    csv="$($KUBECTL -n "${ns}" get subscription "${sub}" -o jsonpath='{.status.installedCSV}' 2>/dev/null || true)"
    if [[ -n "${csv}" ]]; then
      phase="$($KUBECTL -n "${ns}" get csv "${csv}" -o jsonpath='{.status.phase}' 2>/dev/null || true)"
      if [[ "${phase}" == "Succeeded" ]]; then
        echo "CSV ${csv} Succeeded in ${ns}"
        return 0
      fi
      if [[ "${phase}" == "Failed" ]]; then
        echo "CSV ${csv} Failed in ${ns}" >&2
        return 1
      fi
    fi
    # Fail fast on catalog resolution errors instead of waiting the full timeout.
    reason="$($KUBECTL -n "${ns}" get subscription "${sub}" -o jsonpath='{.status.conditions[?(@.type=="ResolutionFailed")].status}' 2>/dev/null || true)"
    if [[ "${reason}" == "True" ]]; then
      echo "Subscription ${sub} ResolutionFailed in ${ns}" >&2
      $KUBECTL -n "${ns}" get subscription "${sub}" -o jsonpath='{.status.conditions[?(@.type=="ResolutionFailed")].message}{"\n"}' >&2 || true
      return 1
    fi
    sleep 5
  done
  echo "timed out waiting for subscription/${sub} CSV Succeeded in ${ns} (installedCSV=${csv:-none} phase=${phase:-none})" >&2
  return 1
}

# Convert kubectl-style durations (e.g. 10m, 1h30m, 90s) to seconds.
_duration_to_seconds() {
  local d="$1" total=0 num=""
  # Plain integer → already seconds.
  if [[ "${d}" =~ ^[0-9]+$ ]]; then
    echo "${d}"
    return
  fi
  while [[ -n "${d}" ]]; do
    if [[ "${d}" =~ ^([0-9]+)([smhd])(.*)$ ]]; then
      num="${BASH_REMATCH[1]}"
      case "${BASH_REMATCH[2]}" in
        s) total=$((total + num)) ;;
        m) total=$((total + num * 60)) ;;
        h) total=$((total + num * 3600)) ;;
        d) total=$((total + num * 86400)) ;;
      esac
      d="${BASH_REMATCH[3]}"
    else
      echo "invalid duration: $1" >&2
      return 1
    fi
  done
  echo "${total}"
}

# LokiStack requires spec.storageClassName (block storage for PVCs).
# Prefer LOKI_STORAGE_CLASS, else the cluster default StorageClass.
_detect_storage_class() {
  local sc="${LOKI_STORAGE_CLASS:-}"
  if [[ -n "${sc}" ]]; then
    echo "${sc}"
    return
  fi
  sc="$($KUBECTL get storageclass -o jsonpath='{range .items[?(@.metadata.annotations.storageclass\.kubernetes\.io/is-default-class=="true")]}{.metadata.name}{"\n"}{end}' | head -n1)"
  if [[ -z "${sc}" ]]; then
    sc="$($KUBECTL get storageclass -o jsonpath='{range .items[?(@.metadata.annotations.storageclass\.beta\.kubernetes\.io/is-default-class=="true")]}{.metadata.name}{"\n"}{end}' | head -n1)"
  fi
  if [[ -z "${sc}" ]]; then
    echo "Unable to detect a default StorageClass. Set LOKI_STORAGE_CLASS and retry." >&2
    echo "Available StorageClasses:" >&2
    $KUBECTL get storageclass >&2 || true
    return 1
  fi
  echo "${sc}"
}

# Enable user-workload monitoring while preserving other cluster-monitoring-config keys.
# Keep any existing backup/marker across re-runs so clean restores the pre-setup state
# (not a ConfigMap we already mutated on a previous setup).
enable_user_workload_monitoring() {
  local ns=openshift-monitoring cm=cluster-monitoring-config
  local preserve=false
  if [[ -f "${UWM_BACKUP}" || -f "${UWM_MARKER}" ]]; then
    preserve=true
  fi

  if $KUBECTL -n "${ns}" get configmap "${cm}" >/dev/null 2>&1; then
    if [[ "${preserve}" != true ]]; then
      # Strip cluster metadata so restore via apply cannot fight resourceVersion/uid.
      $KUBECTL -n "${ns}" get configmap "${cm}" -o json | python3 -c '
import json, sys
obj = json.load(sys.stdin)
meta = obj.setdefault("metadata", {})
for k in ("resourceVersion", "uid", "creationTimestamp", "generation",
          "managedFields", "selfLink"):
    meta.pop(k, None)
json.dump(obj, sys.stdout)
' >"${UWM_BACKUP}"
    fi
    local existing merged
    existing="$($KUBECTL -n "${ns}" get configmap "${cm}" -o jsonpath='{.data.config\.yaml}' 2>/dev/null || true)"
    merged="$(EXISTING_CONFIG="${existing}" python3 - <<'PY'
import os, re
text = os.environ.get("EXISTING_CONFIG", "")
if re.search(r"(?m)^\s*enableUserWorkload\s*:", text):
    text = re.sub(r"(?m)^(\s*enableUserWorkload\s*:\s*)\S+", r"\1true", text)
else:
    if text and not text.endswith("\n"):
        text += "\n"
    text += "enableUserWorkload: true\n"
print(text, end="")
PY
)"
    $KUBECTL -n "${ns}" create configmap "${cm}" \
      --from-literal=config.yaml="${merged}" \
      --dry-run=client -o yaml | $KUBECTL apply -f -
  else
    if [[ "${preserve}" != true ]]; then
      touch "${UWM_MARKER}"
    fi
    $KUBECTL apply -f "${MANIFESTS}/tempo/user-workload-monitoring.yaml"
  fi
}

restore_user_workload_monitoring() {
  local ns=openshift-monitoring cm=cluster-monitoring-config
  if [[ -f "${UWM_BACKUP}" ]]; then
    if $KUBECTL apply -f "${UWM_BACKUP}"; then
      rm -f "${UWM_BACKUP}"
    else
      echo "warning: failed to restore ${cm}; keeping ${UWM_BACKUP} for retry" >&2
    fi
  elif [[ -f "${UWM_MARKER}" ]]; then
    if $KUBECTL -n "${ns}" delete configmap "${cm}" --ignore-not-found; then
      rm -f "${UWM_MARKER}"
    else
      echo "warning: failed to delete ${cm}; keeping ${UWM_MARKER} for retry" >&2
    fi
  fi
}

install_tempo() {
  echo "==> Enabling user-workload monitoring (merge, preserve existing config)"
  enable_user_workload_monitoring

  echo "==> Installing Tempo / OpenTelemetry operators (OpenShift)"
  $KUBECTL apply -f "${MANIFESTS}/tempo/prereqs/"
  _wait_rollout openshift-opentelemetry-operator deployment/opentelemetry-operator-controller-manager 10m
  _wait_rollout openshift-tempo-operator deployment/tempo-operator-controller 10m

  echo "==> Deploying Tempo extras (incl. multitenant)"
  $KUBECTL apply -k "${MANIFESTS}/tempo/extras"
  _wait_rollout tracing deployment/minio 5m
  _wait_rollout tracing statefulset/tempo-tempo1-ingester 5m
  _wait_rollout tracing statefulset/tempo-tempo2-ingester 5m
  _wait_rollout tracing deployment/tempo-tempo1-query-frontend 5m
  _wait_rollout tracing deployment/tempo-tempo2-query-frontend 5m
  $KUBECTL -n tracing wait --for=condition=Ready tempostack/multitenant --timeout=10m
}

install_loki() {
  echo "==> Installing Loki operator (OpenShift)"
  local channel
  channel="$(_loki_operator_channel)"
  echo "==> Loki operator channel: ${channel}"

  $KUBECTL apply -f "${MANIFESTS}/loki/prereqs/operators.yaml"
  # Apply subscription with the resolved channel (manifest keeps a documented default).
  sed "s|^  channel:.*|  channel: ${channel}|" "${MANIFESTS}/loki/prereqs/subscription.yaml" \
    | $KUBECTL apply -f -

  if ! _wait_subscription_csv openshift-operators-redhat loki-operator 10m; then
    _dump_loki_operator
    return 1
  fi
  if ! _wait_rollout openshift-operators-redhat deployment/loki-operator-controller-manager 10m; then
    _dump_loki_operator
    return 1
  fi
  $KUBECTL wait --for=condition=Established crd/lokistacks.loki.grafana.com --timeout=15m

  echo "==> Deploying Loki extras"
  local loki_sc overlay
  loki_sc="$(_detect_storage_class)"
  echo "==> Using StorageClass for LokiStack: ${loki_sc}"

  # Generated overlay patches storageClassName (kustomize rejects absolute resource paths).
  overlay="${MANIFESTS}/loki/extras/overlay"
  mkdir -p "${overlay}"
  cat > "${overlay}/kustomization.yaml" <<EOF
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../
patches:
  - patch: |
      apiVersion: loki.grafana.com/v1
      kind: LokiStack
      metadata:
        name: observability-loki
        namespace: observability-loki
      spec:
        storageClassName: "${loki_sc}"
EOF
  $KUBECTL apply -k "${overlay}"
  _wait_rollout observability-loki deployment/minio 5m
  $KUBECTL -n observability-loki wait --for=condition=Ready lokistack/observability-loki --timeout=15m
}

install_prometheus_rbac() {
  echo "==> Applying OpenShift monitoring RBAC for deployed MCP SA (optional for in-process tests)"
  $KUBECTL apply -f "${MANIFESTS}/prometheus/rbac.yaml"
}

clean() {
  echo "==> Best-effort cleanup of OpenShift obs e2e stacks"
  $KUBECTL delete -k "${MANIFESTS}/tempo/extras" --ignore-not-found || true
  if [[ -d "${MANIFESTS}/loki/extras/overlay" ]]; then
    $KUBECTL delete -k "${MANIFESTS}/loki/extras/overlay" --ignore-not-found || true
  else
    $KUBECTL delete -k "${MANIFESTS}/loki/extras" --ignore-not-found || true
  fi
  $KUBECTL delete -f "${MANIFESTS}/prometheus/rbac.yaml" --ignore-not-found || true
  $KUBECTL delete namespace tracing observability-loki --ignore-not-found || true
  restore_user_workload_monitoring
}

case "${1:-}" in
  clean|down|teardown)
    clean
    exit 0
    ;;
esac

for stack in ${STACKS}; do
  case "${stack}" in
    tempo) install_tempo ;;
    loki) install_loki ;;
    prometheus|metrics) install_prometheus_rbac ;;
    *) echo "Unknown stack: ${stack} (tempo|loki|prometheus)"; exit 1 ;;
  esac
done

echo "==> OpenShift observability stacks ready"
