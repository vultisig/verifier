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
