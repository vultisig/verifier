package api

import (
	"fmt"
	"net/http"
	"strings"
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
		authHeader := c.Request().Header.Get(echo.HeaderAuthorization)
		if authHeader == "" {
			return c.JSON(http.StatusUnauthorized, NewErrorResponse("Authorization header required"))
		}

		// TODO: add user authentication logic

		return next(c)
	}
}

func (s *Server) AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authHeader := c.Request().Header.Get(echo.HeaderAuthorization)
		if authHeader == "" {
			return c.JSON(http.StatusUnauthorized, NewErrorResponse("Authorization header required"))
		}

		tokenStr := authHeader[len("Bearer "):]
		_, err := s.authService.ValidateToken(c.Request().Context(), tokenStr)
		if err != nil {
			s.logger.Warnf("fail to validate token, err: %v", err)
			return c.JSON(http.StatusUnauthorized, NewErrorResponse("Unauthorized"))
		}
		s.logger.Info("Token validated successfully")
		return next(c)
	}
}

// VaultAuthMiddleware verifies JWT tokens and ensures users can only access their own vaults
func (s *Server) VaultAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Get token from header
		authHeader := c.Request().Header.Get(echo.HeaderAuthorization)
		if authHeader == "" {
			return c.JSON(http.StatusUnauthorized, NewErrorResponse("Missing authorization header"))
		}

		// Extract token from Bearer format
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			return c.JSON(http.StatusUnauthorized, NewErrorResponse("Invalid authorization header format"))
		}

		// Validate token and get claims
		claims, err := s.authService.ValidateToken(c.Request().Context(), tokenParts[1])
		if err != nil {
			s.logger.Errorf("Internal error: %v", err)
			return c.JSON(http.StatusInternalServerError, NewErrorResponse("An internal error occurred"))
		}

		// Get requested vault's public key from URL parameter
		// keep in mind, quite some endpoint will not require user to pass `publicKeyECDSA` in the URL
		requestedPublicKey := c.Param("publicKeyECDSA")
		if requestedPublicKey != "" {
			// Verify the token's public key matches the requested vault
			if claims.PublicKey != requestedPublicKey {
				s.logger.Warnf("Access denied: token public key %s does not match requested vault %s",
					claims.PublicKey, requestedPublicKey)
				return c.JSON(http.StatusForbidden, NewErrorResponse("Access denied: token not authorized for this vault"))
			}
		}

		// Store the public key in context for later use
		c.Set("vault_public_key", claims.PublicKey)

		return next(c)
	}
}
func (s *Server) PluginAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authHeader := c.Request().Header.Get(echo.HeaderAuthorization)
		if authHeader == "" {
			return c.JSON(http.StatusUnauthorized, NewErrorResponse("Authorization header required"))
		}

		items := strings.Fields(authHeader)
		if len(items) != 2 || items[0] != "Bearer" {
			return c.JSON(http.StatusUnauthorized, NewErrorResponse("Invalid authorization header format"))
		}
		tokenStr := items[1]
		apiKey, err := s.db.GetAPIKey(c.Request().Context(), tokenStr)
		if err != nil {
			s.logger.Errorf("fail to get API key, err: %v", err)
			return c.JSON(http.StatusInternalServerError, NewErrorResponse("Internal server error"))
		}
		if apiKey.Status == 0 {
			s.logger.Warnf("API key is disabled, id: %s", apiKey.ID)
			return c.JSON(http.StatusForbidden, NewErrorResponse("Invalid API key"))
		}
		if apiKey.ExpiresAt.Before(time.Now()) {
			s.logger.Warnf("API key is expired, id: %s", apiKey.ID)
			return c.JSON(http.StatusForbidden, NewErrorResponse("API key has expired"))
		}
		c.Set("plugin_id", apiKey.PluginID)
		return next(c)
	}
}
