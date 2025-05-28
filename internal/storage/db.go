package storage

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	itypes "github.com/vultisig/verifier/internal/types"
	"github.com/vultisig/verifier/types"
)

type PoolProvider interface {
	Pool() *pgxpool.Pool
}

type Transactor interface {
	PoolProvider
	WithTransaction(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error
}

type DatabaseStorage interface {
	Transactor
	PolicyRepository
	PluginPolicySyncRepository
	VaultTokenRepository
	TransactionRepository
	PricingRepository
	PluginRepository
	CategoryRepository
	TagRepository
	ReviewRepository
	RatingRepository
	Close() error
}

type PolicyRepository interface {
	GetPluginPolicy(ctx context.Context, id uuid.UUID) (types.PluginPolicy, error)
	GetAllPluginPolicies(ctx context.Context, publicKey string, pluginID types.PluginID, take int, skip int) (itypes.PluginPolicyPaginatedList, error)

	DeletePluginPolicyTx(ctx context.Context, dbTx pgx.Tx, id uuid.UUID) error
	InsertPluginPolicyTx(ctx context.Context, dbTx pgx.Tx, policy types.PluginPolicy) (*types.PluginPolicy, error)
	UpdatePluginPolicyTx(ctx context.Context, dbTx pgx.Tx, policy types.PluginPolicy) (*types.PluginPolicy, error)
}

type PluginPolicySyncRepository interface {
	AddPluginPolicySync(ctx context.Context, dbTx pgx.Tx, policy itypes.PluginPolicySync) error
	GetPluginPolicySync(ctx context.Context, id uuid.UUID) (*itypes.PluginPolicySync, error)
	DeletePluginPolicySync(ctx context.Context, id uuid.UUID) error
	GetUnFinishedPluginPolicySyncs(ctx context.Context) ([]itypes.PluginPolicySync, error)
	UpdatePluginPolicySync(ctx context.Context, dbTx pgx.Tx, policy itypes.PluginPolicySync) error
}

type TransactionRepository interface {
	CountTransactions(ctx context.Context, policyID uuid.UUID, status itypes.TransactionStatus, txType string) (int64, error)
	CreateTransactionHistoryTx(ctx context.Context, dbTx pgx.Tx, tx itypes.TransactionHistory) (uuid.UUID, error)
	UpdateTransactionStatusTx(ctx context.Context, dbTx pgx.Tx, txID uuid.UUID, status itypes.TransactionStatus, metadata map[string]interface{}) error
	CreateTransactionHistory(ctx context.Context, tx itypes.TransactionHistory) (uuid.UUID, error)
	UpdateTransactionStatus(ctx context.Context, txID uuid.UUID, status itypes.TransactionStatus, metadata map[string]interface{}) error
	GetTransactionHistory(ctx context.Context, policyID uuid.UUID, transactionType string, take int, skip int) ([]itypes.TransactionHistory, int64, error)
	GetTransactionByHash(ctx context.Context, txHash string) (*itypes.TransactionHistory, error)
}

type PricingRepository interface {
	FindPricingById(ctx context.Context, id uuid.UUID) (*itypes.Pricing, error)
	CreatePricing(ctx context.Context, pricingDto itypes.PricingCreateDto) (*itypes.Pricing, error)
	DeletePricingById(ctx context.Context, id uuid.UUID) error
}

type PluginRepository interface {
	FindPlugins(ctx context.Context, filters itypes.PluginFilters, take int, skip int, sort string) (itypes.PluginsPaginatedList, error)
	FindPluginById(ctx context.Context, dbTx pgx.Tx, id types.PluginID) (*itypes.Plugin, error)
	CreatePlugin(ctx context.Context, dbTx pgx.Tx, pluginDto itypes.PluginCreateDto) (string, error)
	UpdatePlugin(ctx context.Context, id types.PluginID, updates itypes.PluginUpdateDto) (*itypes.Plugin, error)
	DeletePluginById(ctx context.Context, id types.PluginID) error
	AttachTagToPlugin(ctx context.Context, pluginId types.PluginID, tagId string) (*itypes.Plugin, error)
	DetachTagFromPlugin(ctx context.Context, pluginId types.PluginID, tagId string) (*itypes.Plugin, error)

	Pool() *pgxpool.Pool
}

type VaultTokenRepository interface {
	CreateVaultToken(ctx context.Context, token itypes.VaultTokenCreate) (*itypes.VaultToken, error)
	GetVaultToken(ctx context.Context, tokenID string) (*itypes.VaultToken, error)
	RevokeVaultToken(ctx context.Context, tokenID string) error
	RevokeAllVaultTokens(ctx context.Context, publicKey string) error
	UpdateVaultTokenLastUsed(ctx context.Context, tokenID string) error
	GetActiveVaultTokens(ctx context.Context, publicKey string) ([]itypes.VaultToken, error)
}

type CategoryRepository interface {
	FindCategories(ctx context.Context) ([]itypes.Category, error)
}

type TagRepository interface {
	FindTags(ctx context.Context) ([]itypes.Tag, error)
	FindTagById(ctx context.Context, id string) (*itypes.Tag, error)
	FindTagByName(ctx context.Context, name string) (*itypes.Tag, error)
	CreateTag(ctx context.Context, tagDto itypes.CreateTagDto) (*itypes.Tag, error)
}

type ReviewRepository interface {
	CreateReview(ctx context.Context, dbTx pgx.Tx, reviewDto itypes.ReviewCreateDto, pluginId string) (string, error)
	FindReviews(ctx context.Context, pluginId string, take int, skip int, sort string) (itypes.ReviewsDto, error)
	FindReviewById(ctx context.Context, db pgx.Tx, id string) (*itypes.ReviewDto, error)
}

type RatingRepository interface {
	FindRatingByPluginId(ctx context.Context, dbTx pgx.Tx, pluginId string) ([]itypes.PluginRatingDto, error)
	CreateRatingForPlugin(ctx context.Context, dbTx pgx.Tx, pluginId string) error
	UpdateRatingForPlugin(ctx context.Context, dbTx pgx.Tx, pluginId string, reviewRating int) error
}
