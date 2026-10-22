package mustgather

import (
	"fmt"
	"strings"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	mg "github.com/containers/kubernetes-mcp-server/pkg/ocp/mustgather"
	"github.com/google/jsonschema-go/jsonschema"
	"k8s.io/utils/ptr"
)

func initMonitoring() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        "mustgather_monitoring_prometheus_status",
				Description: "Get Prometheus TSDB and runtime status from the must-gather archive",
				Annotations: api.ToolAnnotations{
					Title:        "Prometheus Status",
					ReadOnlyHint: ptr.To(true),
				},
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"archive_id": archiveIDProperty(),
						"replica":    {Type: "string", Description: "Prometheus replica (0, 1, or all). Default: all"},
					},
					Required: []string{"archive_id"},
				},
			},
			RBAC:         api.RBACNone(),
			Handler:      mustgatherMonitoringPrometheusStatus,
			ClusterAware: ptr.To(false),
		},
		{
			Tool: api.Tool{
				Name:        "mustgather_monitoring_prometheus_targets",
				Description: "Get Prometheus scrape targets and their health status from the must-gather archive",
				Annotations: api.ToolAnnotations{
					Title:        "Prometheus Targets",
					ReadOnlyHint: ptr.To(true),
				},
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"archive_id": archiveIDProperty(),
						"replica":    {Type: "string", Description: "Prometheus replica (0, 1, or all). Default: 0"},
						"health":     {Type: "string", Description: "Filter by health status: up, down, unknown (default: all)"},
					},
					Required: []string{"archive_id"},
				},
			},
			RBAC:         api.RBACNone(),
			Handler:      mustgatherMonitoringPrometheusTargets,
			ClusterAware: ptr.To(false),
		},
		{
			Tool: api.Tool{
				Name:        "mustgather_monitoring_prometheus_tsdb",
				Description: "Get detailed Prometheus TSDB statistics including top metrics by series count and label cardinality",
				Annotations: api.ToolAnnotations{
					Title:        "Prometheus TSDB",
					ReadOnlyHint: ptr.To(true),
				},
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"archive_id": archiveIDProperty(),
						"replica":    {Type: "string", Description: "Prometheus replica (0, 1, or all). Default: 0"},
						"limit":      {Type: "integer", Description: "Number of top entries to show per category (default: 10)"},
					},
					Required: []string{"archive_id"},
				},
			},
			RBAC:         api.RBACNone(),
			Handler:      mustgatherMonitoringPrometheusTSDB,
			ClusterAware: ptr.To(false),
		},
		{
			Tool: api.Tool{
				Name:        "mustgather_monitoring_prometheus_alerts",
				Description: "Get active Prometheus alerts from the must-gather archive",
				Annotations: api.ToolAnnotations{
					Title:        "Prometheus Alerts",
					ReadOnlyHint: ptr.To(true),
				},
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"archive_id": archiveIDProperty(),
						"state":      {Type: "string", Description: "Filter by alert state: firing, pending (default: all)"},
					},
					Required: []string{"archive_id"},
				},
			},
			RBAC:         api.RBACNone(),
			Handler:      mustgatherMonitoringPrometheusAlerts,
			ClusterAware: ptr.To(false),
		},
		{
			Tool: api.Tool{
				Name:        "mustgather_monitoring_prometheus_rules",
				Description: "Get Prometheus alerting and recording rules from the must-gather archive",
				Annotations: api.ToolAnnotations{
					Title:        "Prometheus Rules",
					ReadOnlyHint: ptr.To(true),
				},
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"archive_id": archiveIDProperty(),
						"type":       {Type: "string", Description: "Filter by rule type: alerting, recording (default: all)"},
					},
					Required: []string{"archive_id"},
				},
			},
			RBAC:         api.RBACNone(),
			Handler:      mustgatherMonitoringPrometheusRules,
			ClusterAware: ptr.To(false),
		},
	}
}

func mustgatherMonitoringPrometheusStatus(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()
	id := getString(args, "archive_id", "")
	p, err := providerForArchive(params, id)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	replica := getString(args, "replica", "all")
	replicas := mg.GetReplicaNumbers(replica)

	output := api.NewBuilderWithBudget(api.ResultLimit(params.Context))
	output.WriteString("## Prometheus Status\n\n")

	for _, r := range replicas {
		if !output.Fits(0) {
			output.WriteString("\n")
			break
		}
		output.WriteString(fmt.Sprintf("### Replica prometheus-k8s-%d\n\n", r))

		runtime, err := p.GetPrometheusRuntimeInfo(r)
		if err != nil {
			output.WriteString(fmt.Sprintf("Runtime info not available: %v\n\n", err))
		} else {
			output.WriteString("#### Runtime Info\n\n")
			output.WriteString(fmt.Sprintf("Start Time:       %s\n", runtime.StartTime))
			output.WriteString(fmt.Sprintf("Storage Retention: %s\n", runtime.StorageRetention))
			output.WriteString(fmt.Sprintf("Config Reload:    %s (last: %s)\n",
				healthSymbol(fmt.Sprintf("%v", runtime.ReloadConfigSuccess)), runtime.LastConfigTime))
			output.WriteString(fmt.Sprintf("Goroutines:       %d\n", runtime.GoroutineCount))
			output.WriteString(fmt.Sprintf("GOMAXPROCS:       %d\n", runtime.GOMAXPROCS))
			if runtime.CorruptionCount > 0 {
				output.WriteString(fmt.Sprintf("Corruptions:      [WARNING] %d\n", runtime.CorruptionCount))
			}
			output.WriteString("\n")
		}
		if !output.Fits(0) {
			output.WriteString("\n")
			break
		}

		tsdb, err := p.GetPrometheusTSDB(r)
		if err != nil {
			output.WriteString(fmt.Sprintf("TSDB info not available: %v\n\n", err))
		} else {
			output.WriteString("#### TSDB Head Stats\n\n")
			output.WriteString(fmt.Sprintf("Series:      %s\n", formatNumber(tsdb.HeadStats.NumSeries)))
			output.WriteString(fmt.Sprintf("Label Pairs: %s\n", formatNumber(tsdb.HeadStats.NumLabelPairs)))
			output.WriteString(fmt.Sprintf("Chunks:      %s\n", formatNumber(tsdb.HeadStats.ChunkCount)))
			output.WriteString("\n")
		}
	}

	return api.NewToolCallResult(output.String(), nil), nil
}

func mustgatherMonitoringPrometheusTargets(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()
	id := getString(args, "archive_id", "")
	p, err := providerForArchive(params, id)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	replica := getString(args, "replica", "0")
	healthFilter := getString(args, "health", "")

	replicas := mg.GetReplicaNumbers(replica)
	r := replicas[0]

	targets, err := p.GetPrometheusActiveTargets(r)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get Prometheus targets: %w", err)), nil
	}

	var upCount, downCount, unknownCount int
	for i := range targets {
		t := &targets[i]
		switch strings.ToLower(t.Health) {
		case "up":
			upCount++
		case "down":
			downCount++
		default:
			unknownCount++
		}
	}

	output := api.NewBuilderWithBudget(api.ResultLimit(params.Context))
	output.WriteString(fmt.Sprintf("## Prometheus Targets (replica %d)\n\n", r))
	output.WriteString(fmt.Sprintf("Total: %d targets (Up: %d, Down: %d, Unknown: %d)\n\n",
		len(targets), upCount, downCount, unknownCount))

	if healthFilter != "" {
		output.WriteString(fmt.Sprintf("Showing targets with health: %s\n\n", healthFilter))
	}

	for i := range targets {
		if !output.Fits(0) {
			output.WriteString("\n")
			break
		}
		t := &targets[i]
		if healthFilter != "" && !strings.EqualFold(t.Health, healthFilter) {
			continue
		}
		output.WriteString(fmt.Sprintf("%s %s\n", healthSymbol(t.Health), t.ScrapePool))
		output.WriteString(fmt.Sprintf("  URL: %s\n", t.ScrapeURL))
		output.WriteString(fmt.Sprintf("  Last Scrape: %s (duration: %s)\n", t.LastScrape, formatDuration(t.LastScrapeDuration)))
		if t.LastError != "" {
			output.WriteString(fmt.Sprintf("  Error: %s\n", t.LastError))
		}
		output.WriteString("\n")
	}

	return api.NewToolCallResult(output.String(), nil), nil
}

func mustgatherMonitoringPrometheusTSDB(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()
	id := getString(args, "archive_id", "")
	p, err := providerForArchive(params, id)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	replica := getString(args, "replica", "0")
	limit := getInt(args, "limit", 10)

	replicas := mg.GetReplicaNumbers(replica)

	output := api.NewBuilderWithBudget(api.ResultLimit(params.Context))
	output.WriteString("## Prometheus TSDB Statistics\n\n")

	for _, r := range replicas {
		if !output.Fits(0) {
			output.WriteString("\n")
			break
		}
		tsdb, err := p.GetPrometheusTSDB(r)
		if err != nil {
			output.WriteString(fmt.Sprintf("### Replica %d: not available (%v)\n\n", r, err))
			continue
		}

		output.WriteString(fmt.Sprintf("### Replica prometheus-k8s-%d\n\n", r))
		output.WriteString("#### Head Stats\n\n")
		output.WriteString(fmt.Sprintf("Series:      %s\n", formatNumber(tsdb.HeadStats.NumSeries)))
		output.WriteString(fmt.Sprintf("Label Pairs: %s\n", formatNumber(tsdb.HeadStats.NumLabelPairs)))
		output.WriteString(fmt.Sprintf("Chunks:      %s\n\n", formatNumber(tsdb.HeadStats.ChunkCount)))

		if output.Fits(0) {
			output.WriteString(formatNameValueSection("Top Metrics by Series Count", tsdb.SeriesCountByMetricName, limit))
		}
		if output.Fits(0) {
			output.WriteString(formatNameValueSection("Top Labels by Value Count", tsdb.LabelValueCountByLabelName, limit))
		}
		if output.Fits(0) {
			output.WriteString(formatNameValueBytesSection("Top Labels by Memory Usage", tsdb.MemoryInBytesByLabelName, limit))
		}
	}

	return api.NewToolCallResult(output.String(), nil), nil
}

func mustgatherMonitoringPrometheusAlerts(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()
	id := getString(args, "archive_id", "")
	p, err := providerForArchive(params, id)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	stateFilter := getString(args, "state", "")

	rules, err := p.GetPrometheusRules()
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get Prometheus rules: %w", err)), nil
	}

	var alerts []alertEntry
	for _, group := range rules.Groups {
		for i := range group.Rules {
			rule := &group.Rules[i]
			if rule.Type != "alerting" {
				continue
			}
			for j := range rule.Alerts {
				alert := &rule.Alerts[j]
				if stateFilter != "" && !strings.EqualFold(alert.State, stateFilter) {
					continue
				}
				alerts = append(alerts, alertEntry{
					name:     rule.Name,
					state:    alert.State,
					severity: alert.Labels["severity"],
					activeAt: alert.ActiveAt,
					labels:   alert.Labels,
					summary:  alert.Annotations["summary"],
					message:  alert.Annotations["message"],
				})
			}
		}
	}

	output := api.NewBuilderWithBudget(api.ResultLimit(params.Context))
	output.WriteString("## Prometheus Alerts\n\n")
	if stateFilter != "" {
		output.WriteString(fmt.Sprintf("Filter: %s\n\n", stateFilter))
	}
	output.WriteString(fmt.Sprintf("Found %d active alert(s)\n\n", len(alerts)))

	if len(alerts) == 0 {
		output.WriteString("No active alerts found.")
		return api.NewToolCallResult(output.String(), nil), nil
	}

	for _, a := range alerts {
		if !output.Fits(0) {
			output.WriteString("\n")
			break
		}
		stateTag := "[" + strings.ToUpper(a.state) + "]"
		output.WriteString(fmt.Sprintf("%s %s %s\n", severitySymbol(a.severity), stateTag, a.name))
		output.WriteString(fmt.Sprintf("  Active Since: %s\n", a.activeAt))
		if a.summary != "" {
			output.WriteString(fmt.Sprintf("  Summary: %s\n", a.summary))
		} else if a.message != "" {
			output.WriteString(fmt.Sprintf("  Message: %s\n", a.message))
		}
		// Show namespace if present
		if ns, ok := a.labels["namespace"]; ok {
			output.WriteString(fmt.Sprintf("  Namespace: %s\n", ns))
		}
		output.WriteString("\n")
	}

	return api.NewToolCallResult(output.String(), nil), nil
}

func mustgatherMonitoringPrometheusRules(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()
	id := getString(args, "archive_id", "")
	p, err := providerForArchive(params, id)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	typeFilter := getString(args, "type", "")

	rules, err := p.GetPrometheusRules()
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get Prometheus rules: %w", err)), nil
	}

	totalGroups := 0
	totalRules := 0
	for _, group := range rules.Groups {
		found := false
		for _, rule := range group.Rules {
			if typeFilter == "" || strings.EqualFold(rule.Type, typeFilter) {
				totalRules++
				found = true
			}
		}
		if found {
			totalGroups++
		}
	}
	output := api.NewBuilderWithBudget(api.ResultLimit(params.Context))
	output.WriteString("## Prometheus Rules\n\n")
	if typeFilter != "" {
		output.WriteString(fmt.Sprintf("Filter: %s rules only\n\n", typeFilter))
	}
	output.WriteString(fmt.Sprintf("Total: %d groups, %d rules\n\n", totalGroups, totalRules))

	for _, group := range rules.Groups {
		if !output.Fits(0) {
			output.WriteString("\n")
			break
		}
		found := false
		for _, rule := range group.Rules {
			if typeFilter == "" || strings.EqualFold(rule.Type, typeFilter) {
				found = true
				break
			}
		}
		if !found {
			continue
		}

		output.WriteString(fmt.Sprintf("### %s (%s)\n\n", group.Name, group.File))
		output.WriteString(fmt.Sprintf("Eval Interval: %.0fs | Last Eval: %s | Eval Time: %s\n\n",
			group.Interval, group.LastEvaluation, formatDuration(group.EvaluationTime)))

		for i := range group.Rules {
			if !output.Fits(0) {
				output.WriteString("\n")
				break
			}
			r := &group.Rules[i]
			if typeFilter != "" && !strings.EqualFold(r.Type, typeFilter) {
				continue
			}
			output.WriteString(fmt.Sprintf("- %s [%s] %s %s\n", healthSymbol(r.Health), r.Type, r.Name, r.State))
			if r.LastError != "" {
				output.WriteString(fmt.Sprintf("  Error: %s\n", r.LastError))
			}
		}
		output.WriteString("\n")
	}

	return api.NewToolCallResult(output.String(), nil), nil
}

// alertEntry holds formatted alert data for display
type alertEntry struct {
	name     string
	state    string
	severity string
	activeAt string
	labels   map[string]string
	summary  string
	message  string
}

// formatNameValueSection formats a list of name-value pairs as a section
func formatNameValueSection(title string, items []mg.NameValue, limit int) string {
	output := fmt.Sprintf("#### %s\n\n", title)
	if len(items) == 0 {
		return output + "No data available.\n\n"
	}
	count := len(items)
	if limit > 0 && count > limit {
		count = limit
	}
	for i := 0; i < count; i++ {
		output += fmt.Sprintf("%-60s %s\n", truncate(items[i].Name, 60), formatNumber(items[i].Value))
	}
	output += "\n"
	return output
}

// formatNameValueBytesSection formats a list of name-value pairs with byte formatting
func formatNameValueBytesSection(title string, items []mg.NameValue, limit int) string {
	output := fmt.Sprintf("#### %s\n\n", title)
	if len(items) == 0 {
		return output + "No data available.\n\n"
	}
	count := len(items)
	if limit > 0 && count > limit {
		count = limit
	}
	for i := 0; i < count; i++ {
		output += fmt.Sprintf("%-60s %s\n", truncate(items[i].Name, 60), formatBytes(items[i].Value))
	}
	output += "\n"
	return output
}
