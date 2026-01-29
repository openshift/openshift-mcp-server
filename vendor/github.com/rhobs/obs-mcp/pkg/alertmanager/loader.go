package alertmanager

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/alertmanager/api/v2/client/alert"
	"github.com/prometheus/alertmanager/api/v2/client/silence"
	"github.com/prometheus/alertmanager/api/v2/models"
	"github.com/prometheus/client_golang/api"
)

// Loader defines the interface for querying Alertmanager
type Loader interface {
	GetAlerts(ctx context.Context, active, silenced, inhibited, unprocessed *bool, filter []string, receiver string) (models.GettableAlerts, error)
	GetSilences(ctx context.Context, filter []string) (models.GettableSilences, error)
}

// RealLoader implements Loader
type RealLoader struct {
	client *client.AlertmanagerAPI
}

// Ensure RealLoader implements Loader at compile time
var _ Loader = (*RealLoader)(nil)

func NewAlertmanagerClient(apiConfig api.Config) (*RealLoader, error) {
	// Parse the URL to extract scheme and host
	parsedURL, err := url.Parse(apiConfig.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Alertmanager URL: %w", err)
	}

	host := parsedURL.Host
	if host == "" {
		host = strings.TrimPrefix(apiConfig.Address, "//")
	}

	scheme := parsedURL.Scheme
	if scheme == "" {
		scheme = "http"
	}

	cfg := client.DefaultTransportConfig().
		WithHost(host).
		WithSchemes([]string{scheme})

	c := client.NewHTTPClientWithConfig(nil, cfg)

	return &RealLoader{
		client: c,
	}, nil
}

func (a *RealLoader) GetAlerts(ctx context.Context, active, silenced, inhibited, unprocessed *bool, filter []string, receiver string) (models.GettableAlerts, error) {
	params := alert.NewGetAlertsParams().WithContext(ctx)

	if active != nil {
		params = params.WithActive(active)
	}
	if silenced != nil {
		params = params.WithSilenced(silenced)
	}
	if inhibited != nil {
		params = params.WithInhibited(inhibited)
	}
	if unprocessed != nil {
		params = params.WithUnprocessed(unprocessed)
	}
	if len(filter) > 0 {
		params = params.WithFilter(filter)
	}
	if receiver != "" {
		params = params.WithReceiver(&receiver)
	}

	resp, err := a.client.Alert.GetAlerts(params)
	if err != nil {
		return nil, fmt.Errorf("error fetching alerts: %w", err)
	}

	return resp.Payload, nil
}

func (a *RealLoader) GetSilences(ctx context.Context, filter []string) (models.GettableSilences, error) {
	params := silence.NewGetSilencesParams().WithContext(ctx)

	if len(filter) > 0 {
		params = params.WithFilter(filter)
	}

	resp, err := a.client.Silence.GetSilences(params)
	if err != nil {
		return nil, fmt.Errorf("error fetching silences: %w", err)
	}

	return resp.Payload, nil
}
