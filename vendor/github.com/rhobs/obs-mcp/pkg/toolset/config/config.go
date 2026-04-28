package config

import (
	"context"
	"fmt"

	"github.com/BurntSushi/toml"
	"github.com/containers/kubernetes-mcp-server/pkg/api"
	serverconfig "github.com/containers/kubernetes-mcp-server/pkg/config"

	"github.com/rhobs/obs-mcp/pkg/prometheus"
)

const MetricsToolSetName = "metrics"

// AuthMode defines where the bearer token is obtained for authenticating
// against Prometheus and Alertmanager endpoints.
type AuthMode string

const (
	// AuthModeHeader reads the bearer token from the request context (authorization header).
	// This is the default.
	AuthModeHeader AuthMode = "header"

	// AuthModeKubeConfig reads the bearer token from the kubeconfig/REST config only.
	AuthModeKubeConfig AuthMode = "kubeconfig"
)

// Config holds obs-mcp toolset configuration
type Config struct {
	// AuthMode controls where the bearer token is obtained for authenticating
	// against Prometheus and Alertmanager endpoints.
	// Valid values: "header" (default) - read from the request context authorization header,
	//              "kubeconfig" - read from the kubeconfig/REST config.
	AuthMode AuthMode `toml:"auth_mode,omitempty"`
	// PrometheusURL is the URL of the Prometheus/Thanos Querier endpoint.
	// This field is required. Example: "https://thanos-querier-openshift-monitoring.apps.example.com"
	PrometheusURL string `toml:"prometheus_url,omitempty"`

	// AlertmanagerURL is the URL of the Alertmanager endpoint.
	// This field is optional. Example: "https://alertmanager-main-openshift-monitoring.apps.example.com"
	AlertmanagerURL string `toml:"alertmanager_url,omitempty"`

	// Insecure controls whether to skip TLS certificate verification.
	// Default: false (verify certificates)
	Insecure bool `toml:"insecure,omitempty"`

	// Guardrails controls which query safety checks are enabled.
	// Valid values: "all" (default), "none", or comma-separated list of:
	//   - "disallow-explicit-name-label"
	//   - "require-label-matcher"
	//   - "disallow-blanket-regex"
	Guardrails string `toml:"guardrails,omitempty"`

	// MaxMetricCardinality is the maximum allowed series count per metric.
	// Set to 0 to disable this check.
	// Default: 20000
	MaxMetricCardinality uint64 `toml:"max_metric_cardinality,omitempty"`

	// MaxLabelCardinality is the maximum allowed label value count for blanket regex.
	// Only takes effect if disallow-blanket-regex is enabled.
	// Set to 0 to always disallow blanket regex.
	// Default: 500
	MaxLabelCardinality uint64 `toml:"max_label_cardinality,omitempty"`

	// RangeQueryFullResponse controls whether range queries return full data points
	// instead of summary statistics.
	// Default: false (return summary statistics)
	RangeQueryFullResponse bool `toml:"range_query_full_response,omitempty"`
}

var _ api.ExtendedConfig = (*Config)(nil)

// Validate checks that the configuration values are valid.
func (c *Config) Validate() error {
	if c.AuthMode != "" && c.AuthMode != AuthModeHeader && c.AuthMode != AuthModeKubeConfig {
		return fmt.Errorf("invalid auth_mode: %q (valid options: %q, %q)", c.AuthMode, AuthModeHeader, AuthModeKubeConfig)
	}

	if c.Guardrails != "" {
		_, err := prometheus.ParseGuardrails(c.Guardrails)
		if err != nil {
			return fmt.Errorf("invalid guardrails configuration: %w", err)
		}
	}

	return nil
}

// GetAuthMode returns the configured token source, defaulting to TokenSourceHeader.
func (c *Config) GetAuthMode() AuthMode {
	if c.AuthMode == "" {
		return AuthModeHeader
	}
	return c.AuthMode
}

// GetGuardrails returns the parsed guardrails configuration with cardinality limits applied.
func (c *Config) GetGuardrails() (*prometheus.Guardrails, error) {
	guardrailsStr := c.Guardrails
	if guardrailsStr == "" {
		guardrailsStr = "all" // default
	}

	guardrails, err := prometheus.ParseGuardrails(guardrailsStr)
	if err != nil {
		return nil, err
	}

	if guardrails != nil {
		// Apply cardinality limits
		maxMetricCard := c.MaxMetricCardinality
		if maxMetricCard == 0 {
			maxMetricCard = 20000 // default
		}
		guardrails.MaxMetricCardinality = maxMetricCard

		maxLabelCard := c.MaxLabelCardinality
		if maxLabelCard == 0 {
			maxLabelCard = 500 // default
		}
		guardrails.MaxLabelCardinality = maxLabelCard
	}

	return guardrails, nil
}

func obsMCPToolsetParser(_ context.Context, primitive toml.Primitive, md toml.MetaData) (api.ExtendedConfig, error) {
	var cfg Config
	if err := md.PrimitiveDecode(primitive, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func init() {
	serverconfig.RegisterToolsetConfig(MetricsToolSetName, obsMCPToolsetParser)
}
