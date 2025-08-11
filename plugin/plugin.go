package plugin

import (
	rtypes "github.com/vultisig/recipes/types"
	"github.com/vultisig/verifier/types"
)

type Spec interface {
	GetRecipeSpecification() (*rtypes.RecipeSchema, error)
	ValidatePluginPolicy(policyDoc types.PluginPolicy) error
	Suggest(configuration map[string]any) (*rtypes.PolicySuggest, error)
}

// Unimplemented for backward compatibility in the case of new interface methods
type Unimplemented struct {
}

func (*Unimplemented) GetRecipeSpecification() (*rtypes.RecipeSchema, error) {
	return nil, nil
}

func (*Unimplemented) ValidatePluginPolicy(_ types.PluginPolicy) error {
	return nil
}

func (*Unimplemented) Suggest(map[string]any) (*rtypes.PolicySuggest, error) {
	return nil, nil
}
