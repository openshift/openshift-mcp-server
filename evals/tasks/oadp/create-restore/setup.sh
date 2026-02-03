#!/usr/bin/env bash
# Setup: Create a completed backup to restore from
kubectl get namespace openshift-adp || { echo "OADP namespace not found"; exit 1; }
kubectl get crd restores.velero.io || { echo "Velero Restore CRD not found"; exit 1; }

# Create source backup
kubectl apply -f - <<EOF
apiVersion: velero.io/v1
kind: Backup
metadata:
  name: restore-source-backup
  namespace: openshift-adp
spec:
  includedNamespaces:
    - default
  ttl: 1h
EOF

# Wait for backup to complete
kubectl wait --for=jsonpath='{.status.phase}'=Completed backup.velero.io/restore-source-backup -n openshift-adp --timeout=120s || true

# Clean up any existing restore
kubectl delete restore.velero.io app-restore -n openshift-adp --ignore-not-found
