package policy

import (
	"context"

	"github.com/google/uuid"
	"github.com/vultisig/verifier/plugin/storage"
	"github.com/vultisig/verifier/types"
)

type Storage[T any] interface {
	Tx() storage.Tx
	GetPluginPolicy(ctx context.Context, id uuid.UUID) (*types.PluginPolicy, error)
	GetAllPluginPolicies(
		ctx context.Context,
		publicKey string,
		pluginID types.PluginID,
		onlyActive bool,
	) ([]types.PluginPolicy, error)
	DeletePluginPolicy(ctx context.Context, id uuid.UUID) error
	InsertPluginPolicy(ctx context.Context, policy types.PluginPolicy) (*types.PluginPolicy, error)
	UpdatePluginPolicy(ctx context.Context, policy types.PluginPolicy) (*types.PluginPolicy, error)
}
