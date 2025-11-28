package metrics

import (
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vultisig/verifier/plugin/metrics"
)

var (
	// Plugin server HTTP request metrics
	pluginServerHTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "verifier",
			Subsystem: "plugin_server",
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	pluginServerHTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "verifier",
			Subsystem: "plugin_server",
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request latency in seconds",
			Buckets:   prometheus.DefBuckets, // Default: .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10
		},
		[]string{"method", "path"},
	)

	pluginServerHTTPErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "verifier",
			Subsystem: "plugin_server",
			Name:      "http_errors_total",
			Help:      "Total number of HTTP errors (status >= 300)",
		},
		[]string{"method", "path", "status"},
	)
)

// PluginServerMetrics provides methods to update plugin server HTTP metrics
type PluginServerMetrics struct{}

// NewPluginServerMetrics creates a new instance of PluginServerMetrics
func NewPluginServerMetrics() metrics.PluginServerMetrics {
	return &PluginServerMetrics{}
}

// Register registers all plugin server metrics with the provided registry
func (psm *PluginServerMetrics) Register(registry metrics.Registry) {
	registry.MustRegister(
		pluginServerHTTPRequestsTotal,
		pluginServerHTTPRequestDuration,
		pluginServerHTTPErrorsTotal,
	)
}

// HTTPMiddleware returns Echo middleware for HTTP metrics collection
func (psm *PluginServerMetrics) HTTPMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			method := c.Request().Method

			// Prefer route pattern; fall back to raw path for unregistered routes
			routePath := c.Path()
			if routePath == "" {
				routePath = c.Request().URL.Path
			}
			path := normalizePath(routePath)

			// Execute handler
			err := next(c)

			// Compute duration
			duration := time.Since(start).Seconds()

			// Derive final status code
			status := c.Response().Status

			// If handler returned an error, prefer its code
			if he, ok := err.(*echo.HTTPError); ok {
				status = he.Code
			}

			// Fallbacks if still unset
			if status == 0 {
				if err != nil {
					status = 500 // Internal server error
				} else {
					status = 200 // OK
				}
			}

			statusStr := strconv.Itoa(status)

			// Record metrics
			pluginServerHTTPRequestsTotal.WithLabelValues(method, path, statusStr).Inc()
			pluginServerHTTPRequestDuration.WithLabelValues(method, path).Observe(duration)

			// Record errors for non-2xx status codes (4xx + 5xx + weird 1xx/3xx)
			if status < 200 || status >= 300 {
				pluginServerHTTPErrorsTotal.WithLabelValues(method, path, statusStr).Inc()
			}

			return err
		}
	}
}

// RecordHTTPRequest manually records an HTTP request
func (psm *PluginServerMetrics) RecordHTTPRequest(method, path, statusCode string, duration float64) {
	pluginServerHTTPRequestsTotal.WithLabelValues(method, path, statusCode).Inc()
	pluginServerHTTPRequestDuration.WithLabelValues(method, path).Observe(duration)
}

// RecordHTTPError manually records an HTTP error
func (psm *PluginServerMetrics) RecordHTTPError(method, path, statusCode string) {
	pluginServerHTTPErrorsTotal.WithLabelValues(method, path, statusCode).Inc()
}

// normalizePath returns the Echo route pattern to avoid high cardinality metrics
// Echo's c.Path() already provides the route pattern (e.g., "/vault/:pluginId/:publicKeyECDSA")
// rather than actual request paths (e.g., "/vault/123/0x456"), so no transformation needed
func normalizePath(path string) string {
	if path == "" {
		return "unknown"
	}

	// Return the Echo route pattern as-is since it already contains placeholders
	return path
}