package alertmanager

//go:generate mockgen -package=mocks -mock_names=Loader=MockAlertManagerLoader -source=loader.go -destination=../test/mocks/mock_alertmanager_loader.go

import (
	"crypto/tls"
	"crypto/x509"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	runtimeclient "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/openshift/cluster-health-analyzer/pkg/utils"
	"github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/alertmanager/api/v2/client/alert"
	"github.com/prometheus/alertmanager/api/v2/models"
	"github.com/prometheus/client_golang/api"
	prom_config "github.com/prometheus/common/config"
)

// Loader reads alerts from the Alertmanager API
type Loader interface {
	// ActiveAlert reads the active alerts from the Alertmanager
	// and returns them as a slice.
	ActiveAlerts() ([]models.Alert, error)
	// ActiveAlert reads the active alerts with the provided labels from the Alertmanager
	// and returns them as a slice.
	ActiveAlertsWithLabels(labels []string) ([]models.Alert, error)
	// SilencedAlerts reads silenced alerts from the Alertmanager
	// and returns them as a slice
	SilencedAlerts() ([]models.Alert, error)
}

type LoaderConfig struct {
	AlertManagerURL string
	Token           string
	RoundTripper    http.RoundTripper
}

type loader struct {
	cli *client.AlertmanagerAPI
}

// NewLoader creates a new client Alermanager API
func NewLoader(cfg LoaderConfig) (Loader, error) {
	amURL, err := url.Parse(cfg.AlertManagerURL)
	if err != nil {
		return nil, err
	}
	useTls := strings.HasPrefix(cfg.AlertManagerURL, "https://")

	runtime := runtimeclient.New(amURL.Host, path.Join(amURL.Path, "/api/v2"), []string{amURL.Scheme})
	if cfg.RoundTripper != nil {
		runtime.Transport = cfg.RoundTripper
		return &loader{cli: client.New(runtime, strfmt.Default)}, nil
	}
	if useTls {

		token := cfg.Token

		if token == "" {
			tokenBytes, err := readTokenFromFile()
			if err != nil {
				return nil, err
			}
			token = string(tokenBytes)
		}

		amClient, err := initTlsAlertManagerClient(runtime, token)
		if err != nil {
			return nil, err
		}

		return &loader{
			cli: amClient,
		}, nil
	}
	return &loader{cli: client.New(runtime, strfmt.Default)}, nil
}

func initTlsAlertManagerClient(runtime *runtimeclient.Runtime, token string) (*client.AlertmanagerAPI, error) {
	certs, err := createCertPool()
	if err != nil {
		return nil, err
	}
	defaultRt := api.DefaultRoundTripper.(*http.Transport)
	defaultRt.TLSClientConfig = &tls.Config{RootCAs: certs}

	runtime.Transport = prom_config.NewAuthorizationCredentialsRoundTripper(
		"Bearer", prom_config.NewInlineSecret(token), defaultRt)
	return client.New(runtime, strfmt.Default), nil
}

// ActiveAlert reads the active alerts from the Alertmanager
// and returns them as a slice.
func (l *loader) ActiveAlerts() ([]models.Alert, error) {
	return l.loadAlerts(true, false, nil)
}

// ActiveAlert reads the active alerts with the provided labels from the Alertmanager
// and returns them as a slice.
func (l *loader) ActiveAlertsWithLabels(labels []string) ([]models.Alert, error) {
	return l.loadAlerts(true, false, labels)
}

// SilencedAlerts reads silenced alerts from the Alertmanager
// and returns them as a slice
func (l *loader) SilencedAlerts() ([]models.Alert, error) {
	return l.loadAlerts(false, true, nil)
}

// loadAlerts queries the alertmanager with the provided parameters
func (l *loader) loadAlerts(active, silenced bool, labels []string) ([]models.Alert, error) {
	params := alert.NewGetAlertsParams().
		WithActive(&active).
		WithSilenced(&silenced).
		WithInhibited(utils.Ptr(false)).
		WithUnprocessed(utils.Ptr(false)).
		WithFilter(labels)

	alertsOK, err := l.cli.Alert.GetAlerts(params)
	if err != nil {
		return nil, err
	}
	var alerts []models.Alert
	for _, gettableAlert := range alertsOK.Payload {
		alerts = append(alerts, gettableAlert.Alert)
	}

	return alerts, nil
}

func createCertPool() (*x509.CertPool, error) {
	certs := x509.NewCertPool()

	pemData, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt")
	if err != nil {
		slog.Error("Failed to read the CA certificate", "err", err)
		return nil, err
	}
	certs.AppendCertsFromPEM(pemData)
	return certs, nil
}

func readTokenFromFile() ([]byte, error) {
	return os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
}
