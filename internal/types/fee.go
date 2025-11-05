package types

import (
	"github.com/vultisig/recipes/types"
)

type CreditMetadata struct {
	DebitFeeID      uint64 `json:"debit_fee_id"`     // ID of the debit transaction
	TransactionHash string `json:"transaction_hash"` // Transaction hash in blockchain
	Network         string `json:"network"`          // Blockchain network (e.g., "ethereum", "polygon")
}

// TODO: Temporary solution for testing purposes.
// This will be replaced by integrating the fee policy into every relevant policy.
var FeeDefaultPolicy = &types.Policy{
	Rules: []*types.Rule{
		{
			Resource: "ethereum.send",
			Effect:   types.Effect_EFFECT_ALLOW,
			ParameterConstraints: []*types.ParameterConstraint{
				{
					ParameterName: "asset",
					Constraint: &types.Constraint{
						Type: types.ConstraintType_CONSTRAINT_TYPE_FIXED,
						Value: &types.Constraint_FixedValue{
							FixedValue: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
						},
						Required: false,
					},
				},
				{
					ParameterName: "from_address",
					Constraint: &types.Constraint{
						Type:     types.ConstraintType_CONSTRAINT_TYPE_ANY,
						Required: true,
					},
				},
				{
					ParameterName: "amount",
					Constraint: &types.Constraint{
						Type:     types.ConstraintType_CONSTRAINT_TYPE_ANY,
						Required: true,
					},
				},
				{
					ParameterName: "to_address",
					Constraint: &types.Constraint{
						Type: types.ConstraintType_CONSTRAINT_TYPE_MAGIC_CONSTANT,
						Value: &types.Constraint_MagicConstantValue{
							MagicConstantValue: types.MagicConstant_VULTISIG_TREASURY,
						},
						Required: true,
					},
				},
			},
		},
	},
}
