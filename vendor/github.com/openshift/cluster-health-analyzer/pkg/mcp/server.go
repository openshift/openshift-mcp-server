package mcp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type authHeader string

const authHeaderStr authHeader = "kubernetes-authorization"

// MCPHealthServer is a helper and wrapper type
// providing basic methods to run the underlying SSE server
// and to register tools
type MCPHealthServer struct {
	server        *mcp.Server
	addr          string
	tlsCertFile   string
	tlsKeyFile    string
	tokenReviewer *TokenReviewer
}

type MCPHealthServerCfg struct {
	Name    string
	Version string
	Url     string

	PrometheusURL   string
	AlertManagerURL string

	TLSCertFile string
	TLSKeyFile  string

	// DisableAuthForTesting skips TokenReview validation.
	// Only intended for local development.
	DisableAuthForTesting bool
}

// NewMCPHealthServer returns an instance of the MCPHealthServer
func NewMCPHealthServer(cfg MCPHealthServerCfg) (*MCPHealthServer, error) {
	impl := mcp.Implementation{
		Name:    cfg.Name,
		Version: cfg.Version,
	}

	server := mcp.NewServer(&impl, nil)

	incTool := NewIncidentsTool(cfg.PrometheusURL, cfg.AlertManagerURL)
	// get_incidents
	mcp.AddTool(server, &incTool.Tool, mcp.ToolHandlerFor[GetIncidentsParams, any](incTool.IncidentsHandler))

	var reviewer *TokenReviewer
	if !cfg.DisableAuthForTesting {
		restCfg, err := rest.InClusterConfig()
		if err != nil {
			return nil, errors.Join(errors.New("failed to build in-cluster config for token review"), err)
		}
		clientset, err := kubernetes.NewForConfig(restCfg)
		if err != nil {
			return nil, errors.Join(errors.New("failed to create kubernetes client for token review"), err)
		}
		reviewer = NewTokenReviewer(clientset)
		slog.Info("TokenReview authentication enabled")
	} else {
		slog.Warn("Authentication is disabled — do not use in production")
	}

	return &MCPHealthServer{
		server:        server,
		addr:          cfg.Url,
		tlsCertFile:   cfg.TLSCertFile,
		tlsKeyFile:    cfg.TLSKeyFile,
		tokenReviewer: reviewer,
	}, nil
}

// handler builds the root http.Handler for the MCP server, registering the
// healthz endpoint outside the auth middleware and routing everything else
// through it.
func (m *MCPHealthServer) handler() http.Handler {
	mcpHandler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return m.server
	}, nil)

	mux := http.NewServeMux()

	// Register the healthz endpoint directly on the mux, outside the
	// auth middleware, so that kubelet probes can reach it without
	// a bearer token.
	mux.HandleFunc("/healthz", healthzHandler)

	// All other paths go through the auth middleware.
	mux.Handle("/", m.authMiddleware(mcpHandler))

	return mux
}

// Start runs the MCPHealthServer
func (m *MCPHealthServer) Start() error {
	if m.addr == "" {
		return errors.New("empty http address")
	}

	slog.Info("Starting MCP server on ", "address", m.addr)

	h := m.handler()

	if (m.tlsCertFile == "") != (m.tlsKeyFile == "") {
		return errors.New("both TLS certificate and private key files must be configured")
	}

	if m.tlsCertFile != "" && m.tlsKeyFile != "" {
		slog.Info("TLS enabled for MCP server")
		tlsServer := &http.Server{
			Addr:    m.addr,
			Handler: h,
			TLSConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		}
		return tlsServer.ListenAndServeTLS(m.tlsCertFile, m.tlsKeyFile)
	}

	slog.Warn("TLS is not configured, serving over plaintext HTTP")
	return http.ListenAndServe(m.addr, h)
}

// RegisterTool registers a new tool on the MCPHealthServer
func (m *MCPHealthServer) RegisterTool(t *mcp.Tool, handler mcp.ToolHandlerFor[any, any]) {
	mcp.AddTool(m.server, t, handler)
}

// healthzHandler responds with 200 OK for liveness and readiness probes.
func healthzHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprintln(w, "ok"); err != nil {
		slog.Debug("healthz: failed to write response", "err", err)
	}
}

// authMiddleware returns an http.Handler that enriches the request context
// with the kubernetes-authorization token. When a TokenReviewer is configured,
// the token is validated via the Kubernetes TokenReview API before the
// request is forwarded.
func (m *MCPHealthServer) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authCtx := authFromRequest(r.Context(), r)

		if m.tokenReviewer != nil {
			token, err := getTokenFromCtx(authCtx)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			if err := m.tokenReviewer.ValidateToken(r.Context(), token); err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}

		r = r.WithContext(authCtx)
		next.ServeHTTP(w, r)
	})
}

func authFromRequest(ctx context.Context, r *http.Request) context.Context {
	authHeaderValue := r.Header.Get(string(authHeaderStr))
	token, found := strings.CutPrefix(authHeaderValue, "Bearer ")
	if !found {
		slog.Error("Failed to parse kubernetes-authorization header. Prefix Bearer not found.")
		return ctx
	}
	return context.WithValue(ctx, authHeaderStr, token)
}
