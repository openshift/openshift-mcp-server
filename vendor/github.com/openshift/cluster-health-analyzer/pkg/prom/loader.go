package prom

//go:generate mockgen -package=mocks -mock_names=Loader=MockPrometheusLoader -source=loader.go -destination=../test/mocks/mock_prometheus_loader.go

import (
	"context"
	"net/http"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type loader struct {
	api v1.API
}

type Loader interface {
	LoadQuery(ctx context.Context, query string, t time.Time) ([]model.LabelSet, error)
	LoadAlertsRange(ctx context.Context, start, end time.Time, step time.Duration) (RangeVector, error)
	LoadVectorRange(ctx context.Context, query string, start, end time.Time, step time.Duration) (RangeVector, error)
}

func NewLoader(prometheusURL string) (Loader, error) {
	promClient, err := NewPrometheusClient(prometheusURL)
	if err != nil {
		return nil, err
	}
	return &loader{
		api: v1.NewAPI(promClient),
	}, nil
}

func NewLoaderWithToken(prometheusURL, token string) (Loader, error) {
	promClient, err := NewPrometheusClientWithToken(prometheusURL, token)
	if err != nil {
		return nil, err
	}
	return &loader{
		api: v1.NewAPI(promClient),
	}, nil
}

// NewLoaderWithRoundTripper creates a loader using a supplied transport.
func NewLoaderWithRoundTripper(prometheusURL string, roundTripper http.RoundTripper) (Loader, error) {
	promClient, err := NewPrometheusClientWithRoundTripper(prometheusURL, roundTripper)
	if err != nil {
		return nil, err
	}
	return &loader{api: v1.NewAPI(promClient)}, nil
}

func (c *loader) LoadQuery(ctx context.Context, query string, t time.Time) ([]model.LabelSet, error) {
	result, _, err := c.api.Query(ctx, query, t)
	if err != nil {
		return nil, err
	}
	return modelValueToLabelSet(result), nil
}

func (c *loader) LoadAlertsRange(ctx context.Context, start, end time.Time, step time.Duration) (RangeVector, error) {
	result, _, err := c.api.QueryRange(ctx, `ALERTS{alertstate="firing"}`, v1.Range{
		Start: start,
		End:   end,
		Step:  step,
	})
	if err != nil {
		return nil, err
	}
	return modelValueToRangeVector(result, step), nil
}

func (c *loader) LoadVectorRange(ctx context.Context, query string, start, end time.Time, step time.Duration) (RangeVector, error) {
	result, _, err := c.api.QueryRange(ctx, query, v1.Range{
		Start: start,
		End:   end,
		Step:  step,
	})
	if err != nil {
		return nil, err
	}

	return modelValueToRangeVector(result, step), nil
}

func modelValueToRangeVector(mv model.Value, step time.Duration) RangeVector {
	matrix := mv.(model.Matrix)
	ret := make(RangeVector, len(matrix))
	for i, samples := range matrix {
		m := make(model.LabelSet, len(samples.Metric))
		for k, v := range samples.Metric {
			m[k] = v
		}
		ret[i] = Range{
			Metric:  m,
			Samples: samples.Values,
			Step:    step,
		}
	}
	return ret
}

func modelValueToLabelSet(mv model.Value) []model.LabelSet {
	vect := mv.(model.Vector)
	var ret = make([]model.LabelSet, len(vect))
	for i, sample := range vect {
		m := make(model.LabelSet, len(sample.Metric))
		for k, v := range sample.Metric {
			m[k] = v
		}
		ret[i] = m
	}
	return ret
}
