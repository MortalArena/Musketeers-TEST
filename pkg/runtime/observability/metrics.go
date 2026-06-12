package observability

import "github.com/prometheus/client_golang/prometheus"

type Metrics interface {
	Counter(name string, labels map[string]string) Counter
	Gauge(name string, labels map[string]string) Gauge
	Histogram(name string, labels map[string]string) Histogram
}

type Counter interface {
	Inc()
	Add(value float64)
}

type Gauge interface {
	Set(value float64)
	Inc()
	Dec()
	Add(value float64)
	Sub(value float64)
}

type Histogram interface {
	Observe(value float64)
}

type PrometheusMetrics struct {
	counters   map[string]*prometheus.CounterVec
	gauges     map[string]*prometheus.GaugeVec
	histograms map[string]*prometheus.HistogramVec
}

func NewPrometheusMetrics() *PrometheusMetrics {
	return &PrometheusMetrics{
		counters:   make(map[string]*prometheus.CounterVec),
		gauges:     make(map[string]*prometheus.GaugeVec),
		histograms: make(map[string]*prometheus.HistogramVec),
	}
}

func (m *PrometheusMetrics) Counter(name string, labels map[string]string) Counter {
	labelNames := labelNames(labels)
	counterVec, exists := m.counters[name]
	if !exists {
		counterVec = prometheus.NewCounterVec(prometheus.CounterOpts{Name: name}, labelNames)
		prometheus.MustRegister(counterVec)
		m.counters[name] = counterVec
	}
	return &PrometheusCounter{counter: counterVec.With(labels)}
}

func (m *PrometheusMetrics) Gauge(name string, labels map[string]string) Gauge {
	labelNames := labelNames(labels)
	gaugeVec, exists := m.gauges[name]
	if !exists {
		gaugeVec = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: name}, labelNames)
		prometheus.MustRegister(gaugeVec)
		m.gauges[name] = gaugeVec
	}
	return &PrometheusGauge{gauge: gaugeVec.With(labels)}
}

func (m *PrometheusMetrics) Histogram(name string, labels map[string]string) Histogram {
	labelNames := labelNames(labels)
	histogramVec, exists := m.histograms[name]
	if !exists {
		histogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: name, Buckets: prometheus.DefBuckets}, labelNames)
		prometheus.MustRegister(histogramVec)
		m.histograms[name] = histogramVec
	}
	return &PrometheusHistogram{histogram: histogramVec.With(labels)}
}

func labelNames(labels map[string]string) []string {
	names := make([]string, 0, len(labels))
	for k := range labels {
		names = append(names, k)
	}
	return names
}

type PrometheusCounter struct {
	counter prometheus.Counter
}

func (c *PrometheusCounter) Inc()              { c.counter.Inc() }
func (c *PrometheusCounter) Add(value float64) { c.counter.Add(value) }

type PrometheusGauge struct {
	gauge prometheus.Gauge
}

func (g *PrometheusGauge) Set(value float64) { g.gauge.Set(value) }
func (g *PrometheusGauge) Inc()              { g.gauge.Inc() }
func (g *PrometheusGauge) Dec()              { g.gauge.Dec() }
func (g *PrometheusGauge) Add(value float64) { g.gauge.Add(value) }
func (g *PrometheusGauge) Sub(value float64) { g.gauge.Sub(value) }

type PrometheusHistogram struct {
	histogram prometheus.Observer
}

func (h *PrometheusHistogram) Observe(value float64) { h.histogram.Observe(value) }
