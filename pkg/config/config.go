package config

import (
	"bytes"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

const (
	ClusterProviderKubeConfig = "kubeconfig"
	ClusterProviderInCluster  = "in-cluster"
	ClusterProviderDisabled   = "disabled"
)

// StaticConfig is the configuration for the server.
// It allows to configure server specific settings and tools to be enabled or disabled.
type StaticConfig struct {
	DeniedResources []GroupVersionKind `toml:"denied_resources"`

	LogLevel   int    `toml:"log_level,omitempty"`
	Port       string `toml:"port,omitempty"`
	SSEBaseURL string `toml:"sse_base_url,omitempty"`
	KubeConfig string `toml:"kubeconfig,omitempty"`
	ListOutput string `toml:"list_output,omitempty"`
	// When true, expose only tools annotated with readOnlyHint=true
	ReadOnly bool `toml:"read_only,omitempty"`
	// When true, disable tools annotated with destructiveHint=true
	DisableDestructive bool     `toml:"disable_destructive,omitempty"`
	Toolsets           []string `toml:"toolsets,omitempty"`
	EnabledTools       []string `toml:"enabled_tools,omitempty"`
	DisabledTools      []string `toml:"disabled_tools,omitempty"`

	// Authorization-related fields
	// RequireOAuth indicates whether the server requires OAuth for authentication.
	RequireOAuth bool `toml:"require_oauth,omitempty"`
	// OAuthAudience is the valid audience for the OAuth tokens, used for offline JWT claim validation.
	OAuthAudience string `toml:"oauth_audience,omitempty"`
	// ValidateToken indicates whether the server should validate the token against the Kubernetes API Server using TokenReview.
	ValidateToken bool `toml:"validate_token,omitempty"`
	// AuthorizationURL is the URL of the OIDC authorization server.
	// It is used for token validation and for STS token exchange.
	AuthorizationURL string `toml:"authorization_url,omitempty"`
	// DisableDynamicClientRegistration indicates whether dynamic client registration is disabled.
	// If true, the .well-known endpoints will not expose the registration endpoint.
	DisableDynamicClientRegistration bool `toml:"disable_dynamic_client_registration,omitempty"`
	// OAuthScopes are the supported **client** scopes requested during the **client/frontend** OAuth flow.
	OAuthScopes []string `toml:"oauth_scopes,omitempty"`
	// StsClientId is the OAuth client ID used for backend token exchange
	StsClientId string `toml:"sts_client_id,omitempty"`
	// StsClientSecret is the OAuth client secret used for backend token exchange
	StsClientSecret string `toml:"sts_client_secret,omitempty"`
	// StsAudience is the audience for the STS token exchange.
	StsAudience string `toml:"sts_audience,omitempty"`
	// StsScopes is the scopes for the STS token exchange.
	StsScopes            []string `toml:"sts_scopes,omitempty"`
	CertificateAuthority string   `toml:"certificate_authority,omitempty"`
	ServerURL            string   `toml:"server_url,omitempty"`
	// ClusterProviderStrategy is how the server finds clusters.
	// If set to "kubeconfig", the clusters will be loaded from those in the kubeconfig.
	// If set to "in-cluster", the server will use the in cluster config
	ClusterProviderStrategy string `toml:"cluster_provider_strategy,omitempty"`

	// ClusterProvider-specific configurations
	// This map holds raw TOML primitives that will be parsed by registered provider parsers
	ClusterProviderConfigs map[string]toml.Primitive `toml:"cluster_provider_configs,omitempty"`

	// Internal: parsed provider configs (not exposed to TOML package)
	parsedClusterProviderConfigs map[string]ProviderConfig
}

func Default() *StaticConfig {
	return &StaticConfig{
		ListOutput: "table",
		Toolsets:   []string{"core", "config", "helm", "openshift-core"},
	}
}

type GroupVersionKind struct {
	Group   string `toml:"group"`
	Version string `toml:"version"`
	Kind    string `toml:"kind,omitempty"`
}

// Read reads the toml file and returns the StaticConfig.
func Read(configPath string) (*StaticConfig, error) {
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	return ReadToml(configData)
}

// ReadToml reads the toml data and returns the StaticConfig.
func ReadToml(configData []byte) (*StaticConfig, error) {
	config := Default()
	md, err := toml.NewDecoder(bytes.NewReader(configData)).Decode(config)
	if err != nil {
		return nil, err
	}

	if err := config.parseClusterProviderConfigs(md); err != nil {
		return nil, err
	}

	return config, nil
}

func (c *StaticConfig) GetProviderConfig(strategy string) (ProviderConfig, bool) {
	config, ok := c.parsedClusterProviderConfigs[strategy]

	return config, ok
}

func (c *StaticConfig) parseClusterProviderConfigs(md toml.MetaData) error {
	if c.parsedClusterProviderConfigs == nil {
		c.parsedClusterProviderConfigs = make(map[string]ProviderConfig, len(c.ClusterProviderConfigs))
	}

	for strategy, primitive := range c.ClusterProviderConfigs {
		parser, ok := getProviderConfigParser(strategy)
		if !ok {
			continue
		}

		providerConfig, err := parser(primitive, md)
		if err != nil {
			return fmt.Errorf("failed to parse config for ClusterProvider '%s': %w", strategy, err)
		}

		if err := providerConfig.Validate(); err != nil {
			return fmt.Errorf("invalid config file for ClusterProvider '%s': %w", strategy, err)
		}

		c.parsedClusterProviderConfigs[strategy] = providerConfig
	}

	return nil
}
