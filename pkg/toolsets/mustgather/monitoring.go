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
						"replica": {Type: "string", Description: "Prometheus replica (0, 1, or all). Default: all"},
					},
				},
			},
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
						"replica": {Type: "string", Description: "Prometheus replica (0, 1, or all). Default: 0"},
						"health":  {Type: "string", Description: "Filter by health status: up, down, unknown (default: all)"},
					},
				},
			},
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
						"replica": {Type: "string", Description: "Prometheus replica (0, 1, or all). Default: 0"},
						"limit":   {Type: "integer", Description: "Number of top entries to show per category (default: 10)"},
					},
				},
			},
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
						"state": {Type: "string", Description: "Filter by alert state: firing, pending (default: all)"},
					},
				},
			},
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
						"type": {Type: "string", Description: "Filter by rule type: alerting, recording (default: all)"},
					},
				},
			},
			Handler:      mustgatherMonitoringPrometheusRules,
			ClusterAware: ptr.To(false),
		},
		{
			Tool: api.Tool{
				Name:        "mustgather_monitoring_prometheus_config_summary",
				Description: "Get a summary of Prometheus configuration from the must-gather archive",
				Annotations: api.ToolAnnotations{
					Title:        "Prometheus Config",
					ReadOnlyHint: ptr.To(true),
				},
				InputSchema: &jsonschema.Schema{
					Type: "object",
				},
			},
			Handler:      mustgatherMonitoringPrometheusConfigSummary,
			ClusterAware: ptr.To(false),
		},
		{
			Tool: api.Tool{
				Name:        "mustgather_monitoring_alertmanager_status",
				Description: "Get AlertManager cluster status from the must-gather archive",
				Annotations: api.ToolAnnotations{
					Title:        "AlertManager Status",
					ReadOnlyHint: ptr.To(true),
				},
				InputSchema: &jsonschema.Schema{
					Type: "object",
				},
			},
			Handler:      mustgatherMonitoringAlertManagerStatus,
			ClusterAware: ptr.To(false),
		},
	}
}

func mustgatherMonitoringPrometheusStatus(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p, err := getProvider()
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	replica := getString(params.GetArguments(), "replica", "all")
	replicas := mg.GetReplicaNumbers(replica)

	output := "## Prometheus Status\n\n"

	for _, r := range replicas {
		output += fmt.Sprintf("### Replica prometheus-k8s-%d\n\n", r)

		runtime, err := p.GetPrometheusRuntimeInfo(r)
		if err != nil {
			output += fmt.Sprintf("Runtime info not available: %v\n\n", err)
		} else {
			output += "#### Runtime Info\n\n"
			output += fmt.Sprintf("Start Time:       %s\n", runtime.StartTime)
			output += fmt.Sprintf("Storage Retention: %s\n", runtime.StorageRetention)
			output += fmt.Sprintf("Config Reload:    %s (last: %s)\n",
				healthSymbol(fmt.Sprintf("%v", runtime.ReloadConfigSuccess)), runtime.LastConfigTime)
			output += fmt.Sprintf("Goroutines:       %d\n", runtime.GoroutineCount)
			output += fmt.Sprintf("GOMAXPROCS:       %d\n", runtime.GOMAXPROCS)
			if runtime.CorruptionCount > 0 {
				output += fmt.Sprintf("Corruptions:      [WARNING] %d\n", runtime.CorruptionCount)
			}
			output += "\n"
		}

		tsdb, err := p.GetPrometheusTSDB(r)
		if err != nil {
			output += fmt.Sprintf("TSDB info not available: %v\n\n", err)
		} else {
			output += "#### TSDB Head Stats\n\n"
			output += fmt.Sprintf("Series:      %s\n", formatNumber(tsdb.HeadStats.NumSeries))
			output += fmt.Sprintf("Label Pairs: %s\n", formatNumber(tsdb.HeadStats.NumLabelPairs))
			output += fmt.Sprintf("Chunks:      %s\n", formatNumber(tsdb.HeadStats.ChunkCount))
			output += "\n"
		}
	}

	return api.NewToolCallResult(output, nil), nil
}

func mustgatherMonitoringPrometheusTargets(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p, err := getProvider()
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	args := params.GetArguments()
	replica := getString(args, "replica", "0")
	healthFilter := getString(args, "health", "")

	replicas := mg.GetReplicaNumbers(replica)
	r := replicas[0]

	targets, err := p.GetPrometheusActiveTargets(r)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get Prometheus targets: %w", err)), nil
	}

	var upCount, downCount, unknownCount int
	var filtered []mg.ActiveTarget
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
		if healthFilter == "" || strings.EqualFold(t.Health, healthFilter) {
			filtered = append(filtered, *t)
		}
	}

	output := fmt.Sprintf("## Prometheus Targets (replica %d)\n\n", r)
	output += fmt.Sprintf("Total: %d targets (Up: %d, Down: %d, Unknown: %d)\n\n",
		len(targets), upCount, downCount, unknownCount)

	if healthFilter != "" {
		output += fmt.Sprintf("Showing targets with health: %s\n\n", healthFilter)
	}

	for i := range filtered {
		t := &filtered[i]
		output += fmt.Sprintf("%s %s\n", healthSymbol(t.Health), t.ScrapePool)
		output += fmt.Sprintf("  URL: %s\n", t.ScrapeURL)
		output += fmt.Sprintf("  Last Scrape: %s (duration: %s)\n", t.LastScrape, formatDuration(t.LastScrapeDuration))
		if t.LastError != "" {
			output += fmt.Sprintf("  Error: %s\n", t.LastError)
		}
		output += "\n"
	}

	return api.NewToolCallResult(output, nil), nil
}

func mustgatherMonitoringPrometheusTSDB(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p, err := getProvider()
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	args := params.GetArguments()
	replica := getString(args, "replica", "0")
	limit := getInt(args, "limit", 10)

	replicas := mg.GetReplicaNumbers(replica)

	output := "## Prometheus TSDB Statistics\n\n"

	for _, r := range replicas {
		tsdb, err := p.GetPrometheusTSDB(r)
		if err != nil {
			output += fmt.Sprintf("### Replica %d: not available (%v)\n\n", r, err)
			continue
		}

		output += fmt.Sprintf("### Replica prometheus-k8s-%d\n\n", r)
		output += "#### Head Stats\n\n"
		output += fmt.Sprintf("Series:      %s\n", formatNumber(tsdb.HeadStats.NumSeries))
		output += fmt.Sprintf("Label Pairs: %s\n", formatNumber(tsdb.HeadStats.NumLabelPairs))
		output += fmt.Sprintf("Chunks:      %s\n\n", formatNumber(tsdb.HeadStats.ChunkCount))

		output += formatNameValueSection("Top Metrics by Series Count", tsdb.SeriesCountByMetricName, limit)
		output += formatNameValueSection("Top Labels by Value Count", tsdb.LabelValueCountByLabelName, limit)
		output += formatNameValueBytesSection("Top Labels by Memory Usage", tsdb.MemoryInBytesByLabelName, limit)
	}

	return api.NewToolCallResult(output, nil), nil
}

func mustgatherMonitoringPrometheusAlerts(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p, err := getProvider()
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	stateFilter := getString(params.GetArguments(), "state", "")

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

	output := "## Prometheus Alerts\n\n"
	if stateFilter != "" {
		output += fmt.Sprintf("Filter: %s\n\n", stateFilter)
	}
	output += fmt.Sprintf("Found %d active alert(s)\n\n", len(alerts))

	if len(alerts) == 0 {
		return api.NewToolCallResult(output+"No active alerts found.", nil), nil
	}

	for _, a := range alerts {
		stateTag := "[" + strings.ToUpper(a.state) + "]"
		output += fmt.Sprintf("%s %s %s\n", severitySymbol(a.severity), stateTag, a.name)
		output += fmt.Sprintf("  Active Since: %s\n", a.activeAt)
		if a.summary != "" {
			output += fmt.Sprintf("  Summary: %s\n", a.summary)
		} else if a.message != "" {
			output += fmt.Sprintf("  Message: %s\n", a.message)
		}
		// Show namespace if present
		if ns, ok := a.labels["namespace"]; ok {
			output += fmt.Sprintf("  Namespace: %s\n", ns)
		}
		output += "\n"
	}

	return api.NewToolCallResult(output, nil), nil
}

func mustgatherMonitoringPrometheusRules(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p, err := getProvider()
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	typeFilter := getString(params.GetArguments(), "type", "")

	rules, err := p.GetPrometheusRules()
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get Prometheus rules: %w", err)), nil
	}

	output := "## Prometheus Rules\n\n"
	if typeFilter != "" {
		output += fmt.Sprintf("Filter: %s rules only\n\n", typeFilter)
	}

	totalGroups := 0
	totalRules := 0

	for _, group := range rules.Groups {
		var groupRules []mg.Rule
		for i := range group.Rules {
			rule := &group.Rules[i]
			if typeFilter == "" || strings.EqualFold(rule.Type, typeFilter) {
				groupRules = append(groupRules, *rule)
			}
		}
		if len(groupRules) == 0 {
			continue
		}

		totalGroups++
		totalRules += len(groupRules)

		output += fmt.Sprintf("### %s (%s)\n\n", group.Name, group.File)
		output += fmt.Sprintf("Eval Interval: %.0fs | Last Eval: %s | Eval Time: %s\n\n",
			group.Interval, group.LastEvaluation, formatDuration(group.EvaluationTime))

		for i := range groupRules {
			r := &groupRules[i]
			output += fmt.Sprintf("- %s [%s] %s %s\n", healthSymbol(r.Health), r.Type, r.Name, r.State)
			if r.LastError != "" {
				output += fmt.Sprintf("  Error: %s\n", r.LastError)
			}
		}
		output += "\n"
	}

	summary := fmt.Sprintf("Total: %d groups, %d rules\n\n", totalGroups, totalRules)
	// Insert summary after header
	output = strings.Replace(output, "\n\n", "\n\n"+summary, 1)

	return api.NewToolCallResult(output, nil), nil
}

func mustgatherMonitoringPrometheusConfigSummary(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p, err := getProvider()
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	output := "## Prometheus Configuration Summary\n\n"

	config, err := p.GetPrometheusConfig()
	if err != nil {
		output += fmt.Sprintf("Config not available: %v\n\n", err)
	} else {
		// Show a truncated version of the config YAML
		configYAML := config.YAML
		lines := strings.Split(configYAML, "\n")
		if len(lines) > 100 {
			output += fmt.Sprintf("Configuration (%d lines, showing first 100):\n\n", len(lines))
			output += strings.Join(lines[:100], "\n") + "\n...\n\n"
		} else {
			output += "Configuration:\n\n" + configYAML + "\n\n"
		}
	}

	flags, err := p.GetPrometheusFlags()
	if err != nil {
		output += fmt.Sprintf("Flags not available: %v\n\n", err)
	} else {
		output += "### Key Flags\n\n"
		keyFlags := []string{
			"storage.tsdb.retention.time",
			"storage.tsdb.retention.size",
			"storage.tsdb.path",
			"web.listen-address",
			"web.external-url",
			"rules.alert.for-outage-tolerance",
			"rules.alert.for-grace-period",
		}
		for _, key := range keyFlags {
			if val, ok := flags[key]; ok {
				output += fmt.Sprintf("%-40s %s\n", key, val)
			}
		}
		output += "\n"
	}

	return api.NewToolCallResult(output, nil), nil
}

func mustgatherMonitoringAlertManagerStatus(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	p, err := getProvider()
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	status, err := p.GetAlertManagerStatus()
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get AlertManager status: %w", err)), nil
	}

	output := "## AlertManager Status\n\n"
	output += fmt.Sprintf("Cluster Status: %s\n", healthSymbol(status.Cluster.Status))
	output += fmt.Sprintf("Uptime: %s\n\n", status.Uptime)

	output += "### Version\n\n"
	output += fmt.Sprintf("Version:    %s\n", status.VersionInfo.Version)
	output += fmt.Sprintf("Revision:   %s\n", status.VersionInfo.Revision)
	output += fmt.Sprintf("Branch:     %s\n", status.VersionInfo.Branch)
	output += fmt.Sprintf("Build Date: %s\n", status.VersionInfo.BuildDate)
	output += fmt.Sprintf("Go Version: %s\n\n", status.VersionInfo.GoVersion)

	if len(status.Cluster.Peers) > 0 {
		output += "### Cluster Peers\n\n"
		for _, peer := range status.Cluster.Peers {
			output += fmt.Sprintf("- %s (%s)\n", peer.Name, peer.Address)
		}
		output += "\n"
	}

	return api.NewToolCallResult(output, nil), nil
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
