package kubernetes

import (
	"context"

	"golang.org/x/oauth2"
	authenticationv1api "k8s.io/api/authentication/v1"
)

type TokenVerifier interface {
	VerifyToken(ctx context.Context, cluster, token, audience string) (*authenticationv1api.UserInfo, []string, error)
}

// TokenExchanger provides per-target token exchange capabilities.
// This is used for scenarios like cross-realm Keycloak token exchange
// where a hub token needs to be exchanged for a managed cluster token.
type TokenExchanger interface {
	// HasTargetTokenExchange returns true if per-target token exchange is configured for the given target.
	HasTargetTokenExchange(target string) bool
	// ExchangeTokenForTarget exchanges the given token for a target-specific token.
	// Returns the exchanged token, or an error if exchange fails.
	ExchangeTokenForTarget(ctx context.Context, target, token string) (*oauth2.Token, error)
}
