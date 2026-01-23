package netedge

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

type NetEdgeClient struct {
	bearerToken          string
	prometheusURL        string
	prometheusInsecure   bool
	certificateAuthority string
}

// NewNetEdgeClient creates a new NetEdgeClient instance
func NewNetEdgeClient(configProvider api.ExtendedConfigProvider, kubernetes *rest.Config) *NetEdgeClient {
	client := &NetEdgeClient{bearerToken: kubernetes.BearerToken}

	// TODO: We might want a specific config section for netedge later,
	// for now we can assume we might get some config or default to standard prometheus locations
	// but following the kiali pattern:
	if cfg, ok := configProvider.GetToolsetConfig("netedge"); ok {
		if nc, ok := cfg.(*Config); ok && nc != nil {
			client.prometheusURL = nc.PrometheusURL
			client.prometheusInsecure = nc.PrometheusInsecure
			client.certificateAuthority = nc.CertificateAuthority
		}
	}
	return client
}

func (c *NetEdgeClient) createHTTPClient() *http.Client {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: c.prometheusInsecure,
	}

	if caValue := strings.TrimSpace(c.certificateAuthority); caValue != "" {
		caPEM, err := os.ReadFile(caValue)
		if err != nil {
			klog.Errorf("failed to read CA certificate from file %s: %v; proceeding without custom CA", caValue, err)
		} else {
			var certPool *x509.CertPool
			if systemPool, err := x509.SystemCertPool(); err == nil && systemPool != nil {
				certPool = systemPool
			} else {
				certPool = x509.NewCertPool()
			}
			if ok := certPool.AppendCertsFromPEM(caPEM); ok {
				tlsConfig.RootCAs = certPool
			}
		}
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}
}

func (c *NetEdgeClient) authorizationHeader() string {
	if c == nil {
		return ""
	}
	token := strings.TrimSpace(c.bearerToken)
	if token == "" {
		return ""
	}
	if strings.HasPrefix(token, "Bearer ") {
		return token
	}
	return "Bearer " + token
}

func (c *NetEdgeClient) executeRequest(ctx context.Context, method, endpoint string, queryParams map[string]string) (string, error) {
	// Construct URL
	// TODO: Discovery of Prometheus URL. For now, assuming standard OCP monitoring location or passed in config
	// If not set, we might default to the openshift-monitoring service URL if we are in-cluster

	// Validating base URL
	baseURLStr := c.prometheusURL
	if baseURLStr == "" {
		// Fallback or error? For now error if not configured, or maybe dynamic discovery?
		// In OCP, it's typically https://prometheus-k8s.openshift-monitoring.svc:9091
		// But we need to be careful about accessibility.
		return "", fmt.Errorf("prometheus URL not configured")
	}

	u, err := url.Parse(baseURLStr)
	if err != nil {
		return "", fmt.Errorf("invalid prometheus base URL: %w", err)
	}

	// Join path
	u.Path, err = url.JoinPath(u.Path, endpoint)
	if err != nil {
		return "", fmt.Errorf("failed to join path: %w", err)
	}

	// Add query params
	q := u.Query()
	for k, v := range queryParams {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	klog.V(4).Infof("NetEdge API call: %s %s", method, u.String())

	req, err := http.NewRequestWithContext(ctx, method, u.String(), nil)
	if err != nil {
		return "", err
	}

	authHeader := c.authorizationHeader()
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	client := c.createHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("netedge API error status %d: %s", resp.StatusCode, string(respBody))
	}

	return string(respBody), nil
}
