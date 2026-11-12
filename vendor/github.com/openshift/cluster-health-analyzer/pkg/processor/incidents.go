package processor

import (
	"fmt"
	"log/slog"
	"math"
	"slices"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/common/model"

	"github.com/openshift/cluster-health-analyzer/pkg/common"
	"github.com/openshift/cluster-health-analyzer/pkg/prom"
)

type Interval struct {
	Metric model.LabelSet
	Start  model.Time
	End    model.Time
}

func (i Interval) String() string {
	return fmt.Sprintf("(%s -> %s): %s", i.Start.Time(), i.End.Time(), i.Metric)
}

type GroupedInterval struct {
	Interval
	GroupMatcher *GroupMatcher
}

func (gi GroupedInterval) String() string {
	return fmt.Sprintf("%s: %s", gi.GroupMatcher.RootGroupID, gi.Interval)
}

type Change struct {
	Timestamp model.Time
	Intervals []Interval
}

func (c Change) String() string {
	var str string
	str += fmt.Sprintf("Timestamp: %s\n", c.Timestamp.Time())
	for _, i := range c.Intervals {
		str += fmt.Sprintln(i)
	}
	return str
}

type ChangeSet []Change

var noMatchAlerts = []common.LabelsSubsetMatcher{
	{Labels: model.LabelSet{"alertname": "Watchdog", "namespace": "openshift-monitoring"}},
	{Labels: model.LabelSet{"alertname": "AlertmanagerReceiversNotConfigured", "namespace": "openshift-monitoring"}},
}

func MetricsIntervals(rangeVector prom.RangeVector) []Interval {
	if len(rangeVector) == 0 {
		return nil
	}
	step := rangeVector[0].Step

	ret := make([]Interval, 0)
	for _, r := range rangeVector {
		if len(r.Samples) == 0 {
			continue
		}
		start := r.Samples[0].Timestamp
		end := start

		for i := 1; i < len(r.Samples); i++ {
			sample := r.Samples[i]
			if sample.Timestamp.Sub(end) > step {
				// The end of the previous interval.
				ret = append(ret, Interval{Metric: r.Metric, Start: start, End: end})
				// Start of the new interval.
				start = sample.Timestamp
				end = start
			} else {
				// Current interval continues.
				end = sample.Timestamp
			}
		}
		// The last interval.
		ret = append(ret, Interval{Metric: r.Metric, Start: start, End: end})
	}
	return ret
}

// MetricsChanges returns a list of changes in the alerts.
//
// The changes are grouped by the timestamp of the change and sorted
// by the timestamp.
func MetricsChanges(rangeVector prom.RangeVector) ChangeSet {
	intervals := MetricsIntervals(rangeVector)
	if len(intervals) == 0 {
		return nil
	}

	var ret ChangeSet

	sort.Slice(intervals, func(i, j int) bool {
		return intervals[i].Start.Before(intervals[j].Start)
	})
	currGroup := make([]Interval, 0)
	currTime := intervals[0].Start

	for _, i := range intervals {
		if currTime == i.Start {
			// The same start - group together.
			currGroup = append(currGroup, i)
		} else {
			// Different start: save the current group and start a new one.
			ret = append(ret, Change{Timestamp: currTime, Intervals: currGroup})

			currGroup = []Interval{i}
			currTime = i.Start
		}
	}
	// Save the last group.
	ret = append(ret, Change{Timestamp: currTime, Intervals: currGroup})
	return ret
}

func (ch ChangeSet) String() string {
	var str string
	for i, c := range ch {
		if i > 0 {
			str += "\n"
		}
		str += c.String()
	}
	return str
}

// Group Matchers

type GroupMatcher struct {
	GroupID     string
	RootGroupID string
	Start       model.Time
	Modified    model.Time
	End         model.Time

	Distance float64
	Matchers []common.LabelsSubsetMatcher
}

func (g GroupMatcher) String() string {
	return fmt.Sprintf("GroupID: %s, RootGroupID: %s, Start: %s, Modified: %s, End: %s, Distance: %f, Matchers: %v",
		g.GroupID, g.RootGroupID, g.Start.Time(), g.Modified.Time(), g.End.Time(), g.Distance, g.Matchers)
}

func (g GroupMatcher) isSubsetOf(other *GroupMatcher) bool {
	if g.Distance != other.Distance {
		return false
	}

	// Check if all matchers of g are in other.
	for _, m := range g.Matchers {
		contains := false
		for _, om := range other.Matchers {
			if om.Equals(m) {
				contains = true
				break
			}
		}
		if !contains {
			return false
		}
	}
	return true
}

func (g *GroupMatcher) expandMatchers(matchers []common.LabelsSubsetMatcher) {
	for _, m := range matchers {
		// Check if the matcher is already in the group.
		// If not, add it.
		found := false

		for _, gm := range g.Matchers {
			if gm.Equals(m) {
				found = true
				break
			}
		}

		if !found {
			g.Matchers = append(g.Matchers, m)
		}
	}
}

type match struct {
	GroupMatcher *GroupMatcher
	TimeDist     time.Duration
}

func newGroupMatcherSubset(labels model.LabelSet, keys []model.LabelName, distance float64) *GroupMatcher {
	labels = getMapSubset(labels, keys...)

	return &GroupMatcher{
		Matchers: []common.LabelsSubsetMatcher{{Labels: labels}},
		Distance: distance,
	}
}

func newGroupMatcherExact(labels model.LabelSet) *GroupMatcher {
	return &GroupMatcher{
		Matchers: []common.LabelsSubsetMatcher{{Labels: labels}},
		Distance: 0,
	}
}

func watchdogAlert(i Interval) bool {
	return i.Metric["alertname"] == "Watchdog" &&
		i.Metric["namespace"] == "openshift-monitoring"
}

func alertFuzzyLabels(i Interval) model.LabelSet {
	for _, m := range noMatchAlerts {
		// For certain alerts, we don't want to do any fuzzy matching.
		if match, _ := m.Matches(i.Metric); match {
			return nil
		}
	}
	// TODO: add option for some alerts to match some known pairs, but not others.
	// E.g. APIRemovedInNextReleaseInUse and APIRemovedInNextEUSReleaseInUse
	return getMapSubset(i.Metric, "alertname", "namespace")
}

// alertGroupMatchers returns a list of matchers for the alert.
// This includes exact matcher with 0 distance, as well as various fuzzy matchers
// based on the alert labels.
func alertGroupMatchers(interval Interval) []*GroupMatcher {
	labels := interval.Metric
	groups := []*GroupMatcher{
		newGroupMatcherExact(labels),
		// Match on main subset of labels - should be still close enough.
		newGroupMatcherSubset(labels, []model.LabelName{"namespace", "alertname", "service", "job", "container"}, 1),
	}

	for k, v := range alertFuzzyLabels(interval) {
		groups = append(groups,
			newGroupMatcherSubset(model.LabelSet{k: v}, []model.LabelName{k}, 2),
		)
	}
	for _, g := range groups {
		g.Start = interval.Start
		g.Modified = interval.Start
		g.End = interval.End
	}
	return groups
}

type GroupsCollection struct {
	Groups []*GroupMatcher
}

func (gc *GroupsCollection) AddGroup(g *GroupMatcher) {
	gc.Groups = append(gc.Groups, g)
}

func (gc *GroupsCollection) ProcessIntervalsBatch(intervals []Interval) []GroupedInterval {
	slog.Info("Processing", "intervals", len(intervals), "groups", len(gc.Groups))
	groupedIntervals, unmatched := gc.tryMatchIntervals(intervals)

	if len(unmatched) > 0 {
		// Create new groups for the unmatched intervals.
		newGroupedIntervals := gc.addIntervalsGroups(unmatched, nil)
		groupedIntervals = append(groupedIntervals, newGroupedIntervals...)
	}

	return groupedIntervals
}

func (gc *GroupsCollection) processHistoricalAlerts(alertsRange prom.RangeVector) {
	changes := MetricsChanges(alertsRange)

	for _, change := range changes {
		gc.ProcessIntervalsBatch(change.Intervals)
	}
}

func (gc *GroupsCollection) ProcessAlertsBatch(alerts []model.LabelSet, timestamp time.Time) []model.LabelSet {
	modelT := model.TimeFromUnixNano(timestamp.UnixNano())

	intervals := make([]Interval, 0, len(alerts))
	for _, a := range alerts {
		intervals = append(intervals, Interval{
			Metric: a,
			Start:  modelT,
			End:    modelT,
		})
	}

	groupedIntervals := gc.ProcessIntervalsBatch(intervals)

	ret := make([]model.LabelSet, 0, len(alerts))
	for _, gi := range groupedIntervals {
		alert := gi.Metric
		if gi.GroupMatcher != nil {
			alert["group_id"] = model.LabelValue(gi.GroupMatcher.RootGroupID)
		}
		ret = append(ret, alert)
	}
	return ret
}

// PruneGroups removes groups that can't be matched anymore.
//
// It calculates the threshold based on the provided time and removes groups.
func (gc *GroupsCollection) PruneGroups(t time.Time) {
	// Directs matches have longer retention times.
	gc.pruneGroupsBefore(0, 0, t.Add(-1*directMatchLongTimeDelta))
	// Fuzzy matches have shorter retention times.
	gc.pruneGroupsBefore(1, math.Inf(1), t.Add(-1*fuzzyMatchTimeDelta))
}

func (gc *GroupsCollection) pruneGroupsBefore(minDistance, maxDistance float64, t time.Time) {
	mt := model.TimeFromUnixNano(t.UnixNano())

	newGroups := make([]*GroupMatcher, 0, len(gc.Groups))

	for _, g := range gc.Groups {
		if g.Distance >= minDistance && g.Distance <= maxDistance && g.Modified.Before(mt) {
			continue
		}
		newGroups = append(newGroups, g)
	}
	gc.Groups = newGroups
}

func (gc *GroupsCollection) tryMatchIntervals(intervals []Interval) ([]GroupedInterval, []Interval) {
	var ret []GroupedInterval
	var unmatched []Interval
	for _, i := range intervals {
		matchedGroup := gc.bestMatch(i)
		if matchedGroup == nil {
			unmatched = append(unmatched, i)
			continue
		}

		if matchedGroup.Distance > 0 {
			// We don't update modified time for flapping alerts,
			matchedGroup.Modified = i.Start
		}
		matchedGroup.End = max(matchedGroup.End, i.End)

		newGroupedIntervals := gc.addIntervalsGroups([]Interval{i}, matchedGroup)
		ret = append(ret, newGroupedIntervals...)
	}
	return ret, unmatched
}

func (gc *GroupsCollection) newRootGroup(i Interval, inactive bool) *GroupMatcher {
	rootGroupID := uuid.New().String()

	ret := GroupMatcher{
		GroupID:     rootGroupID,
		RootGroupID: rootGroupID,
		Start:       i.Start,
		Modified:    i.Start,
		End:         i.End,
		Distance:    math.Inf(1),
	}
	if inactive {
		// For inactive group, we set the modified to 0 so that it doesn't match
		// any new alert.
		ret.Modified = 0
	}

	gc.AddGroup(&ret)
	return &ret
}

func (gc *GroupsCollection) addIntervalsGroups(intervals []Interval, groupMatcher *GroupMatcher) []GroupedInterval {
	if len(intervals) == 0 {
		return nil
	}
	newGc := &GroupsCollection{}

	isWatchdogGroup := false
	for _, i := range intervals {
		if watchdogAlert(i) {
			isWatchdogGroup = true
			break
		}
	}

	ret := make([]GroupedInterval, 0, len(intervals))
	if groupMatcher == nil && !isWatchdogGroup {
		// If not provided, create a new root group for all intervals in this batch.
		// We don't do this if watchdog is present in the group, as it indicates
		// the alerts are together by accident (perhaps due to a restart or data
		// outage).
		groupMatcher = newGc.newRootGroup(intervals[0], isWatchdogGroup)
	}

	for _, i := range intervals {
		var iGroupMatcher *GroupMatcher
		iGroupMatcher = groupMatcher

		if iGroupMatcher == nil {
			iGroupMatcher = newGc.bestMatch(i)
		}

		if iGroupMatcher == nil {
			iGroupMatcher = newGc.newRootGroup(i, isWatchdogGroup)
		}

		// If we didn't have a direct match, add additional fuzzy matchers
		// for this interval. If Distance is 0, we assume the fuzzy matchers
		// to be already present.
		if iGroupMatcher.Distance > 0 {
			newGroupCands := alertGroupMatchers(i)
			for _, g := range newGroupCands {
				if g.Distance == iGroupMatcher.Distance && iGroupMatcher.isSubsetOf(g) {
					iGroupMatcher.expandMatchers(g.Matchers)
					if g.Distance > 0 {
						// We don't update modified time for flapping alerts,
						// as we don't consider that being a significant change
						// for the group.
						iGroupMatcher.Modified = i.Start
					}
					iGroupMatcher.End = max(iGroupMatcher.End, i.End)
				} else {
					g.RootGroupID = iGroupMatcher.RootGroupID
					newGc.AddGroup(g)
				}
			}
		}

		ret = append(ret, GroupedInterval{i, iGroupMatcher})
	}
	for _, g := range newGc.Groups {
		gc.AddGroup(g)
	}
	return ret
}

var (
	// Unless we have a direct match, we try fuzzy matching.
	fuzzyMatchTimeDelta = 24 * time.Hour

	// If we have no match yet, we try to match on the time, but just very close events.
	timeMatchTimeDelta = 15 * time.Minute

	// No match yet: look for direct matches deeper in the past.
	directMatchLongTimeDelta = 5 * 24 * time.Hour
)

func (gc *GroupsCollection) bestMatch(interval Interval) *GroupMatcher {
	matches := gc.matches(interval)
	var directLongMatch *match
	var shortCandidates []match
	var shortMatch *match
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].TimeDist < matches[j].TimeDist
	})

	for _, m := range matches {
		if m.TimeDist <= fuzzyMatchTimeDelta {
			shortCandidates = append(shortCandidates, m)
			continue
		}

		if m.TimeDist <= directMatchLongTimeDelta && m.GroupMatcher.Distance == 0 {
			directLongMatch = &m
			// Given matches are sorted by time and we crossed the fuzzyMatchTimeDelta,
			// there is no change to match anything better at this point.
			break
		}
	}

	if len(shortCandidates) > 0 {
		// Try to find the match with the smallest distance.
		// In case of the same distance, it's fine for the first match to win,
		// as we sort the matches by time initially.
		shortMatch = &shortCandidates[0]
		for i := 1; i < len(shortCandidates); i++ {
			if shortCandidates[i].GroupMatcher.Distance < shortMatch.GroupMatcher.Distance {
				shortMatch = &shortCandidates[i]
			}
		}
	}

	if shortMatch != nil {
		return shortMatch.GroupMatcher
	}
	if directLongMatch != nil {
		return directLongMatch.GroupMatcher
	}

	return nil
}

func (gc *GroupsCollection) matches(interval Interval) []match {
	var ret []match
	allLabels := interval.Metric
	fuzzyLabels := alertFuzzyLabels(interval)
	for _, g := range gc.Groups {
		var timeDist time.Duration
		if g.Distance == 0 {
			// for direct matches, we compare with the end of the interval
			timeDist = interval.Start.Sub(g.End)
		} else {
			// In fuzzy matching, we compare with the last time the group was modified.
			timeDist = interval.Start.Sub(g.Modified)
		}

		if interval.Start < g.Start {
			// We don't consider groups from the future
			continue
		}

		// Pure time-based grouping
		if g.Distance == math.Inf(1) && timeDist <= timeMatchTimeDelta {
			ret = append(ret, match{g, timeDist})
			continue
		}

		labels := allLabels
		// For fuzzy matching, we use only a subset of labels that can be overriden
		// on per-alert basis.
		if g.Distance >= 2 {
			labels = fuzzyLabels
		}
		for _, m := range g.Matchers {
			if matched, _ := m.Matches(labels); matched {
				ret = append(ret, match{g, timeDist})
				// We found a match for this group: no need to check other matchers.
				break
			}
		}
	}
	return ret
}

/// Previous Incidents Matcher
///
/// The previous incidents matcher is used to match the current groups
/// with the previous incidents. This allows preserving the group UUIDs
/// after the restart of the analyzer.

type previousIncident struct {
	matcher *common.LabelsSubsetMatcher
	uuid    string
	start   model.Time
	end     model.Time
}

const previousIncidentsTolerance = 10 * time.Minute

type previousIncidentsMatcher struct {
	incidentsByStart []*previousIncident
	tolerance        time.Duration
}

func (pim *previousIncidentsMatcher) atTime(t model.Time) []*previousIncident {
	ret := make([]*previousIncident, 0)
	// Add some tolerance when comparing the start time.
	startT := t.Add(pim.tolerance)
	// For end time, subtract the tolerance, in case the incident ended
	// before the current time.
	endT := t.Add(-pim.tolerance)

	startIdx := sort.Search(len(pim.incidentsByStart), func(i int) bool {
		// find first incident that started after the given time.
		return !pim.incidentsByStart[i].start.Before(startT)
	})
	// We want to include the incident that started just before the current time.
	startIdx = max(0, startIdx)
	for i := 0; i < startIdx; i++ {
		incident := pim.incidentsByStart[i]
		if incident.end.After(endT) {
			ret = append(ret, incident)
		}
	}

	return ret
}

func (pim *previousIncidentsMatcher) match(labels model.LabelSet, time model.Time) *previousIncident {
	candidates := pim.atTime(time)

	for _, c := range candidates {
		ok, _ := c.matcher.Matches(labels)

		if ok {
			return c
		}
	}
	return nil
}

func newPreviousIncidentsMatcher(healthMapRV prom.RangeVector) *previousIncidentsMatcher {
	componentsMapChanges := MetricsChanges(healthMapRV)
	prevIncidents := make([]*previousIncident, 0, len(componentsMapChanges))
	for _, change := range componentsMapChanges {
		for _, interval := range change.Intervals {
			labels := interval.Metric
			prevIncidents = append(prevIncidents, &previousIncident{
				matcher: &common.LabelsSubsetMatcher{Labels: common.SrcLabels(model.Metric(labels))},
				uuid:    string(labels["group_id"]),
				start:   interval.Start,
				end:     interval.End,
			})
		}
	}

	incidentsByStart := slices.Clone(prevIncidents)
	sort.Slice(incidentsByStart, func(i, j int) bool {
		return incidentsByStart[i].start.Before(incidentsByStart[j].start)
	})

	return &previousIncidentsMatcher{
		incidentsByStart: incidentsByStart,
		tolerance:        previousIncidentsTolerance,
	}
}

func (gc *GroupsCollection) UpdateGroupUUIDs(healthMapRV prom.RangeVector) {
	unmappedGroups := make(map[string][]*GroupMatcher)
	mappedGroupIDs := make(map[string]struct{})

	// Prepare map of groups by the root group ID.
	for _, g := range gc.Groups {
		groups, ok := unmappedGroups[g.RootGroupID]
		if !ok {
			unmappedGroups[g.RootGroupID] = []*GroupMatcher{g}
		} else {
			unmappedGroups[g.RootGroupID] = append(groups, g)
		}
	}

	prevIncidentsMatcher := newPreviousIncidentsMatcher(healthMapRV)

	for _, g := range gc.Groups {
		// Check if the group is still unmapped.
		if _, ok := unmappedGroups[g.RootGroupID]; !ok {
			continue
		}

		for _, m := range g.Matchers {
			// Using End instead of Start, as the previous incidents might not be covering
			// the whole duration of the group.
			prevIncident := prevIncidentsMatcher.match(m.Labels, g.End)
			if prevIncident != nil {
				newGroupID := prevIncident.uuid
				oldGroupID := g.RootGroupID
				// Replace all occurrences of old group ID with the new one and.
				for _, g := range unmappedGroups[oldGroupID] {
					g.RootGroupID = newGroupID
					mappedGroupIDs[newGroupID] = struct{}{}
				}
				// Remove the old group from the list of unmapped groups.
				delete(unmappedGroups, oldGroupID)
				break
			}
		}
	}
}
