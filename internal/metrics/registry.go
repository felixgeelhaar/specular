package metrics

import (
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Default is the default metrics instance
	Default *Metrics
	once    sync.Once
)

// InitDefault initializes the default metrics instance
// This should be called once at application startup
func InitDefault() *Metrics {
	once.Do(func() {
		Default = NewMetrics(prometheus.DefaultRegisterer)
	})
	return Default
}

// GetDefault returns the default metrics instance
// If not initialized, it will initialize it first
func GetDefault() *Metrics {
	if Default == nil {
		return InitDefault()
	}
	return Default
}

// NewRegistry creates a new Prometheus registry with metrics
func NewRegistry() (*prometheus.Registry, *Metrics) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)
	return reg, m
}

// Handler returns an HTTP handler for Prometheus metrics endpoint
func Handler() http.Handler {
	return promhttp.Handler()
}

// HandlerFor returns an HTTP handler for a specific registry
func HandlerFor(reg prometheus.Gatherer, opts promhttp.HandlerOpts) http.Handler {
	return promhttp.HandlerFor(reg, opts)
}

// Reset clears the default metrics instance (useful for testing)
func Reset() {
	Default = nil
	once = sync.Once{}
}
