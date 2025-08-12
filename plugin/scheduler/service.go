package scheduler

import (
	"context"

	"github.com/google/uuid"
	"github.com/vultisig/verifier/types"
)

type Service interface {
	Create(ctx context.Context, policy types.PluginPolicy) error
	Update(ctx context.Context, oldPolicy, newPolicy types.PluginPolicy) error
	Delete(ctx context.Context, policyID uuid.UUID) error
}
