package storage

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	itypes "github.com/vultisig/verifier/internal/types"
	"github.com/vultisig/verifier/types"
)

type DatabaseStorage interface {
	Close() error

	FindUserById(ctx context.Context, userId string) (*itypes.User, error)
	FindUserByName(ctx context.Context, username string) (*itypes.UserWithPassword, error)

	// Vault Token methods
	CreateVaultToken(ctx context.Context, token itypes.VaultTokenCreate) (*itypes.VaultToken, error)
	GetVaultToken(ctx context.Context, tokenID string) (*itypes.VaultToken, error)
	RevokeVaultToken(ctx context.Context, tokenID string) error
	RevokeAllVaultTokens(ctx context.Context, publicKey string) error
	UpdateVaultTokenLastUsed(ctx context.Context, tokenID string) error
	GetActiveVaultTokens(ctx context.Context, publicKey string) ([]itypes.VaultToken, error)

	GetPluginPolicy(ctx context.Context, id uuid.UUID) (types.PluginPolicy, error)
	GetAllPluginPolicies(ctx context.Context, publicKey string, pluginID types.PluginID) ([]types.PluginPolicy, error)
	DeletePluginPolicyTx(ctx context.Context, dbTx pgx.Tx, id uuid.UUID) error
	InsertPluginPolicyTx(ctx context.Context, dbTx pgx.Tx, policy types.PluginPolicy) (*types.PluginPolicy, error)
	UpdatePluginPolicyTx(ctx context.Context, dbTx pgx.Tx, policy types.PluginPolicy) (*types.PluginPolicy, error)

	FindPricingById(ctx context.Context, id uuid.UUID) (*itypes.Pricing, error)
	CreatePricing(ctx context.Context, pricingDto itypes.PricingCreateDto) (*itypes.Pricing, error)
	DeletePricingById(ctx context.Context, id uuid.UUID) error

	CountTransactions(ctx context.Context, policyID uuid.UUID, status itypes.TransactionStatus, txType string) (int64, error)
	CreateTransactionHistoryTx(ctx context.Context, dbTx pgx.Tx, tx itypes.TransactionHistory) (uuid.UUID, error)
	UpdateTransactionStatusTx(ctx context.Context, dbTx pgx.Tx, txID uuid.UUID, status itypes.TransactionStatus, metadata map[string]interface{}) error
	CreateTransactionHistory(ctx context.Context, tx itypes.TransactionHistory) (uuid.UUID, error)
	UpdateTransactionStatus(ctx context.Context, txID uuid.UUID, status itypes.TransactionStatus, metadata map[string]interface{}) error
	GetTransactionHistory(ctx context.Context, policyID uuid.UUID, transactionType string, take int, skip int) ([]itypes.TransactionHistory, error)
	GetTransactionByHash(ctx context.Context, txHash string) (*itypes.TransactionHistory, error)

	FindPlugins(ctx context.Context, take int, skip int, sort string) (itypes.PluginsDto, error)
	FindPluginById(ctx context.Context, id types.PluginID) (*itypes.Plugin, error)
	CreatePlugin(ctx context.Context, pluginDto itypes.PluginCreateDto) (*itypes.Plugin, error)
	UpdatePlugin(ctx context.Context, id types.PluginID, updates itypes.PluginUpdateDto) (*itypes.Plugin, error)
	DeletePluginById(ctx context.Context, id types.PluginID) error

	FindCategories(ctx context.Context) ([]itypes.Category, error)

	AddPluginPolicySync(ctx context.Context, dbTx pgx.Tx, policy itypes.PluginPolicySync) error
	GetPluginPolicySync(ctx context.Context, id uuid.UUID) (*itypes.PluginPolicySync, error)
	DeletePluginPolicySync(ctx context.Context, id uuid.UUID) error
	GetUnFinishedPluginPolicySyncs(ctx context.Context) ([]itypes.PluginPolicySync, error)
	UpdatePluginPolicySync(ctx context.Context, dbTx pgx.Tx, policy itypes.PluginPolicySync) error

	Pool() *pgxpool.Pool
}
