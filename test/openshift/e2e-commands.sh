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

bash test/openshift/keycloak-setup.sh

# Source Keycloak env vars if setup produced them (skipped on older clusters)
if [[ -f /tmp/keycloak-env.sh ]]; then
    # shellcheck disable=SC1091
    source /tmp/keycloak-env.sh
fi

make e2e-ci-setup
make e2e-ci-test
