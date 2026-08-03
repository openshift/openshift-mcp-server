#!/usr/bin/env bash
set -euo pipefail

kubectl get pods -n openshift-ovn-kubernetes -l app=ovnkube-node --field-selector=status.phase=Running -o name | grep -q . || {
  echo "ERROR: No running ovnkube-node pods found"
  exit 1
}
