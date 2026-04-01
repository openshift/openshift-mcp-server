#!/usr/bin/env bash
set -e

# Check if the deployment exists
if ! kubectl get deployment web-app-deployment -n webshop-frontend &>/dev/null; then
  echo "Deployment 'web-app-deployment' does not exist in namespace 'webshop-frontend'"
  exit 1
fi

# Wait for the current Deployment revision to become available. Do not use
# `kubectl wait pods -l app=web-app`: that waits for every Pod with the label,
# including stale ReplicaSet Pods that stay Pending forever after a bad rollout.
echo "Waiting for deployment rollout to finish..."
TIMEOUT="120s"
if ! kubectl rollout status deployment/web-app-deployment -n webshop-frontend --timeout="$TIMEOUT"; then
  echo "Deployment did not become ready after fixing scheduling/workload"
  exit 1
fi

# Verify that the service now has endpoints
ENDPOINTS=$(kubectl get endpoints web-app-service -n webshop-frontend -o jsonpath='{.subsets[0].addresses}')
if [[ -z "$ENDPOINTS" ]]; then
  echo "Service still has no endpoints after fixing the deployment"
  exit 1
fi
