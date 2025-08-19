package types

import (
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"
	rtypes "github.com/vultisig/recipes/types"
	"google.golang.org/protobuf/proto"
)

// This type should be used externally when creating or updating a plugin policy. It keeps the protobuf encoded billing recipe as a string which is used to verify a signature.
type PluginPolicy struct {
	ID            uuid.UUID       `json:"id" validate:"required"`
	PublicKey     string          `json:"public_key" validate:"required"`
	PluginID      PluginID        `json:"plugin_id" validate:"required"`
	PluginVersion string          `json:"plugin_version" validate:"required"`
	PolicyVersion int             `json:"policy_version" validate:"required"`
	Signature     string          `json:"signature" validate:"required"`
	Recipe        string          `json:"recipe" validate:"required"`  // base64 encoded recipe protobuf bytes
	Billing       []BillingPolicy `json:"billing" validate:"required"` // This will be populated later
	Active        bool            `json:"active" validate:"required"`
}

func (p *PluginPolicy) GetRecipe() (*rtypes.Policy, error) {
	if p.Recipe == "" {
		return nil, fmt.Errorf("recipe is empty")
	}

	var recipe rtypes.Policy
	policyBytes, err := base64.StdEncoding.DecodeString(p.Recipe)
	if err != nil {
		return nil, fmt.Errorf("failed to decode policy recipe: %w", err)
	}

	if err := proto.Unmarshal(policyBytes, &recipe); err != nil {
		return nil, fmt.Errorf("failed to unmarshal recipe: %w", err)
	}

	return &recipe, nil
}
