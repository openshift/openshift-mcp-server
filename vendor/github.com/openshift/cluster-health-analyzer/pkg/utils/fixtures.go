package utils

import (
	"time"

	"github.com/prometheus/common/model"

	"github.com/openshift/cluster-health-analyzer/pkg/prom"
)

// RelativeInterval represents a labeled interval with relative start and end times.
//
// It can be used to generate various data structures for testing purposes, such
// as [prom.Range] or [processor.Interval].
type RelativeInterval struct {
	Labels model.LabelSet
	Start  int // relative start in minutes
	End    int // relative end in minutes
}

// ToRange converts the relative interval to a prom.Range.
//
// The origin is used to calculate the absolute timestamps in the samples.
func RelativeIntervalToRange(ri RelativeInterval, origin model.Time, step time.Duration) prom.Range {
	samples := make([]model.SamplePair, 0)
	for i := ri.Start; i < ri.End; i += int(step.Minutes()) {
		samples = append(samples, model.SamplePair{
			Timestamp: origin.Add(time.Duration(i) * time.Minute),
			Value:     1})
	}
	return prom.Range{
		Metric:  ri.Labels,
		Samples: samples,
		Step:    step,
	}
}

// RelativeIntervalsToRangeVectors converts a slice of relative intervals
// to a prom.RangeVector.
func RelativeIntervalsToRangeVectors(
	intervals []RelativeInterval, origin model.Time, step time.Duration) prom.RangeVector {
	ret := make(prom.RangeVector, len(intervals))
	for i, ri := range intervals {
		ret[i] = RelativeIntervalToRange(ri, origin, step)
	}
	return ret
}
