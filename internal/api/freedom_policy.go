package api

import (
	"math/big"

	rtypes "github.com/vultisig/recipes/types"
)

// FreedomPluginID is the synthetic plugin identifier used for freedom signing.
// Freedom vaults are stored under this key on VultiServer.
const FreedomPluginID = "vultisig.freedom"

// usdcAddress is the canonical USDC contract address on Ethereum mainnet.
const usdcAddress = "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"

// uint256Max is 2^256 - 1.
var uint256Max, _ = new(big.Int).SetString(
	"115792089237316195423570985008687907853269984665640564039457584007913129639935", 10,
)

// FreedomPolicy is the static co-signer policy for the freedom signing surface.
//
// Design intent:
//   - DENY rules fire first (deny-wins semantics in the recipes engine).
//   - ALLOW rules define the permitted set within those limits.
//   - Every ALLOW rule that names a specific contract targets that contract via
//     TARGET_TYPE_ADDRESS; native ETH transfers use TARGET_TYPE_UNSPECIFIED
//     (any recipient, amount-capped).
var FreedomPolicy = &rtypes.Policy{
	Id:   "vultisig.freedom.v1",
	Name: "Freedom Policy v1",
	Rules: []*rtypes.Rule{
		// --- DENY rules (evaluated first) ---

		// Reject unbounded ERC-20 USDC approvals (amount == uint256.max).
		// A legitimate DApp should never need unlimited approval; this blocks
		// a class of approval-draining attacks.
		{
			Id:       "deny-usdc-approve-unbounded",
			Resource: "ethereum.erc20.approve",
			Effect:   rtypes.Effect_EFFECT_DENY,
			Target: &rtypes.Target{
				TargetType: rtypes.TargetType_TARGET_TYPE_ADDRESS,
				Target:     &rtypes.Target_Address{Address: usdcAddress},
			},
			ParameterConstraints: []*rtypes.ParameterConstraint{
				{
					ParameterName: "spender",
					Constraint:    &rtypes.Constraint{Type: rtypes.ConstraintType_CONSTRAINT_TYPE_ANY},
				},
				{
					ParameterName: "amount",
					Constraint: &rtypes.Constraint{
						Type:  rtypes.ConstraintType_CONSTRAINT_TYPE_FIXED,
						Value: &rtypes.Constraint_FixedValue{FixedValue: uint256Max.String()},
					},
				},
			},
		},

		// --- ALLOW rules ---

		// Allow native ETH transfers up to 0.01 ETH (1e16 wei) to any recipient.
		{
			Id:       "allow-eth-transfer",
			Resource: "ethereum.eth.transfer",
			Effect:   rtypes.Effect_EFFECT_ALLOW,
			Target: &rtypes.Target{
				// TARGET_TYPE_UNSPECIFIED = no recipient restriction.
				TargetType: rtypes.TargetType_TARGET_TYPE_UNSPECIFIED,
			},
			ParameterConstraints: []*rtypes.ParameterConstraint{
				{
					ParameterName: "amount",
					Constraint: &rtypes.Constraint{
						Type:  rtypes.ConstraintType_CONSTRAINT_TYPE_MAX,
						Value: &rtypes.Constraint_MaxValue{MaxValue: "10000000000000000"}, // 0.01 ETH
					},
				},
			},
		},

		// Allow USDC ERC-20 transfers up to $10 (1e7, 6 decimals) to any recipient.
		{
			Id:       "allow-usdc-transfer",
			Resource: "ethereum.erc20.transfer",
			Effect:   rtypes.Effect_EFFECT_ALLOW,
			Target: &rtypes.Target{
				TargetType: rtypes.TargetType_TARGET_TYPE_ADDRESS,
				Target:     &rtypes.Target_Address{Address: usdcAddress},
			},
			ParameterConstraints: []*rtypes.ParameterConstraint{
				{
					ParameterName: "recipient",
					Constraint:    &rtypes.Constraint{Type: rtypes.ConstraintType_CONSTRAINT_TYPE_ANY},
				},
				{
					ParameterName: "amount",
					Constraint: &rtypes.Constraint{
						Type:  rtypes.ConstraintType_CONSTRAINT_TYPE_MAX,
						Value: &rtypes.Constraint_MaxValue{MaxValue: "10000000"}, // $10 USDC
					},
				},
			},
		},

		// Allow USDC ERC-20 approvals up to $10 (10_000_000, 6 decimals) — matching
		// the transfer cap exactly. The freedom use case (approve + aave supply bundle)
		// requires a bounded approve equal to the deposit amount, so a $10 ceiling
		// covers every legitimate flow while bounding the blast radius to $10.
		//
		// IMPORTANT: the previous ceiling was uint256.max-1, which let a compromised
		// agent issue approve(drainer, ~∞) and then transferFrom the entire balance.
		// The DENY-uint256.max rule is intentionally kept as a belt-and-suspenders
		// backstop, but the primary protection is now this tight MaxValue cap.
		{
			Id:       "allow-usdc-approve",
			Resource: "ethereum.erc20.approve",
			Effect:   rtypes.Effect_EFFECT_ALLOW,
			Target: &rtypes.Target{
				TargetType: rtypes.TargetType_TARGET_TYPE_ADDRESS,
				Target:     &rtypes.Target_Address{Address: usdcAddress},
			},
			ParameterConstraints: []*rtypes.ParameterConstraint{
				{
					ParameterName: "spender",
					Constraint:    &rtypes.Constraint{Type: rtypes.ConstraintType_CONSTRAINT_TYPE_ANY},
				},
				{
					ParameterName: "amount",
					Constraint: &rtypes.Constraint{
						Type:  rtypes.ConstraintType_CONSTRAINT_TYPE_MAX,
						Value: &rtypes.Constraint_MaxValue{MaxValue: "10000000"}, // $10 USDC (6 decimals)
					},
				},
			},
		},
	},
}
