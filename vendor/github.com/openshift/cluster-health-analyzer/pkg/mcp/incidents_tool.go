package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/openshift/cluster-health-analyzer/pkg/alertmanager"
	"github.com/openshift/cluster-health-analyzer/pkg/common"
	"github.com/openshift/cluster-health-analyzer/pkg/processor"
	"github.com/openshift/cluster-health-analyzer/pkg/prom"
	"github.com/prometheus/alertmanager/api/v2/models"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

var (
	// Format response with instructions for better LLM interpretation
	getIncidentsResponseTemplate = `<DATA>
%s
</DATA>
<INSTRUCTIONS>
- An incident is a group of related alerts. Base your analysis on the alerts to understand the incident. 
- Don't confuse or mix the concepts of incident and alert during your explanation.
- For each incident, analyze its alerts to identify the affected components and the core problem. 
- Whenever you print an incident ID, add also a short one-sentence summary of the incident (e.g. "etcd degradation", "ingress failure")
- If the user asks about a problem you cannot find in the data, do not guess. State that you cannot find the cause and simply list the incidents.
</INSTRUCTIONS>`
)

const (
	// GetIncidentsDescription describes the get_incidents MCP tool.
	GetIncidentsDescription = `List the current firing incidents in the cluster. 
		One incident is a group of related alerts that are likely triggered by the same root cause.
		Use this tool to analyze the cluster health status and determine why a component is failing or degraded.
		`

	getIncidentsToolName  = "get_incidents"
	minTimeRangeHours     = 1
	defaultTimeRangeHours = 360

	clusterIDStr = "clusterID"
	defaultStr   = "default"
	silencedStr  = "silenced"
)

type IncidentTool struct {
	Tool mcp.Tool
	cfg  incidentToolCfg
	// the followings allow to use mocked instance of needed clients for testing
	getPrometheusLoaderFn   func(string, string) (prom.Loader, error)
	getAlertManagerLoaderFn func(string, string) (alertmanager.Loader, error)
}

type incidentToolCfg struct {
	promURL         string
	alertManagerURL string
}

type GetIncidentsParams struct {
	TimeRange   float64 `json:"time_range"`
	MinSeverity string  `json:"min_severity"`
}

// NormalizeGetIncidentsParams applies defaults to omitted arguments and
// validates supplied hours and severity. The supplied flags distinguish
// omission from an explicit zero or empty value.
func NormalizeGetIncidentsParams(params GetIncidentsParams, timeRangeSupplied, minSeveritySupplied bool) (GetIncidentsParams, error) {
	if !timeRangeSupplied {
		params.TimeRange = defaultTimeRangeHours
	}
	if !(params.TimeRange >= minTimeRangeHours && params.TimeRange <= defaultTimeRangeHours) {
		return GetIncidentsParams{}, fmt.Errorf("time_range must be between %d and %d hours", minTimeRangeHours, defaultTimeRangeHours)
	}
	if !minSeveritySupplied {
		params.MinSeverity = processor.Warning.String()
	}
	switch strings.ToLower(params.MinSeverity) {
	case processor.Healthy.String(), processor.Warning.String(), processor.Critical.String():
	default:
		return GetIncidentsParams{}, errors.New("min_severity must be info, warning, or critical")
	}
	return params, nil
}

var (
	paramsByTool = map[string]map[string]*jsonschema.Schema{
		getIncidentsToolName: {
			"time_range": {
				Type:        "number",
				Default:     json.RawMessage([]byte(strconv.Itoa(defaultTimeRangeHours))),
				Description: "Maximum age of incidents to include in hours (max 360 for 15 days). Default: 360",
				Minimum:     jsonschema.Ptr(float64(minTimeRangeHours)),
				Maximum:     jsonschema.Ptr(float64(defaultTimeRangeHours)),
			},
			"min_severity": {
				Type:        "string",
				Default:     json.RawMessage([]byte(strconv.Quote(processor.Warning.String()))),
				Pattern:     fmt.Sprintf("(?i)^(%s|%s|%s)$", processor.Healthy.String(), processor.Warning.String(), processor.Critical.String()),
				Description: "Minimum severity level to be applied as filter for incidents. Allowed values, from lower severity to higher severity, can be: info, warning and critical. Default: warning.",
			},
		},
	}

	defaultMcpGetIncidentsTool = mcp.Tool{
		Name:        getIncidentsToolName,
		Description: GetIncidentsDescription,
		Annotations: &mcp.ToolAnnotations{
			Title:        "Provides information about Incidents in the cluster",
			ReadOnlyHint: true,
		},
		InputSchema: &jsonschema.Schema{
			Type:       "object",
			Properties: paramsByTool[getIncidentsToolName],
		},
	}
)

// NewIncidentsTool creates a new MCP tool for the incidents
func NewIncidentsTool(promURL, alertmanagerURL string) IncidentTool {
	return IncidentTool{
		Tool: defaultMcpGetIncidentsTool,
		cfg: incidentToolCfg{
			promURL:         promURL,
			alertManagerURL: alertmanagerURL,
		},
		getPrometheusLoaderFn:   defaultPrometheusLoader,
		getAlertManagerLoaderFn: defaultAlertManagerLoader,
	}
}

// IncidentsHandler is the main handler for the Incidents. It connects to the
// in-cluster Prometheus and queries the Incidents metrics.
func (i *IncidentTool) IncidentsHandler(ctx context.Context, request *mcp.CallToolRequest, params GetIncidentsParams) (*mcp.CallToolResult, any, error) {
	slog.Info("Incidents tool received request with ", "params", params)
	params, err := NormalizeGetIncidentsParams(params, params.TimeRange != 0, params.MinSeverity != "")
	if err != nil {
		return nil, nil, err
	}
	token, err := getTokenFromCtx(ctx)
	if err != nil {
		slog.Error(err.Error())
		return nil, nil, err
	}

	amLoader, err := i.getAlertManagerLoaderFn(i.cfg.alertManagerURL, token)
	if err != nil {
		slog.Error("Failed to initialize AlertManager client", "error", err)
		return nil, nil, err
	}

	promLoader, err := i.getPrometheusLoaderFn(i.cfg.promURL, token)
	if err != nil {
		slog.Error("Failed to initialize Prometheus client", "error", err)
		return nil, nil, err
	}

	return i.GetIncidents(ctx, params, promLoader, amLoader)
}

// GetIncidents retrieves incidents using the supplied clients, without
// depending on how the caller obtained its credentials.
func (i *IncidentTool) GetIncidents(ctx context.Context, params GetIncidentsParams, promLoader prom.Loader, amLoader alertmanager.Loader) (*mcp.CallToolResult, any, error) {
	timeRange := float64(defaultTimeRangeHours)
	if params.TimeRange > 0 {
		timeRange = params.TimeRange
	}

	// the method ParseHealthValue will default to warning in the case of not recognized severity
	minSeverity := processor.ParseHealthValue(params.MinSeverity)

	timeNow := time.Now()
	queryTimeRange := v1.Range{
		Start: timeNow.Add(-time.Duration(timeRange * float64(time.Hour))),
		End:   timeNow,
		Step:  300 * time.Second,
	}

	val, err := promLoader.LoadVectorRange(ctx, processor.ClusterHealthComponentsMap, queryTimeRange.Start, queryTimeRange.End, queryTimeRange.Step)
	if err != nil {
		slog.Error("Received error response from Prometheus", "error", err)
		return nil, nil, err
	}

	silences, err := amLoader.SilencedAlerts()
	if err != nil {
		slog.Error("Failed retrieving silenced alerts from AlertManager", "error", err)
		return nil, nil, err
	}
	clusterIDconsoleURL, err := getConsoleURL(ctx, promLoader)
	if err != nil {
		slog.Error("Failed retrieving console URL from metrics", "error", err)
	}
	incidentsMap, err := i.transformPromValueToIncident(val, queryTimeRange, clusterIDconsoleURL)
	if err != nil {
		slog.Error("Failed to transform metric data", "error", err)
		return nil, nil, err
	}

	incidents := filterIncidentsBySeverity(
		getAlertDataForIncidents(ctx, incidentsMap, silences, promLoader, queryTimeRange),
		minSeverity,
	)

	r := Response{
		Incidents: Incidents{
			Total:     len(incidents),
			Incidents: incidents,
		},
	}

	data, err := json.Marshal(r)
	if err != nil {
		slog.Error("Failed to marshal the Incident data", "error", err)
		return nil, nil, err
	}

	response := fmt.Sprintf(getIncidentsResponseTemplate, string(data))
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: response},
		},
	}, nil, nil
}

// formatToRFC3339 formats a time to RFC3339 string, returns empty string for zero time
func formatToRFC3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// processSampleTime calculates the delta between the two samples and if it's greater
// than the range step then the endTime is set, otherwise it returns zero endTime
func processSampleTime(firstSample, lastSample model.SamplePair, qRange v1.Range) (time.Time, time.Time) {
	startTime := firstSample.Timestamp.Time()
	var endTime time.Time

	if qRange.End.Sub(lastSample.Timestamp.Time()).Seconds() > qRange.Step.Seconds() {
		endTime = lastSample.Timestamp.Time()
	}
	return startTime, endTime
}

// transformPromValueToIncident transforms the metrics data to map of incidents
func (i *IncidentTool) transformPromValueToIncident(dataVec prom.RangeVector,
	qRange v1.Range,
	clusterIDConsoleURL map[string]string) (map[string]Incident, error) {

	incidents := make(map[string]Incident, len(dataVec))
	for _, v := range dataVec {

		alertSeverity := v.Metric["src_severity"]
		alertName := v.Metric["src_alertname"]

		if alertSeverity == "none" {
			slog.Debug("Skipping unknown severity ", "alert", alertName, "severity", alertSeverity)
			continue
		}

		lastSample := v.Samples[len(v.Samples)-1]
		firstSample := v.Samples[0]
		startTime, endTime := processSampleTime(firstSample, lastSample, qRange)

		labels := common.SrcLabels(model.Metric(v.Metric))

		healthyVal := processor.HealthValue(lastSample.Value)
		groupId := string(v.Metric["group_id"])
		component := string(v.Metric["component"])
		clusterName := string(v.Metric["cluster"])
		clusterID := string(v.Metric[clusterIDStr])

		if existingInc, ok := incidents[groupId]; ok {
			existingInc.ComponentsSet[component] = struct{}{}
			existingInc.AffectedComponents = slices.Collect(maps.Keys(existingInc.ComponentsSet))
			sort.Strings(existingInc.AffectedComponents)

			if _, ok := existingInc.AlertsSet[labels.String()]; !ok {
				existingInc.AlertsSet[labels.String()] = struct{}{}
				existingInc.Alerts = append(existingInc.Alerts, labels)
			}

			if healthyVal > processor.ParseHealthValue(existingInc.Severity) {
				existingInc.Severity = healthyVal.String()
			}
			err := existingInc.UpdateStartTime(startTime)
			if err != nil {
				slog.Error("Failed to parse the start time of an incident ", "error", err)
				continue
			}
			err = existingInc.UpdateEndTime(endTime)
			if err != nil {
				slog.Error("Failed to parse the end time of an incident ", "error", err)
				continue
			}
			existingInc.UpdateStatus()
			incidents[existingInc.GroupId] = existingInc
		} else {
			incident := Incident{
				Cluster:   clusterName,
				ClusterID: clusterID,
				GroupId:   groupId,
				Severity:  healthyVal.String(),
				StartTime: formatToRFC3339(startTime),
				EndTime:   formatToRFC3339(endTime),
				ComponentsSet: map[string]struct{}{
					component: {},
				},
				AffectedComponents: []string{component},
				Alerts:             []model.LabelSet{labels},
				AlertsSet: map[string]struct{}{
					labels.String(): {},
				},
			}
			if clusterIDConsoleURL != nil {
				if clusterID != "" {
					incident.URL = fmt.Sprintf("%s/monitoring/incidents?groupId=%s", clusterIDConsoleURL[clusterID], groupId)
				} else {
					incident.URL = fmt.Sprintf("%s/monitoring/incidents?groupId=%s", clusterIDConsoleURL[defaultStr], groupId)
				}
			}
			incident.UpdateStatus()
			incidents[groupId] = incident
		}
	}
	return incidents, nil
}

// getTokenFromCtx gets the authorization header from the
// provided context
func getTokenFromCtx(ctx context.Context) (string, error) {
	k8sToken := ctx.Value(authHeaderStr)
	k8TokenStr, ok := k8sToken.(string)
	if !ok {
		return "", fmt.Errorf("failed to convert the authorization token to string")
	}
	return k8TokenStr, nil
}

// getAlertDataForIncidents queries Prometheus for firing alerts from the last 15 days (to have
// some starting time) and then maps (the alert identifier is composed by name and namespace)
// the active alerts to the provided map of incidents. It returns slice of the incidents.
func getAlertDataForIncidents(ctx context.Context, incidents map[string]Incident, silences []models.Alert, promAPI prom.Loader, qRange v1.Range) []Incident {
	alertData, err := promAPI.LoadVectorRange(ctx, `ALERTS{alertstate!="pending"}`, qRange.Start, qRange.End, qRange.Step)
	if err != nil {
		slog.Error("Failed to query firing alerts", "error", err)
		return nil
	}

	silencedAlertsMap := make(map[string][]models.Alert, len(silences))
	for _, silencedAlert := range silences {
		if alertname, ok := silencedAlert.Labels["alertname"]; ok {
			silencedAlertsMap[alertname] = append(silencedAlertsMap[alertname], silencedAlert)
		}
	}

	var alerts []model.LabelSet
	for i := range alertData {
		sample := alertData[i]
		metric := model.LabelSet(sample.Metric)
		firstSample := sample.Samples[0]
		lastSample := sample.Samples[len(sample.Samples)-1]
		startTime, endTime := processSampleTime(firstSample, lastSample, qRange)

		metric["start_time"] = model.LabelValue(formatToRFC3339(startTime))
		if !endTime.IsZero() {
			metric["end_time"] = model.LabelValue(formatToRFC3339(endTime))
			metric["alertstate"] = "resolved"
		} else {
			metric["alertstate"] = "firing"
		}
		alerts = append(alerts, metric)
	}

	var incidentsSlice []Incident
	for _, inc := range incidents {
		updatedAlertsMap := make(map[string]model.LabelSet, len(inc.Alerts))

		for _, alertInIncident := range inc.Alerts {
			subsetMatcher := common.LabelsSubsetMatcher{Labels: alertInIncident}
			for _, firingAlert := range alerts {
				// check for multicluster/ACM environment
				if inc.ClusterID != "" {
					clusterIDMatch := string(firingAlert[clusterIDStr]) == inc.ClusterID
					// if the alert cluster ID does not match incident cluster ID, skip
					if !clusterIDMatch {
						continue
					}
				}
				match, _ := subsetMatcher.Matches(firingAlert)
				if match {

					// the silencedAlertsMap is precomputed in order to contain all the silences grouped by alertname
					// [Alert1] = [{alertname="Alert1", namespace="foo"}, alertname="Alert1", namespace="bar"]
					alertname := string(firingAlert["alertname"])
					namespace := string(firingAlert["namespace"])
					severity := string(firingAlert["severity"])

					key := fmt.Sprintf("%s|%s|%s", alertname, namespace, severity)

					silenced := false
					if isAlertSilenced(firingAlert, silencedAlertsMap[alertname]) {
						silenced = true
					}

					updatedAlert := cleanupLabels(firingAlert)

					// If multiple alerts shares the same triple (alertname, namespace, severity) within
					// the same incident, these should be collapsed in a unique row. (same logic applied on server command)
					// The desired behaviour is to attach `silenced="true"` only if all colliding alerts are silenced, otherwise false.
					if _, f := updatedAlertsMap[key]; f {

						// if an alert, already labels cleaned, was already registered in the map
						// we should verify if it was marked as silenced
						lastSilenced, err := strconv.ParseBool(string(updatedAlertsMap[key][silencedStr]))
						if err != nil {
							slog.Error("failed to parse bool", "error", err)
							return nil
						}
						// the && operator allow us to get the following behaviour
						// if all are silenced the ending property will be true
						// if even just one is not silenced the ending property will be false
						updatedAlert[silencedStr] = model.LabelValue(fmt.Sprintf("%t", lastSilenced && silenced))
						updatedAlertsMap[key] = updatedAlert
					} else {
						updatedAlert[silencedStr] = model.LabelValue(fmt.Sprintf("%t", silenced))
						updatedAlertsMap[key] = updatedAlert
					}
				}
			}
		}

		inc.Alerts = slices.Collect(maps.Values(updatedAlertsMap))

		// sorting introduced to resolve unit tests flakyness
		slices.SortFunc(inc.Alerts, func(ls1, ls2 model.LabelSet) int {
			return strings.Compare(string(ls1["start_time"]), string(ls2["start_time"]))
		})

		incidentsSlice = append(incidentsSlice, inc)
	}
	return incidentsSlice
}

// cleanupLabels removes and renames some of the
// labels from the set and returns new LabelSet
func cleanupLabels(m model.LabelSet) model.LabelSet {
	updatedLS := m.Clone()
	updatedLS["status"] = updatedLS["alertstate"]
	updatedLS["name"] = updatedLS["alertname"]
	if cID := updatedLS[clusterIDStr]; cID != "" {
		updatedLS["cluster_id"] = cID
	}
	delete(updatedLS, "__name__")
	delete(updatedLS, "prometheus")
	delete(updatedLS, "alertstate")
	delete(updatedLS, "alertname")
	delete(updatedLS, "pod")
	delete(updatedLS, clusterIDStr)
	return updatedLS
}

// getConsoleURL queries the "console_url" metric from the Prometheus.
// If there is no metric value, it returns nil and error.
// If the "clusterID" label exists in the metric, it returns a map of cluster ID to
// console URL mappin.
// If the "clusterID" label doesn't exist in the metric, it returns a map
// with one key-value pair with "default" key and
// the console URL.
func getConsoleURL(ctx context.Context, prom prom.Loader) (map[string]string, error) {
	val, err := prom.LoadQuery(ctx, "console_url", time.Now())
	if err != nil {
		return nil, err
	}

	if len(val) == 0 {
		return nil, fmt.Errorf("console_url not found")
	}
	if _, ok := val[0][clusterIDStr]; ok {
		result := make(map[string]string, len(val))
		for _, v := range val {
			result[string(v[clusterIDStr])] = string(v["url"])
		}
		return result, nil
	}

	return map[string]string{
		defaultStr: string(val[0]["url"]),
	}, nil
}

func defaultPrometheusLoader(promURL, token string) (prom.Loader, error) {
	return prom.NewLoaderWithToken(promURL, token)
}

func defaultAlertManagerLoader(alertManagerURL, token string) (alertmanager.Loader, error) {
	return alertmanager.NewLoader(alertmanager.LoaderConfig{
		AlertManagerURL: alertManagerURL,
		Token:           token,
	})
}

func isAlertSilenced(alert model.LabelSet, silences []models.Alert) bool {
	// The labels defined in an Alertmanager silence are an intersection of the labels on an alert
	// If all the common labels by alert and the silence matches we can assume that alert is silenced

	for _, silence := range silences {
		if silence.Labels == nil {
			continue
		}

		// Convert silence labels to model.LabelSet
		silenceLabels := make(model.LabelSet)
		for silenceLabel, silenceValue := range silence.Labels {
			silenceLabels[model.LabelName(silenceLabel)] = model.LabelValue(silenceValue)
		}

		// Use LabelsIntersectionMatcher to check if silence labels match the alert
		matcher := common.LabelsIntersectionMatcher{Labels: silenceLabels}
		match, _ := matcher.Matches(alert)
		if match {
			return true
		}
	}

	return false
}

func filterIncidentsBySeverity(incidents []Incident, minSeverity processor.HealthValue) []Incident {
	filteredList := make([]Incident, 0)
	for _, inc := range incidents {
		incSeverity := processor.ParseHealthValue(inc.Severity)

		if incSeverity < minSeverity {
			continue
		}

		filteredList = append(filteredList, inc)
	}
	return filteredList
}
