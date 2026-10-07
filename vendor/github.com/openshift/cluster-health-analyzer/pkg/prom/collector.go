package prom

import (
	"sync"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
)

type Metric struct {
	Labels model.LabelSet
	Value  float64
}

type metricSet struct {
	mtx     sync.RWMutex
	metrics []Metric
	name    string
	help    string
}

// MetricSet is an expasion of prometheus.Collector interface that allows batch
// updates of metrics. Useful when processing a set of metrics that are later
// exposed to Prometheus via different metric.
type MetricSet interface {
	prom.Collector
	Update(metrics []Metric)
}

func NewMetricSet(name, help string) *metricSet {
	return &metricSet{name: name, help: help}
}

func (m *metricSet) Update(metrics []Metric) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.metrics = metrics
}

func (m *metricSet) Reset() {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.metrics = nil
}

func (m *metricSet) Collect(ch chan<- prom.Metric) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	for _, metric := range m.metrics {
		labels := make([]string, 0, len(metric.Labels))
		values := make([]string, 0, len(metric.Labels))
		for k, v := range metric.Labels {
			if k == "__name__" {
				continue
			}

			labels = append(labels, string(k))
			values = append(values, string(v))
		}
		desc := prom.NewDesc(m.name, m.help, labels, nil)
		ch <- prom.MustNewConstMetric(desc, prom.GaugeValue, metric.Value, values...)
	}
}

func (m *metricSet) Describe(ch chan<- *prom.Desc) {
	ch <- prom.NewDesc(m.name, m.help, nil, nil)
}
