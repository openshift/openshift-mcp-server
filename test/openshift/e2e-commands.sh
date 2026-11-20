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

# Observability: install Tempo/Loki for shared TestObs*, run TestObs*, then
# OpenShift-only Route discovery / CMO metric checks. Always tear stacks down.
cleanup_observability_e2e() {
    echo "==> Cleaning up observability e2e stacks"
    make clean-observability-e2e || true
}
trap cleanup_observability_e2e EXIT

# Install stacks individually so a missing operator (e.g. loki-operator not in
# the OCP catalog) or a partial rollout does not block the entire job.
if ! ./hack/e2e/observability/setup-openshift.sh tempo; then
    echo "WARNING: Tempo setup failed; Tempo TestObs* cases will be skipped"
    export SKIP_OBS_TEMPO=1
fi
if ! ./hack/e2e/observability/setup-openshift.sh loki; then
    echo "WARNING: Loki setup failed; Loki TestObs* cases will be skipped"
    export SKIP_OBS_LOKI=1
fi

make e2e-ci-test-obs
make test-observability-e2e-routes
