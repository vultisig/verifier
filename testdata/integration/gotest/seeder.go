package gotest

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/jackc/pgx/v5/pgxpool"
	vaultType "github.com/vultisig/commondata/go/vultisig/vault/v1"
	recipetypes "github.com/vultisig/recipes/types"
	"github.com/vultisig/vultisig-go/common"
	"google.golang.org/protobuf/proto"
)

type S3Config struct {
	Endpoint  string
	Region    string
	AccessKey string
	SecretKey string
	Bucket    string
}

type SeederConfig struct {
	DSN              string
	S3               S3Config
	Fixture          *FixtureData
	Plugins          []PluginConfig
	EncryptionSecret string
}

type Seeder struct {
	config SeederConfig
}

func NewSeeder(cfg SeederConfig) *Seeder {
	return &Seeder{config: cfg}
}

func (s *Seeder) SeedDatabase(ctx context.Context) error {
	pool, err := pgxpool.New(ctx, s.config.DSN)
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

	log.Println("Seeding integration database...")

	for _, plugin := range s.config.Plugins {
		log.Printf("  Inserting plugin: %s...\n", plugin.ID)

		_, err := tx.Exec(ctx, `
			INSERT INTO plugins (id, title, description, server_endpoint, category, audited)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (id) DO UPDATE SET
				title = EXCLUDED.title,
				description = EXCLUDED.description,
				server_endpoint = EXCLUDED.server_endpoint,
				category = EXCLUDED.category,
				audited = EXCLUDED.audited,
				updated_at = NOW()
		`, plugin.ID, plugin.Title, plugin.Description, plugin.ServerEndpoint,
			plugin.Category, plugin.Audited)
		if err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("failed to insert plugin %s: %w", plugin.ID, err)
		}

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

		log.Printf("  Plugin %s seeded\n", plugin.ID)
	}

	vaultPubkey := s.config.Fixture.Vault.PublicKey
	if vaultPubkey != "" {
		tokenID := "integration-token-1"
		now := time.Now()
		expiresAt := now.Add(365 * 24 * time.Hour)

		log.Printf("  Inserting vault token...\n")

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

		log.Println("  Inserting test policies...")

		targetAddr := "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0"
		permissiveRecipe, err := generatePermissivePolicy(targetAddr)
		if err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("failed to generate permissive policy: %w", err)
		}

		for i, plugin := range s.config.Plugins {
			policyID := fmt.Sprintf("00000000-0000-0000-0000-0000000000%02d", i+11)

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

			log.Printf("  Policy %s seeded for plugin %s\n", policyID, plugin.ID)
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Println("Database seeding completed!")
	return nil
}

func (s *Seeder) SeedVaults(ctx context.Context) error {
	vaultData, err := base64.StdEncoding.DecodeString(s.config.Fixture.Vault.VaultB64)
	if err != nil {
		return fmt.Errorf("failed to decode vault_b64: %w", err)
	}

	encryptedVault, err := common.EncryptVault(s.config.EncryptionSecret, vaultData)
	if err != nil {
		return fmt.Errorf("failed to encrypt vault: %w", err)
	}

	vaultContainer := &vaultType.VaultContainer{
		Version:     1,
		Vault:       base64.StdEncoding.EncodeToString(encryptedVault),
		IsEncrypted: true,
	}

	containerBytes, err := proto.Marshal(vaultContainer)
	if err != nil {
		return fmt.Errorf("failed to marshal vault container: %w", err)
	}

	vaultBackup := []byte(base64.StdEncoding.EncodeToString(containerBytes))

	sess, err := session.NewSession(&aws.Config{
		Endpoint:         aws.String(s.config.S3.Endpoint),
		Region:           aws.String(s.config.S3.Region),
		Credentials:      credentials.NewStaticCredentials(s.config.S3.AccessKey, s.config.S3.SecretKey, ""),
		S3ForcePathStyle: aws.Bool(true),
	})
	if err != nil {
		return fmt.Errorf("failed to create S3 session: %w", err)
	}

	s3Client := s3.New(sess)

	log.Println("Seeding vault fixtures to S3/MinIO...")

	for _, plugin := range s.config.Plugins {
		key := fmt.Sprintf("%s-%s.vult", plugin.ID, s.config.Fixture.Vault.PublicKey)

		_, err := s3Client.PutObject(&s3.PutObjectInput{
			Bucket:      aws.String(s.config.S3.Bucket),
			Key:         aws.String(key),
			Body:        bytes.NewReader(vaultBackup),
			ContentType: aws.String("application/octet-stream"),
		})
		if err != nil {
			return fmt.Errorf("failed to upload %s: %w", key, err)
		}

		log.Printf("  Uploaded: %s\n", key)
	}

	billingKey := fmt.Sprintf("vultisig-fees-feee-%s.vult", s.config.Fixture.Vault.PublicKey)
	_, err = s3Client.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(s.config.S3.Bucket),
		Key:         aws.String(billingKey),
		Body:        bytes.NewReader(vaultBackup),
		ContentType: aws.String("application/octet-stream"),
	})
	if err != nil {
		return fmt.Errorf("failed to upload billing vault %s: %w", billingKey, err)
	}

	log.Printf("  Uploaded billing vault: %s\n", billingKey)
	log.Println("Vault seeding completed!")
	return nil
}

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
