package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	authenticationv1api "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"github.com/BurntSushi/toml"
	"github.com/containers/kubernetes-mcp-server/pkg/config"
)

const (
	ACMHubTargetParameterName    = "cluster"
	ClusterProviderACM           = "acm"
	ClusterProviderACMKubeConfig = "acm-kubeconfig"
)

// ACMProviderConfig holds ACM-specific configuration that users can set in config.toml
type ACMProviderConfig struct {
	// The host for the ACM cluster proxy addon
	// If using the acm-kubeconfig strategy, this should be the route for the proxy
	// If using the acm strategy, this should be the service for the proxy
	ClusterProxyAddonHost string `toml:"cluster_proxy_addon_host,omitempty"`

	// Whether to skip verifying the TLS certs from the cluster proxy
	ClusterProxyAddonSkipTLSVerify bool `toml:"cluster_proxy_addon_skip_tls_verify"`

	// The CA file for the cluster proxy addon
	ClusterProxyAddonCAFile string `toml:"cluster_proxy_addon_ca_file,omitempty"`
}

func (c *ACMProviderConfig) Validate() error {
	var err error = nil

	if c.ClusterProxyAddonHost == "" {
		err = errors.Join(err, fmt.Errorf("cluster_proxy_addon_host is required"))
	}

	if !c.ClusterProxyAddonSkipTLSVerify && c.ClusterProxyAddonCAFile == "" {
		err = errors.Join(err, fmt.Errorf("cluster_proxy_addon_ca_file is required if tls verification is not disabled"))
	}

	return err
}

func (c *ACMProviderConfig) ResolveClusterProxyAddonCAFilePath(ctx context.Context) {
	path := config.ConfigDirPathFromContext(ctx)
	c.ClusterProxyAddonCAFile = filepath.Join(path, c.ClusterProxyAddonCAFile)
}

type ACMKubeConfigProviderConfig struct {
	ACMProviderConfig

	// Name of the context in the kubeconfig file to look for acm access credentials in.
	// Should point to the "hub" cluster.
	ContextName string `toml:"context_name,omitempty"`
}

func (c *ACMKubeConfigProviderConfig) Validate() error {
	err := c.ACMProviderConfig.Validate()

	if c.ContextName == "" {
		err = errors.Join(err, fmt.Errorf("context_name is required if acm-kubeconfig strategy is used"))
	}

	return err
}

func parseAcmConfig(ctx context.Context, primitive toml.Primitive, md toml.MetaData) (config.ProviderConfig, error) {
	cfg := &ACMProviderConfig{}
	if err := md.PrimitiveDecode(primitive, cfg); err != nil {
		return nil, err
	}

	cfg.ResolveClusterProxyAddonCAFilePath(ctx)

	return cfg, nil
}

func parseAcmKubeConfigConfig(ctx context.Context, primitive toml.Primitive, md toml.MetaData) (config.ProviderConfig, error) {
	cfg := &ACMKubeConfigProviderConfig{}
	if err := md.PrimitiveDecode(primitive, &cfg); err != nil {
		return nil, err
	}

	cfg.ResolveClusterProxyAddonCAFilePath(ctx)

	return cfg, nil
}

type acmHubClusterProvider struct {
	hubManager         *Manager // for the main "hub" cluster
	clusterManagers    map[string]*Manager
	clusters           []string
	clusterProxyHost   string
	skipTLSVerify      bool
	clusterProxyCAFile string
	watchKubeConfig    bool // whether or not the kubeconfig should be watched for changes

	// Context for cancelling the watch goroutine
	watchCtx     context.Context
	watchCancel  context.CancelFunc
	watchStarted bool // Track if watch is already running

	// Resource version from last list operation to use for watch
	lastResourceVersion string
}

var _ Provider = &acmHubClusterProvider{}

func init() {
	RegisterProvider(ClusterProviderACM, newACMHubClusterProvider)
	RegisterProvider(ClusterProviderACMKubeConfig, newACMKubeConfigClusterProvider)

	config.RegisterProviderConfig(ClusterProviderACM, parseAcmConfig)
	config.RegisterProviderConfig(ClusterProviderACMKubeConfig, parseAcmKubeConfigConfig)
}

// IsACMHub checks if the current cluster is an ACM hub by looking for ACM CRDs
// This is included here instead of in other files so that it doesn't create conflicts
// with upstream changes
func (m *Manager) IsACMHub() bool {
	discoveryClient, err := m.ToDiscoveryClient()
	if err != nil {
		klog.V(3).Infof("failed to get discovery client for ACM detection: %v", err)
		return false
	}

	_, apiLists, err := discoveryClient.ServerGroupsAndResources()
	if err != nil {
		klog.V(3).Infof("failed to discover server resources for ACM detection: %v", err)
		return false
	}

	for _, apiList := range apiLists {
		if apiList.GroupVersion == "cluster.open-cluster-management.io/v1" {
			for _, resource := range apiList.APIResources {
				if resource.Kind == "ManagedCluster" {
					klog.V(2).Info("Detected ACM hub cluster")
					return true
				}
			}
		}
	}

	return false
}

func newACMHubClusterProvider(cfg *config.StaticConfig) (Provider, error) {
	m, err := NewInClusterManager(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create in-cluster Kubernetes Manager for acm-hub cluster provider strategy: %w", err)
	}

	providerCfg, ok := cfg.GetProviderConfig(ClusterProviderACM)
	if !ok {
		return nil, fmt.Errorf("missing required config for strategy '%s'", ClusterProviderACM)
	}

	return newACMClusterProvider(m, providerCfg.(*ACMProviderConfig), false)
}

func newACMKubeConfigClusterProvider(cfg *config.StaticConfig) (Provider, error) {
	providerCfg, ok := cfg.GetProviderConfig(ClusterProviderACMKubeConfig)
	if !ok {
		return nil, fmt.Errorf("missing required config for strategy '%s'", ClusterProviderACMKubeConfig)
	}

	acmKubeConfigProviderCfg := providerCfg.(*ACMKubeConfigProviderConfig)
	baseManager, err := NewKubeconfigManager(cfg, acmKubeConfigProviderCfg.ContextName)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create manager to hub cluster specified by acm_context_name %s: %w",
			acmKubeConfigProviderCfg.ContextName,
			err,
		)
	}

	return newACMClusterProvider(baseManager, &acmKubeConfigProviderCfg.ACMProviderConfig, true)
}

func newACMClusterProvider(m *Manager, cfg *ACMProviderConfig, watchKubeConfig bool) (Provider, error) {
	if !m.IsACMHub() {
		return nil, fmt.Errorf("not deployed in an ACM hub cluster")
	}

	// Create cancellable context for the watch goroutine
	watchCtx, watchCancel := context.WithCancel(context.Background())

	provider := &acmHubClusterProvider{
		hubManager:         m,
		clusterManagers:    make(map[string]*Manager),
		watchKubeConfig:    watchKubeConfig,
		watchCtx:           watchCtx,
		watchCancel:        watchCancel,
		clusterProxyHost:   cfg.ClusterProxyAddonHost,
		clusterProxyCAFile: cfg.ClusterProxyAddonCAFile,
		skipTLSVerify:      cfg.ClusterProxyAddonSkipTLSVerify,
	}

	ctx := context.Background()
	if err := provider.refreshClusters(ctx); err != nil {
		klog.Warningf("Failed to discover managed clusters: %v", err)
	}

	klog.V(2).Infof("ACM hub provider initialized with %d managed clusters", len(provider.clusters))
	return provider, nil
}

func (p *acmHubClusterProvider) IsOpenShift(ctx context.Context) bool {
	return p.hubManager.IsOpenShift(ctx)
}

func (p *acmHubClusterProvider) VerifyToken(ctx context.Context, target, token, audience string) (*authenticationv1api.UserInfo, []string, error) {
	manager, err := p.managerForCluster(target)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get manager for cluster '%s', unable to verify token", target)
	}
	return manager.VerifyToken(ctx, token, audience)
}

func (p *acmHubClusterProvider) GetDerivedKubernetes(ctx context.Context, target string) (*Kubernetes, error) {
	if target == "" || target == "hub" {
		return p.hubManager.Derived(ctx)
	}

	manager, err := p.managerForCluster(target)
	if err != nil {
		return nil, err
	}

	return manager.Derived(ctx)
}

func (p *acmHubClusterProvider) GetDefaultTarget() string {
	return "hub"
}

func (p *acmHubClusterProvider) GetTargets(_ context.Context) ([]string, error) {
	return p.clusters, nil
}

func (p *acmHubClusterProvider) GetTargetParameterName() string {
	return ACMHubTargetParameterName
}

func (p *acmHubClusterProvider) WatchTargets(onTargetsChanged func() error) {
	if p.watchKubeConfig {
		p.hubManager.WatchKubeConfig(onTargetsChanged)
	}

	// Only start watch if not already running
	if !p.watchStarted {
		p.watchStarted = true
		go p.watchManagedClusters(onTargetsChanged)
	}
}

func (p *acmHubClusterProvider) Close() {
	// Cancel the watch goroutine first
	if p.watchCancel != nil {
		p.watchCancel()
	}

	// Reset watch state
	p.watchStarted = false

	p.hubManager.Close()

	for _, manager := range p.clusterManagers {
		if manager != nil {
			manager.Close()
		}
	}
}

func (p *acmHubClusterProvider) watchManagedClusters(onTargetsChanged func() error) {
	gvr := schema.GroupVersionResource{
		Group:    "cluster.open-cluster-management.io",
		Version:  "v1",
		Resource: "managedclusters",
	}

	// Exponential backoff configuration
	const (
		initialDelay = 1 * time.Second
		maxDelay     = 5 * time.Minute
		backoffRate  = 2.0
	)

	delay := initialDelay

	for {
		// Check if the context has been cancelled before starting a new watch
		select {
		case <-p.watchCtx.Done():
			klog.V(2).Info("Watch goroutine cancelled, exiting")
			return
		default:
		}

		watchInterface, err := p.hubManager.dynamicClient.Resource(gvr).Watch(p.watchCtx, metav1.ListOptions{
			ResourceVersion: p.lastResourceVersion,
		})
		if err != nil {
			klog.Errorf("Failed to start watch on managed clusters: %v", err)

			// Apply exponential backoff
			klog.V(2).Infof("Waiting %v before retrying watch", delay)
			time.Sleep(delay)

			// Increase delay for next retry, but cap at maxDelay
			delay = time.Duration(float64(delay) * backoffRate)
			delay = min(delay, maxDelay)
			continue
		}

		// Reset delay on successful watch start
		delay = initialDelay
		klog.V(2).Info("Started watching managed clusters for changes")

		for event := range watchInterface.ResultChan() {
			switch event.Type {
			case watch.Added, watch.Deleted, watch.Modified:
				clusterName := "unknown"
				if obj, ok := event.Object.(*unstructured.Unstructured); ok {
					clusterName = obj.GetName()
				}
				klog.V(3).Infof("Managed cluster %s: %s", event.Type, clusterName)

				// Notify about target changes
				if err := onTargetsChanged(); err != nil {
					klog.Warningf("Error in onTargetsChanged callback: %v", err)
				}
			}
		}

		// Clean up the watch interface before restarting
		watchInterface.Stop()
		klog.Warning("Managed clusters watch closed, restarting...")
		// Don't reset delay here since this could be due to connectivity issues
	}
}

func (p *acmHubClusterProvider) refreshClusters(ctx context.Context) error {
	dynamicClient := p.hubManager.dynamicClient

	gvr := schema.GroupVersionResource{
		Group:    "cluster.open-cluster-management.io",
		Version:  "v1",
		Resource: "managedclusters",
	}

	result, err := dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list cluster managers: %w", err)
	}

	clusters := make([]string, 0, len(result.Items))
	for _, item := range result.Items {
		name := item.GetName()
		if name != "" {
			clusters = append(clusters, name)
		}
	}

	p.clusters = clusters
	p.lastResourceVersion = result.GetResourceVersion()
	klog.V(3).Infof("discovered %d managed clusters: %v (resourceVersion: %s)", len(clusters), clusters, p.lastResourceVersion)

	return nil
}

func (p *acmHubClusterProvider) managerForCluster(cluster string) (*Manager, error) {
	if manager, exists := p.clusterManagers[cluster]; exists && manager != nil {
		return manager, nil
	}

	proxyConfig := rest.CopyConfig(p.hubManager.cfg)
	proxyHost := fmt.Sprintf("https://%s/%s", p.clusterProxyHost, cluster)
	proxyConfig.Host = proxyHost

	if p.skipTLSVerify {
		proxyConfig.TLSClientConfig = rest.TLSClientConfig{
			Insecure: true,
		}
	} else {
		proxyConfig.TLSClientConfig = rest.TLSClientConfig{
			CAFile: p.clusterProxyCAFile,
		}
	}

	// Create modified clientCmdConfig to match the proxy configuration
	hubRawConfig, err := p.hubManager.clientCmdConfig.RawConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get hub kubeconfig: %w", err)
	}

	// Create a copy and modify the server URL to match the proxy
	proxyRawConfig := hubRawConfig.DeepCopy()

	// Update all clusters in the config to use the proxy host
	for _, clusterConfig := range proxyRawConfig.Clusters {
		clusterConfig.Server = proxyHost
		if p.skipTLSVerify {
			clusterConfig.InsecureSkipTLSVerify = true
			clusterConfig.CertificateAuthority = ""
			clusterConfig.CertificateAuthorityData = nil
		} else {
			clusterConfig.CertificateAuthority = p.clusterProxyCAFile
			clusterConfig.CertificateAuthorityData = nil
			clusterConfig.InsecureSkipTLSVerify = false
		}
	}

	manager := &Manager{
		cfg:             proxyConfig,
		staticConfig:    p.hubManager.staticConfig,
		clientCmdConfig: clientcmd.NewDefaultClientConfig(*proxyRawConfig, nil),
	}

	if err := p.initializeManager(manager); err != nil {
		return nil, fmt.Errorf("failed to initialize manager for cluster %s: %w", cluster, err)
	}

	// Cache the manager before returning
	p.clusterManagers[cluster] = manager
	return manager, nil
}

func (p *acmHubClusterProvider) initializeManager(m *Manager) error {
	var err error

	m.accessControlClientSet, err = NewAccessControlClientset(m.cfg, m.staticConfig)
	if err != nil {
		return err
	}

	m.discoveryClient = memory.NewMemCacheClient(m.accessControlClientSet.DiscoveryClient())

	m.accessControlRESTMapper = NewAccessControlRESTMapper(
		restmapper.NewDeferredDiscoveryRESTMapper(m.discoveryClient),
		m.staticConfig,
	)

	m.dynamicClient, err = dynamic.NewForConfig(m.cfg)
	if err != nil {
		return err
	}

	return nil
}
