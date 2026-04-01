#!/usr/bin/env bash
set -euo pipefail

NS=crashloop-test
DEP=app

if ! kubectl rollout status "deployment/${DEP}" -n "${NS}" --timeout=180s; then
  echo "Deployment ${DEP} did not finish rolling out"
  kubectl get pods,deploy -n "${NS}" -o wide 2>/dev/null || true
  exit 1
fi

# replicas=1; take newest pod if list order is messy during RS handoff
pod=$(kubectl get pods -n "${NS}" -l app=crashloop -o name --sort-by=.metadata.creationTimestamp 2>/dev/null | tail -n1 | cut -d/ -f2)
if [[ -z "${pod}" ]]; then
  echo "No pod with label app=crashloop in ${NS}"
  kubectl get pods -n "${NS}" -o wide 2>/dev/null || true
  exit 1
fi

restarts=$(kubectl get pod -n "${NS}" "${pod}" -o jsonpath='{.status.containerStatuses[0].restartCount}' 2>/dev/null || echo "0")
sleep 10
new_restarts=$(kubectl get pod -n "${NS}" "${pod}" -o jsonpath='{.status.containerStatuses[0].restartCount}' 2>/dev/null || echo "0")
if [[ "${restarts}" != "${new_restarts}" ]]; then
  echo "Restart count still increasing (${restarts} -> ${new_restarts}); pod not stable"
  exit 1
fi

exit 0
