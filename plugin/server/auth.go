package server

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

type Auth struct {
	token []byte
}

func NewAuth(token string) *Auth {
	return &Auth{
		token: []byte(token),
	}
}

func (a *Auth) Middleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authHeader := c.Request().Header.Get(echo.HeaderAuthorization)
		if authHeader == "" {
			return echo.NewHTTPError(http.StatusUnauthorized, "missing authorization header")
		}

		const prefix = "Bearer "
		if !strings.HasPrefix(authHeader, prefix) {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization header format")
		}

		token := authHeader[len(prefix):]
		if token == "" {
			return echo.NewHTTPError(http.StatusUnauthorized, "missing token")
		}

		if !a.validateToken(token) {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
		}

		return next(c)
	}
}

// validateToken performs constant-time comparison to prevent timing attacks
func (a *Auth) validateToken(token string) bool {
	tokenBytes := []byte(token)
	return subtle.ConstantTimeCompare(a.token, tokenBytes) == 1
}
