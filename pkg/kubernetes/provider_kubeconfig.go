package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/BurntSushi/toml"
	"golang.org/x/oauth2"

	"github.com/containers/kubernetes-mcp-server/pkg/config"
	"github.com/containers/kubernetes-mcp-server/pkg/sts"
	authenticationv1api "k8s.io/api/authentication/v1"
	"k8s.io/klog/v2"
)

// KubeConfigTargetParameterName is the parameter name used to specify
// the kubeconfig context when using the kubeconfig cluster provider strategy.
const KubeConfigTargetParameterName = "context"

// KubeConfigProviderConfig holds kubeconfig-specific configuration
type KubeConfigProviderConfig struct {
	// TokenExchangeStrategy specifies which token exchange protocol to use
	// Valid values: "keycloak-v1", "rfc8693", "external-account"
	// Default: "" (no per-context token exchange)
	TokenExchangeStrategy string `toml:"token_exchange_strategy,omitempty"`

	// Contexts holds per-context token exchange configuration
	// The key is the context name from the kubeconfig file
	Contexts map[string]sts.TargetSTSConfig `toml:"contexts,omitempty"`
}

func (c *KubeConfigProviderConfig) Validate() error {
	return nil
}

func parseKubeConfigProviderConfig(_ context.Context, primitive toml.Primitive, md toml.MetaData) (config.ProviderConfig, error) {
	cfg := &KubeConfigProviderConfig{}
	if err := md.PrimitiveDecode(primitive, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// kubeConfigClusterProvider implements Provider for managing multiple
// Kubernetes clusters using different contexts from a kubeconfig file.
// It lazily initializes managers for each context as they are requested.
type kubeConfigClusterProvider struct {
	defaultContext string
	managers       map[string]*Manager

	// Per-context token exchange configuration
	tokenExchanger    sts.TokenExchanger
	contextSTSConfigs map[string]sts.TargetSTSConfig
	httpClient        *http.Client
}

var _ Provider = &kubeConfigClusterProvider{}

func init() {
	RegisterProvider(config.ClusterProviderKubeConfig, newKubeConfigClusterProvider)
	config.RegisterProviderConfig(config.ClusterProviderKubeConfig, parseKubeConfigProviderConfig)
}

// newKubeConfigClusterProvider creates a provider that manages multiple clusters
// via kubeconfig contexts.
// Internally, it leverages a KubeconfigManager for each context, initializing them
// lazily when requested.
func newKubeConfigClusterProvider(cfg *config.StaticConfig) (Provider, error) {
	m, err := NewKubeconfigManager(cfg, "")
	if err != nil {
		if errors.Is(err, ErrorKubeconfigInClusterNotAllowed) {
			return nil, fmt.Errorf("kubeconfig ClusterProviderStrategy is invalid for in-cluster deployments: %v", err)
		}
		return nil, err
	}

	rawConfig, err := m.clientCmdConfig.RawConfig()
	if err != nil {
		return nil, err
	}

	allClusterManagers := map[string]*Manager{
		rawConfig.CurrentContext: m, // we already initialized a manager for the default context, let's use it
	}

	for name := range rawConfig.Contexts {
		if name == rawConfig.CurrentContext {
			continue // already initialized this, don't want to set it to nil
		}

		allClusterManagers[name] = nil
	}

	provider := &kubeConfigClusterProvider{
		defaultContext: rawConfig.CurrentContext,
		managers:       allClusterManagers,
	}

	// Initialize per-context token exchange if configured
	providerCfg, ok := cfg.GetProviderConfig(config.ClusterProviderKubeConfig)
	if ok {
		kubeConfigCfg := providerCfg.(*KubeConfigProviderConfig)
		if kubeConfigCfg.TokenExchangeStrategy != "" {
			tokenExchanger, err := sts.GetTokenExchanger(kubeConfigCfg.TokenExchangeStrategy)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize token exchanger: %w", err)
			}
			provider.tokenExchanger = tokenExchanger
			provider.contextSTSConfigs = kubeConfigCfg.Contexts
			provider.httpClient = &http.Client{
				Timeout: 30 * time.Second,
			}
			klog.V(2).Infof("Per-context token exchange enabled with strategy: %s", kubeConfigCfg.TokenExchangeStrategy)
		}
	}

	return provider, nil
}

func (p *kubeConfigClusterProvider) managerForContext(context string) (*Manager, error) {
	m, ok := p.managers[context]
	if ok && m != nil {
		return m, nil
	}

	baseManager := p.managers[p.defaultContext]

	m, err := NewKubeconfigManager(baseManager.staticConfig, context)
	if err != nil {
		return nil, err
	}

	p.managers[context] = m

	return m, nil
}

func (p *kubeConfigClusterProvider) IsOpenShift(ctx context.Context) bool {
	return p.managers[p.defaultContext].IsOpenShift(ctx)
}

func (p *kubeConfigClusterProvider) VerifyToken(ctx context.Context, context, token, audience string) (*authenticationv1api.UserInfo, []string, error) {
	m, err := p.managerForContext(context)
	if err != nil {
		return nil, nil, err
	}
	return m.VerifyToken(ctx, token, audience)
}

func (p *kubeConfigClusterProvider) GetTargets(_ context.Context) ([]string, error) {
	contextNames := make([]string, 0, len(p.managers))
	for contextName := range p.managers {
		contextNames = append(contextNames, contextName)
	}

	return contextNames, nil
}

func (p *kubeConfigClusterProvider) GetTargetParameterName() string {
	return KubeConfigTargetParameterName
}

func (p *kubeConfigClusterProvider) GetDerivedKubernetes(ctx context.Context, context string) (*Kubernetes, error) {
	m, err := p.managerForContext(context)
	if err != nil {
		return nil, err
	}
	return m.Derived(ctx)
}

func (p *kubeConfigClusterProvider) GetDefaultTarget() string {
	return p.defaultContext
}

func (p *kubeConfigClusterProvider) WatchTargets(onKubeConfigChanged func() error) {
	m := p.managers[p.defaultContext]

	m.WatchKubeConfig(onKubeConfigChanged)
}

func (p *kubeConfigClusterProvider) Close() {
	m := p.managers[p.defaultContext]

	m.Close()
}

// HasTargetTokenExchange returns true if per-target token exchange is configured for the given context.
func (p *kubeConfigClusterProvider) HasTargetTokenExchange(target string) bool {
	if p.tokenExchanger == nil || p.contextSTSConfigs == nil {
		return false
	}
	_, ok := p.contextSTSConfigs[target]
	return ok
}

// ExchangeTokenForTarget exchanges the given token for a target-specific token.
func (p *kubeConfigClusterProvider) ExchangeTokenForTarget(ctx context.Context, target, token string) (*oauth2.Token, error) {
	if p.tokenExchanger == nil {
		return nil, fmt.Errorf("token exchanger not configured")
	}

	stsCfg, ok := p.contextSTSConfigs[target]
	if !ok {
		return nil, fmt.Errorf("no token exchange configuration for context %s", target)
	}

	exchangedToken, err := p.tokenExchanger.Exchange(ctx, p.httpClient, stsCfg, token)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed for context %s: %w", target, err)
	}

	klog.V(3).Infof("Successfully exchanged token for context %s", target)
	return exchangedToken, nil
}
