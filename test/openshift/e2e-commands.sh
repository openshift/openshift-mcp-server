#!/bin/bash
set -euo pipefail

export KUBECONFIG="${SHARED_DIR}/kubeconfig"

# The ipi-aws workflow provides oc; the e2e suite expects kubectl
if ! command -v kubectl >/dev/null 2>&1; then
    mkdir -p /tmp/bin
    ln -s "$(command -v oc)" /tmp/bin/kubectl
    export PATH="/tmp/bin:${PATH}"
fi

# The test pod runs in the build-farm cluster which sets in-cluster env vars.
# The MCP server would auto-detect these and talk to the build farm instead of
# the IPI cluster. Unset them so it falls back to KUBECONFIG.
unset KUBERNETES_SERVICE_HOST KUBERNETES_SERVICE_PORT

export MCP_SERVER_IMAGE="${IMAGE_OPENSHIFT_MCP_SERVER}"

make e2e-ci-setup
make e2e-ci-test

# Observability e2e: install Tempo/Loki (+ platform monitoring already present),
# run the full suite, always tear stacks down (ephemeral CI cluster).
cleanup_observability_e2e() {
    echo "==> Cleaning up observability e2e stacks"
    make clean-observability-e2e || true
}
trap cleanup_observability_e2e EXIT

# Install stacks individually so a missing operator (e.g. loki-operator not in
# the OCP catalog) does not block the entire job.  The Go tests call
# requireTempoStacks()/requireLokiStack() and gracefully skip when the
# corresponding CRDs or instances are absent.
./hack/e2e/observability/setup-openshift.sh tempo || echo "WARNING: Tempo setup failed; Tempo tests will be skipped"
./hack/e2e/observability/setup-openshift.sh loki  || echo "WARNING: Loki setup failed; Loki tests will be skipped"

make test-observability-e2e
