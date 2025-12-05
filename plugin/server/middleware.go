package server

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// DefaultMiddlewares returns default middlewares with Victoria Metrics compatible logging
func DefaultMiddlewares() []echo.MiddlewareFunc {
	return []echo.MiddlewareFunc{
		VictoriaMetricsLogger(),
		middleware.Recover(),
		middleware.CORS(),
		middleware.BodyLimit("2M"),
		middleware.RateLimiter(middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{Rate: 5, Burst: 30, ExpiresIn: 5 * time.Minute},
		)),
	}
}

// VictoriaMetricsLogger creates a middleware that logs HTTP requests in Victoria Metrics compatible format
func VictoriaMetricsLogger() echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:      true,
		LogStatus:   true,
		LogMethod:   true,
		LogLatency:  true,
		LogRemoteIP: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			// Use fmt.Printf to output JSON in Victoria Metrics compatible format
			logEntry := map[string]interface{}{
				"_msg":         "HTTP request",
				"timestamp":    v.StartTime.Format("2006-01-02T15:04:05.000Z07:00"),
				"level":        "info",
				"method":       v.Method,
				"uri":          v.URI,
				"status":       v.Status,
				"latency_ms":   v.Latency.Milliseconds(),
				"remote_ip":    v.RemoteIP,
				"user_agent":   c.Request().UserAgent(),
				"bytes_in":     v.ContentLength,
				"service":      "plugin-server",
			}
			
			if v.Error != nil {
				logEntry["error"] = v.Error.Error()
			}
			
			jsonBytes, _ := json.Marshal(logEntry)
			fmt.Println(string(jsonBytes))
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
