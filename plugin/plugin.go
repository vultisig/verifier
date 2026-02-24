package plugin

import (
	"context"
	"errors"

	rtypes "github.com/vultisig/recipes/types"
	"github.com/vultisig/verifier/types"
)

type Spec interface {
	GetRecipeSpecification() (*rtypes.RecipeSchema, error)
	ValidatePluginPolicy(policyDoc types.PluginPolicy) error
	// Suggest generates policy suggestions based on configuration.
	// Context is required as implementations may make RPC calls to external services.
	Suggest(ctx context.Context, configuration map[string]any) (*rtypes.PolicySuggest, error)
	// GetPluginID returns the unique identifier for this plugin.
	GetPluginID() string
	// GetSkills returns markdown describing the plugin's capabilities for AI agents.
	GetSkills() string
}

type BuildTxHandler interface {
	HandleBuildTx(ctx context.Context, body []byte) (any, error)
}

// Unimplemented for backward compatibility in the case of new interface methods
type Unimplemented struct {
}

func (*Unimplemented) GetRecipeSpecification() (*rtypes.RecipeSchema, error) {
	return nil, errors.New("not implemented")
}

func (*Unimplemented) ValidatePluginPolicy(_ types.PluginPolicy) error {
	return errors.New("not implemented")
}

func (*Unimplemented) Suggest(_ context.Context, _ map[string]any) (*rtypes.PolicySuggest, error) {
	return nil, errors.New("not implemented")
}

func (*Unimplemented) GetPluginID() string {
	return ""
}

func (*Unimplemented) GetSkills() string {
	return ""
}
