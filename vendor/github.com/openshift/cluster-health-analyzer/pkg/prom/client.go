package prom

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/prometheus/client_golang/api"
	prom_config "github.com/prometheus/common/config"
)

func NewPrometheusClient(prometheusURL string) (api.Client, error) {
	useTls := strings.HasPrefix(prometheusURL, "https://")

	if !useTls {
		return NewPrometheusClientWithToken(prometheusURL, "")
	}
	tokenBytes, err := readTokenFromFile()
	if err != nil {
		slog.Error("Failed to read the service account token", "err", err)
		return nil, err
	}
	token := string(tokenBytes)
	return NewPrometheusClientWithToken(prometheusURL, token)
}

func NewPrometheusClientWithToken(prometheusURL string, token string) (api.Client, error) {
	if !regexp.MustCompile(`^(http|https)://`).MatchString(prometheusURL) {
		return nil, errors.New("invalid URL: must start with https:// or http://")
	}

	api_config := api.Config{
		Address: prometheusURL,
	}

	use_tls := strings.HasPrefix(prometheusURL, "https://")
	if use_tls {
		certs, err := createCertPool()
		if err != nil {
			return nil, err
		}

		defaultRt := api.DefaultRoundTripper.(*http.Transport)
		defaultRt.TLSClientConfig = &tls.Config{RootCAs: certs}

		api_config.RoundTripper = prom_config.NewAuthorizationCredentialsRoundTripper(
			"Bearer", prom_config.NewInlineSecret(token), defaultRt)
	} else {
		slog.Warn("Connecting to Prometheus without TLS")
	}

	return api.NewClient(api_config)
}

// NewPrometheusClientWithRoundTripper creates a client using a supplied
// transport for authentication and TLS.
func NewPrometheusClientWithRoundTripper(prometheusURL string, roundTripper http.RoundTripper) (api.Client, error) {
	if roundTripper == nil {
		return nil, errors.New("round tripper is required")
	}
	if !regexp.MustCompile(`^(http|https)://`).MatchString(prometheusURL) {
		return nil, errors.New("invalid URL: must start with https:// or http://")
	}
	return api.NewClient(api.Config{
		Address:      prometheusURL,
		RoundTripper: roundTripper,
	})
}

func createCertPool() (*x509.CertPool, error) {
	certs := x509.NewCertPool()

	pemData, err := os.ReadFile(`/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt`)
	if err != nil {
		slog.Error("Failed to read the CA certificate", "err", err)
		return nil, err
	}
	certs.AppendCertsFromPEM(pemData)
	return certs, nil
}

func readTokenFromFile() ([]byte, error) {
	return os.ReadFile(`/var/run/secrets/kubernetes.io/serviceaccount/token`)
}
