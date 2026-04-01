#!/usr/bin/env bash
set -euo pipefail

# rollout status waits until status matches spec (better than condition=Available during scale-down churn).
NS=scale-down-test
DEP=web-service
TIMEOUT="180s"

want=""
for _ in $(seq 1 90); do
  want=$(kubectl get deployment "$DEP" -n "$NS" -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "")
  [[ "$want" == "1" ]] && break
  sleep 1
done
if [[ "$want" != "1" ]]; then
  echo "Expected spec.replicas=1 after agent scale-down (waited 90s), got '${want}'"
  kubectl get deployment "$DEP" -n "$NS" -o wide 2>/dev/null || true
  exit 1
fi

if ! kubectl rollout status "deployment/$DEP" -n "$NS" --timeout="$TIMEOUT"; then
  echo "Rollout did not finish within $TIMEOUT"
  kubectl get deploy,pods -n "$NS" -o wide 2>/dev/null || true
  exit 1
fi

av="$(kubectl get deployment "$DEP" -n "$NS" -o jsonpath='{.status.availableReplicas}' 2>/dev/null || echo "")"
if [[ "$av" != "1" ]]; then
  echo "Expected status.availableReplicas=1, got '${av}'"
  kubectl get deploy,pods -n "$NS" -o wide 2>/dev/null || true
  exit 1
fi

exit 0
