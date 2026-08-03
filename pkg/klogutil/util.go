package klogutil

import (
	"context"
	"sync/atomic"

	"k8s.io/klog/v2"
)

var otelLogSinkActive atomic.Bool

// SetOtelLogSinkActive records whether the OTel log bridge is active.
// When false, FromContext skips the WithValues("ctx", …) call, avoiding a
// per-call allocation and keeping the text log output free of the placeholder.
func SetOtelLogSinkActive(active bool) { otelLogSinkActive.Store(active) }

// Attr is a single, complete key/value log attribute. Because every constructor
// returns a complete pair, attributes compose through the variadic logr/klog
// logging API without the "even number of varargs" footgun: keys and values
// cannot get out of sync.
type Attr struct {
	K string
	V any
}

// FromContext returns the logger associated with ctx, enriched so that the
// OpenTelemetry log bridge (otellogr) can extract the active trace span.
//
// logr.LogSink.Info/Error have no context.Context parameter, so
// klog.FromContext(ctx).Info(...) discards the span carried by ctx. Calling
// WithValues("ctx", ctx) stores the context inside the otellogr sink (via
// its convertKVs path), which then passes it to Emit — giving the OTel SDK
// the span context it needs for trace-log correlation.
//
// The context is wrapped in otelCtx so that text loggers that don't strip
// context values (e.g. in tests without the teeSink) produce a harmless
// placeholder instead of formatting the full context chain, which could
// contain sensitive data such as HTTP request headers.
func FromContext(ctx context.Context) klog.Logger {
	logger := klog.FromContext(ctx)
	if !otelLogSinkActive.Load() {
		return logger
	}
	return logger.WithValues("ctx", otelCtx{ctx})
}

// otelCtx wraps a context.Context so that text formatters produce a fixed
// placeholder instead of traversing the context chain. It still satisfies
// context.Context, so otellogr's convertKVs detects it via type assertion
// and extracts the underlying context for trace-log correlation.
type otelCtx struct{ context.Context }

func (otelCtx) GoString() string { return "<otel-ctx>" }
func (otelCtx) String() string   { return "<otel-ctx>" }

// Err returns an Attr carrying the error message under the OpenTelemetry
// semantic-convention key "exception.message". It centralizes that key so there
// is a single source of truth across every error log site. A nil error yields an
// empty message rather than panicking.
func Err(err error) Attr {
	if err == nil {
		return Attr{"exception.message", ""}
	}
	return Attr{"exception.message", err.Error()}
}

// Field returns an Attr for an arbitrary key/value pair.
func Field(k string, v any) Attr { return Attr{k, v} }

// toKV flattens a slice of Attr into the alternating key/value slice expected by
// the logr/klog variadic logging API.
func toKV(attrs []Attr) []any {
	kv := make([]any, 0, len(attrs)*2)
	for _, a := range attrs {
		kv = append(kv, a.K, a.V)
	}
	return kv
}

// LogInfo logs msg with the provided attributes on the given logger. Pass a
// V-leveled logger (e.g. logger.V(4)) to preserve verbosity gating.
func LogInfo(logger klog.Logger, msg string, attrs ...Attr) {
	logger.Info(msg, toKV(attrs)...)
}

// LogWarn logs msg with a log.severity=WARN tag and the provided attributes.
// This is done because the klog.Logger interface has no native Warn method. Pass
// a V-leveled logger (e.g. logger.V(1)) to preserve verbosity gating.
func LogWarn(logger klog.Logger, msg string, attrs ...Attr) {
	logger.Info(msg, append([]any{"log.severity", "WARN"}, toKV(attrs)...)...)
}
