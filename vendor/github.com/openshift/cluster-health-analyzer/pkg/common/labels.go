package common

import (
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/prometheus/common/model"
)

const SrcLabelPrefix = "src_"

// SrcLabels returns a map of labels that are not internal.
//
// These labels are used for matching underlying metrics (e.g. alerts).
func SrcLabels(labels model.Metric) model.LabelSet {
	ret := make(model.LabelSet)
	for k, v := range labels {
		keyStr := string(k)
		if strings.HasPrefix(keyStr, SrcLabelPrefix) {
			ret[k[len(SrcLabelPrefix):]] = v
		}
	}
	return ret
}

// labelsMatcher represents a matcher definition for a set of labels.
// It matches if all of the label matchers match the labels.
type LabelsSubsetMatcher struct {
	Labels model.LabelSet
}

func (l LabelsSubsetMatcher) Matches(labels model.LabelSet) (bool, []model.LabelName) {
	var keys []model.LabelName
	for k, m := range l.Labels {
		if v, ok := labels[k]; !ok || v != m {
			return false, nil
		}
		keys = append(keys, k)
	}
	return true, keys
}

func (l LabelsSubsetMatcher) Equals(other LabelsMatcher) bool {
	o, ok := other.(LabelsSubsetMatcher)
	if !ok {
		return false
	}

	if len(l.Labels) != len(o.Labels) {
		return false
	}

	// The length of the maps is the same, so if it's subset of the other, it's equal.
	ret, _ := l.Matches(o.Labels)
	return ret
}

type LabelsMatcher interface {
	Matches(labels model.LabelSet) (match bool, keys []model.LabelName)
	Equals(other LabelsMatcher) bool
}

func NewLabelsMatcher(key string, matcher ValueMatcher) LabelsMatcher {
	return labelMatcher{key: key, matcher: matcher}
}

func NewStringValuesMatcher(keys ...string) ValueMatcher {
	return stringMatcher(keys)
}

func NewRegexValuesMatcher(regexes ...*regexp.Regexp) ValueMatcher {
	return regexpMatcher(regexes)
}

// labelMatcher represents a matcher definition for a label.
type labelMatcher struct {
	key     string
	matcher ValueMatcher
}

// Matches implements the LabelsMatcher interface.
func (l labelMatcher) Matches(labels model.LabelSet) (bool, []model.LabelName) {
	if l.matcher.Matches(string(labels[model.LabelName(l.key)])) {
		return true, []model.LabelName{model.LabelName(l.key)}
	}
	return false, nil
}

// Equals implements the LabelsMatcher interface.
func (l labelMatcher) Equals(other LabelsMatcher) bool {
	ol, ok := other.(labelMatcher)
	if !ok {
		return false
	}
	return l.key == ol.key && l.matcher.Equals(ol.matcher)
}

// ValueMatcher represents a matcher for a specific value.
//
// Multiple implementations are provided for different types of matchers.
type ValueMatcher interface {
	Matches(value string) bool
	Equals(other ValueMatcher) bool
}

// stringMatcher is a matcher for a list of strings.
//
// It matches if the value is in the list of strings.
type stringMatcher []string

func (s stringMatcher) Matches(value string) bool {
	for _, v := range s {
		if v == value {
			return true
		}
	}
	return false
}

// Equals implements the ValueMatcher interface.
func (s stringMatcher) Equals(other ValueMatcher) bool {
	o, ok := other.(stringMatcher)
	if !ok {
		return false
	}
	return equalsNoOrder(s, o)
}

// regexpMatcher is a matcher for a list of regular expressions.
//
// It matches if the value matches any of the regular expressions.
type regexpMatcher []*regexp.Regexp

func (r regexpMatcher) Matches(value string) bool {
	for _, re := range r {
		if re.MatchString(value) {
			return true
		}
	}
	return false
}

// Equals implements the ValueMatcher interface.
func (r regexpMatcher) Equals(other ValueMatcher) bool {
	o, ok := other.(regexpMatcher)
	if !ok {
		return false
	}
	s1 := make([]string, 0, len(r))
	for _, re := range r {
		s1 = append(s1, re.String())
	}
	s2 := make([]string, 0, len(o))
	for _, re := range o {
		s2 = append(s2, re.String())
	}
	return equalsNoOrder(s1, s2)
}

func equalsNoOrder(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	seen := make(map[string]int, len(a))
	for _, v := range a {
		seen[v]++
	}
	for _, v := range b {
		if seen[v] == 0 {
			return false
		}
		seen[v]--
	}
	return true
}

// LabelsIntersectionMatcher implements LabelsMatcher by spefically matching the intersection between sets
type LabelsIntersectionMatcher struct {
	Labels model.LabelSet
}

func (l LabelsIntersectionMatcher) Matches(labels model.LabelSet) (bool, []model.LabelName) {
	intersection := make(map[model.LabelName]struct{})

	for k, v := range l.Labels {
		if label, f := labels[k]; f {

			// if they are not equal intersection does not match
			if label != v {
				return false, nil
			}

			// if they are equal save the model.LabelName of common labels
			intersection[k] = struct{}{}
		}
	}

	if len(intersection) == 0 {
		// no intersection
		return false, nil
	}

	return true, slices.Collect(maps.Keys(intersection))
}

func (l LabelsIntersectionMatcher) Equals(other LabelsMatcher) bool {
	o, ok := other.(LabelsIntersectionMatcher)
	if !ok {
		return false
	}

	return maps.Equal(l.Labels, o.Labels)
}
