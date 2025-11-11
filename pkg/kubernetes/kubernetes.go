package kubernetes

import (
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/containers/kubernetes-mcp-server/pkg/helm"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

type HeaderKey string

const (
	CustomAuthorizationHeader = HeaderKey("kubernetes-authorization")
	OAuthAuthorizationHeader  = HeaderKey("Authorization")

	CustomUserAgent = "kubernetes-mcp-server/bearer-token-auth"
)

type CloseWatchKubeConfig func() error

type Kubernetes struct {
	manager *Manager
}

// AccessControlClientset returns the access-controlled clientset
// This ensures that any denied resources configured in the system are properly enforced
func (k *Kubernetes) AccessControlClientset() *AccessControlClientset {
	return k.manager.accessControlClientSet
}

var Scheme = scheme.Scheme
var ParameterCodec = runtime.NewParameterCodec(Scheme)

func (k *Kubernetes) NewHelm() *helm.Helm {
	// This is a derived Kubernetes, so it already has the Helm initialized
	return helm.NewHelm(k.manager)
}

// ToRESTConfig returns the REST configuration from the underlying manager
func (k *Kubernetes) ToRESTConfig() (*rest.Config, error) {
	return k.manager.ToRESTConfig()
}

// GetOrCreateOpenShiftAIClient returns a cached OpenShift AI client instance from the underlying manager
// clientFactory should be a function that creates the OpenShift AI client: func(*rest.Config, interface{}) (interface{}, error)
func (k *Kubernetes) GetOrCreateOpenShiftAIClient(clientFactory func(*rest.Config, interface{}) (interface{}, error)) (interface{}, error) {
	return k.manager.GetOrCreateOpenShiftAIClient(clientFactory)
}
