package observability

import (
	"errors"
	"testing"
)

func TestZapLogger(t *testing.T) {
	logger, err := NewZapLogger("debug")
	if err != nil {
		t.Fatalf("NewZapLogger returned error: %v", err)
	}
	logger.Info("info", map[string]any{"key": "value"})
	logger.Warn("warn", nil)
	logger.Error("error", errors.New("boom"), nil)
	if logger.WithField("did", "did:ia:test").WithFields(nil) == nil {
		t.Fatal("expected logger")
	}
}

func TestPrometheusMetrics(t *testing.T) {
	metrics := NewPrometheusMetrics()
	counter := metrics.Counter("nr_test_counter", map[string]string{"label": "a"})
	counter.Inc()
	counter.Add(2)

	gauge := metrics.Gauge("nr_test_gauge", map[string]string{"label": "a"})
	gauge.Set(1)
	gauge.Inc()
	gauge.Dec()
	gauge.Add(0.5)
	gauge.Sub(0.5)

	metrics.Histogram("nr_test_histogram", map[string]string{"label": "a"}).Observe(1)
}
