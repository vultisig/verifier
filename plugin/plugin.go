package plugin

import (
	rtypes "github.com/vultisig/recipes/types"
	"github.com/vultisig/verifier/types"
)

type Spec interface {
	GetRecipeSpecification() (*rtypes.RecipeSchema, error)
	ValidatePluginPolicy(policyDoc types.PluginPolicy) error
}
