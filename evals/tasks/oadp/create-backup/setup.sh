#!/usr/bin/env bash
# Setup: Ensure OADP is installed and create the target namespace
kubectl get namespace openshift-adp || { echo "OADP namespace not found"; exit 1; }
kubectl get crd backups.velero.io || { echo "Velero Backup CRD not found"; exit 1; }

# Create a test namespace to back up
kubectl delete namespace oadp-eval-app --ignore-not-found
kubectl create namespace oadp-eval-app
kubectl run nginx --image=nginx --namespace=oadp-eval-app

# Ensure no backup with this name already exists
kubectl delete backup app-backup -n openshift-adp --ignore-not-found
