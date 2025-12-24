package externalsecrets

import (
	"slices"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	internalk8s "github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets"
)

// Toolset provides tools for managing the External Secrets Operator for Red Hat OpenShift.
// This includes operator installation, configuration, SecretStore/ExternalSecret management,
// debugging, and status monitoring.
//
// References:
// - https://docs.redhat.com/en/documentation/openshift_container_platform/latest/html/security_and_compliance/external-secrets-operator-for-red-hat-openshift
// - https://external-secrets.io/latest/
// - https://github.com/openshift/external-secrets-operator
// - https://github.com/openshift/external-secrets
type Toolset struct{}

var _ api.Toolset = (*Toolset)(nil)

func (t *Toolset) GetName() string {
	return "external-secrets"
}

func (t *Toolset) GetDescription() string {
	return "Tools for managing External Secrets Operator for Red Hat OpenShift - operator installation, configuration, SecretStore/ExternalSecret management, and debugging"
}

func (t *Toolset) GetTools(_ internalk8s.Openshift) []api.ServerTool {
	return slices.Concat(
		initOperatorTools(),
		initStoreTools(),
		initSecretTools(),
		initStatusTools(),
	)
}

func init() {
	toolsets.Register(&Toolset{})
}
