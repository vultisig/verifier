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

// This is used to populate the Billing field of a PluginPolicy from the Recipe field. It does not validate this information against the plugin pricing.
func (p *PluginPolicy) ParseBillingFromRecipe() error {

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

		var feeType PricingType
		switch feePolicy.Type {
		case rtypes.FeeType_FEE_TYPE_UNSPECIFIED:
			return fmt.Errorf("fee type is unspecified")
		case rtypes.FeeType_ONCE:
			feeType = PricingTypeOnce
		case rtypes.FeeType_TRANSACTION:
			feeType = PricingTypePerTx
		case rtypes.FeeType_RECURRING:
			feeType = PricingTypeRecurring
		default:
			return fmt.Errorf("invalid fee type: %v", feePolicy.Type)
		}

		var pricingFrequency *PricingFrequency
		if feeType == PricingTypeRecurring {
			switch feePolicy.Frequency {
			case rtypes.BillingFrequency_BILLING_FREQUENCY_UNSPECIFIED:
				return fmt.Errorf("invalid frequency: %v", feePolicy.Frequency)
			case rtypes.BillingFrequency_DAILY:
				_freq := PricingFrequencyDaily
				pricingFrequency = &_freq
			case rtypes.BillingFrequency_WEEKLY:
				_freq := PricingFrequencyWeekly
				pricingFrequency = &_freq
			case rtypes.BillingFrequency_BIWEEKLY:
				_freq := PricingFrequencyBiweekly
				pricingFrequency = &_freq
			case rtypes.BillingFrequency_MONTHLY:
				_freq := PricingFrequencyMonthly
				pricingFrequency = &_freq
			default:
				return fmt.Errorf("invalid frequency: %v", feePolicy.Frequency)
			}
		}

		p.Billing = append(p.Billing, BillingPolicy{
			ID:        id,
			Type:      feeType,
			Frequency: pricingFrequency,
			StartDate: feePolicy.StartDate.AsTime(),
			Amount:    uint64(feePolicy.Amount),
			Asset:     "usdc", // Multiple currencies not currently supported in fee policy recipes or elsewhere so for now we can hard code it, and later, extract from the protobuf encoded fee policies
		})
	}
	return nil
}
