package util

import (
	"errors"

	rtypes "github.com/vultisig/recipes/types"
)

func ValidateRecipeSchema(recipe rtypes.RecipeSchema) error {
	if recipe.PluginId == "" {
		return errors.New("plugin ID is required")
	}
	if recipe.PluginName == "" {
		return errors.New("plugin name is required")
	}
	if len(recipe.SupportedResources) == 0 {
		return errors.New("at least one supported resource is required")
	}
	return nil
}
