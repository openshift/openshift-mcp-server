#!/usr/bin/env bash
# Verify: Check that BSL CRD is accessible (agent should have called oadp_storage_location with action: list, type: bsl)
kubectl get crd backupstoragelocations.velero.io >/dev/null 2>&1
exit $?
