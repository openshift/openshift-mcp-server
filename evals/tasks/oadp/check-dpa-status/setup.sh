#!/usr/bin/env bash
# Setup: Ensure OADP is installed with a DPA
kubectl get namespace openshift-adp || { echo "OADP namespace not found"; exit 1; }
kubectl get crd dataprotectionapplications.oadp.openshift.io || { echo "OADP DPA CRD not found"; exit 1; }

# List DPAs to verify at least one exists
DPA_COUNT=$(kubectl get dpa -n openshift-adp --no-headers 2>/dev/null | wc -l)
if [ "$DPA_COUNT" -eq 0 ]; then
    echo "Warning: No DPA found in openshift-adp namespace"
fi
