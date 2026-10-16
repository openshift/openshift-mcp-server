package mcp

import (
	"context"
	"fmt"

	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// TokenReviewer validates bearer tokens using the Kubernetes TokenReview API.
type TokenReviewer struct {
	client kubernetes.Interface
}

// NewTokenReviewer creates a TokenReviewer with the given Kubernetes client.
func NewTokenReviewer(client kubernetes.Interface) *TokenReviewer {
	return &TokenReviewer{client: client}
}

// ValidateToken submits a TokenReview for the given token and returns an
// error when the token is invalid or nil when the token is authenticated.
func (t *TokenReviewer) ValidateToken(ctx context.Context, token string) error {
	review := &authenticationv1.TokenReview{
		Spec: authenticationv1.TokenReviewSpec{
			Token: token,
		},
	}

	result, err := t.client.AuthenticationV1().TokenReviews().Create(ctx, review, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("token review request failed: %w", err)
	}

	if !result.Status.Authenticated {
		return fmt.Errorf("token authentication failed: %s", result.Status.Error)
	}

	return nil
}
