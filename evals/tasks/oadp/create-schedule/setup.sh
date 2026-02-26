#!/usr/bin/env bash
# Setup: Ensure OADP is installed
kubectl get namespace openshift-adp || { echo "OADP namespace not found"; exit 1; }
kubectl get crd schedules.velero.io || { echo "Velero Schedule CRD not found"; exit 1; }

# Create target namespace
kubectl create namespace production --dry-run=client -o yaml | kubectl apply -f -

# Clean up any existing schedule with this name
kubectl delete schedule nightly-backup -n openshift-adp --ignore-not-found
