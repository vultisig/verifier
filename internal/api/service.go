package api

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/vultisig/vultisig-go/common"

	itypes "github.com/vultisig/verifier/internal/types"
)

// GetServiceFeeStatus returns fee/trial status for a vault identified by public_key query param.
// Called by server-to-server clients (e.g. MCP server) with X-Service-Key auth.
func (s *Server) GetServiceFeeStatus(c echo.Context) error {
	publicKey := c.QueryParam("public_key")
	if publicKey == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgRequiredPublicKey))
	}

	status, err := s.feeService.GetUserFees(c.Request().Context(), publicKey)
	if err != nil {
		s.logger.WithError(err).Error("GetServiceFeeStatus: failed to get user fees")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgGetUserFeesFailed))
	}

	err = s.db.WithTransaction(c.Request().Context(), func(ctx context.Context, tx pgx.Tx) error {
		status.IsTrialActive, status.TrialRemaining, err = s.db.IsTrialActive(ctx, tx, publicKey)
		return err
	})
	if err != nil {
		s.logger.WithError(err).Warn("GetServiceFeeStatus: failed to check trial info")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgGetUserTrialInfo))
	}

	return c.JSON(http.StatusOK, status)
}

// GetServiceInstalledPlugins returns installed plugins for a vault identified by public_key query param.
// Called by server-to-server clients (e.g. MCP server) with X-Service-Key auth.
func (s *Server) GetServiceInstalledPlugins(c echo.Context) error {
	publicKey := c.QueryParam("public_key")
	if publicKey == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgRequiredPublicKey))
	}

	pluginList, err := s.db.FindPlugins(c.Request().Context(), itypes.PluginFilters{}, 1000, 0, "")
	if err != nil {
		return s.internal(c, msgGetPluginsFailed, err)
	}

	var installed itypes.PluginsPaginatedList
	installed.Plugins = make([]itypes.Plugin, 0, len(pluginList.Plugins))

	for _, plugin := range pluginList.Plugins {
		fileName := common.GetVaultBackupFilename(publicKey, string(plugin.ID))
		exist, err := s.vaultStorage.Exist(fileName)
		if err != nil {
			s.logger.WithError(err).Errorf("GetServiceInstalledPlugins: failed to check vault existence for plugin %s", plugin.ID)
			continue
		}
		if exist {
			installed.Plugins = append(installed.Plugins, plugin)
			installed.TotalCount++
		}
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, installed))
}
