package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

func (s *Server) statsdMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		start := time.Now()
		err := next(c)
		duration := time.Since(start).Milliseconds()

		// Send metrics to statsd
		_ = s.sdClient.Incr("http.requests", []string{"path:" + c.Path()}, 1)
		_ = s.sdClient.Timing("http.response_time", time.Duration(duration)*time.Millisecond, []string{"path:" + c.Path()}, 1)
		_ = s.sdClient.Incr("http.status."+fmt.Sprint(c.Response().Status), []string{"path:" + c.Path(), "method:" + c.Request().Method}, 1)

		return err
	}
}

func (s *Server) userAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			return c.JSON(http.StatusUnauthorized, NewErrorResponse("Authorization header required"))
		}

		// TODO: add user authentication logic

		return next(c)
	}
}

func (s *Server) AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			return c.JSON(http.StatusUnauthorized, NewErrorResponse("Authorization header required"))
		}

		tokenStr := authHeader[len("Bearer "):]
		_, err := s.authService.ValidateToken(tokenStr)
		if err != nil {
			s.logger.Warnf("fail to validate token, err: %v", err)
			return c.JSON(http.StatusUnauthorized, NewErrorResponse("Unauthorized"))
		}
		s.logger.Info("Token validated successfully")
		return next(c)
	}
}
