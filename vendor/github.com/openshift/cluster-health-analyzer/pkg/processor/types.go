package processor

// This file contains support types used by the processor.

import (
	"hash/fnv"
	"slices"
	"strings"

	"github.com/openshift/cluster-health-analyzer/pkg/common"
	"github.com/prometheus/common/model"
)

// # Component Health Map

// ComponentHealthMap represents mapping of a specific source to the component's health.
type ComponentHealthMap struct {
	Layer     string         // Layer of the component
	Component string         // Component name
	SrcType   SrcType        // Type of the source (alert, cluster_operator_condition)
	SrcLabels model.LabelSet // Identifying labels of the source
	GroupId   string         // Group ID of the component
	Health    HealthValue    // Health value of the component
	Silenced  string         // Whether the alert is silenced or not
}

// SrcType represents the type of the source.
type SrcType string

const (
	Alert                    SrcType = "alert"
	ClusterOperatorCondition SrcType = "cluster_operator_condition"
)

// HealthValue represents the health value of the component.
type HealthValue int

const (
	Healthy  HealthValue = 0
	Warning  HealthValue = 1
	Critical HealthValue = 2

	SrcLabelPrefix = "src_"
)

func (h HealthValue) String() string {
	switch h {
	case Healthy:
		return "info"
	case Warning:
		return "warning"
	case Critical:
		return "critical"
	default:
		return "none"
	}
}
func ParseHealthValue(s string) HealthValue {
	switch strings.ToLower(s) {
	case "info":
		return Healthy
	case "warning":
		return Warning
	case "critical":
		return Critical
	default:
		// We don't recognize the severity, so we'll default to warning
		return Warning
	}
}

// hashLabelValues returns a hash of the labels of the component.
//
// This is used to uniquely identify the component when deduplicating.
func (c ComponentHealthMap) hashLabelValues() uint64 {
	h := fnv.New64a()
	labels := c.Labels()
	keys := make([]model.LabelName, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	for _, k := range keys {
		v := labels[k]
		h.Write([]byte(k))
		h.Write([]byte(v))
	}
	return h.Sum64()
}

// Labels to be exported for the mapping.
func (c ComponentHealthMap) Labels() model.LabelSet {
	metaLabels := model.LabelSet{
		"layer":     model.LabelValue(c.Layer),
		"component": model.LabelValue(c.Component),
		"type":      model.LabelValue(c.SrcType),
		"group_id":  model.LabelValue(c.GroupId),
		"silenced":  model.LabelValue(c.Silenced),
	}

	labels := make(model.LabelSet, len(c.SrcLabels)+len(metaLabels))
	for k, v := range metaLabels {
		labels[k] = v
	}

	for k, v := range c.SrcLabels {
		labels[SrcLabelPrefix+k] = v
	}
	return labels
}

// # Component Matcher

// componentMatcher represents a matcher definition for a component.
//
// It matches if any of the label matchers match the labels.
type componentMatcher struct {
	component string
	matchers  []common.LabelsMatcher
}

// findComponent tries to dtermine a component for given labels using the provided matchers.
//
// It returns the component and the keys that matched.
// If no match is found, it returns an empty component and nil keys.
func findComponent(compMatchers []componentMatcher, labels model.LabelSet) (
	component string, keys []model.LabelName) {
	for _, compMatcher := range compMatchers {
		for _, labelsMatcher := range compMatcher.matchers {
			if matches, keys := labelsMatcher.Matches(labels); matches {
				return compMatcher.component, keys
			}
		}
	}
	return "", nil
}

// componentMatcherFn is a function that tries matching provided labels to a component.
// It returns the layer, component and the keys from the labels that were used for matching.
// If no match is found, it returns an empty layer, component and nil keys.
type componentMatcherFn func(labels model.LabelSet) (layer, comp model.LabelValue, keys []model.LabelName)

func evalMatcherFns(fns []componentMatcherFn, labels model.LabelSet) (
	layer, comp string, labelsSubset model.LabelSet) {
	for _, fn := range fns {
		if layer, comp, keys := fn(labels); layer != "" {
			return string(layer), string(comp), getLabelsSubset(labels, keys...)
		}
	}
	return "Others", "Others", getLabelsSubset(labels)
}

// getLabelsSubset returns a subset of the labels with given keys.
func getLabelsSubset(m model.LabelSet, keys ...model.LabelName) model.LabelSet {
	keys = append([]model.LabelName{"namespace", "alertname", "severity"}, keys...)
	return getMapSubset(m, keys...)
}

// getLabelsSubset returns a subset of the labels with given keys.
func getMapSubset(m model.LabelSet, keys ...model.LabelName) model.LabelSet {
	subset := make(model.LabelSet, len(keys))
	for _, key := range keys {
		if val, ok := m[key]; ok {
			subset[key] = val
		}
	}
	return subset
}
