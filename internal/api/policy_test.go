package api

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/google/uuid"
	rtypes "github.com/vultisig/recipes/types"
	"google.golang.org/protobuf/proto"

	"github.com/vultisig/verifier/types"
)

func TestValidatePluginPolicy_Success(t *testing.T) {
	recipe := &rtypes.Policy{
		Id:   uuid.New().String(),
		Name: "Test Recipe",
	}
	recipeBytes, err := proto.Marshal(recipe)
	if err != nil {
		t.Fatalf("failed to marshal recipe: %v", err)
	}
	encodedRecipe := base64.StdEncoding.EncodeToString(recipeBytes)

	policy := types.PluginPolicy{
		ID:            uuid.New(),
		PublicKey:     "test_public_key",
		PluginID:      types.PluginID("test_plugin_id"),
		PluginVersion: "1.0.0",
		PolicyVersion: 1,
		Signature:     "test_signature",
		Recipe:        encodedRecipe,
		Billing: []types.BillingPolicy{
			{
				ID:        uuid.New(),
				Type:      string(types.BILLING_TYPE_TX),
				Frequency: "monthly",
				StartDate: time.Now(),
				Amount:    1000000,
			},
		},
		Active: true,
	}

	server := &Server{}
	err = server.validatePluginPolicy(policy)
	if err != nil {
		t.Errorf("validatePluginPolicy() returned error: %v", err)
	}

	_, err = policy.GetRecipe()
	if err != nil {
		t.Errorf("GetRecipe() returned error: %v", err)
	}
}
