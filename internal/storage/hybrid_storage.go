package storage

import (
	"context"

	"github.com/jackc/pgx/v5"

	iconfig "github.com/vultisig/verifier/internal/config"
	itypes "github.com/vultisig/verifier/internal/types"
	"github.com/vultisig/verifier/types"
)

type HybridStorage struct {
	DatabaseStorage
	pluginData *iconfig.PluginData
}

func NewHybridStorage(db DatabaseStorage, pluginData *iconfig.PluginData) *HybridStorage {
	return &HybridStorage{
		DatabaseStorage: db,
		pluginData:      pluginData,
	}
}

func (h *HybridStorage) FindPlugins(ctx context.Context, filters itypes.PluginFilters, take int, skip int, sort string) (*itypes.PluginsPaginatedList, error) {
	return h.pluginData.FindPlugins(filters, take, skip, sort)
}

func (h *HybridStorage) FindPluginById(ctx context.Context, dbTx pgx.Tx, id types.PluginID) (*itypes.Plugin, error) {
	return h.pluginData.FindPluginById(id)
}
