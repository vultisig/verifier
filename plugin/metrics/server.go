package metrics

import (
	"github.com/labstack/echo/v4"
)

// PluginServerMetrics interface for collecting plugin server HTTP metrics
type PluginServerMetrics interface {
	// Register registers all metrics with the provided registry
	Register(registry Registry)

	// HTTP metrics recording methods
	RecordHTTPRequest(method, path, status string, duration float64)
	RecordHTTPError(method, path, status string)
	
	// Middleware integration
	HTTPMiddleware() echo.MiddlewareFunc
}

// NilPluginServerMetrics is a no-op implementation for when metrics are disabled
type NilPluginServerMetrics struct{}

// NewNilPluginServerMetrics creates a no-op metrics implementation
func NewNilPluginServerMetrics() PluginServerMetrics {
	return &NilPluginServerMetrics{}
}

// All methods are no-ops - safe to call, do nothing
func (n *NilPluginServerMetrics) Register(registry Registry) {}

func (n *NilPluginServerMetrics) RecordHTTPRequest(method, path, status string, duration float64) {}
func (n *NilPluginServerMetrics) RecordHTTPError(method, path, status string) {}

func (n *NilPluginServerMetrics) HTTPMiddleware() echo.MiddlewareFunc {
	// Return pass-through middleware that does nothing
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return next
	}
}