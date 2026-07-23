#!/usr/bin/env bash
set -e

if ! kubectl get nodes -l '!node-role.kubernetes.io/control-plane,!node-role.kubernetes.io/master' --no-headers | grep -q .; then
    echo "Error: No non-control-plane nodes found"
    exit 1
fi

# Create a test namespace with a pod that generates some TCP traffic
kubectl delete namespace tcpdump-test --ignore-not-found
kubectl create namespace tcpdump-test

cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: test-client
  namespace: tcpdump-test
spec:
  securityContext:
    runAsNonRoot: true
    seccompProfile:
      type: RuntimeDefault
  containers:
  - name: nginx
    image: quay.io/nginx/nginx-unprivileged:latest
    ports:
    - containerPort: 8080
    securityContext:
      allowPrivilegeEscalation: false
      capabilities:
        drop:
          - ALL
      runAsNonRoot: true
      seccompProfile:
        type: RuntimeDefault
EOF

# Wait for pod to be ready
kubectl wait --for=condition=Ready pod/test-client -n tcpdump-test --timeout=60s

echo "Setup complete: test pod created"
exit 0
