package server

import (
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
)

// DefaultMiddlewares returns default middlewares with Victoria Metrics compatible logging.
// The logger should be configured with JSONFormatter and FieldMap to use "_msg" field.
func DefaultMiddlewares(logger *logrus.Logger) []echo.MiddlewareFunc {
	return []echo.MiddlewareFunc{
		VictoriaMetricsLogger(logger),
		middleware.Recover(),
		middleware.CORS(),
		middleware.BodyLimit("2M"),
		middleware.RateLimiter(middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{Rate: 5, Burst: 30, ExpiresIn: 5 * time.Minute},
		)),
	}
}

// VictoriaMetricsLogger creates a middleware that logs HTTP requests using the provided logrus logger.
// The logger should be configured with JSONFormatter and FieldMap to use "_msg" field.
func VictoriaMetricsLogger(logger *logrus.Logger) echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		// Skipper: func(c echo.Context) bool {
		// 	return c.Path() == "/healthz"
		// },
		LogURI:           true,
		LogStatus:        true,
		LogMethod:        true,
		LogLatency:       true,
		LogRemoteIP:      true,
		LogError:         true,
		LogContentLength: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			entry := logger.WithFields(logrus.Fields{
				"method":     v.Method,
				"uri":        v.URI,
				"status":     v.Status,
				"latency_ms": v.Latency.Milliseconds(),
				"remote_ip":  v.RemoteIP,
				"user_agent": c.Request().UserAgent(),
				"bytes_in":   v.ContentLength,
				"service":    "plugin-server",
			})

			if v.Error != nil {
				entry = entry.WithError(v.Error)
			}

			entry.Info("HTTP request")
			return nil
		},
	})
}

func (s *Server) VerifierAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	if s.authMiddleware == nil {
		return next
	}
	return s.authMiddleware(next)
}
