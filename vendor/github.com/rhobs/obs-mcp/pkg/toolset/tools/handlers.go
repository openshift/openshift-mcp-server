package tools

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/prometheus/common/model"

	"github.com/rhobs/obs-mcp/pkg/prometheus"
)

// Output types for tool results

// ListMetricsOutput defines the output schema for the list_metrics tool.
type ListMetricsOutput struct {
	Metrics []string `json:"metrics"`
}

// InstantQueryOutput defines the output schema for the execute_instant_query tool.
type InstantQueryOutput struct {
	ResultType string          `json:"resultType"`
	Result     []InstantResult `json:"result"`
	Warnings   []string        `json:"warnings,omitempty"`
}

// InstantResult represents a single instant query result.
type InstantResult struct {
	Metric map[string]string `json:"metric"`
	Value  []any             `json:"value"`
}

// RangeQueryOutput defines the output schema for the execute_range_query tool.
type RangeQueryOutput struct {
	ResultType string         `json:"resultType"`
	Result     []SeriesResult `json:"result"`
	Warnings   []string       `json:"warnings,omitempty"`
}

// SeriesResult represents a single time series result from a range query.
type SeriesResult struct {
	Metric map[string]string `json:"metric"`
	Values [][]any           `json:"values"`
}

// LabelNamesOutput defines the output schema for the get_label_names tool.
type LabelNamesOutput struct {
	Labels []string `json:"labels"`
}

// LabelValuesOutput defines the output schema for the get_label_values tool.
type LabelValuesOutput struct {
	Values []string `json:"values"`
}

// SeriesOutput defines the output schema for the get_series tool.
type SeriesOutput struct {
	Series      []map[string]string `json:"series"`
	Cardinality int                 `json:"cardinality"`
}

// Helper function to create error results
func errorResult(msg string) (*api.ToolCallResult, error) {
	slog.Info("Query execution error: " + msg)
	return api.NewToolCallResult("", fmt.Errorf("%s", msg)), nil
}

// Helper function to get string argument with default
func getStringArg(params api.ToolHandlerParams, key string, defaultValue string) string {
	if val, ok := params.GetArguments()[key].(string); ok && val != "" {
		return val
	}
	return defaultValue
}

// ListMetricsHandler handles the listing of available Prometheus metrics.
func ListMetricsHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	slog.Info("ListMetricsHandler called")

	promClient, err := getPromClient(params)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to create Prometheus client: %s", err.Error()))
	}

	metrics, err := promClient.ListMetrics(params.Context)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to list metrics: %s", err.Error()))
	}

	slog.Info("ListMetricsHandler executed successfully", "resultLength", len(metrics))
	slog.Debug("ListMetricsHandler results", "results", metrics)

	output := ListMetricsOutput{Metrics: metrics}
	result, err := json.Marshal(output)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to marshal metrics: %s", err.Error()))
	}

	return api.NewToolCallResult(string(result), nil), nil
}

// ExecuteInstantQueryHandler handles the execution of Prometheus instant queries.
func ExecuteInstantQueryHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	slog.Info("ExecuteInstantQueryHandler called")

	promClient, err := getPromClient(params)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to create Prometheus client: %s", err.Error()))
	}

	// Get required query parameter
	query, ok := params.GetArguments()["query"].(string)
	if !ok || query == "" {
		return errorResult("query parameter is required and must be a string")
	}

	// Get optional time parameter
	timeStr := getStringArg(params, "time", "")

	var queryTime time.Time
	if timeStr == "" {
		queryTime = time.Now()
	} else {
		queryTime, err = prometheus.ParseTimestamp(timeStr)
		if err != nil {
			return errorResult(fmt.Sprintf("invalid time format: %s", err.Error()))
		}
	}

	// Execute the instant query
	result, err := promClient.ExecuteInstantQuery(params.Context, query, queryTime)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to execute instant query: %s", err.Error()))
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
				Value:  []any{float64(sample.Timestamp) / 1000, sample.Value.String()},
			}
		}
	} else {
		slog.Info("ExecuteInstantQueryHandler executed successfully (unknown format)", "result", result)
	}

	if warnings, ok := result["warnings"].([]string); ok {
		output.Warnings = warnings
	}

	jsonResult, err := json.Marshal(output)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to marshal result: %s", err.Error()))
	}

	return api.NewToolCallResult(string(jsonResult), nil), nil
}

// ExecuteRangeQueryHandler handles the execution of Prometheus range queries.
func ExecuteRangeQueryHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	slog.Info("ExecuteRangeQueryHandler called")

	promClient, err := getPromClient(params)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to create Prometheus client: %s", err.Error()))
	}

	// Get required query parameter
	query, ok := params.GetArguments()["query"].(string)
	if !ok || query == "" {
		return errorResult("query parameter is required and must be a string")
	}

	// Get required step parameter
	step, ok := params.GetArguments()["step"].(string)
	if !ok || step == "" {
		return errorResult("step parameter is required and must be a string")
	}

	// Parse step duration
	stepDuration, err := model.ParseDuration(step)
	if err != nil {
		return errorResult(fmt.Sprintf("invalid step format: %s", err.Error()))
	}

	// Get optional parameters
	startStr := getStringArg(params, "start", "")
	endStr := getStringArg(params, "end", "")
	durationStr := getStringArg(params, "duration", "")

	// Validate parameter combinations
	if startStr != "" && endStr != "" && durationStr != "" {
		return errorResult("cannot specify both start/end and duration parameters")
	}

	if (startStr != "" && endStr == "") || (startStr == "" && endStr != "") {
		return errorResult("both start and end must be provided together")
	}

	var startTime, endTime time.Time

	// Handle duration-based query (default to 1h if nothing specified)
	if durationStr != "" || (startStr == "" && endStr == "") {
		if durationStr == "" {
			durationStr = "1h"
		}

		duration, err := model.ParseDuration(durationStr)
		if err != nil {
			return errorResult(fmt.Sprintf("invalid duration format: %s", err.Error()))
		}

		endTime = time.Now()
		startTime = endTime.Add(-time.Duration(duration))
	} else {
		// Handle explicit start/end times
		startTime, err = prometheus.ParseTimestamp(startStr)
		if err != nil {
			return errorResult(fmt.Sprintf("invalid start time format: %s", err.Error()))
		}

		endTime, err = prometheus.ParseTimestamp(endStr)
		if err != nil {
			return errorResult(fmt.Sprintf("invalid end time format: %s", err.Error()))
		}
	}

	// Execute the range query
	result, err := promClient.ExecuteRangeQuery(params.Context, query, startTime, endTime, time.Duration(stepDuration))
	if err != nil {
		return errorResult(fmt.Sprintf("failed to execute range query: %s", err.Error()))
	}

	// Convert to structured output
	output := RangeQueryOutput{
		ResultType: fmt.Sprintf("%v", result["resultType"]),
	}

	resMatrix, ok := result["result"].(model.Matrix)
	if ok {
		slog.Info("ExecuteRangeQueryHandler executed successfully", "resultLength", resMatrix.Len())
		slog.Debug("ExecuteRangeQueryHandler results", "results", resMatrix)

		output.Result = make([]SeriesResult, len(resMatrix))
		for i, series := range resMatrix {
			labels := make(map[string]string)
			for k, v := range series.Metric {
				labels[string(k)] = string(v)
			}
			values := make([][]any, len(series.Values))
			for j, sample := range series.Values {
				values[j] = []any{float64(sample.Timestamp) / 1000, sample.Value.String()}
			}
			output.Result[i] = SeriesResult{
				Metric: labels,
				Values: values,
			}
		}
	} else {
		slog.Info("ExecuteRangeQueryHandler executed successfully (unknown format)", "result", result)
	}

	if warnings, ok := result["warnings"].([]string); ok {
		output.Warnings = warnings
	}

	// Convert to JSON for fallback text
	jsonResult, err := json.Marshal(output)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to marshal result: %s", err.Error()))
	}

	return api.NewToolCallResult(string(jsonResult), nil), nil
}

// GetLabelNamesHandler handles the retrieval of label names.
func GetLabelNamesHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	slog.Info("GetLabelNamesHandler called")

	promClient, err := getPromClient(params)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to create Prometheus client: %s", err.Error()))
	}

	// Get optional parameters
	metric := getStringArg(params, "metric", "")
	startStr := getStringArg(params, "start", "")
	endStr := getStringArg(params, "end", "")

	// Default to last hour if not specified
	var startTime, endTime time.Time
	if startStr == "" && endStr == "" {
		endTime = time.Now()
		startTime = endTime.Add(-prometheus.ListMetricsTimeRange)
	} else {
		if startStr != "" {
			startTime, err = prometheus.ParseTimestamp(startStr)
			if err != nil {
				return errorResult(fmt.Sprintf("invalid start time format: %s", err.Error()))
			}
		}
		if endStr != "" {
			endTime, err = prometheus.ParseTimestamp(endStr)
			if err != nil {
				return errorResult(fmt.Sprintf("invalid end time format: %s", err.Error()))
			}
		}
	}

	// Get label names
	labels, err := promClient.GetLabelNames(params.Context, metric, startTime, endTime)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to get label names: %s", err.Error()))
	}

	output := LabelNamesOutput{Labels: labels}

	slog.Info("GetLabelNamesHandler executed successfully", "labelCount", len(labels))
	slog.Debug("GetLabelNamesHandler results", "results", labels)

	jsonResult, err := json.Marshal(output)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to marshal label names: %s", err.Error()))
	}

	return api.NewToolCallResult(string(jsonResult), nil), nil
}

// GetLabelValuesHandler handles the retrieval of label values.
func GetLabelValuesHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	slog.Info("GetLabelValuesHandler called")

	promClient, err := getPromClient(params)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to create Prometheus client: %s", err.Error()))
	}

	// Get required label parameter
	label, ok := params.GetArguments()["label"].(string)
	if !ok || label == "" {
		return errorResult("label parameter is required and must be a string")
	}

	// Get optional parameters
	metric := getStringArg(params, "metric", "")
	startStr := getStringArg(params, "start", "")
	endStr := getStringArg(params, "end", "")

	// Default to last hour if not specified
	var startTime, endTime time.Time
	if startStr == "" && endStr == "" {
		endTime = time.Now()
		startTime = endTime.Add(-prometheus.ListMetricsTimeRange)
	} else {
		if startStr != "" {
			startTime, err = prometheus.ParseTimestamp(startStr)
			if err != nil {
				return errorResult(fmt.Sprintf("invalid start time format: %s", err.Error()))
			}
		}
		if endStr != "" {
			endTime, err = prometheus.ParseTimestamp(endStr)
			if err != nil {
				return errorResult(fmt.Sprintf("invalid end time format: %s", err.Error()))
			}
		}
	}

	// Get label values
	values, err := promClient.GetLabelValues(params.Context, label, metric, startTime, endTime)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to get label values: %s", err.Error()))
	}

	output := LabelValuesOutput{Values: values}

	slog.Info("GetLabelValuesHandler executed successfully", "valueCount", len(values))
	slog.Debug("GetLabelValuesHandler results", "results", values)

	jsonResult, err := json.Marshal(output)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to marshal label values: %s", err.Error()))
	}

	return api.NewToolCallResult(string(jsonResult), nil), nil
}

// GetSeriesHandler handles the retrieval of time series.
func GetSeriesHandler(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	slog.Info("GetSeriesHandler called")

	promClient, err := getPromClient(params)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to create Prometheus client: %s", err.Error()))
	}

	// Get required matches parameter
	matchesStr, ok := params.GetArguments()["matches"].(string)
	if !ok || matchesStr == "" {
		return errorResult("matches parameter is required and must be a string")
	}

	// Parse matches - could be comma-separated
	matches := []string{matchesStr}

	// Get optional parameters
	startStr := getStringArg(params, "start", "")
	endStr := getStringArg(params, "end", "")

	// Default to last hour if not specified
	var startTime, endTime time.Time
	if startStr == "" && endStr == "" {
		endTime = time.Now()
		startTime = endTime.Add(-prometheus.ListMetricsTimeRange)
	} else {
		if startStr != "" {
			startTime, err = prometheus.ParseTimestamp(startStr)
			if err != nil {
				return errorResult(fmt.Sprintf("invalid start time format: %s", err.Error()))
			}
		}
		if endStr != "" {
			endTime, err = prometheus.ParseTimestamp(endStr)
			if err != nil {
				return errorResult(fmt.Sprintf("invalid end time format: %s", err.Error()))
			}
		}
	}

	// Get series
	series, err := promClient.GetSeries(params.Context, matches, startTime, endTime)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to get series: %s", err.Error()))
	}

	output := SeriesOutput{
		Series:      series,
		Cardinality: len(series),
	}

	slog.Info("GetSeriesHandler executed successfully", "cardinality", len(series))
	slog.Debug("GetSeriesHandler results", "results", series)

	jsonResult, err := json.Marshal(output)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to marshal series: %s", err.Error()))
	}

	return api.NewToolCallResult(string(jsonResult), nil), nil
}
