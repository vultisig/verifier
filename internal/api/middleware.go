package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	"github.com/vultisig/verifier/internal/service"
)

func (s *Server) AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authHeader := c.Request().Header.Get(echo.HeaderAuthorization)
		if authHeader == "" {
			return c.JSON(http.StatusUnauthorized, NewErrorResponseWithMessage(msgMissingAuthHeader))
		}

		// Extract token from Bearer format
		tokenParts := strings.Fields(authHeader)
		if len(tokenParts) != 2 || !strings.EqualFold(tokenParts[0], "Bearer") {
			return c.JSON(http.StatusUnauthorized, NewErrorResponseWithMessage(msgInvalidAuthHeader))
		}
		tokenStr := tokenParts[1]

		_, err := s.authService.ValidateToken(c.Request().Context(), tokenStr)
		if err != nil {
			s.logger.Warnf("fail to validate token, err: %v", err)
			return c.JSON(http.StatusUnauthorized, NewErrorResponseWithMessage(msgUnauthorized))
		}
		s.logger.Info("Token validated successfully")
		return next(c)
	}
}

// VaultAuthMiddleware verifies JWT tokens and ensures users can only access their own vaults.
func (s *Server) VaultAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if s.cfg.Auth.Enabled != nil && !*s.cfg.Auth.Enabled {
			s.logger.Info("Auth is disabled, skipping token validation")
			return next(c)
		}

		authHeader := c.Request().Header.Get(echo.HeaderAuthorization)
		if authHeader == "" {
			return c.JSON(http.StatusUnauthorized, NewErrorResponseWithMessage(msgMissingAuthHeader))
		}

		tokenParts := strings.Fields(authHeader)
		if len(tokenParts) != 2 || !strings.EqualFold(tokenParts[0], "Bearer") {
			return c.JSON(http.StatusUnauthorized, NewErrorResponseWithMessage(msgInvalidAuthHeader))
		}
		tokenStr := tokenParts[1]

		claims, err := s.authService.ValidateToken(c.Request().Context(), tokenStr)
		if err != nil {
			s.logger.Warnf("fail to validate token, err: %v", err)
			return c.JSON(http.StatusUnauthorized, NewErrorResponseWithMessage(msgUnauthorized))
		}

		if claims.TokenType != service.TokenTypeAccess {
			s.logger.Warnf("invalid token type: expected access token, got: %s", claims.TokenType)
			return c.JSON(http.StatusUnauthorized, NewErrorResponseWithMessage("access token required"))
		}

		// Get requested vault's public key from URL parameter
		// keep in mind, quite some endpoint will not require user to pass `publicKeyECDSA` in the URL
		requestedPublicKey := c.Param("publicKeyECDSA")
		if requestedPublicKey != "" {
			// Verify the token's public key matches the requested vault
			if claims.PublicKey != requestedPublicKey {
				s.logger.Warnf("Access denied: token public key %s does not match requested vault %s",
					claims.PublicKey, requestedPublicKey)
				return c.JSON(http.StatusForbidden, NewErrorResponseWithMessage(msgAccessDenied))
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
			return c.JSON(http.StatusUnauthorized, NewErrorResponseWithMessage(msgMissingAuthHeader))
		}

		items := strings.Fields(authHeader)
		if len(items) != 2 || !strings.EqualFold(items[0], "Bearer") {
			return c.JSON(http.StatusUnauthorized, NewErrorResponseWithMessage(msgInvalidAuthHeader))
		}
		tokenStr := items[1]
		apiKey, err := s.db.GetAPIKey(c.Request().Context(), tokenStr)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				s.logger.Warnf("API key not found: %v", tokenStr)
				return c.JSON(http.StatusUnauthorized, NewErrorResponseWithMessage(msgAPIKeyNotFound))
			}
			s.logger.WithError(err).Error("fail to get API key")
			return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgInternalError))
		}
		if apiKey.Status == 0 {
			s.logger.Warnf("API key is disabled, id: %s", apiKey.ID)
			return c.JSON(http.StatusUnauthorized, NewErrorResponseWithMessage(msgDisabledAPIKey))
		}
		if apiKey.ExpiresAt != nil {
			if apiKey.ExpiresAt.Before(time.Now()) {
				s.logger.Warnf("API key is expired, id: %s", apiKey.ID)
				return c.JSON(http.StatusUnauthorized, NewErrorResponseWithMessage(msgExpiredAPIKey))
			}
		}
		c.Set("plugin_id", apiKey.PluginID)
		return next(c)
	}
}
