package prometheus

import (
	"crypto/tls"
	"crypto/x509"
	"os"
	"strings"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

// ClientOption is a function that configures a Client.
type ClientOption func(*Client)

// WithBearerToken sets the bearer token for authentication.
func WithBearerToken(token string) ClientOption {
	return func(c *Client) {
		c.bearerToken = strings.TrimSpace(token)
	}
}

// WithBearerTokenFromRESTConfig extracts and sets the bearer token from a Kubernetes REST config.
// It tries the token directly first, then falls back to reading from a token file.
func WithBearerTokenFromRESTConfig(config *rest.Config) ClientOption {
	return func(c *Client) {
		if config == nil {
			return
		}

		// Try bearer token directly
		if config.BearerToken != "" {
			c.bearerToken = config.BearerToken
			return
		}

		// Try bearer token file
		if config.BearerTokenFile != "" {
			token, err := os.ReadFile(config.BearerTokenFile)
			if err != nil {
				klog.V(2).Infof("Failed to read token file %s: %v", config.BearerTokenFile, err)
				return
			}
			c.bearerToken = strings.TrimSpace(string(token))
		}
	}
}

// WithTLSFromRESTConfig configures TLS using the CA from a Kubernetes REST config.
// It tries CAData first, then CAFile, then system cert pool, and finally falls back to insecure.
func WithTLSFromRESTConfig(config *rest.Config) ClientOption {
	return func(c *Client) {
		if config == nil {
			return
		}

		// Try to build a cert pool with the cluster CA
		var certPool *x509.CertPool
		var caLoaded bool

		// First, try to load CA from REST config's CAData
		if len(config.CAData) > 0 {
			// Start with system cert pool if available
			if systemPool, err := x509.SystemCertPool(); err == nil && systemPool != nil {
				certPool = systemPool
			} else {
				certPool = x509.NewCertPool()
			}
			if ok := certPool.AppendCertsFromPEM(config.CAData); ok {
				c.tlsConfig.RootCAs = certPool
				caLoaded = true
				klog.V(4).Info("Loaded cluster CA from REST config CAData")
			} else {
				klog.V(2).Info("Failed to parse CA certificates from REST config CAData")
			}
		}

		// If CAData wasn't available or didn't work, try CAFile
		if !caLoaded && config.CAFile != "" {
			caPEM, err := os.ReadFile(config.CAFile)
			if err != nil {
				klog.V(2).Infof("Failed to read CA file %s: %v", config.CAFile, err)
			} else {
				// Start with system cert pool if available
				if systemPool, err := x509.SystemCertPool(); err == nil && systemPool != nil {
					certPool = systemPool
				} else {
					certPool = x509.NewCertPool()
				}
				if ok := certPool.AppendCertsFromPEM(caPEM); ok {
					c.tlsConfig.RootCAs = certPool
					caLoaded = true
					klog.V(4).Infof("Loaded cluster CA from file %s", config.CAFile)
				} else {
					klog.V(2).Infof("Failed to parse CA certificates from file %s", config.CAFile)
				}
			}
		}

		// If no CA was loaded, try system cert pool alone (for routes with public CAs)
		if !caLoaded {
			if systemPool, err := x509.SystemCertPool(); err == nil && systemPool != nil {
				c.tlsConfig.RootCAs = systemPool
				klog.V(4).Info("Using system certificate pool for TLS verification")
			} else {
				// Last resort: skip verification with a warning
				klog.Warning("No cluster CA available and system cert pool failed; using insecure TLS (skip verification)")
				c.tlsConfig.InsecureSkipVerify = true
			}
		}
	}
}

// WithCustomCA configures TLS using a custom CA certificate file.
func WithCustomCA(caFile string) ClientOption {
	return func(c *Client) {
		caFile = strings.TrimSpace(caFile)
		if caFile == "" {
			return
		}

		caPEM, err := os.ReadFile(caFile)
		if err != nil {
			klog.Errorf("Failed to read CA certificate from file %s: %v; proceeding without custom CA", caFile, err)
			return
		}

		// Start with the host system pool when possible so we don't drop system roots
		var certPool *x509.CertPool
		if systemPool, err := x509.SystemCertPool(); err == nil && systemPool != nil {
			certPool = systemPool
		} else {
			certPool = x509.NewCertPool()
		}
		if ok := certPool.AppendCertsFromPEM(caPEM); ok {
			c.tlsConfig.RootCAs = certPool
		} else {
			klog.V(0).Infof("Failed to append provided certificate authority; proceeding without custom CA")
		}
	}
}

// WithInsecure configures whether to skip TLS verification.
func WithInsecure(insecure bool) ClientOption {
	return func(c *Client) {
		c.tlsConfig.InsecureSkipVerify = insecure
	}
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.timeout = timeout
	}
}

// newDefaultTLSConfig creates a default TLS configuration.
func newDefaultTLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
}
