package timeout

import (
	"context"
	"time"
)

// MaxTimeout is the maximum timeout in seconds for the command execution.
const MaxTimeout = 300 * time.Second

// TimeoutParams is a type that contains the timeout seconds for the command execution.
type TimeoutParams struct {
	// TimeoutSeconds is the timeout in seconds for the command execution.
	// If not specified, server default timeout is used. The maximum value
	// is MaxTimeout.
	TimeoutSeconds uint32 `json:"timeout_seconds,omitempty"`
}

// ToDuration converts the timeout seconds to a duration. The maximum value
// is MaxTimeout.
func (t *TimeoutParams) ToDuration() time.Duration {
	duration := time.Duration(t.TimeoutSeconds) * time.Second
	return min(duration, MaxTimeout)
}

// WithTimeout returns a context with a timeout set to the timeout seconds. If the timeout is 0,
// the original context is returned and the cancel function is nil.
func (t *TimeoutParams) WithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if t.TimeoutSeconds == 0 {
		return ctx, nil
	}
	return context.WithTimeout(ctx, t.ToDuration())
}
