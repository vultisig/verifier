package logging

import (
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

func LoggerMiddleware(logger *logrus.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			res := c.Response()
			start := time.Now()

			err := next(c)
			if err != nil {
				c.Error(err)
			}

			if req.RequestURI == "/healthz" {
				return nil
			}

			latency := time.Since(start)

			logger.WithFields(logrus.Fields{
				"remote_ip":     c.RealIP(),
				"host":          req.Host,
				"method":        req.Method,
				"uri":           req.RequestURI,
				"user_agent":    req.UserAgent(),
				"status":        res.Status,
				"latency":       latency.Microseconds(),
				"latency_human": latency.String(),
				"bytes_in":      req.ContentLength,
				"bytes_out":     res.Size,
			}).Info("HTTP request")

			return nil
		}
	}
}
