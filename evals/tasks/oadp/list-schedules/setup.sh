#!/usr/bin/env bash
# Setup: Create a test schedule
kubectl get namespace openshift-adp || { echo "OADP namespace not found"; exit 1; }
kubectl get crd schedules.velero.io || { echo "Velero Schedule CRD not found"; exit 1; }

kubectl apply -f - <<EOF
apiVersion: velero.io/v1
kind: Schedule
metadata:
  name: eval-daily-schedule
  namespace: openshift-adp
spec:
  schedule: "0 2 * * *"
  template:
    includedNamespaces:
      - default
    ttl: 720h
EOF
