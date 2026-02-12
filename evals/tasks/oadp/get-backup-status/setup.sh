#!/usr/bin/env bash
# Setup: Ensure OADP is installed and create a test backup
kubectl get namespace openshift-adp || { echo "OADP namespace not found"; exit 1; }

# Create a test backup for status check
kubectl apply -f - <<EOF
apiVersion: velero.io/v1
kind: Backup
metadata:
  name: status-check-backup
  namespace: openshift-adp
spec:
  includedNamespaces:
    - default
  ttl: 1h
EOF

# Wait for backup to have some status
sleep 5
