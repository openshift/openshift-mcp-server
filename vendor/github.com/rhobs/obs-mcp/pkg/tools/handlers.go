package tools

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"strings"
	"time"

	ammodels "github.com/prometheus/alertmanager/api/v2/models"
	"github.com/prometheus/common/model"
	"k8s.io/utils/ptr"

	"github.com/rhobs/obs-mcp/pkg/alertmanager"
	"github.com/rhobs/obs-mcp/pkg/prometheus"
	"github.com/rhobs/obs-mcp/pkg/resultutil"
)

const (
	// millisecondsPerSecond converts Prometheus millisecond timestamps to seconds.
	millisecondsPerSecond = 1000
)

// GetString is a helper to extract a string parameter with a default value
func GetString(params map[string]any, key, defaultValue string) string {
	if val, ok := params[key]; ok {
		if str, ok := val.(string); ok && str != "" {
			return str
		}
	}
	return defaultValue
}

// GetBoolPtr is a helper to extract an optional boolean parameter as a pointer
func GetBoolPtr(params map[string]any, key string) *bool {
	if val, ok := params[key]; ok {
		if b, ok := val.(bool); ok {
			return &b
		}
	}
	return nil
}

// parseDefaultTimeRange parses optional start/end time strings,
// defaulting to the last hour if both are empty.
func parseDefaultTimeRange(start, end string) (startTime, endTime time.Time, err error) {
	if start == "" && end == "" {
		endTime = time.Now()
		startTime = endTime.Add(-prometheus.ListMetricsTimeRange)
		return startTime, endTime, nil
	}

	if start != "" {
		startTime, err = prometheus.ParseTimestamp(start)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid start time format: %w", err)
		}
	}
	if end != "" {
		endTime, err = prometheus.ParseTimestamp(end)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid end time format: %w", err)
		}
	}
	return startTime, endTime, nil
}

// parseFilterString splits a comma-separated filter string into trimmed parts.
func parseFilterString(filter string) []string {
	if filter == "" {
		return nil
	}
	parts := strings.Split(filter, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// convertAlert converts an Alertmanager GettableAlert to the Alert output type.
func convertAlert(a *ammodels.GettableAlert) Alert {
	labels := make(map[string]string)
	maps.Copy(labels, a.Labels)

	annotations := make(map[string]string)
	maps.Copy(annotations, a.Annotations)

	var silencedBy, inhibitedBy []string
	var state string
	if a.Status != nil {
		if a.Status.SilencedBy != nil {
			silencedBy = a.Status.SilencedBy
		}
		if a.Status.InhibitedBy != nil {
			inhibitedBy = a.Status.InhibitedBy
		}
		state = ptr.Deref(a.Status.State, "")
	}
	if silencedBy == nil {
		silencedBy = []string{}
	}
	if inhibitedBy == nil {
		inhibitedBy = []string{}
	}

	var startsAt, endsAt string
	if a.StartsAt != nil {
		startsAt = a.StartsAt.String()
	}
	if a.EndsAt != nil {
		endsAt = a.EndsAt.String()
	}

	return Alert{
		Labels:      labels,
		Annotations: annotations,
		StartsAt:    startsAt,
		EndsAt:      endsAt,
		Status: AlertStatus{
			State:       state,
			SilencedBy:  silencedBy,
			InhibitedBy: inhibitedBy,
		},
	}
}

// convertMatcher converts an Alertmanager Matcher to the Matcher output type.
func convertMatcher(m *ammodels.Matcher) Matcher {
	isEqual := true
	if m.IsEqual != nil {
		isEqual = *m.IsEqual
	}
	return Matcher{
		Name:    ptr.Deref(m.Name, ""),
		Value:   ptr.Deref(m.Value, ""),
		IsRegex: m.IsRegex != nil && *m.IsRegex,
		IsEqual: isEqual,
	}
}

// convertSilence converts an Alertmanager GettableSilence to the Silence output type.
func convertSilence(s *ammodels.GettableSilence) Silence {
	matchers := make([]Matcher, len(s.Matchers))
	for i, m := range s.Matchers {
		matchers[i] = convertMatcher(m)
	}

	var state string
	if s.Status != nil {
		state = ptr.Deref(s.Status.State, "")
	}

	var startsAt, endsAt string
	if s.StartsAt != nil {
		startsAt = s.StartsAt.String()
	}
	if s.EndsAt != nil {
		endsAt = s.EndsAt.String()
	}

	return Silence{
		ID: ptr.Deref(s.ID, ""),
		Status: SilenceStatus{
			State: state,
		},
		Matchers:  matchers,
		StartsAt:  startsAt,
		EndsAt:    endsAt,
		CreatedBy: ptr.Deref(s.CreatedBy, ""),
		Comment:   ptr.Deref(s.Comment, ""),
	}
}

func BuildListMetricsInput(args map[string]any) ListMetricsInput {
	return ListMetricsInput{
		NameRegex: GetString(args, "name_regex", ""),
	}
}

func BuildInstantQueryInput(args map[string]any) InstantQueryInput {
	return InstantQueryInput{
		Query: GetString(args, "query", ""),
		Time:  GetString(args, "time", ""),
	}
}

func BuildRangeQueryInput(args map[string]any) RangeQueryInput {
	return RangeQueryInput{
		Query:    GetString(args, "query", ""),
		Step:     GetString(args, "step", ""),
		Start:    GetString(args, "start", ""),
		End:      GetString(args, "end", ""),
		Duration: GetString(args, "duration", ""),
	}
}

func BuildLabelNamesInput(args map[string]any) LabelNamesInput {
	return LabelNamesInput{
		Metric: GetString(args, "metric", ""),
		Start:  GetString(args, "start", ""),
		End:    GetString(args, "end", ""),
	}
}

func BuildLabelValuesInput(args map[string]any) LabelValuesInput {
	return LabelValuesInput{
		Label:  GetString(args, "label", ""),
		Metric: GetString(args, "metric", ""),
		Start:  GetString(args, "start", ""),
		End:    GetString(args, "end", ""),
	}
}

func BuildSeriesInput(args map[string]any) SeriesInput {
	return SeriesInput{
		Matches: GetString(args, "matches", ""),
		Start:   GetString(args, "start", ""),
		End:     GetString(args, "end", ""),
	}
}

func BuildAlertsInput(args map[string]any) AlertsInput {
	return AlertsInput{
		Active:      GetBoolPtr(args, "active"),
		Silenced:    GetBoolPtr(args, "silenced"),
		Inhibited:   GetBoolPtr(args, "inhibited"),
		Unprocessed: GetBoolPtr(args, "unprocessed"),
		Filter:      GetString(args, "filter", ""),
		Receiver:    GetString(args, "receiver", ""),
	}
}

func BuildSilencesInput(args map[string]any) SilencesInput {
	return SilencesInput{
		Filter: GetString(args, "filter", ""),
	}
}

func BuildGenerateSLOInput(args map[string]any) GenerateSLOInput {
	return GenerateSLOInput{
		Target:                  GetString(args, "target", ""),
		Window:                  GetString(args, "window", "28d"),
		AvailabilityTarget:      GetString(args, "availability_target", "99.9"),
		LatencyTargetDuration:   GetString(args, "latency_target_duration", "5s"),
		LatencyTargetPercentile: GetString(args, "latency_target_percentile", "99.9"),
	}
}

// ListMetricsHandler handles the listing of available Prometheus metrics.
func ListMetricsHandler(ctx context.Context, promClient prometheus.Loader, input ListMetricsInput) *resultutil.Result {
	slog.Info("ListMetricsHandler called")
	slog.Debug("ListMetricsHandler params", "input", input)

	// Validate required parameters
	if input.NameRegex == "" {
		return resultutil.NewErrorResult(fmt.Errorf("name_regex parameter is required and must be a string"))
	}

	metrics, err := promClient.ListMetrics(ctx, input.NameRegex)
	if err != nil {
		slog.Error("failed to list metrics", "error", err)
		return resultutil.NewErrorResult(fmt.Errorf("failed to list metrics: %w", err))
	}

	slog.Info("ListMetricsHandler executed successfully", "resultLength", len(metrics))
	slog.Debug("ListMetricsHandler results", "results", metrics)

	output := ListMetricsOutput{Metrics: metrics}
	return resultutil.NewSuccessResult(output)
}

// ExecuteRangeQueryHandler handles the execution of Prometheus range queries.
func ExecuteRangeQueryHandler(ctx context.Context, promClient prometheus.Loader, input RangeQueryInput, fullResponse bool) *resultutil.Result {
	slog.Info("ExecuteRangeQueryHandler called")
	slog.Debug("ExecuteRangeQueryHandler params", "input", input)

	// Validate required parameters
	if input.Query == "" {
		return resultutil.NewErrorResult(fmt.Errorf("query parameter is required and must be a string"))
	}
	if input.Step == "" {
		return resultutil.NewErrorResult(fmt.Errorf("step parameter is required and must be a string"))
	}

	// Parse step duration
	stepDuration, err := model.ParseDuration(input.Step)
	if err != nil {
		return resultutil.NewErrorResult(fmt.Errorf("invalid step format: %w", err))
	}

	// Validate parameter combinations
	if input.Start != "" && input.End != "" && input.Duration != "" {
		return resultutil.NewErrorResult(fmt.Errorf("cannot specify both start/end and duration parameters"))
	}

	if (input.Start != "" && input.End == "") || (input.Start == "" && input.End != "") {
		return resultutil.NewErrorResult(fmt.Errorf("both start and end must be provided together"))
	}

	var startTime, endTime time.Time

	// Handle duration-based query (default to 1h if nothing specified)
	if input.Duration != "" || (input.Start == "" && input.End == "") {
		durationStr := input.Duration
		if durationStr == "" {
			durationStr = "1h"
		}

		duration, err := model.ParseDuration(durationStr)
		if err != nil {
			return resultutil.NewErrorResult(fmt.Errorf("invalid duration format: %w", err))
		}

		endTime = time.Now()
		startTime = endTime.Add(-time.Duration(duration))
	} else {
		// Handle explicit start/end times
		startTime, err = prometheus.ParseTimestamp(input.Start)
		if err != nil {
			return resultutil.NewErrorResult(fmt.Errorf("invalid start time format: %w", err))
		}

		endTime, err = prometheus.ParseTimestamp(input.End)
		if err != nil {
			return resultutil.NewErrorResult(fmt.Errorf("invalid end time format: %w", err))
		}
	}

	// Execute the range query
	result, err := promClient.ExecuteRangeQuery(ctx, input.Query, startTime, endTime, time.Duration(stepDuration))
	if err != nil {
		return resultutil.NewErrorResult(fmt.Errorf("failed to execute range query: %w", err))
	}

	// Convert to structured output
	output := RangeQueryOutput{
		ResultType: fmt.Sprintf("%v", result["resultType"]),
	}

	resMatrix, ok := result["result"].(model.Matrix)
	if ok {
		slog.Info("ExecuteRangeQueryHandler executed successfully", "resultLength", resMatrix.Len())
		slog.Debug("ExecuteRangeQueryHandler results", "results", resMatrix)

		if fullResponse {
			// Return full data
			output.Result = make([]SeriesResult, len(resMatrix))
			for i, series := range resMatrix {
				labels := make(map[string]string)
				for k, v := range series.Metric {
					labels[string(k)] = string(v)
				}
				values := make([][]any, len(series.Values))
				for j, sample := range series.Values {
					values[j] = []any{float64(sample.Timestamp) / millisecondsPerSecond, sample.Value.String()}
				}
				output.Result[i] = SeriesResult{
					Metric: labels,
					Values: values,
				}
			}
		} else {
			// Return summary statistics instead of full data
			output.Summary = make([]SeriesResultSummary, len(resMatrix))
			for i, series := range resMatrix {
				output.Summary[i] = CalculateSeriesSummary(series.Metric, series.Values)
			}
		}
	} else {
		slog.Info("ExecuteRangeQueryHandler executed successfully (unknown format)", "result", result)
	}

	if warnings, ok := result["warnings"].([]string); ok {
		output.Warnings = warnings
	}

	return resultutil.NewSuccessResult(output)
}

// ExecuteInstantQueryHandler handles the execution of Prometheus instant queries.
func ExecuteInstantQueryHandler(ctx context.Context, promClient prometheus.Loader, input InstantQueryInput) *resultutil.Result {
	slog.Info("ExecuteInstantQueryHandler called")
	slog.Debug("ExecuteInstantQueryHandler params", "input", input)

	// Validate required parameters
	if input.Query == "" {
		return resultutil.NewErrorResult(fmt.Errorf("query parameter is required and must be a string"))
	}

	var queryTime time.Time
	var err error
	if input.Time == "" {
		queryTime = time.Now()
	} else {
		queryTime, err = prometheus.ParseTimestamp(input.Time)
		if err != nil {
			return resultutil.NewErrorResult(fmt.Errorf("invalid time format: %w", err))
		}
	}

	// Execute the instant query
	result, err := promClient.ExecuteInstantQuery(ctx, input.Query, queryTime)
	if err != nil {
		return resultutil.NewErrorResult(fmt.Errorf("failed to execute instant query: %w", err))
	}

	// Convert to structured output
	output := InstantQueryOutput{
		ResultType: fmt.Sprintf("%v", result["resultType"]),
	}

	resVector, ok := result["result"].(model.Vector)
	if ok {
		slog.Info("ExecuteInstantQueryHandler executed successfully", "resultLength", len(resVector))
		slog.Debug("ExecuteInstantQueryHandler results", "results", resVector)

		output.Result = make([]InstantResult, len(resVector))
		for i, sample := range resVector {
			labels := make(map[string]string)
			for k, v := range sample.Metric {
				labels[string(k)] = string(v)
			}
			output.Result[i] = InstantResult{
				Metric: labels,
				Value:  []any{float64(sample.Timestamp) / millisecondsPerSecond, sample.Value.String()},
			}
		}
	} else {
		slog.Info("ExecuteInstantQueryHandler executed successfully (unknown format)", "result", result)
	}

	if warnings, ok := result["warnings"].([]string); ok {
		output.Warnings = warnings
	}

	return resultutil.NewSuccessResult(output)
}

// GetLabelNamesHandler handles the retrieval of label names.
func GetLabelNamesHandler(ctx context.Context, promClient prometheus.Loader, input LabelNamesInput) *resultutil.Result {
	slog.Info("GetLabelNamesHandler called")
	slog.Debug("GetLabelNamesHandler params", "input", input)

	startTime, endTime, err := parseDefaultTimeRange(input.Start, input.End)
	if err != nil {
		return resultutil.NewErrorResult(err)
	}

	// Get label names
	labels, err := promClient.GetLabelNames(ctx, input.Metric, startTime, endTime)
	if err != nil {
		return resultutil.NewErrorResult(fmt.Errorf("failed to get label names: %w", err))
	}

	slog.Info("GetLabelNamesHandler executed successfully", "labelCount", len(labels))
	slog.Debug("GetLabelNamesHandler results", "results", labels)

	output := LabelNamesOutput{Labels: labels}
	return resultutil.NewSuccessResult(output)
}

// GetLabelValuesHandler handles the retrieval of label values.
func GetLabelValuesHandler(ctx context.Context, promClient prometheus.Loader, input LabelValuesInput) *resultutil.Result {
	slog.Info("GetLabelValuesHandler called")
	slog.Debug("GetLabelValuesHandler params", "input", input)

	// Validate required parameters
	if input.Label == "" {
		return resultutil.NewErrorResult(fmt.Errorf("label parameter is required and must be a string"))
	}

	startTime, endTime, err := parseDefaultTimeRange(input.Start, input.End)
	if err != nil {
		return resultutil.NewErrorResult(err)
	}

	// Get label values
	values, err := promClient.GetLabelValues(ctx, input.Label, input.Metric, startTime, endTime)
	if err != nil {
		return resultutil.NewErrorResult(fmt.Errorf("failed to get label values: %w", err))
	}

	slog.Info("GetLabelValuesHandler executed successfully", "valueCount", len(values))
	slog.Debug("GetLabelValuesHandler results", "results", values)

	output := LabelValuesOutput{Values: values}
	return resultutil.NewSuccessResult(output)
}

// GetSeriesHandler handles the retrieval of time series.
func GetSeriesHandler(ctx context.Context, promClient prometheus.Loader, input SeriesInput) *resultutil.Result {
	slog.Info("GetSeriesHandler called")
	slog.Debug("GetSeriesHandler params", "input", input)

	// Validate required parameters
	if input.Matches == "" {
		return resultutil.NewErrorResult(fmt.Errorf("matches parameter is required and must be a string"))
	}

	// Parse matches - could be comma-separated
	matches := []string{input.Matches}
	// If it contains comma outside of braces, split it
	// For simplicity, treat the entire string as one match for now
	// Users can make multiple calls if needed

	startTime, endTime, err := parseDefaultTimeRange(input.Start, input.End)
	if err != nil {
		return resultutil.NewErrorResult(err)
	}

	// Get series
	series, err := promClient.GetSeries(ctx, matches, startTime, endTime)
	if err != nil {
		return resultutil.NewErrorResult(fmt.Errorf("failed to get series: %w", err))
	}

	slog.Info("GetSeriesHandler executed successfully", "cardinality", len(series))
	slog.Debug("GetSeriesHandler results", "results", series)

	output := SeriesOutput{
		Series:      series,
		Cardinality: len(series),
	}
	return resultutil.NewSuccessResult(output)
}

// GetAlertsHandler handles the retrieval of alerts from Alertmanager.
func GetAlertsHandler(ctx context.Context, amClient alertmanager.Loader, input AlertsInput) *resultutil.Result {
	slog.Info("GetAlertsHandler called")
	slog.Debug("GetAlertsHandler params", "input", input)

	alerts, err := amClient.GetAlerts(ctx, input.Active, input.Silenced, input.Inhibited, input.Unprocessed, parseFilterString(input.Filter), input.Receiver)
	if err != nil {
		return resultutil.NewErrorResult(fmt.Errorf("failed to get alerts: %w", err))
	}

	output := AlertsOutput{
		Alerts: make([]Alert, len(alerts)),
	}
	for i, alert := range alerts {
		output.Alerts[i] = convertAlert(alert)
	}

	slog.Info("GetAlertsHandler executed successfully", "alertCount", len(alerts))
	slog.Debug("GetAlertsHandler results", "results", output.Alerts)

	return resultutil.NewSuccessResult(output)
}

// GetSilencesHandler handles the retrieval of silences from Alertmanager.
func GetSilencesHandler(ctx context.Context, amClient alertmanager.Loader, input SilencesInput) *resultutil.Result {
	slog.Info("GetSilencesHandler called")
	slog.Debug("GetSilencesHandler params", "input", input)

	silences, err := amClient.GetSilences(ctx, parseFilterString(input.Filter))
	if err != nil {
		return resultutil.NewErrorResult(fmt.Errorf("failed to get silences: %w", err))
	}

	output := SilencesOutput{
		Silences: make([]Silence, len(silences)),
	}
	for i, silence := range silences {
		output.Silences[i] = convertSilence(silence)
	}

	slog.Info("GetSilencesHandler executed successfully", "silenceCount", len(silences))
	slog.Debug("GetSilencesHandler results", "results", output.Silences)

	return resultutil.NewSuccessResult(output)
}

// GenerateSLOHandler generates SLO expressions for error rate and latency monitoring.
func GenerateSLOHandler(ctx context.Context, promClient prometheus.Loader, input GenerateSLOInput) *resultutil.Result {
	slog.Info("GenerateSLOHandler called")
	slog.Debug("GenerateSLOHandler params", "input", input)

	// Validate required parameters
	if input.Target == "" {
		return resultutil.NewErrorResult(fmt.Errorf("target parameter is required"))
	}

	// Initialize output
	output := GenerateSLOOutput{
		Target:             input.Target,
		Window:             input.Window,
		AvailabilityTarget: input.AvailabilityTarget,
		DiscoveredMetrics:  []string{},
		Recommendations:    []string{},
	}

	// Step 1: Discover error/request metrics
	errorMetrics, err := discoverErrorMetrics(ctx, promClient, input.Target)
	if err != nil {
		slog.Warn("failed to discover error metrics", "error", err)
	} else {
		output.DiscoveredMetrics = append(output.DiscoveredMetrics, errorMetrics...)

		// Generate error rate SLO if we found metrics
		if len(errorMetrics) > 0 {
			errorSLO := generateErrorRateSLO(errorMetrics, input)
			if errorSLO != nil {
				output.ErrorRateSLO = errorSLO
			}
		}
	}

	// Step 2: Discover latency metrics
	latencyMetrics, err := discoverLatencyMetrics(ctx, promClient, input.Target)
	if err != nil {
		slog.Warn("failed to discover latency metrics", "error", err)
	} else {
		output.DiscoveredMetrics = append(output.DiscoveredMetrics, latencyMetrics...)

		// Generate latency SLO if we found metrics
		if len(latencyMetrics) > 0 {
			latencySLO := generateLatencySLO(latencyMetrics, input)
			if latencySLO != nil {
				output.LatencySLO = latencySLO
			}
		}
	}

	// Step 3: Generate error budget if we have error rate SLO
	if output.ErrorRateSLO != nil {
		output.ErrorBudget = generateErrorBudget(input)
	}

	// Step 4: Add recommendations
	output.Recommendations = generateRecommendations(output)

	if len(output.DiscoveredMetrics) == 0 {
		return resultutil.NewErrorResult(fmt.Errorf("no relevant metrics found for target '%s'. Try using list_metrics to explore available metrics", input.Target))
	}

	slog.Info("GenerateSLOHandler executed successfully", "metricsFound", len(output.DiscoveredMetrics))
	return resultutil.NewSuccessResult(output)
}

// discoverErrorMetrics finds metrics related to HTTP requests, errors, or status codes.
func discoverErrorMetrics(ctx context.Context, promClient prometheus.Loader, target string) ([]string, error) {
	patterns := []string{
		fmt.Sprintf(".*%s.*request.*", target),
		fmt.Sprintf(".*%s.*http.*", target),
		fmt.Sprintf(".*%s.*rpc.*", target),
		fmt.Sprintf(".*request.*%s.*", target),
		fmt.Sprintf(".*http.*%s.*", target),
	}

	var allMetrics []string
	for _, pattern := range patterns {
		metrics, err := promClient.ListMetrics(ctx, pattern)
		if err != nil {
			continue
		}
		allMetrics = append(allMetrics, metrics...)
	}

	// Deduplicate
	seen := make(map[string]bool)
	var unique []string
	for _, m := range allMetrics {
		if !seen[m] {
			seen[m] = true
			unique = append(unique, m)
		}
	}

	return unique, nil
}

// discoverLatencyMetrics finds metrics related to request duration or latency.
func discoverLatencyMetrics(ctx context.Context, promClient prometheus.Loader, target string) ([]string, error) {
	patterns := []string{
		fmt.Sprintf(".*%s.*duration.*", target),
		fmt.Sprintf(".*%s.*latency.*", target),
		fmt.Sprintf(".*duration.*%s.*", target),
		fmt.Sprintf(".*latency.*%s.*", target),
	}

	var allMetrics []string
	for _, pattern := range patterns {
		metrics, err := promClient.ListMetrics(ctx, pattern)
		if err != nil {
			continue
		}
		allMetrics = append(allMetrics, metrics...)
	}

	// Deduplicate
	seen := make(map[string]bool)
	var unique []string
	for _, m := range allMetrics {
		if !seen[m] {
			seen[m] = true
			unique = append(unique, m)
		}
	}

	return unique, nil
}

// generateErrorRateSLO creates error rate SLO expressions from discovered metrics.
func generateErrorRateSLO(metrics []string, input GenerateSLOInput) *SLOExpression {
	// Try to find total and error metrics
	var totalMetric, errorMetric string

	for _, m := range metrics {
		if strings.Contains(m, "_total") && (strings.Contains(m, "request") || strings.Contains(m, "http")) {
			totalMetric = m
		}
		if strings.Contains(m, "error") || strings.Contains(m, "fail") {
			errorMetric = m
		}
	}

	if totalMetric == "" && len(metrics) > 0 {
		// Use the first metric that looks like a counter
		for _, m := range metrics {
			if strings.HasSuffix(m, "_total") || strings.HasSuffix(m, "_count") {
				totalMetric = m
				break
			}
		}
	}

	if totalMetric == "" {
		return nil
	}

	availabilityFloat := 99.9
	fmt.Sscanf(input.AvailabilityTarget, "%f", &availabilityFloat)
	errorBudget := 100 - availabilityFloat

	slo := &SLOExpression{
		Type:          "availability",
		TotalMetric:   totalMetric,
		SuccessMetric: totalMetric,
	}

	// Generate queries based on available metrics
	if errorMetric != "" {
		slo.SuccessMetric = fmt.Sprintf("%s - %s", totalMetric, errorMetric)
		slo.SLOQuery = fmt.Sprintf(`(
  sum(rate(%s[%s]))
  /
  sum(rate(%s[%s]))
) * 100`,
			slo.SuccessMetric, input.Window,
			totalMetric, input.Window)

		slo.Explanation = fmt.Sprintf("This SLO tracks availability by measuring the ratio of successful requests to total requests over a %s window. Target: %s%% (error budget: %g%%)",
			input.Window, input.AvailabilityTarget, errorBudget)
	} else {
		// Assume we need to use status code labels or similar
		slo.SLOQuery = fmt.Sprintf(`(
  sum(rate(%s{code=~"2..|3.."}[%s]))
  /
  sum(rate(%s[%s]))
) * 100`,
			totalMetric, input.Window,
			totalMetric, input.Window)

		slo.Explanation = fmt.Sprintf("This SLO tracks availability by measuring the ratio of 2xx/3xx status codes to total requests over a %s window. Target: %s%% (error budget: %g%%). Note: Adjust the status code regex based on your metric labels.",
			input.Window, input.AvailabilityTarget, errorBudget)

		slo.Labels = []string{"code", "status", "status_code"}
	}

	// Generate burn rate query
	slo.BurnRateQuery = fmt.Sprintf(`(
  1 - (
    sum(rate(%s[1h]))
    /
    sum(rate(%s[1h]))
  )
) / %g`,
		slo.SuccessMetric, totalMetric, errorBudget/100)

	// Generate alert query
	slo.AlertQuery = fmt.Sprintf(`(
  1 - (
    sum(rate(%s[1h]))
    /
    sum(rate(%s[1h]))
  )
) > %g`,
		slo.SuccessMetric, totalMetric, errorBudget/100)

	return slo
}

// generateLatencySLO creates latency SLO expressions from discovered metrics.
func generateLatencySLO(metrics []string, input GenerateSLOInput) *SLOExpression {
	// Find histogram or summary metrics
	var latencyMetric string

	for _, m := range metrics {
		if strings.HasSuffix(m, "_bucket") || strings.HasSuffix(m, "_sum") {
			latencyMetric = strings.TrimSuffix(strings.TrimSuffix(m, "_bucket"), "_sum")
			break
		}
	}

	if latencyMetric == "" && len(metrics) > 0 {
		latencyMetric = metrics[0]
	}

	if latencyMetric == "" {
		return nil
	}

	percentile := input.LatencyTargetPercentile
	percentileFloat := 0.999
	fmt.Sscanf(percentile, "%f", &percentileFloat)
	percentileVal := percentileFloat / 100.0

	slo := &SLOExpression{
		Type:          "latency",
		LatencyMetric: latencyMetric,
	}

	// Check if it's a histogram (has _bucket suffix in metrics list)
	isHistogram := false
	for _, m := range metrics {
		if strings.HasSuffix(m, "_bucket") {
			isHistogram = true
			break
		}
	}

	if isHistogram {
		// Use histogram_quantile for histograms
		slo.SLOQuery = fmt.Sprintf(`histogram_quantile(%g,
  sum(rate(%s_bucket[%s])) by (le)
)`,
			percentileVal, latencyMetric, input.Window)

		slo.Explanation = fmt.Sprintf("This SLO tracks the p%s latency over a %s window using histogram metrics. Target: requests complete within %s at the %s percentile.",
			percentile, input.Window, input.LatencyTargetDuration, percentile)

		// For burn rate, check if current latency exceeds target
		slo.BurnRateQuery = fmt.Sprintf(`(
  histogram_quantile(%g, sum(rate(%s_bucket[1h])) by (le))
  >
  %s
)`,
			percentileVal, latencyMetric, input.LatencyTargetDuration)

		slo.AlertQuery = fmt.Sprintf(`histogram_quantile(%g,
  sum(rate(%s_bucket[5m])) by (le)
) > %s`,
			percentileVal, latencyMetric, input.LatencyTargetDuration)

	} else {
		// Assume it's a summary or simple metric
		slo.SLOQuery = fmt.Sprintf(`%s{quantile="%g"}`, latencyMetric, percentileVal)

		slo.Explanation = fmt.Sprintf("This SLO tracks the p%s latency using summary metrics. Target: requests complete within %s at the %s percentile. Note: Adjust the quantile label based on your metric structure.",
			percentile, input.LatencyTargetDuration, percentile)

		slo.Labels = []string{"quantile"}
		slo.BurnRateQuery = fmt.Sprintf(`%s{quantile="%g"} > %s`, latencyMetric, percentileVal, input.LatencyTargetDuration)
		slo.AlertQuery = slo.BurnRateQuery
	}

	return slo
}

// generateErrorBudget creates error budget calculations.
func generateErrorBudget(input GenerateSLOInput) *ErrorBudget {
	availabilityFloat := 99.9
	fmt.Sscanf(input.AvailabilityTarget, "%f", &availabilityFloat)
	errorBudget := 100 - availabilityFloat

	// Parse window to calculate allowed downtime
	windowDuration, _ := model.ParseDuration(input.Window)
	totalSeconds := time.Duration(windowDuration).Seconds()
	allowedDowntimeSeconds := totalSeconds * (errorBudget / 100.0)

	var downtimeStr string
	if allowedDowntimeSeconds < 60 {
		downtimeStr = fmt.Sprintf("%.1f seconds", allowedDowntimeSeconds)
	} else if allowedDowntimeSeconds < 3600 {
		downtimeStr = fmt.Sprintf("%.1f minutes", allowedDowntimeSeconds/60)
	} else if allowedDowntimeSeconds < 86400 {
		downtimeStr = fmt.Sprintf("%.1f hours", allowedDowntimeSeconds/3600)
	} else {
		downtimeStr = fmt.Sprintf("%.1f days", allowedDowntimeSeconds/86400)
	}

	return &ErrorBudget{
		TotalBudget:     fmt.Sprintf("%g%%", errorBudget),
		AllowedDowntime: downtimeStr,
		RemainingQuery: fmt.Sprintf(`# Error budget remaining (0-1 scale, where 1 = full budget available)
# This requires tracking actual errors over the window
1 - (
  sum(increase(errors_total[%s]))
  /
  (sum(increase(requests_total[%s])) * %g)
)`, input.Window, input.Window, errorBudget/100),
		BurnRateQuery: fmt.Sprintf(`# Current error rate divided by error budget
# Burn rate > 1 means consuming budget faster than sustainable
(
  sum(rate(errors_total[1h]))
  /
  sum(rate(requests_total[1h]))
) / %g`, errorBudget/100),
		Explanation: fmt.Sprintf("With a %s%% availability target over %s, you have an error budget of %s%% (%s of allowed downtime). A burn rate of 1.0 means you're consuming the budget at exactly the sustainable rate. Higher burn rates indicate you're spending the budget too quickly.",
			input.AvailabilityTarget, input.Window, fmt.Sprintf("%g", errorBudget), downtimeStr),
	}
}

// generateRecommendations provides recommendations based on the discovered metrics.
func generateRecommendations(output GenerateSLOOutput) []string {
	var recommendations []string

	if output.ErrorRateSLO == nil {
		recommendations = append(recommendations, "No error rate metrics found. Consider instrumenting your application to track request success/failure rates.")
	} else {
		if len(output.ErrorRateSLO.Labels) > 0 {
			recommendations = append(recommendations, fmt.Sprintf("Verify that the metric has the suggested labels: %s. Adjust the queries based on actual label names.", strings.Join(output.ErrorRateSLO.Labels, ", ")))
		}
		recommendations = append(recommendations, "Consider setting up multi-window multi-burn-rate alerts (e.g., 2% budget burn in 1h AND 5% burn in 6h) to catch issues early while reducing false positives.")
	}

	if output.LatencySLO == nil {
		recommendations = append(recommendations, "No latency metrics found. Consider instrumenting your application with histogram or summary metrics to track request duration.")
	} else {
		recommendations = append(recommendations, "For histogram metrics, ensure buckets are properly configured to capture your target latency (e.g., include buckets at 1s, 2s, 5s for a 5s target).")
	}

	if output.ErrorRateSLO != nil || output.LatencySLO != nil {
		recommendations = append(recommendations, "Use execute_range_query with these expressions to visualize SLO compliance over time.")
		recommendations = append(recommendations, "Create Prometheus recording rules for these SLO queries to improve query performance.")
		recommendations = append(recommendations, "Consider implementing a 4-signal monitoring approach: latency, traffic, errors, and saturation (RED/USE methods).")
	}

	return recommendations
}
