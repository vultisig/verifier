package types

import (
	"time"

	"github.com/google/uuid"
)

type BillingPolicy struct {
	ID        uuid.UUID `json:"id" validate:"required"`
	Type      string    `json:"type" validate:"required"`   // "tx", "recurring" or "once"
	Frequency string    `json:"frequency"`                  // only "monthly" for now
	StartDate time.Time `json:"start_date"`                 // Number of a month, e.g., "1" for the first month. Only allow 1 for now
	Amount    int       `json:"amount" validate:"required"` // Amount in the smallest unit, e.g., "1000000" for 0.01 VULTI
}

// This type should be used externally when creating or updating a plugin policy. It keeps the protobuf encoded billing recipe as a string which is used to verify a signature.
type PluginPolicyCreateUpdate struct {
	ID            uuid.UUID `json:"id" validate:"required"`
	PublicKey     string    `json:"public_key" validate:"required"`
	PluginID      PluginID  `json:"plugin_id" validate:"required"`
	PluginVersion string    `json:"plugin_version" validate:"required"`
	PolicyVersion string    `json:"policy_version" validate:"required"`
	Signature     string    `json:"signature" validate:"required"`
	Recipe        string    `json:"recipe" validate:"required"`         // base64 encoded recipe protobuf bytes
	BillingRecipe string    `json:"billing_recipe" validate:"required"` // base64 encoded billing recipe protobuf bytes
	Active        bool      `json:"active" validate:"required"`
}

func (p *PluginPolicyCreateUpdate) ToPluginPolicy() PluginPolicy {
	return PluginPolicy{
		ID:            p.ID,
		PublicKey:     p.PublicKey,
		PluginID:      p.PluginID,
		PluginVersion: p.PluginVersion,
		PolicyVersion: p.PolicyVersion,
		Signature:     p.Signature,
		Recipe:        p.Recipe,
		Billing:       []BillingPolicy{}, // TODO garry This will be populated later
		Active:        p.Active,
	}
}

// PluginPolicy is our internal typing and return DTO for plugin policies. It shows the previously protobuf encoded billing recipe as a slice of BillingPolicy objects which closer aligns with the DB.
type PluginPolicy struct {
	ID            uuid.UUID       `json:"id" validate:"required"`
	PublicKey     string          `json:"public_key" validate:"required"`
	PluginID      PluginID        `json:"plugin_id" validate:"required"`
	PluginVersion string          `json:"plugin_version" validate:"required"`
	PolicyVersion int             `json:"policy_version" validate:"required"`
	Signature     string          `json:"signature" validate:"required"`
	Recipe        string          `json:"recipe" validate:"required"` // base64 encoded recipe protobuf bytes
	Billing       []BillingPolicy `json:"billing" validate:"required"`
	Active        bool            `json:"active" validate:"required"`
}

func (p *PluginPolicy) ToPluginPolicyCreateUpdate() PluginPolicyCreateUpdate {
	return PluginPolicyCreateUpdate{
		ID:            p.ID,
		PublicKey:     p.PublicKey,
		PluginID:      p.PluginID,
		PluginVersion: p.PluginVersion,
		PolicyVersion: p.PolicyVersion,
		Signature:     p.Signature,
		Recipe:        p.Recipe,
		BillingRecipe: "", // TODO garry This will be populated later
		Active:        p.Active,
	}
}

func (p *PluginPolicy) ToPluginPolicyCreateUpdate() PluginPolicyCreateUpdate {
	return PluginPolicyCreateUpdate{
		ID:            p.ID,
		PublicKey:     p.PublicKey,
		PluginID:      p.PluginID,
		PluginVersion: p.PluginVersion,
		PolicyVersion: p.PolicyVersion,
		Signature:     p.Signature,
		Recipe:        p.Recipe,
		BillingRecipe: "", // TODO garry This will be populated later
		Active:        p.Active,
	}
}
