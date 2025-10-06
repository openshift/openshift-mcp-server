package cmd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/spf13/cobra"

	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/textlogger"
	"k8s.io/kubectl/pkg/util/i18n"
	"k8s.io/kubectl/pkg/util/templates"

	"github.com/kiali/kiali-mcp-server/pkg/config"
	internalhttp "github.com/kiali/kiali-mcp-server/pkg/http"
	internalk8s "github.com/kiali/kiali-mcp-server/pkg/kubernetes"
	"github.com/kiali/kiali-mcp-server/pkg/mcp"
	"github.com/kiali/kiali-mcp-server/pkg/output"
	"github.com/kiali/kiali-mcp-server/pkg/toolsets"
	"github.com/kiali/kiali-mcp-server/pkg/version"
)

var (
	long     = templates.LongDesc(i18n.T("Kubernetes Model Context Protocol (MCP) server"))
	examples = templates.Examples(i18n.T(`
# show this help
kiali-mcp-server -h

# shows version information
kiali-mcp-server --version

# start STDIO server
kiali-mcp-server

# start a SSE server on port 8080
kiali-mcp-server --port 8080

# start a SSE server on port 8443 with a public HTTPS host of example.com
kiali-mcp-server --port 8443 --sse-base-url https://example.com:8443
`))
)

type MCPServerOptions struct {
	Version              bool
	LogLevel             int
	Port                 string
	SSEPort              int
	HttpPort             int
	SSEBaseUrl           string
	Kubeconfig           string
	Toolsets             []string
	ListOutput           string
	ReadOnly             bool
	DisableDestructive   bool
	RequireOAuth         bool
	OAuthAudience        string
	ValidateToken        bool
	AuthorizationURL     string
	CertificateAuthority string
	ServerURL            string
	KialiServerURL       string
	KialiInsecure        bool

	ConfigPath   string
	StaticConfig *config.StaticConfig

	genericiooptions.IOStreams
}

func NewMCPServerOptions(streams genericiooptions.IOStreams) *MCPServerOptions {
	return &MCPServerOptions{
		IOStreams:    streams,
		StaticConfig: config.Default(),
	}
}

func NewMCPServer(streams genericiooptions.IOStreams) *cobra.Command {
	o := NewMCPServerOptions(streams)
	cmd := &cobra.Command{
		Use:     "kubernetes-mcp-server [command] [options]",
		Short:   "Kubernetes Model Context Protocol (MCP) server",
		Long:    long,
		Example: examples,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(c); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}
			if err := o.Run(); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&o.Version, "version", o.Version, "Print version information and quit")
	cmd.Flags().IntVar(&o.LogLevel, "log-level", o.LogLevel, "Set the log level (from 0 to 9)")
	cmd.Flags().StringVar(&o.ConfigPath, "config", o.ConfigPath, "Path of the config file.")
	cmd.Flags().IntVar(&o.SSEPort, "sse-port", o.SSEPort, "Start a SSE server on the specified port")
	cmd.Flag("sse-port").Deprecated = "Use --port instead"
	cmd.Flags().IntVar(&o.HttpPort, "http-port", o.HttpPort, "Start a streamable HTTP server on the specified port")
	cmd.Flag("http-port").Deprecated = "Use --port instead"
	cmd.Flags().StringVar(&o.Port, "port", o.Port, "Start a streamable HTTP and SSE HTTP server on the specified port (e.g. 8080)")
	cmd.Flags().StringVar(&o.SSEBaseUrl, "sse-base-url", o.SSEBaseUrl, "SSE public base URL to use when sending the endpoint message (e.g. https://example.com)")
	cmd.Flags().StringVar(&o.Kubeconfig, "kubeconfig", o.Kubeconfig, "Path to the kubeconfig file to use for authentication")
	cmd.Flags().StringSliceVar(&o.Toolsets, "toolsets", o.Toolsets, "Comma-separated list of MCP toolsets to use (available toolsets: "+strings.Join(toolsets.ToolsetNames(), ", ")+"). Defaults to "+strings.Join(o.StaticConfig.Toolsets, ", ")+".")
	cmd.Flags().StringVar(&o.ListOutput, "list-output", o.ListOutput, "Output format for resource list operations (one of: "+strings.Join(output.Names, ", ")+"). Defaults to "+o.StaticConfig.ListOutput+".")
	cmd.Flags().BoolVar(&o.ReadOnly, "read-only", o.ReadOnly, "If true, only tools annotated with readOnlyHint=true are exposed")
	cmd.Flags().BoolVar(&o.DisableDestructive, "disable-destructive", o.DisableDestructive, "If true, tools annotated with destructiveHint=true are disabled")
	cmd.Flags().BoolVar(&o.RequireOAuth, "require-oauth", o.RequireOAuth, "If true, requires OAuth authorization as defined in the Model Context Protocol (MCP) specification. This flag is ignored if transport type is stdio")
	_ = cmd.Flags().MarkHidden("require-oauth")
	cmd.Flags().StringVar(&o.OAuthAudience, "oauth-audience", o.OAuthAudience, "OAuth audience for token claims validation. Optional. If not set, the audience is not validated. Only valid if require-oauth is enabled.")
	_ = cmd.Flags().MarkHidden("oauth-audience")
	cmd.Flags().BoolVar(&o.ValidateToken, "validate-token", o.ValidateToken, "If true, validates the token against the Kubernetes API Server using TokenReview. Optional. If not set, the token is not validated. Only valid if require-oauth is enabled.")
	_ = cmd.Flags().MarkHidden("validate-token")
	cmd.Flags().StringVar(&o.AuthorizationURL, "authorization-url", o.AuthorizationURL, "OAuth authorization server URL for protected resource endpoint. If not provided, the Kubernetes API server host will be used. Only valid if require-oauth is enabled.")
	_ = cmd.Flags().MarkHidden("authorization-url")
	cmd.Flags().StringVar(&o.ServerURL, "server-url", o.ServerURL, "Server URL of this application. Optional. If set, this url will be served in protected resource metadata endpoint and tokens will be validated with this audience. If not set, expected audience is kubernetes-mcp-server. Only valid if require-oauth is enabled.")
	_ = cmd.Flags().MarkHidden("server-url")
	cmd.Flags().StringVar(&o.CertificateAuthority, "certificate-authority", o.CertificateAuthority, "Certificate authority path to verify certificates. Optional. Only valid if require-oauth is enabled.")
	_ = cmd.Flags().MarkHidden("certificate-authority")
	cmd.Flags().StringVar(&o.KialiServerURL, "kiali-server-url", o.KialiServerURL, "Kiali server URL for protected resource endpoint. If not provided, the Kiali server will not be used. Only valid if require-oauth is enabled.")
	cmd.Flags().BoolVar(&o.KialiInsecure, "kiali-insecure", o.KialiInsecure, "If true, uses insecure TLS for the Kiali server. Optional. Only valid if require-oauth is enabled.")

	return cmd
}

func (m *MCPServerOptions) Complete(cmd *cobra.Command) error {
	if m.ConfigPath != "" {
		cnf, err := config.Read(m.ConfigPath)
		if err != nil {
			return err
		}
		m.StaticConfig = cnf
	}

	m.loadFlags(cmd)

	m.initializeLogging()

	if m.StaticConfig.RequireOAuth && m.StaticConfig.Port == "" {
		// RequireOAuth is not relevant flow for STDIO transport
		m.StaticConfig.RequireOAuth = false
	}

	return nil
}

func (m *MCPServerOptions) loadFlags(cmd *cobra.Command) {
	if cmd.Flag("log-level").Changed {
		m.StaticConfig.LogLevel = m.LogLevel
	}
	if cmd.Flag("port").Changed {
		m.StaticConfig.Port = m.Port
	} else if cmd.Flag("sse-port").Changed {
		m.StaticConfig.Port = strconv.Itoa(m.SSEPort)
	} else if cmd.Flag("http-port").Changed {
		m.StaticConfig.Port = strconv.Itoa(m.HttpPort)
	}
	if cmd.Flag("sse-base-url").Changed {
		m.StaticConfig.SSEBaseURL = m.SSEBaseUrl
	}
	if cmd.Flag("kubeconfig").Changed {
		m.StaticConfig.KubeConfig = m.Kubeconfig
	}
	if cmd.Flag("list-output").Changed {
		m.StaticConfig.ListOutput = m.ListOutput
	}
	if cmd.Flag("read-only").Changed {
		m.StaticConfig.ReadOnly = m.ReadOnly
	}
	if cmd.Flag("disable-destructive").Changed {
		m.StaticConfig.DisableDestructive = m.DisableDestructive
	}
	if cmd.Flag("toolsets").Changed {
		m.StaticConfig.Toolsets = m.Toolsets
	}
	if cmd.Flag("require-oauth").Changed {
		m.StaticConfig.RequireOAuth = m.RequireOAuth
	}
	if cmd.Flag("oauth-audience").Changed {
		m.StaticConfig.OAuthAudience = m.OAuthAudience
	}
	if cmd.Flag("validate-token").Changed {
		m.StaticConfig.ValidateToken = m.ValidateToken
	}
	if cmd.Flag("authorization-url").Changed {
		m.StaticConfig.AuthorizationURL = m.AuthorizationURL
	}
	if cmd.Flag("server-url").Changed {
		m.StaticConfig.ServerURL = m.ServerURL
	}
	if cmd.Flag("certificate-authority").Changed {
		m.StaticConfig.CertificateAuthority = m.CertificateAuthority
	}
	if cmd.Flag("kiali-server-url").Changed {
		m.StaticConfig.KialiServerURL = m.KialiServerURL
	}
	if cmd.Flag("kiali-insecure").Changed {
		m.StaticConfig.KialiInsecure = m.KialiInsecure
	}
}

func (m *MCPServerOptions) initializeLogging() {
	flagSet := flag.NewFlagSet("klog", flag.ContinueOnError)
	klog.InitFlags(flagSet)
	if m.StaticConfig.Port == "" {
		// disable klog output for stdio mode
		// this is needed to avoid klog writing to stderr and breaking the protocol
		_ = flagSet.Parse([]string{"-logtostderr=false", "-alsologtostderr=false", "-stderrthreshold=FATAL"})
		return
	}
	loggerOptions := []textlogger.ConfigOption{textlogger.Output(m.Out)}
	if m.StaticConfig.LogLevel >= 0 {
		loggerOptions = append(loggerOptions, textlogger.Verbosity(m.StaticConfig.LogLevel))
		_ = flagSet.Parse([]string{"--v", strconv.Itoa(m.StaticConfig.LogLevel)})
	}
	logger := textlogger.NewLogger(textlogger.NewConfig(loggerOptions...))
	klog.SetLoggerWithOptions(logger)
}

func (m *MCPServerOptions) Validate() error {
	if m.Port != "" && (m.SSEPort > 0 || m.HttpPort > 0) {
		return fmt.Errorf("--port is mutually exclusive with deprecated --http-port and --sse-port flags")
	}
	if output.FromString(m.StaticConfig.ListOutput) == nil {
		return fmt.Errorf("invalid output name: %s, valid names are: %s", m.StaticConfig.ListOutput, strings.Join(output.Names, ", "))
	}
	if err := toolsets.Validate(m.StaticConfig.Toolsets); err != nil {
		return err
	}
	if !m.StaticConfig.RequireOAuth && (m.StaticConfig.ValidateToken || m.StaticConfig.OAuthAudience != "" || m.StaticConfig.AuthorizationURL != "" || m.StaticConfig.ServerURL != "" || m.StaticConfig.CertificateAuthority != "") {
		return fmt.Errorf("validate-token, oauth-audience, authorization-url, server-url and certificate-authority are only valid if require-oauth is enabled. Missing --port may implicitly set require-oauth to false")
	}
	if m.StaticConfig.AuthorizationURL != "" {
		u, err := url.Parse(m.StaticConfig.AuthorizationURL)
		if err != nil {
			return err
		}
		if u.Scheme != "https" && u.Scheme != "http" {
			return fmt.Errorf("--authorization-url must be a valid URL")
		}
		if u.Scheme == "http" {
			klog.Warningf("authorization-url is using http://, this is not recommended production use")
		}
	}

	// If kiali toolset is enabled, require kiali_server_url to be set
	hasKiali := false
	for _, ts := range m.StaticConfig.Toolsets {
		if strings.TrimSpace(ts) == "kiali" {
			hasKiali = true
			break
		}
	}
	if hasKiali && strings.TrimSpace(m.StaticConfig.KialiServerURL) == "" {
		// Try to discover the Kiali URL before starting the server
		// Build a temporary Kubernetes manager from current static config
		k8sMgr, err := internalk8s.NewManager(m.StaticConfig)
		if err == nil && k8sMgr.IsOpenShift(context.Background()) {
			if url, dErr := k8sMgr.DiscoverRouteURLForService(context.Background(), "istio-system", "kiali"); dErr == nil && strings.TrimSpace(url) != "" {
				klog.V(0).Infof("auto-discovered Kiali URL: %s", url)
				m.StaticConfig.KialiServerURL = url
			} else if dErr != nil {
				klog.V(3).Infof("auto-discovery of Kiali URL failed: %v", dErr)
			}
		}
		if strings.TrimSpace(m.StaticConfig.KialiServerURL) == "" {
			return fmt.Errorf("kiali_server_url must be set when 'kiali' toolset is enabled and auto-discovery failed")
		}
	} else {
		klog.V(0).Infof("Kiali URL defined: %s", m.StaticConfig.KialiServerURL)
	}

	// Validate reachability of KialiServerURL (configured or discovered)
	if hasKiali && strings.TrimSpace(m.StaticConfig.KialiServerURL) != "" {
		u, err := url.Parse(m.StaticConfig.KialiServerURL)
		if err != nil {
			return fmt.Errorf("invalid kiali_server_url: %w", err)
		}
		transport := &http.Transport{}
		if m.StaticConfig.KialiInsecure {
			transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // allowed via configuration
		}
		client := &http.Client{Transport: transport, Timeout: 10 * time.Second}
		req, rErr := http.NewRequestWithContext(context.Background(), http.MethodGet, strings.TrimRight(m.StaticConfig.KialiServerURL, "/"), nil)
		if rErr != nil {
			return fmt.Errorf("failed to create request to kiali_server_url: %w", rErr)
		}
		resp, hErr := client.Do(req)
		if hErr != nil {
			var unknownAuthErr x509.UnknownAuthorityError
			var hostnameErr x509.HostnameError
			var certInvalidErr x509.CertificateInvalidError
			if u.Scheme == "https" && !m.StaticConfig.KialiInsecure && (errors.As(hErr, &unknownAuthErr) || errors.As(hErr, &hostnameErr) || errors.As(hErr, &certInvalidErr)) {
				// Auto-enable insecure for self-signed/untrusted certs and retry once
				klog.V(0).Infof("TLS verification failed for Kiali at %s: %v. Proceeding with insecure TLS (kiali_insecure=true)", m.StaticConfig.KialiServerURL, hErr)
				m.StaticConfig.KialiInsecure = true
				transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // allowed via configuration
				client.Transport = transport
				resp2, hErr2 := client.Do(req)
				if hErr2 != nil {
					return fmt.Errorf("failed to reach Kiali at %s even with insecure TLS: %v", m.StaticConfig.KialiServerURL, hErr2)
				}
				_ = resp2.Body.Close()
				klog.V(0).Infof("Kiali URL reachable (https, insecure): HTTP %d", resp2.StatusCode)
				return nil
			}
			return fmt.Errorf("failed to reach Kiali at %s: %v", m.StaticConfig.KialiServerURL, hErr)
		}
		_ = resp.Body.Close()
		klog.V(0).Infof("Kiali URL reachable (%s): HTTP %d", u.Scheme, resp.StatusCode)
	}
	return nil
}

func (m *MCPServerOptions) Run() error {
	klog.V(1).Infof("Starting %s", version.BinaryName)
	klog.V(1).Infof(" - Config: %s", m.ConfigPath)
	klog.V(1).Infof(" - Toolsets: %s", strings.Join(m.StaticConfig.Toolsets, ", "))
	klog.V(1).Infof(" - ListOutput: %s", m.StaticConfig.ListOutput)
	klog.V(1).Infof(" - Read-only mode: %t", m.StaticConfig.ReadOnly)
	klog.V(1).Infof(" - Disable destructive tools: %t", m.StaticConfig.DisableDestructive)

	if m.Version {
		_, _ = fmt.Fprintf(m.Out, "%s\n", version.Version)
		return nil
	}

	var oidcProvider *oidc.Provider
	if m.StaticConfig.AuthorizationURL != "" {
		ctx := context.Background()
		if m.StaticConfig.CertificateAuthority != "" {
			httpClient := &http.Client{}
			caCert, err := os.ReadFile(m.StaticConfig.CertificateAuthority)
			if err != nil {
				return fmt.Errorf("failed to read CA certificate from %s: %w", m.StaticConfig.CertificateAuthority, err)
			}
			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCert) {
				return fmt.Errorf("failed to append CA certificate from %s to pool", m.StaticConfig.CertificateAuthority)
			}

			if caCertPool.Equal(x509.NewCertPool()) {
				caCertPool = nil
			}

			transport := &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: caCertPool,
				},
			}
			httpClient.Transport = transport
			ctx = oidc.ClientContext(ctx, httpClient)
		}
		provider, err := oidc.NewProvider(ctx, m.StaticConfig.AuthorizationURL)
		if err != nil {
			return fmt.Errorf("unable to setup OIDC provider: %w", err)
		}
		oidcProvider = provider
	}

	mcpServer, err := mcp.NewServer(mcp.Configuration{StaticConfig: m.StaticConfig})
	if err != nil {
		return fmt.Errorf("failed to initialize MCP server: %w", err)
	}
	defer mcpServer.Close()

	if m.StaticConfig.Port != "" {
		ctx := context.Background()
		return internalhttp.Serve(ctx, mcpServer, m.StaticConfig, oidcProvider)
	}

	if err := mcpServer.ServeStdio(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	return nil
}
