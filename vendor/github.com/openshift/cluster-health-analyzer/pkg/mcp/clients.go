package mcp

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/openshift/cluster-health-analyzer/pkg/alertmanager"
	"github.com/openshift/cluster-health-analyzer/pkg/prom"
	"k8s.io/client-go/rest"
)

// NewIncidentLoaders creates HTTPS loaders using the caller's REST credentials
// and the supplied backend CA, rather than the Kubernetes API CA.
func NewIncidentLoaders(restConfig *rest.Config, prometheusURL, alertmanagerURL, caFile string) (prom.Loader, alertmanager.Loader, error) {
	promTransport, err := backendTransport(restConfig, prometheusURL, caFile)
	if err != nil {
		return nil, nil, fmt.Errorf("prometheus transport: %w", err)
	}
	alertTransport, err := backendTransport(restConfig, alertmanagerURL, caFile)
	if err != nil {
		return nil, nil, fmt.Errorf("alertmanager transport: %w", err)
	}
	promLoader, err := prom.NewLoaderWithRoundTripper(prometheusURL, promTransport)
	if err != nil {
		return nil, nil, fmt.Errorf("prometheus client: %w", err)
	}
	alertLoader, err := alertmanager.NewLoader(alertmanager.LoaderConfig{
		AlertManagerURL: alertmanagerURL,
		RoundTripper:    alertTransport,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("alertmanager client: %w", err)
	}
	return promLoader, alertLoader, nil
}

// backendTransport builds an HTTPS transport with the caller's REST credentials
// and the supplied backend CA.
func backendTransport(restConfig *rest.Config, endpoint, caFile string) (http.RoundTripper, error) {
	if restConfig == nil {
		return nil, errors.New("no REST config available")
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	if u.Host == "" || u.Scheme != "https" {
		return nil, fmt.Errorf("backend URL %q must use HTTPS", endpoint)
	}
	cfg := rest.CopyConfig(restConfig)
	cfg.Host = endpoint
	cfg.WrapTransport = nil // The Kubernetes API wrapper must not intercept backend requests.
	cfg.Transport = nil
	cfg.Impersonate = rest.ImpersonationConfig{}
	cfg.Insecure = false
	cfg.CAData = nil
	cfg.CAFile = caFile
	cfg.ServerName = ""
	return rest.TransportFor(cfg)
}
