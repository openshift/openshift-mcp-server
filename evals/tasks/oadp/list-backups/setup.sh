#!/usr/bin/env bash
# Setup: Ensure OADP is installed and create a test backup
kubectl get namespace openshift-adp || { echo "OADP namespace not found"; exit 1; }
kubectl get crd backups.velero.io || { echo "Velero Backup CRD not found"; exit 1; }

# Create a test backup for the eval
kubectl apply -f - <<EOF
apiVersion: velero.io/v1
kind: Backup
metadata:
  name: eval-test-backup
  namespace: openshift-adp
spec:
  includedNamespaces:
    - default
  ttl: 1h
EOF
