package observer

import (
	"sync"
	"time"
)

// MetricType defines the type of metric.
type MetricType string

const (
	MetricTypeCounter   MetricType = "COUNTER"
	MetricTypeGauge     MetricType = "GAUGE"
	MetricTypeHistogram MetricType = "HISTOGRAM"
)

// Metric represents a single measurement.
type Metric struct {
	Name      string
	Type      MetricType
	Value     float64
	Labels    map[string]string
	Timestamp time.Time
}

// MetricsCollector defines the interface for collecting and exporting metrics.
type MetricsCollector interface {
	Inc(name string, labels map[string]string)
	Gauge(name string, value float64, labels map[string]string)
	Observe(name string, value float64, labels map[string]string)
	GetMetrics() []Metric
}

// DefaultMetricsCollector is an in-memory implementation of MetricsCollector.
type DefaultMetricsCollector struct {
	metrics []Metric
	mu      sync.RWMutex
}

func NewDefaultMetricsCollector() *DefaultMetricsCollector {
	return &DefaultMetricsCollector{
		metrics: make([]Metric, 0),
	}
}

func (c *DefaultMetricsCollector) Inc(name string, labels map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metrics = append(c.metrics, Metric{
		Name:      name,
		Type:      MetricTypeCounter,
		Value:     1,
		Labels:    labels,
		Timestamp: time.Now(),
	})
}

func (c *DefaultMetricsCollector) Gauge(name string, value float64, labels map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metrics = append(c.metrics, Metric{
		Name:      name,
		Type:      MetricTypeGauge,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
	})
}

func (c *DefaultMetricsCollector) Observe(name string, value float64, labels map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metrics = append(c.metrics, Metric{
		Name:      name,
		Type:      MetricTypeHistogram,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
	})
}

func (c *DefaultMetricsCollector) GetMetrics() []Metric {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]Metric, len(c.metrics))
	copy(result, c.metrics)
	return result
}
