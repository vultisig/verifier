package plugin

import (
	"context"

	"github.com/vultisig/mobile-tss-lib/tss"
	rtypes "github.com/vultisig/recipes/types"
	"github.com/vultisig/verifier/types"
)

type Plugin interface {
	GetRecipeSpecification() *rtypes.RecipeSchema
	ValidatePluginPolicy(policyDoc types.PluginPolicy) error
	ProposeTransactions(policy types.PluginPolicy) ([]types.PluginKeysignRequest, error)
	ValidateProposedTransactions(policy types.PluginPolicy, txs []types.PluginKeysignRequest) error
	SigningComplete(ctx context.Context, signature tss.KeysignResponse, signRequest types.PluginKeysignRequest, policy types.PluginPolicy) error
}
