#!/usr/bin/env bash
# Setup: Ensure OADP is installed
kubectl get namespace openshift-adp || { echo "OADP namespace not found"; exit 1; }
kubectl get crd backupstoragelocations.velero.io || { echo "Velero BSL CRD not found"; exit 1; }

# OADP DPA should have created at least one BSL, but verify we can list them
echo "Checking for existing BSLs..."
kubectl get backupstoragelocations -n openshift-adp
