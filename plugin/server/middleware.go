package server

import (
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func DefaultMiddlewares() []echo.MiddlewareFunc {
	return []echo.MiddlewareFunc{
		middleware.Logger(),
		middleware.Recover(),
		middleware.CORS(),
		middleware.BodyLimit("2M"),
		middleware.RateLimiter(middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{Rate: 5, Burst: 30, ExpiresIn: 5 * time.Minute},
		)),
	}
}

func (s *Server) VerifierAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	if s.authMiddleware == nil {
		return next
	}
	return s.authMiddleware(next)
}
