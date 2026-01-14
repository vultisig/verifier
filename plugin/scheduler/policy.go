package scheduler

import (
	"context"

	"github.com/google/uuid"
	"github.com/vultisig/verifier/types"
)

type PolicyFetcher interface {
	GetPluginPolicy(ctx context.Context, id uuid.UUID) (*types.PluginPolicy, error)
	UpdatePluginPolicy(ctx context.Context, policy types.PluginPolicy) (*types.PluginPolicy, error)
}

type SafetyManager interface {
	EnforceKeysign(ctx context.Context, pluginID string) error
}
