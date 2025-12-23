package main

import (
	"encoding/base64"
	"fmt"

	"github.com/vultisig/recipes/types"
	"google.golang.org/protobuf/proto"
)

func main() {
	// Create a permissive policy that allows all transactions
	// This policy has rules for common EVM resources
	policy := &types.Policy{
		Id:          "permissive-test-policy",
		Name:        "Permissive Test Policy",
		Description: "Allows all transactions for testing",
		Version:     1,
		Author:      "integration-test",
		Rules: []*types.Rule{
			{
				Id:          "allow-ethereum-eth-transfer",
				Resource:    "ethereum.eth.transfer",
				Effect:      types.Effect_EFFECT_ALLOW,
				Description: "Allow Ethereum transfers",
				Target: &types.Target{
					TargetType: types.TargetType_TARGET_TYPE_ADDRESS,
					Target: &types.Target_Address{
						Address: "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
					},
				},
				ParameterConstraints: []*types.ParameterConstraint{
					{
						ParameterName: "amount",
						Constraint: &types.Constraint{
							Type: types.ConstraintType_CONSTRAINT_TYPE_ANY,
						},
					},
				},
			},
			{
				Id:          "allow-ethereum-erc20-transfer",
				Resource:    "ethereum.erc20.transfer",
				Effect:      types.Effect_EFFECT_ALLOW,
				Description: "Allow ERC20 transfers",
				Target: &types.Target{
					TargetType: types.TargetType_TARGET_TYPE_ADDRESS,
					Target: &types.Target_Address{
						Address: "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
					},
				},
			},
			{
				Id:          "allow-ethereum-erc20-approve",
				Resource:    "ethereum.erc20.approve",
				Effect:      types.Effect_EFFECT_ALLOW,
				Description: "Allow ERC20 approvals",
				Target: &types.Target{
					TargetType: types.TargetType_TARGET_TYPE_ADDRESS,
					Target: &types.Target_Address{
						Address: "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
					},
				},
			},
		},
	}

	// Serialize to protobuf binary
	data, err := proto.Marshal(policy)
	if err != nil {
		panic(err)
	}

	// Encode to base64
	b64 := base64.StdEncoding.EncodeToString(data)
	fmt.Println(b64)
}
