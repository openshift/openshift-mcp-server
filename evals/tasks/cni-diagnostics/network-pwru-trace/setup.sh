#!/usr/bin/env bash
set -e

if ! kubectl get nodes -l '!node-role.kubernetes.io/control-plane,!node-role.kubernetes.io/master' --no-headers | grep -q .; then
    echo "Error: No non-control-plane nodes found"
    exit 1
fi

# Create a test pod to generate traffic
kubectl delete namespace pwru-test --ignore-not-found
kubectl create namespace pwru-test

cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: traced-pod
  namespace: pwru-test
spec:
  securityContext:
    runAsNonRoot: true
    seccompProfile:
      type: RuntimeDefault
  containers:
  - name: busybox
    image: quay.io/quay/busybox:latest
    command: ["sleep", "3600"]
    securityContext:
      allowPrivilegeEscalation: false
      capabilities:
        drop:
          - ALL
      runAsNonRoot: true
      seccompProfile:
        type: RuntimeDefault
EOF

kubectl wait --for=condition=Ready pod/traced-pod -n pwru-test --timeout=60s

# Get pod IP for tracing
POD_IP=$(kubectl get pod traced-pod -n pwru-test -o jsonpath='{.status.podIP}')
echo "Pod IP for tracing: $POD_IP"

echo "Setup complete: test pod created for pwru tracing"
exit 0
