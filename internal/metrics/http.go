package metrics

import (
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// HTTP request metrics
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "verifier",
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests by method, path, and status code",
		},
		[]string{"method", "path", "status_code"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "verifier",
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "Duration of HTTP requests by method and path",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	httpActiveRequests = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "verifier",
			Subsystem: "http",
			Name:      "active_requests",
			Help:      "Number of currently active HTTP requests by method and path",
		},
		[]string{"method", "path"},
	)
)

// HTTPMetrics provides methods to update HTTP-related metrics
type HTTPMetrics struct{}

// NewHTTPMetrics creates a new instance of HTTPMetrics
func NewHTTPMetrics() *HTTPMetrics {
	return &HTTPMetrics{}
}

// Register registers all HTTP metrics with the provided registry
func (hm *HTTPMetrics) Register(registry *prometheus.Registry) {
	registry.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		httpActiveRequests,
	)
}

// Middleware returns an Echo middleware for HTTP metrics collection
func (hm *HTTPMetrics) Middleware() echo.MiddlewareFunc {
	return echo.MiddlewareFunc(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Skip metrics collection if HTTPMetrics is nil
			if hm == nil {
				return next(c)
			}

			start := time.Now()
			method := c.Request().Method
			path := c.Path()

			// Increment active requests
			httpActiveRequests.WithLabelValues(method, path).Inc()
			defer httpActiveRequests.WithLabelValues(method, path).Dec()

			// Process the request
			err := next(c)

			// Record metrics
			duration := time.Since(start).Seconds()
			statusCode := strconv.Itoa(c.Response().Status)

			httpRequestsTotal.WithLabelValues(method, path, statusCode).Inc()
			httpRequestDuration.WithLabelValues(method, path).Observe(duration)

			return err
		}
	})
}

// RecordRequest manually records an HTTP request (for cases where middleware isn't used)
func (hm *HTTPMetrics) RecordRequest(method, path, statusCode string, duration float64) {
	httpRequestsTotal.WithLabelValues(method, path, statusCode).Inc()
	httpRequestDuration.WithLabelValues(method, path).Observe(duration)
}
