package processor

// This file contains logic for mapping prometheus alerts to component health maps.

import (
	"slices"

	"github.com/prometheus/common/model"
)

// MapAlerts maps prometheus alerts to component health maps.
func MapAlerts(alerts []model.LabelSet) []ComponentHealthMap {
	healthMaps := make([]ComponentHealthMap, 0, len(alerts))
	for _, alert := range alerts {
		healthMap := getAlertHealthMap(alert)
		healthMaps = append(healthMaps, healthMap)
	}
	return healthMaps
}

// getAlertHealthMap maps a prometheus alert to a component health map.
func getAlertHealthMap(a model.LabelSet) ComponentHealthMap {
	// Check if alert is a node alert
	layer, component, labels := determineComponent(a)

	healthMap := ComponentHealthMap{
		Layer:     layer,
		Component: component,
		SrcType:   Alert,
		SrcLabels: labels,
	}

	healthMap.GroupId = string(a["group_id"])
	healthMap.Health = ParseHealthValue(string(a["severity"]))
	healthMap.Silenced = string(a["silenced"])

	return healthMap
}

// determineComponent determines the component of a prometheus alert.
//
// It uses various strategies to determine the component.
func determineComponent(a model.LabelSet) (layer, component string, labels model.LabelSet) {
	// Check if alert is a node alert.
	return evalMatcherFns([]componentMatcherFn{
		cvoAlertsMatcher,
		computeMatcher,
		coreMatcher,
		workloadMatcher,
	}, a)
}

var cvoAlerts = []model.LabelValue{"ClusterOperatorDown", "ClusterOperatorDegraded"}

func cvoAlertsMatcher(labels model.LabelSet) (layer, comp model.LabelValue, keys []model.LabelName) {
	if slices.Contains(cvoAlerts, labels["alertname"]) {
		component := labels["name"]
		if component == "" {
			component = "version"
		}
		return "core", component, nil
	}
	return "", "", nil
}

func computeMatcher(labels model.LabelSet) (layer, comp model.LabelValue, keys []model.LabelName) {
	for _, nodeAlert := range nodeAlerts {
		if labels["alertname"] == nodeAlert {
			// TODO: determine node identifier as a component
			// TODO: split nodes to controlplane and worker nodes
			component := "compute"
			return "compute", model.LabelValue(component), nil
		}
	}
	return "", "", nil
}

func coreMatcher(labels model.LabelSet) (layer, comp model.LabelValue, keys []model.LabelName) {
	// Try matching against core components.
	if component, keys := findComponent(coreMatchers, labels); component != "" {
		return "core", model.LabelValue(component), keys
	}
	return "", "", nil
}

func workloadMatcher(labels model.LabelSet) (layer, comp model.LabelValue, keys []model.LabelName) {
	// Try matching against workload components.
	if component, keys := findComponent(workloadMatchers, labels); component != "" {
		return "workload", model.LabelValue(component), keys
	}
	return "", "", nil
}
