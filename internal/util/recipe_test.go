package util

import (
	"testing"

	rtypes "github.com/vultisig/recipes/types"
)

func TestValidateRecipeSchema(t *testing.T) {
	tests := []struct {
		name   string
		recipe rtypes.RecipeSchema
		hasErr bool
	}{
		{
			name: "missing plugin ID",
			recipe: rtypes.RecipeSchema{
				PluginId:   "",
				PluginName: "Test Plugin",
				SupportedResources: []*rtypes.ResourcePattern{
					&rtypes.ResourcePattern{},
				},
			},
			hasErr: true,
		},
		{
			name: "missing plugin name",
			recipe: rtypes.RecipeSchema{
				PluginId:   "plugin-123",
				PluginName: "",
				SupportedResources: []*rtypes.ResourcePattern{
					&rtypes.ResourcePattern{},
				},
			},
			hasErr: true,
		},
		{
			name: "missing supported resources",
			recipe: rtypes.RecipeSchema{
				PluginId:           "plugin-123",
				PluginName:         "Test Plugin",
				SupportedResources: []*rtypes.ResourcePattern{},
			},
			hasErr: true,
		},
		{
			name: "valid recipe",
			recipe: rtypes.RecipeSchema{
				PluginId:   "plugin-123",
				PluginName: "Test Plugin",
				SupportedResources: []*rtypes.ResourcePattern{
					&rtypes.ResourcePattern{},
				},
			},
			hasErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRecipeSchema(tt.recipe)
			if tt.hasErr && err == nil {

				t.Errorf("expected error, got nil")
			}
			if !tt.hasErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}
