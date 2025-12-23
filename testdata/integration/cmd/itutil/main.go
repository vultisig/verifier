package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	recipetypes "github.com/vultisig/recipes/types"
	"github.com/vultisig/verifier/config"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

// itutil - Integration Test Utility
// Unified CLI tool for integration test helpers

// generatePermissivePolicy creates a permissive policy that allows ETH transfers,
// ERC20 transfers, and ERC20 approvals to any target address.
func generatePermissivePolicy(targetAddr string) (string, error) {
	policy := &recipetypes.Policy{
		Id:          "permissive-test-policy",
		Name:        "Permissive Test Policy",
		Description: "Allows all transactions for testing",
		Version:     1,
		Author:      "integration-test",
		Rules: []*recipetypes.Rule{
			{
				Id:          "allow-ethereum-eth-transfer",
				Resource:    "ethereum.eth.transfer",
				Effect:      recipetypes.Effect_EFFECT_ALLOW,
				Description: "Allow Ethereum transfers",
				Target: &recipetypes.Target{
					TargetType: recipetypes.TargetType_TARGET_TYPE_ADDRESS,
					Target: &recipetypes.Target_Address{
						Address: targetAddr,
					},
				},
				ParameterConstraints: []*recipetypes.ParameterConstraint{
					{
						ParameterName: "amount",
						Constraint: &recipetypes.Constraint{
							Type: recipetypes.ConstraintType_CONSTRAINT_TYPE_ANY,
						},
					},
				},
			},
			{
				Id:          "allow-ethereum-erc20-transfer",
				Resource:    "ethereum.erc20.transfer",
				Effect:      recipetypes.Effect_EFFECT_ALLOW,
				Description: "Allow ERC20 transfers",
				Target: &recipetypes.Target{
					TargetType: recipetypes.TargetType_TARGET_TYPE_ADDRESS,
					Target: &recipetypes.Target_Address{
						Address: targetAddr,
					},
				},
			},
			{
				Id:          "allow-ethereum-erc20-approve",
				Resource:    "ethereum.erc20.approve",
				Effect:      recipetypes.Effect_EFFECT_ALLOW,
				Description: "Allow ERC20 approvals",
				Target: &recipetypes.Target{
					TargetType: recipetypes.TargetType_TARGET_TYPE_ADDRESS,
					Target: &recipetypes.Target_Address{
						Address: targetAddr,
					},
				},
			},
		},
	}

	data, err := proto.Marshal(policy)
	if err != nil {
		return "", fmt.Errorf("failed to marshal policy: %w", err)
	}

	return base64.StdEncoding.EncodeToString(data), nil
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "itutil",
		Short: "Integration test utility for Vultisig Verifier",
		Long:  "A unified CLI tool providing helpers for integration testing",
	}

	rootCmd.AddCommand(jwtCmd())
	rootCmd.AddCommand(evmFixtureCmd())
	rootCmd.AddCommand(policyB64Cmd())
	rootCmd.AddCommand(seedDBCmd())
	rootCmd.AddCommand(seedVaultCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// JWT Token Command

type Claims struct {
	PublicKey string `json:"public_key"`
	TokenID   string `json:"token_id"`
	jwt.RegisteredClaims
}

func jwtCmd() *cobra.Command {
	var secret, pubkey, tokenID string
	var expireHours int

	cmd := &cobra.Command{
		Use:   "jwt",
		Short: "Generate a JWT token for policy endpoints",
		RunE: func(cmd *cobra.Command, args []string) error {
			if secret == "" || pubkey == "" {
				return fmt.Errorf("--secret and --pubkey are required")
			}

			expirationTime := time.Now().Add(time.Duration(expireHours) * time.Hour)
			claims := &Claims{
				PublicKey: pubkey,
				TokenID:   tokenID,
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(expirationTime),
					IssuedAt:  jwt.NewNumericDate(time.Now()),
				},
			}

			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			tokenString, err := token.SignedString([]byte(secret))
			if err != nil {
				return fmt.Errorf("failed to create token: %w", err)
			}

			fmt.Println(tokenString)
			return nil
		},
	}

	cmd.Flags().StringVar(&secret, "secret", "", "JWT signing secret")
	cmd.Flags().StringVar(&pubkey, "pubkey", "", "Vault public key")
	cmd.Flags().StringVar(&tokenID, "token-id", "integration-token-1", "Token ID")
	cmd.Flags().IntVar(&expireHours, "expire-hours", 24, "Token expiration in hours")

	return cmd
}

// EVM Fixture Command

// DynamicFeeTxWithoutSignature mirrors the recipes package structure
type DynamicFeeTxWithoutSignature struct {
	ChainID    *big.Int
	Nonce      uint64
	GasTipCap  *big.Int
	GasFeeCap  *big.Int
	Gas        uint64
	To         *common.Address `rlp:"nil"`
	Value      *big.Int
	Data       []byte
	AccessList ethtypes.AccessList
}

func evmFixtureCmd() *cobra.Command {
	var chainID int64
	var to string
	var valueWei string
	var gas uint64
	var nonce uint64
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "evm-fixture",
		Short: "Generate EVM transaction and message fixtures",
		RunE: func(cmd *cobra.Command, args []string) error {
			chainIDBig := big.NewInt(chainID)
			toAddr := common.HexToAddress(to)

			value := new(big.Int)
			if valueWei != "" {
				if _, ok := value.SetString(valueWei, 10); !ok {
					return fmt.Errorf("invalid value: %q is not a valid base-10 integer", valueWei)
				}
			} else {
				value.SetInt64(1000000000000000) // 0.001 ETH default
			}

			// Create a DynamicFee (EIP-1559) transaction
			tx := ethtypes.NewTx(&ethtypes.DynamicFeeTx{
				ChainID:   chainIDBig,
				Nonce:     nonce,
				GasTipCap: big.NewInt(1000000000),  // 1 gwei
				GasFeeCap: big.NewInt(20000000000), // 20 gwei
				Gas:       gas,
				To:        &toAddr,
				Value:     value,
				Data:      nil,
			})

			// Manually encode the unsigned transaction without signature fields
			unsignedTx := DynamicFeeTxWithoutSignature{
				ChainID:    chainIDBig,
				Nonce:      nonce,
				GasTipCap:  big.NewInt(1000000000),
				GasFeeCap:  big.NewInt(20000000000),
				Gas:        gas,
				To:         &toAddr,
				Value:      value,
				Data:       nil,
				AccessList: ethtypes.AccessList{},
			}

			// RLP encode the unsigned transaction
			txBytes, err := rlp.EncodeToBytes(unsignedTx)
			if err != nil {
				return fmt.Errorf("failed to RLP encode: %w", err)
			}

			// Prepend the transaction type byte (2 for DynamicFeeTx)
			typedTxBytes := append([]byte{byte(ethtypes.DynamicFeeTxType)}, txBytes...)
			txB64 := base64.StdEncoding.EncodeToString(typedTxBytes)

			// Hash-to-sign must match what verifier computes from tx bytes
			signer := ethtypes.LatestSignerForChainID(chainIDBig)
			hash := signer.Hash(tx)
			msgBytes := hash.Bytes()
			msgB64 := base64.StdEncoding.EncodeToString(msgBytes)

			// Compute SHA256 of the message bytes for the hash field
			msgSha256 := sha256.Sum256(msgBytes)
			msgSha256B64 := base64.StdEncoding.EncodeToString(msgSha256[:])

			switch outputFormat {
			case "shell":
				fmt.Printf("TX_B64=%s\n", txB64)
				fmt.Printf("MSG_B64=%s\n", msgB64)
				fmt.Printf("MSG_SHA256_B64=%s\n", msgSha256B64)
			case "json":
				fmt.Printf(`{"tx_b64":"%s","msg_b64":"%s","msg_sha256_b64":"%s"}`+"\n", txB64, msgB64, msgSha256B64)
			default:
				fmt.Println(txB64)
				fmt.Println(msgB64)
				fmt.Println(msgSha256B64)
			}

			return nil
		},
	}

	cmd.Flags().Int64Var(&chainID, "chain-id", 1, "Chain ID (1=mainnet)")
	cmd.Flags().StringVar(&to, "to", "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0", "Recipient address")
	cmd.Flags().StringVar(&valueWei, "value", "", "Value in wei (default: 0.001 ETH)")
	cmd.Flags().Uint64Var(&gas, "gas", 21000, "Gas limit")
	cmd.Flags().Uint64Var(&nonce, "nonce", 0, "Transaction nonce")
	cmd.Flags().StringVar(&outputFormat, "output", "shell", "Output format: shell, json, or plain")

	return cmd
}

// Policy B64 Command

func policyB64Cmd() *cobra.Command {
	var allowAll bool
	var targetAddr string

	cmd := &cobra.Command{
		Use:   "policy-b64",
		Short: "Generate base64-encoded policy protobuf",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !allowAll {
				return fmt.Errorf("only --allow-all is currently supported")
			}

			policyB64, err := generatePermissivePolicy(targetAddr)
			if err != nil {
				return err
			}

			fmt.Println(policyB64)
			return nil
		},
	}

	cmd.Flags().BoolVar(&allowAll, "allow-all", false, "Generate a permissive policy that allows all")
	cmd.Flags().StringVar(&targetAddr, "target", "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0", "Target address for rules")

	return cmd
}

// Seed DB Command

type ProposedYAML struct {
	Plugins []PluginConfig `yaml:"plugins"`
}

type PluginConfig struct {
	ID             string `yaml:"id"`
	Title          string `yaml:"title"`
	Description    string `yaml:"description"`
	ServerEndpoint string `yaml:"server_endpoint"`
	Category       string `yaml:"category"`
	LogoURL        string `yaml:"logo_url"`
	ThumbnailURL   string `yaml:"thumbnail_url"`
	Audited        bool   `yaml:"audited"`
}

func seedDBCmd() *cobra.Command {
	var proposedFile string
	var fixtureFile string
	var dsn string

	cmd := &cobra.Command{
		Use:   "seed-db",
		Short: "Seed the database with plugins from proposed.yaml and vault token from fixture",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			// Determine DSN
			if dsn == "" {
				cfg, err := config.ReadVerifierConfig()
				if err != nil {
					return fmt.Errorf("failed to read config (set --dsn or VERIFIER_DSN): %w", err)
				}
				dsn = cfg.Database.DSN
			}

			// Read proposed.yaml
			proposedYAML, err := os.ReadFile(proposedFile)
			if err != nil {
				return fmt.Errorf("failed to read %s: %w", proposedFile, err)
			}

			var proposed ProposedYAML
			if err := yaml.Unmarshal(proposedYAML, &proposed); err != nil {
				return fmt.Errorf("failed to parse %s: %w", proposedFile, err)
			}

			// Read fixture.json to get vault public key
			var vaultPubkey string
			if fixtureFile != "" {
				fixtureData, err := os.ReadFile(fixtureFile)
				if err != nil {
					return fmt.Errorf("failed to read fixture file: %w", err)
				}
				var fixture FixtureJSON
				if err := json.Unmarshal(fixtureData, &fixture); err != nil {
					return fmt.Errorf("failed to parse fixture JSON: %w", err)
				}
				vaultPubkey = fixture.Vault.PublicKey
			}

			pool, err := pgxpool.New(ctx, dsn)
			if err != nil {
				return fmt.Errorf("failed to connect to database: %w", err)
			}
			defer pool.Close()

			tx, err := pool.Begin(ctx)
			if err != nil {
				return fmt.Errorf("failed to begin transaction: %w", err)
			}

			defer func() {
				if r := recover(); r != nil {
					tx.Rollback(ctx)
					panic(r)
				}
			}()

			log.Println("🌱 Seeding integration database with plugins from", proposedFile)

			for _, plugin := range proposed.Plugins {
				log.Printf("  📦 Inserting plugin: %s...\n", plugin.ID)

				_, err := tx.Exec(ctx, `
					INSERT INTO plugins (id, title, description, server_endpoint, category, logo_url, thumbnail_url, audited)
					VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
					ON CONFLICT (id) DO UPDATE SET
						title = EXCLUDED.title,
						description = EXCLUDED.description,
						server_endpoint = EXCLUDED.server_endpoint,
						category = EXCLUDED.category,
						logo_url = EXCLUDED.logo_url,
						thumbnail_url = EXCLUDED.thumbnail_url,
						audited = EXCLUDED.audited,
						updated_at = NOW()
				`, plugin.ID, plugin.Title, plugin.Description, plugin.ServerEndpoint,
					plugin.Category, plugin.LogoURL, plugin.ThumbnailURL, plugin.Audited)

				if err != nil {
					tx.Rollback(ctx)
					return fmt.Errorf("failed to insert plugin %s: %w", plugin.ID, err)
				}

				// Insert API key for the plugin
				apiKey := fmt.Sprintf("integration-test-apikey-%s", plugin.ID)
				_, err = tx.Exec(ctx, `
					INSERT INTO plugin_apikey (id, plugin_id, apikey, created_at, expires_at, status)
					VALUES (gen_random_uuid(), $1, $2, NOW(), NULL, 1)
					ON CONFLICT DO NOTHING
				`, plugin.ID, apiKey)

				if err != nil {
					tx.Rollback(ctx)
					return fmt.Errorf("failed to insert API key for plugin %s: %w", plugin.ID, err)
				}

				log.Printf("  ✅ Plugin %s seeded (API Key: %s)\n", plugin.ID, apiKey)
			}

			// Insert vault token for JWT authentication (if fixture was provided)
			if vaultPubkey != "" {
				tokenID := "integration-token-1" // Must match jwt command default
				now := time.Now()
				expiresAt := now.Add(365 * 24 * time.Hour) // 1 year expiry

				log.Printf("  🔑 Inserting vault token for pubkey: %s...\n", vaultPubkey[:16]+"...")

				_, err = tx.Exec(ctx, `
					INSERT INTO vault_tokens (token_id, public_key, expires_at, last_used_at)
					VALUES ($1, $2, $3, $4)
					ON CONFLICT (token_id) DO UPDATE SET
						public_key = EXCLUDED.public_key,
						expires_at = EXCLUDED.expires_at,
						last_used_at = EXCLUDED.last_used_at,
						revoked_at = NULL,
						updated_at = NOW()
				`, tokenID, vaultPubkey, expiresAt, now)

				if err != nil {
					tx.Rollback(ctx)
					return fmt.Errorf("failed to insert vault token: %w", err)
				}

				log.Printf("  ✅ Vault token seeded (token_id: %s)\n", tokenID)

				// Insert test policies for each plugin (for plugin-signer tests)
				log.Println("  📋 Inserting test policies...")

				// Target address for policy rules (allows transfers to this address)
				targetAddr := "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0"

				// Generate permissive policy dynamically
				permissiveRecipe, err := generatePermissivePolicy(targetAddr)
				if err != nil {
					tx.Rollback(ctx)
					return fmt.Errorf("failed to generate permissive policy: %w", err)
				}

				for i, plugin := range proposed.Plugins {
					// Generate deterministic policy ID based on plugin index
					policyID := fmt.Sprintf("00000000-0000-0000-0000-0000000000%02d", i+11)

					// Insert a permissive test policy
					_, err = tx.Exec(ctx, `
						INSERT INTO plugin_policies (id, public_key, plugin_id, plugin_version, policy_version, signature, recipe, active)
						VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
						ON CONFLICT (id) DO UPDATE SET
							public_key = EXCLUDED.public_key,
							plugin_id = EXCLUDED.plugin_id,
							plugin_version = EXCLUDED.plugin_version,
							policy_version = EXCLUDED.policy_version,
							signature = EXCLUDED.signature,
							recipe = EXCLUDED.recipe,
							active = EXCLUDED.active,
							updated_at = NOW()
					`, policyID, vaultPubkey, plugin.ID, "1.0.0", 1,
						"0x0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
						permissiveRecipe,
						true)

					if err != nil {
						tx.Rollback(ctx)
						return fmt.Errorf("failed to insert policy for plugin %s: %w", plugin.ID, err)
					}

					log.Printf("  ✅ Policy %s seeded for plugin %s\n", policyID, plugin.ID)
				}
			}

			if err := tx.Commit(ctx); err != nil {
				return fmt.Errorf("failed to commit transaction: %w", err)
			}

			log.Println("✅ Integration database seeding completed!")
			log.Printf("   Total plugins seeded: %d\n", len(proposed.Plugins))
			return nil
		},
	}

	cmd.Flags().StringVar(&proposedFile, "proposed", "proposed.yaml", "Path to proposed.yaml")
	cmd.Flags().StringVar(&fixtureFile, "fixture", "testdata/integration/fixture.json", "Path to fixture.json (for vault token)")
	cmd.Flags().StringVar(&dsn, "dsn", "", "Database DSN (defaults to config)")

	return cmd
}

// Seed Vault Command

type FixtureJSON struct {
	Vault struct {
		PublicKey string `json:"public_key"`
		VaultB64  string `json:"vault_b64"`
	} `json:"vault"`
}

func seedVaultCmd() *cobra.Command {
	var fixtureFile string
	var proposedFile string
	var s3Endpoint string
	var s3Region string
	var s3AccessKey string
	var s3SecretKey string
	var s3Bucket string

	cmd := &cobra.Command{
		Use:   "seed-vault",
		Short: "Seed vault fixtures to S3/MinIO for each plugin",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Read fixture
			fixtureData, err := os.ReadFile(fixtureFile)
			if err != nil {
				return fmt.Errorf("failed to read fixture file: %w", err)
			}

			var fixture FixtureJSON
			if err := json.Unmarshal(fixtureData, &fixture); err != nil {
				return fmt.Errorf("failed to parse fixture JSON: %w", err)
			}

			// Decode vault data
			vaultData, err := base64.StdEncoding.DecodeString(fixture.Vault.VaultB64)
			if err != nil {
				return fmt.Errorf("failed to decode vault_b64: %w", err)
			}

			// Read proposed.yaml to get plugin IDs
			proposedData, err := os.ReadFile(proposedFile)
			if err != nil {
				return fmt.Errorf("failed to read proposed.yaml: %w", err)
			}

			var proposed ProposedYAML
			if err := yaml.Unmarshal(proposedData, &proposed); err != nil {
				return fmt.Errorf("failed to parse proposed.yaml: %w", err)
			}

			// Create S3 client
			sess, err := session.NewSession(&aws.Config{
				Endpoint:         aws.String(s3Endpoint),
				Region:           aws.String(s3Region),
				Credentials:      credentials.NewStaticCredentials(s3AccessKey, s3SecretKey, ""),
				S3ForcePathStyle: aws.Bool(true),
			})
			if err != nil {
				return fmt.Errorf("failed to create S3 session: %w", err)
			}

			s3Client := s3.New(sess)

			log.Println("🗄️  Seeding vault fixtures to S3/MinIO...")

			// Upload vault for each plugin, tracking failures
			var failedUploads []string
			for _, plugin := range proposed.Plugins {
				key := fmt.Sprintf("%s-%s.vult", plugin.ID, fixture.Vault.PublicKey)

				_, err := s3Client.PutObject(&s3.PutObjectInput{
					Bucket:      aws.String(s3Bucket),
					Key:         aws.String(key),
					Body:        bytes.NewReader(vaultData),
					ContentType: aws.String("application/octet-stream"),
				})

				if err != nil {
					log.Printf("  ❌ Failed to upload %s: %v\n", key, err)
					failedUploads = append(failedUploads, key)
					continue
				}

				log.Printf("  ✅ Uploaded: %s\n", key)
			}

			if len(failedUploads) > 0 {
				return fmt.Errorf("failed to upload %d vault(s): %v", len(failedUploads), failedUploads)
			}

			log.Println("✅ Vault seeding completed!")
			return nil
		},
	}

	cmd.Flags().StringVar(&fixtureFile, "fixture", "testdata/integration/fixture.json", "Path to fixture.json")
	cmd.Flags().StringVar(&proposedFile, "proposed", "proposed.yaml", "Path to proposed.yaml")
	cmd.Flags().StringVar(&s3Endpoint, "s3-endpoint", "http://localhost:9000", "S3/MinIO endpoint")
	cmd.Flags().StringVar(&s3Region, "s3-region", "us-east-1", "S3 region")
	cmd.Flags().StringVar(&s3AccessKey, "s3-access-key", "minioadmin", "S3 access key")
	cmd.Flags().StringVar(&s3SecretKey, "s3-secret-key", "minioadmin", "S3 secret key")
	cmd.Flags().StringVar(&s3Bucket, "s3-bucket", "vultisig-verifier", "S3 bucket name")

	return cmd
}
