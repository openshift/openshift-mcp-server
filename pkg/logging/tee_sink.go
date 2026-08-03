package logging

import (
	"context"

	"github.com/go-logr/logr"
)

// teeSink forwards log records to two logr.LogSink implementations.
// The primary sink is the text logger (file/stderr/stdout); the secondary
// is typically the OTel log bridge. Failures in the secondary are invisible
// — OTel export problems must never block local logging.
type teeSink struct {
	primary   logr.LogSink
	secondary logr.LogSink
}

var _ logr.LogSink = (*teeSink)(nil)
var _ logr.CallDepthLogSink = (*teeSink)(nil)

func (t *teeSink) Init(info logr.RuntimeInfo) {
	// Add one frame to account for the teeSink forwarding method.
	info.CallDepth++
	t.primary.Init(info)
	t.secondary.Init(info)
}

func (t *teeSink) Enabled(level int) bool {
	return t.primary.Enabled(level) || t.secondary.Enabled(level)
}

func (t *teeSink) Info(level int, msg string, keysAndValues ...any) {
	if t.primary.Enabled(level) {
		t.primary.Info(level, msg, stripContextValues(keysAndValues)...)
	}
	if t.secondary.Enabled(level) {
		t.secondary.Info(level, msg, keysAndValues...)
	}
}

func (t *teeSink) Error(err error, msg string, keysAndValues ...any) {
	t.primary.Error(err, msg, stripContextValues(keysAndValues)...)
	t.secondary.Error(err, msg, keysAndValues...)
}

func (t *teeSink) WithValues(keysAndValues ...any) logr.LogSink {
	return &teeSink{
		primary:   t.primary.WithValues(stripContextValues(keysAndValues)...),
		secondary: t.secondary.WithValues(keysAndValues...),
	}
}

// stripContextValues removes context.Context key-value pairs so the text
// logger never tries to format a context (which produces noisy output like
// "ctx=context.Background.WithCancel.WithValue(...)"). The secondary sink
// (otellogr) still receives the unfiltered slice and extracts the context
// for trace-log correlation.
func stripContextValues(keysAndValues []any) []any {
	result := make([]any, 0, len(keysAndValues))
	for i := 0; i < len(keysAndValues)-1; i += 2 {
		if _, ok := keysAndValues[i+1].(context.Context); ok {
			continue
		}
		result = append(result, keysAndValues[i], keysAndValues[i+1])
	}
	// Preserve a trailing key with no value (odd-length input).
	if len(keysAndValues)%2 != 0 {
		result = append(result, keysAndValues[len(keysAndValues)-1])
	}
	return result
}

func (t *teeSink) WithName(name string) logr.LogSink {
	return &teeSink{
		primary:   t.primary.WithName(name),
		secondary: t.secondary.WithName(name),
	}
}

// WithCallDepth implements logr.CallDepthLogSink so that klog's internal
// call-depth adjustments propagate to both underlying sinks, keeping
// caller-reported source locations accurate.
func (t *teeSink) WithCallDepth(depth int) logr.LogSink {
	return &teeSink{
		primary:   sinkWithCallDepth(t.primary, depth),
		secondary: sinkWithCallDepth(t.secondary, depth),
	}
}

func sinkWithCallDepth(sink logr.LogSink, depth int) logr.LogSink {
	if cd, ok := sink.(logr.CallDepthLogSink); ok {
		return cd.WithCallDepth(depth)
	}
	return sink
}
