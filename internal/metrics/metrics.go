package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// Registry holds the Prometheus metrics registry and HTTP handler
type Registry struct {
	registry *prometheus.Registry
	handler  http.Handler
	logger   *logrus.Logger
}

// NewRegistry creates a new metrics registry
func NewRegistry(logger *logrus.Logger) *Registry {
	registry := prometheus.NewRegistry()
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})

	return &Registry{
		registry: registry,
		handler:  handler,
		logger:   logger.WithField("component", "metrics").Logger,
	}
}

// Register adds a collector to the registry
func (r *Registry) Register(collector prometheus.Collector) error {
	return r.registry.Register(collector)
}

// MustRegister adds a collector to the registry and panics on error
func (r *Registry) MustRegister(collectors ...prometheus.Collector) {
	r.registry.MustRegister(collectors...)
}

// Handler returns the HTTP handler for the /metrics endpoint
func (r *Registry) Handler() http.Handler {
	return r.handler
}

// ServeHTTP implements http.Handler
func (r *Registry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.handler.ServeHTTP(w, req)
}