#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="statefulset-test"
STS_NAME="db"
POD_NAME="db-0"
EXPECTED_CONTENT="initial_data"

if ! kubectl get namespace "${NAMESPACE}" &>/dev/null; then
  echo "Namespace ${NAMESPACE} does not exist"
  exit 1
fi

if [[ $(kubectl get pvc -n "${NAMESPACE}" --no-headers 2>/dev/null | wc -l) -eq 0 ]]; then
  echo "No PVCs in ${NAMESPACE}; StatefulSet volumeClaimTemplates missing or wrong namespace"
  kubectl get sts,pods,pvc -n "${NAMESPACE}" -o wide 2>/dev/null || true
  exit 1
fi

echo "Waiting for all PVCs to be Bound"
if ! kubectl wait pvc --all -n "${NAMESPACE}" --for=jsonpath='{.status.phase}'=Bound --timeout=300s; then
  echo "PVCs did not reach Bound (check StorageClass / provisioner)"
  kubectl get pvc -n "${NAMESPACE}" -o wide 2>/dev/null || true
  exit 1
fi

echo "Ensuring db-1 and db-2 are deleted after scale-down"
for ordinal in 1 2; do
  if kubectl get pod "db-${ordinal}" -n "${NAMESPACE}" &>/dev/null; then
    kubectl wait "pod/db-${ordinal}" -n "${NAMESPACE}" --for=delete --timeout=180s
  fi
done

echo "Waiting for StatefulSet rollout (includes db-0 Ready)"
if ! kubectl rollout status "statefulset/${STS_NAME}" -n "${NAMESPACE}" --timeout=240s; then
  echo "StatefulSet ${STS_NAME} did not stabilize"
  kubectl get pods,sts,pvc -n "${NAMESPACE}" -o wide 2>/dev/null || true
  exit 1
fi

replicas=$(kubectl get sts "${STS_NAME}" -n "${NAMESPACE}" -o jsonpath='{.spec.replicas}')
if [[ "${replicas}" != "1" ]]; then
  echo "Expected spec.replicas=1, got '${replicas}'"
  exit 1
fi

echo "Verifying /data/test in ${POD_NAME}"
data=$(kubectl exec "${POD_NAME}" -n "${NAMESPACE}" -- cat /data/test | tr -d '\r')
if [[ "${data}" != "${EXPECTED_CONTENT}" ]]; then
  echo "Data missing or wrong in ${POD_NAME} (want '${EXPECTED_CONTENT}', got '${data}')"
  exit 1
fi
