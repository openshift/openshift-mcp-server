// Package processor contains the core logic for processing alerts and
// updating the health map metrics.
package processor

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/openshift/cluster-health-analyzer/pkg/alertmanager"
	"github.com/openshift/cluster-health-analyzer/pkg/common"
	"github.com/openshift/cluster-health-analyzer/pkg/prom"
	"github.com/prometheus/alertmanager/api/v2/models"
	"github.com/prometheus/common/model"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	ClusterHealthComponentsMap = "cluster_health_components_map"

	AlertNameLabelKey = "alertname"
)

// processor is the component responsible for continuously loading alerts from source
// and coordinates updating the exported metrics.
type processor struct {
	// healthMapMetrics maps input signal (alerts) to components, incidents
	// and normalized severity.
	healthMapMetrics prom.MetricSet

	// componentsMetrics maps components to their ranking via the metric value.
	componentsMetrics prom.MetricSet

	// groupSeverityCountMetrics exposes the current counts of group_ids by severity.
	groupSeverityCountMetrics prom.MetricSet

	// interval is the time interval between processing iterations.
	interval time.Duration

	loader           prom.Loader
	amLoader         alertmanager.Loader
	groupsCollection *GroupsCollection
}

type ProcessorConfig struct {
	Interval        time.Duration
	PromURL         string
	AlertManagerURL string
}

func NewProcessor(cfg ProcessorConfig, healthMapMetrics, componentsMetrics prom.MetricSet, groupSeverityCountMetrics prom.MetricSet) (*processor, error) {
	promLoader, err := prom.NewLoader(cfg.PromURL)
	if err != nil {
		return nil, err
	}

	amLoader, err := alertmanager.NewLoader(alertmanager.LoaderConfig{
		AlertManagerURL: cfg.AlertManagerURL,
	})
	if err != nil {
		return nil, err
	}

	return &processor{
		healthMapMetrics:          healthMapMetrics,
		componentsMetrics:         componentsMetrics,
		groupSeverityCountMetrics: groupSeverityCountMetrics,
		interval:                  cfg.Interval,
		loader:                    promLoader,
		amLoader:                  amLoader,
	}, nil
}

// Start starts the processor in a goroutine and returns immediately.
func (p *processor) Start(ctx context.Context) {
	go p.Run(ctx)
}

// initGroupsCollection initializes the groups collection by loading the alerts.
//
// The alerts are loaded for the given time range and step and prepares the structure
// for assigning group-ids to the alerts.
func (p *processor) InitGroupsCollection(ctx context.Context, start, end time.Time, step time.Duration) error {
	slog.Info("Initializing groups collection", "start", start, "end", end, "step", step)
	p.groupsCollection = &GroupsCollection{}

	slog.Info("Loading alerts range")
	alertsRange, err := p.loader.LoadAlertsRange(ctx, start, end, step)
	if err != nil {
		return err
	}
	slog.Info("Loaded alerts range", "len", len(alertsRange))

	// Warm up the groups collection with historical alerts.
	slog.Info("Processing historical alerts")
	p.groupsCollection.processHistoricalAlerts(alertsRange)

	slog.Info("Loading health map range")
	healthMapRV, err := p.loader.LoadVectorRange(ctx, ClusterHealthComponentsMap, start, end, step)
	if err != nil {
		return err
	}
	slog.Info("Loaded health map range", "len", len(healthMapRV))

	slog.Info("Updating group-ids")
	p.groupsCollection.UpdateGroupUUIDs(healthMapRV)

	return nil
}

// Run runs the processor and blocks until canceled via the ctx.
func (p *processor) Run(ctx context.Context) {
	// wait.Until provides the core for the repeated execution of the Process method
	wait.Until(func() {
		// wait.ExponentialBackoffWithContext provides a backoff mechanism
		// in case of errors during the Process method execution.
		err := wait.ExponentialBackoffWithContext(
			ctx,
			wait.Backoff{Duration: time.Second, Steps: 4, Factor: 1.5},
			func(ctx context.Context) (bool, error) {
				slog.Info("Start processing")

				err := p.Process(ctx)
				if err != nil {
					slog.Error("Error processing", "err", err)
					// We don't return an error here because we want to keep retrying.
					return false, nil
				}

				slog.Info("End processing")
				return true, nil
			})
		if err != nil {
			slog.Error("Error processing", "err", err)
		}
	}, p.interval, ctx.Done())
}

// dedupHealthMaps deduplicates the health maps by combining the health values.
//
// The deduplication is done by hashing the label values of the health maps.
// For duplicates, the health value is combined by taking the maximum of the two.
func dedupHealthMaps(healthMaps []ComponentHealthMap) []ComponentHealthMap {
	hashMap := make(map[uint64]ComponentHealthMap, len(healthMaps))

	for _, healthMap := range healthMaps {
		hash := healthMap.hashLabelValues()
		existing, ok := hashMap[hash]
		if ok {
			existing.Health = max(existing.Health, healthMap.Health)
		} else {
			hashMap[hash] = healthMap
		}
	}

	deduped := make([]ComponentHealthMap, 0, len(hashMap))
	for _, healthMap := range hashMap {
		deduped = append(deduped, healthMap)
	}
	return deduped
}

func (p *processor) assignAlertsToGroups(alerts []model.LabelSet, t time.Time) []model.LabelSet {
	processedAlerts := p.groupsCollection.ProcessAlertsBatch(alerts, t)

	// Prune the groups collection to remove old groups.
	p.groupsCollection.PruneGroups(t)
	return processedAlerts
}

// Process performs a single iteration of the processor.
func (p *processor) Process(ctx context.Context) error {
	err := p.updateHealthMap(ctx)
	if err != nil {
		return err
	}

	p.updateComponentsMetrics()

	return nil
}

func (p *processor) updateHealthMap(ctx context.Context) error {
	t := time.Now()
	alerts, err := p.loadAlerts(ctx, t)
	if err != nil {
		return err
	}

	healthMap := MapAlerts(alerts)
	healthMap = dedupHealthMaps(healthMap)

	healthMapMetrics := make([]prom.Metric, 0, len(healthMap))
	for _, healthMap := range healthMap {
		healthMapMetrics = append(healthMapMetrics, prom.Metric{
			Labels: healthMap.Labels(),
			Value:  float64(healthMap.Health),
		})
	}
	p.healthMapMetrics.Update(healthMapMetrics)

	severityCountsMetrics := p.computeSeverityCountMetrics(healthMap)
	p.groupSeverityCountMetrics.Update(severityCountsMetrics)

	return nil
}

func (p *processor) loadAlerts(ctx context.Context, t time.Time) ([]model.LabelSet, error) {
	alerts, err := p.loader.LoadQuery(ctx, `ALERTS{alertstate="firing"}`, t)
	if err != nil {
		return nil, err
	}

	if p.groupsCollection != nil {
		alerts = p.assignAlertsToGroups(alerts, t)
	}

	alerts, err = p.evaluateSilences(alerts)
	if err != nil {
		return nil, err
	}

	return alerts, nil
}

func (p *processor) evaluateSilences(alerts []model.LabelSet) ([]model.LabelSet, error) {
	type silenceEvalGroup struct {
		groupID        string
		alertName      string
		namespace      string
		severity       string
		originalAlerts []model.LabelSet
	}

	// get all silenced alerts from alertmanager
	silenced, err := p.amLoader.SilencedAlerts()
	if err != nil {
		return nil, err
	}

	// convert slice to temporary map for better lookup
	silencedAlertsMap := make(map[string][]models.Alert, len(silenced))
	for _, silencedAlert := range silenced {
		labelKey := silencedAlert.Labels[AlertNameLabelKey]
		silencedAlertsMap[labelKey] = append(silencedAlertsMap[labelKey], silencedAlert)
	}

	// Group alerts by consolidated key
	silenceEvalGroups := make(map[string]*silenceEvalGroup)

	for _, alert := range alerts {
		key := fmt.Sprintf(
			"%s|%s|%s|%s",
			string(alert["group_id"]),
			string(alert["alertname"]),
			string(alert["namespace"]),
			string(alert["severity"]),
		)

		if group, exists := silenceEvalGroups[key]; exists {
			group.originalAlerts = append(group.originalAlerts, alert)
			continue
		}

		silenceEvalGroups[key] = &silenceEvalGroup{
			groupID:        string(alert["group_id"]),
			alertName:      string(alert["alertname"]),
			namespace:      string(alert["namespace"]),
			severity:       string(alert["severity"]),
			originalAlerts: []model.LabelSet{alert},
		}
	}

	// Evaluate silence for each group
	for _, group := range silenceEvalGroups {
		allSilenced := true
		for _, alert := range group.originalAlerts {
			if !isAlertSilenced(alert, silencedAlertsMap) {
				allSilenced = false
				break
			}
		}

		silencedValue := fmt.Sprintf("%t", allSilenced)
		for i := range group.originalAlerts {
			group.originalAlerts[i]["silenced"] = model.LabelValue(silencedValue)
		}
	}

	return alerts, nil
}

func (p *processor) computeSeverityCountMetrics(alertsHealthMap []ComponentHealthMap) []prom.Metric {
	severityCount := countSeverities(alertsHealthMap)

	metrics := make([]prom.Metric, 0, len(severityCount))
	for severity, count := range severityCount {
		metrics = append(metrics, prom.Metric{
			Labels: model.LabelSet{
				"severity": model.LabelValue(severity),
			},
			Value: float64(count),
		})
	}

	return metrics
}

func countSeverities(healthMaps []ComponentHealthMap) map[string]int {
	healthValues := getCurrentMaxHealthValues(healthMaps)
	severities := convertHealthValuesToSeverities(healthValues)

	count := make(map[string]int)
	for _, severity := range severities {
		count[severity]++
	}

	return count
}

func getCurrentMaxHealthValues(healthMaps []ComponentHealthMap) map[string]HealthValue {
	healthValues := make(map[string]HealthValue)
	for _, alert := range healthMaps {
		groupID := alert.GroupId
		if groupID == "" {
			continue
		}
		healthValues[groupID] = max(healthValues[groupID], alert.Health)
	}
	return healthValues
}

func convertHealthValuesToSeverities(healthValues map[string]HealthValue) []string {
	severities := make([]string, 0, len(healthValues))
	for _, health := range healthValues {
		severities = append(severities, health.String())
	}
	return severities
}

func (p *processor) updateComponentsMetrics() {
	ranks := BuildComponentRanks()

	metrics := make([]prom.Metric, 0)
	for _, r := range ranks {
		metrics = append(metrics, prom.Metric{
			Labels: model.LabelSet{
				"layer":     model.LabelValue(r.Layer),
				"component": model.LabelValue(r.Component),
			},
			Value: float64(r.Rank),
		})
	}
	p.componentsMetrics.Update(metrics)
}

func isAlertSilenced(alert model.LabelSet, silences map[string][]models.Alert) bool {
	alertName := string(alert[AlertNameLabelKey])
	silencedAlerts, nameIsSilenced := silences[alertName]
	if !nameIsSilenced {
		return false
	}
	// An `alertname` can apply to multiple alerts with different labels.
	// We must iterate through the labels of each alert with the same `alertname`
	// to find the specific silence that applies.
	// For example, `{alertname="Alert1", namespace="foo"}` and
	// `{alertname="Alert1", namespace="bar"}` are distinct alerts.
	// A silence on `{alertname="Alert1", namespace="foo"}` will only silence the first alert.
	for _, silencedAlert := range silencedAlerts {
		// Convert silence labels to model.LabelSet
		silenceLabels := make(model.LabelSet)
		for k, v := range silencedAlert.Labels {
			silenceLabels[model.LabelName(k)] = model.LabelValue(v)
		}
		// Use LabelsIntersectionMatcher to check for an equal intersection between the silence and the alert
		matcher := common.LabelsIntersectionMatcher{Labels: silenceLabels}
		match, _ := matcher.Matches(alert)
		if match {
			return true
		}
	}
	return false
}

type ComponentRank struct {
	Layer     string
	Component string
	Rank      int
}

func BuildComponentRanks() []ComponentRank {
	components := make(map[string]ComponentRank)
	components["compute"] = ComponentRank{Layer: "compute", Component: "compute", Rank: 1}

	for i, m := range coreMatchers {
		components[m.component] = ComponentRank{Layer: "core", Component: m.component, Rank: 10 + i*5}
	}

	for i, m := range workloadMatchers {
		components[m.component] = ComponentRank{Layer: "workload", Component: m.component, Rank: 1000 + i*5}
	}

	ret := make([]ComponentRank, 0, len(components))
	for _, c := range components {
		ret = append(ret, c)
	}
	return ret
}
