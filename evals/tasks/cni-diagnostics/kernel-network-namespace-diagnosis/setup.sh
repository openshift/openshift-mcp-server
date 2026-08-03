#!/usr/bin/env bash
set -e

WORKER_NODE=$(kubectl get nodes -l '!node-role.kubernetes.io/control-plane,!node-role.kubernetes.io/master' -o jsonpath='{.items[0].metadata.name}')
if [ -z "$WORKER_NODE" ]; then
    echo "Error: No non-control-plane nodes found"
    exit 1
fi

# Create test namespace and pod
kubectl delete namespace netns-test --ignore-not-found
kubectl create namespace netns-test

cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: external-test
  namespace: netns-test
spec:
  nodeName: ${WORKER_NODE}
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

kubectl wait --for=condition=Ready pod/external-test -n netns-test --timeout=60s

echo "Setup complete: test pod scheduled on worker node ${WORKER_NODE}"
exit 0
