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
	GetAllPluginPolicies(ctx context.Context, pluginType string, publicKey string, take int, skip int) (itypes.PluginPolicyPaginatedList, error)
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

	FindPlugins(ctx context.Context, filters itypes.PluginFilters, take int, skip int, sort string) (itypes.PluginsPaginatedList, error)
	FindPluginById(ctx context.Context, dbTx pgx.Tx, id types.PluginID) (*itypes.Plugin, error)
	CreatePlugin(ctx context.Context, dbTx pgx.Tx, pluginDto itypes.PluginCreateDto) (string, error)
	UpdatePlugin(ctx context.Context, id types.PluginID, updates itypes.PluginUpdateDto) (*itypes.Plugin, error)
	DeletePluginById(ctx context.Context, id types.PluginID) error
	AttachTagToPlugin(ctx context.Context, pluginId types.PluginID, tagId string) (*itypes.Plugin, error)
	DetachTagFromPlugin(ctx context.Context, pluginId types.PluginID, tagId string) (*itypes.Plugin, error)

	FindCategories(ctx context.Context) ([]itypes.Category, error)

	FindTags(ctx context.Context) ([]itypes.Tag, error)
	FindTagById(ctx context.Context, id string) (*itypes.Tag, error)
	FindTagByName(ctx context.Context, name string) (*itypes.Tag, error)
	CreateTag(ctx context.Context, tagDto itypes.CreateTagDto) (*itypes.Tag, error)

	AddPluginPolicySync(ctx context.Context, dbTx pgx.Tx, policy itypes.PluginPolicySync) error
	GetPluginPolicySync(ctx context.Context, id uuid.UUID) (*itypes.PluginPolicySync, error)
	DeletePluginPolicySync(ctx context.Context, id uuid.UUID) error
	GetUnFinishedPluginPolicySyncs(ctx context.Context) ([]itypes.PluginPolicySync, error)
	UpdatePluginPolicySync(ctx context.Context, dbTx pgx.Tx, policy itypes.PluginPolicySync) error

	Pool() *pgxpool.Pool
}
