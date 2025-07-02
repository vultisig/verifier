package types

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
	rtypes "github.com/vultisig/recipes/types"
	"google.golang.org/protobuf/proto"
)

type BILLING_TYPE string

const (
	BILLING_TYPE_TX        BILLING_TYPE = "tx"
	BILLING_TYPE_RECURRING BILLING_TYPE = "recurring"
	BILLING_TYPE_ONCE      BILLING_TYPE = "once"
)

type Fee struct {
	ID                    uuid.UUID  `json:"id"`                       // The unique id of the fee incurred
	PublicKey             string     `json:"public_key"`               // The public key "account" connected to the fee
	PluginID              PluginID   `json:"plugin_id"`                // The plugin ID that has incurred the fee
	PolicyID              uuid.UUID  `json:"policy_id"`                // The policy ID that has incurred the fee
	PluginPolicyBillingID uuid.UUID  `json:"plugin_policy_billing_id"` // The plugin policy billing ID that has incurred the fee. This is because a plugin policy may have several billing "rules" associated with it.
	TransactionID         uuid.UUID  `json:"transaction_id"`           // The transaction ID that has incurred the fee
	Amount                int        `json:"amount"`                   // The amount of the fee in the smallest unit, e.g., "1000000" for 0.01 VULTI
	Type                  string     `json:"type"`                     // "tx", "recurring" or "once". Only availble on the fees_view table
	CreatedAt             time.Time  `json:"created_at"`
	ChargedAt             time.Time  `json:"charged_at"`
	CollectedAt           *time.Time `json:"collected_at"`
}

// TODO update internal type references
type BillingPolicy struct {
	ID        uuid.UUID `json:"id" validate:"required"`
	Type      string    `json:"type" validate:"required"`   // "tx", "recurring" or "once"
	Frequency string    `json:"frequency"`                  // only "monthly" for now
	StartDate time.Time `json:"start_date"`                 // Number of a month, e.g., "1" for the first month. Only allow 1 for now
	Amount    int       `json:"amount" validate:"required"` // Amount in the smallest unit, e.g., "1000000" for 0.01 VULTI
}

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

func (p *PluginPolicy) PopulateBilling() error {

	p.Billing = []BillingPolicy{}

	recipe, err := p.GetRecipe()
	if err != nil {
		return fmt.Errorf("failed to get recipe: %w", err)
	}

	for _, feePolicy := range recipe.FeePolicies {
		if feePolicy.Id == "" {
			feePolicy.Id = uuid.New().String()
		}
		id, err := uuid.Parse(feePolicy.Id)
		if err != nil {
			return fmt.Errorf("failed to parse fee policy ID: %w", err)
		}
		p.Billing = append(p.Billing, BillingPolicy{
			ID:        id,
			Type:      string(feePolicy.Type),
			Frequency: string(feePolicy.Frequency),
			StartDate: feePolicy.StartDate.AsTime(),
			Amount:    int(feePolicy.Amount),
		})
	}
	return nil
}
