package tokenexchange

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google/externalaccount"
)

// Token exchange strategy constants
const (
	TokenExchangeStrategyKeycloakV1      = "keycloak-v1"
	TokenExchangeStrategyRFC8693         = "rfc8693"
	TokenExchangeStrategyExternalAccount = "external-account"
)

// OAuth 2.0 Token Exchange grant types and token types (RFC 8693)
const (
	GrantTypeTokenExchange = "urn:ietf:params:oauth:grant-type:token-exchange"
	TokenTypeAccessToken   = "urn:ietf:params:oauth:token-type:access_token"
	TokenTypeJWT           = "urn:ietf:params:oauth:token-type:jwt"
)

// Token exchange form field keys
const (
	FormKeyGrantType          = "grant_type"
	FormKeySubjectToken       = "subject_token"
	FormKeySubjectTokenType   = "subject_token_type"
	FormKeySubjectIssuer      = "subject_issuer"
	FormKeyAudience           = "audience"
	FormKeyClientID           = "client_id"
	FormKeyClientSecret       = "client_secret"
	FormKeyScope              = "scope"
	FormKeyRequestedTokenType = "requested_token_type"
)

// TargetSTSConfig holds per-target token exchange configuration.
// This is used by providers that support per-cluster or per-context token exchange
// (e.g., ACM hub provider for managed clusters, kubeconfig provider for contexts).
type TargetSTSConfig struct {
	// TokenURL is the token endpoint for this target's realm
	TokenURL string `toml:"token_url"`
	// ClientID is the OAuth client ID in the target realm
	ClientID string `toml:"client_id"`
	// ClientSecret is the OAuth client secret in the target realm
	ClientSecret string `toml:"client_secret"`
	// Audience is the target audience for the exchanged token
	Audience string `toml:"audience"`
	// SubjectTokenType specifies the token type of the subject token
	// For same-realm: "urn:ietf:params:oauth:token-type:access_token"
	// For cross-realm: "urn:ietf:params:oauth:token-type:jwt"
	SubjectTokenType string `toml:"subject_token_type"`
	// SubjectIssuer is the IDP alias for cross-realm token exchange (Keycloak V1 only)
	// Only required when exchanging tokens across Keycloak realms
	SubjectIssuer string `toml:"subject_issuer,omitempty"`
	// Scopes are optional scopes to request during token exchange
	Scopes []string `toml:"scopes,omitempty"`
	// CAFile is the path to a CA certificate file for TLS verification
	// Used when the token endpoint uses a certificate signed by a private CA
	CAFile string `toml:"ca_file,omitempty"`
	// InsecureSkipTLSVerify disables TLS certificate verification (not recommended for production)
	InsecureSkipTLSVerify bool `toml:"insecure_skip_tls_verify,omitempty"`
}

// HTTPClient creates an HTTP client configured with the TLS settings from this config.
// If neither CAFile nor InsecureSkipTLSVerify is set, returns a default client.
func (c *TargetSTSConfig) HTTPClient() (*http.Client, error) {
	// If no TLS customization needed, return default client with timeout
	if c.CAFile == "" && !c.InsecureSkipTLSVerify {
		return &http.Client{Timeout: 30 * time.Second}, nil
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if c.InsecureSkipTLSVerify {
		tlsConfig.InsecureSkipVerify = true
	}

	if c.CAFile != "" {
		caCert, err := os.ReadFile(c.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file %s: %w", c.CAFile, err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate from %s", c.CAFile)
		}
		tlsConfig.RootCAs = caCertPool
	}

	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}, nil
}

// TokenExchanger defines the interface for token exchange strategies
type TokenExchanger interface {
	Exchange(ctx context.Context, httpClient *http.Client, cfg TargetSTSConfig, subjectToken string) (*oauth2.Token, error)
}

// tokenExchangeResponse represents the OAuth 2.0 token exchange response
type tokenExchangeResponse struct {
	AccessToken     string `json:"access_token"`
	TokenType       string `json:"token_type"`
	ExpiresIn       int    `json:"expires_in"`
	RefreshToken    string `json:"refresh_token,omitempty"`
	Scope           string `json:"scope,omitempty"`
	IssuedTokenType string `json:"issued_token_type,omitempty"`
}

// GetTokenExchanger returns a TokenExchanger for the given strategy
func GetTokenExchanger(strategy string) (TokenExchanger, error) {
	switch strategy {
	case TokenExchangeStrategyKeycloakV1:
		return &keycloakV1Exchanger{}, nil
	case TokenExchangeStrategyRFC8693:
		return &rfc8693Exchanger{}, nil
	case TokenExchangeStrategyExternalAccount:
		return &externalAccountExchanger{}, nil
	default:
		return nil, fmt.Errorf("unknown token exchange strategy: %s", strategy)
	}
}

// staticSubjectTokenSupplier implements externalaccount.SubjectTokenSupplier
type staticSubjectTokenSupplier struct {
	token string
}

func (s *staticSubjectTokenSupplier) SubjectToken(_ context.Context, _ externalaccount.SupplierOptions) (string, error) {
	return s.token, nil
}

var _ externalaccount.SubjectTokenSupplier = &staticSubjectTokenSupplier{}

// keycloakV1Exchanger implements Keycloak V1 token exchange with subject_issuer support
type keycloakV1Exchanger struct{}

func (e *keycloakV1Exchanger) Exchange(ctx context.Context, httpClient *http.Client, cfg TargetSTSConfig, subjectToken string) (*oauth2.Token, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	data := url.Values{}
	data.Set(FormKeyGrantType, GrantTypeTokenExchange)
	data.Set(FormKeySubjectToken, subjectToken)
	data.Set(FormKeySubjectTokenType, cfg.SubjectTokenType)
	data.Set(FormKeyAudience, cfg.Audience)
	data.Set(FormKeyClientID, cfg.ClientID)

	if cfg.ClientSecret != "" {
		data.Set(FormKeyClientSecret, cfg.ClientSecret)
	}

	// subject_issuer is the key differentiator for Keycloak V1 cross-realm exchange
	if cfg.SubjectIssuer != "" {
		data.Set(FormKeySubjectIssuer, cfg.SubjectIssuer)
	}

	if len(cfg.Scopes) > 0 {
		data.Set(FormKeyScope, strings.Join(cfg.Scopes, " "))
	}

	return doTokenExchange(ctx, httpClient, cfg.TokenURL, data)
}

// rfc8693Exchanger implements standard RFC 8693 token exchange
type rfc8693Exchanger struct{}

func (e *rfc8693Exchanger) Exchange(ctx context.Context, httpClient *http.Client, cfg TargetSTSConfig, subjectToken string) (*oauth2.Token, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	data := url.Values{}
	data.Set(FormKeyGrantType, GrantTypeTokenExchange)
	data.Set(FormKeySubjectToken, subjectToken)
	data.Set(FormKeySubjectTokenType, cfg.SubjectTokenType)
	data.Set(FormKeyAudience, cfg.Audience)
	data.Set(FormKeyRequestedTokenType, TokenTypeAccessToken)
	data.Set(FormKeyClientID, cfg.ClientID)

	if cfg.ClientSecret != "" {
		data.Set(FormKeyClientSecret, cfg.ClientSecret)
	}

	if len(cfg.Scopes) > 0 {
		data.Set(FormKeyScope, strings.Join(cfg.Scopes, " "))
	}

	return doTokenExchange(ctx, httpClient, cfg.TokenURL, data)
}

// externalAccountExchanger wraps the Google externalaccount library
type externalAccountExchanger struct{}

func (e *externalAccountExchanger) Exchange(ctx context.Context, httpClient *http.Client, cfg TargetSTSConfig, subjectToken string) (*oauth2.Token, error) {
	if httpClient != nil {
		ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)
	}

	ts, err := externalaccount.NewTokenSource(ctx, externalaccount.Config{
		TokenURL:             cfg.TokenURL,
		ClientID:             cfg.ClientID,
		ClientSecret:         cfg.ClientSecret,
		Audience:             cfg.Audience,
		SubjectTokenType:     cfg.SubjectTokenType,
		SubjectTokenSupplier: &staticSubjectTokenSupplier{token: subjectToken},
		Scopes:               cfg.Scopes,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create external account token source: %w", err)
	}

	return ts.Token()
}

// doTokenExchange performs the HTTP POST for token exchange
func doTokenExchange(ctx context.Context, httpClient *http.Client, tokenURL string, data url.Values) (*oauth2.Token, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token exchange request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token exchange response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp tokenExchangeResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token exchange response: %w", err)
	}

	token := &oauth2.Token{
		AccessToken:  tokenResp.AccessToken,
		TokenType:    tokenResp.TokenType,
		RefreshToken: tokenResp.RefreshToken,
	}

	if tokenResp.ExpiresIn > 0 {
		token.Expiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	return token, nil
}
