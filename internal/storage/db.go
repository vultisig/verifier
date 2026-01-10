package storage

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	itypes "github.com/vultisig/verifier/internal/types"
	"github.com/vultisig/verifier/plugin/tx_indexer/pkg/rpc"
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
	PricingRepository
	PluginRepository
	FeeRepository
	TagRepository
	ReviewRepository
	RatingRepository
	ApiKeyRepository
	ReportRepository
	ControlFlagsRepository
	Close() error
}

type PolicyRepository interface {
	GetPluginPolicy(ctx context.Context, id uuid.UUID) (*types.PluginPolicy, error)
	GetPluginPolicies(ctx context.Context, publicKey string, pluginIds []types.PluginID, includeInactive bool) ([]types.PluginPolicy, error)
	GetPluginInstallationsCount(ctx context.Context, pluginID types.PluginID) (itypes.PluginTotalCount, error)
	GetAllPluginPolicies(ctx context.Context, publicKey string, pluginID types.PluginID, take int, skip int, activeFilter *bool) (*itypes.PluginPolicyPaginatedList, error)
	DeletePluginPolicyTx(ctx context.Context, dbTx pgx.Tx, id uuid.UUID) error
	InsertPluginPolicyTx(ctx context.Context, dbTx pgx.Tx, policy types.PluginPolicy) (*types.PluginPolicy, error)
	UpdatePluginPolicyTx(ctx context.Context, dbTx pgx.Tx, policy types.PluginPolicy) (*types.PluginPolicy, error)
	DeleteAllPolicies(ctx context.Context, dbTx pgx.Tx, pluginID types.PluginID, publicKey string) error
}

type FeeRepository interface {
	GetFeeById(ctx context.Context, id uint64) (*types.Fee, error)
	GetFeesByPublicKey(ctx context.Context, publicKey string) ([]*types.Fee, error)
	GetFeesByPluginID(ctx context.Context, pluginID types.PluginID, publicKey string, skip, take uint32) ([]itypes.FeeWithStatus, uint32, error)
	GetPluginBillingSummary(ctx context.Context, publicKey string) ([]itypes.PluginBillingSummaryRow, error)
	GetPricingsByPluginIDs(ctx context.Context, pluginIDs []string) (map[string][]itypes.PricingInfo, error)
	InsertFee(ctx context.Context, dbTx pgx.Tx, fee *types.Fee) (uint64, error)
	InsertPluginInstallation(ctx context.Context, dbTx pgx.Tx, pluginID types.PluginID, publicKey string) error
	MarkFeesCollected(ctx context.Context, dbTx pgx.Tx, feeIDs []uint64, txHash string, totalAmount uint64) error
	GetUserFees(ctx context.Context, publicKey string) (*types.UserFeeStatus, error)
	UpdateBatchStatus(ctx context.Context, dbTx pgx.Tx, txHash string, status *rpc.TxOnChainStatus) error
	IsTrialActive(ctx context.Context, dbTx pgx.Tx, pubKey string) (bool, time.Duration, error)
}

type PluginPolicySyncRepository interface {
	AddPluginPolicySync(ctx context.Context, dbTx pgx.Tx, policy itypes.PluginPolicySync) error
	GetPluginPolicySync(ctx context.Context, id uuid.UUID) (*itypes.PluginPolicySync, error)
	DeletePluginPolicySync(ctx context.Context, id uuid.UUID) error
	GetUnFinishedPluginPolicySyncs(ctx context.Context) ([]itypes.PluginPolicySync, error)
	UpdatePluginPolicySync(ctx context.Context, dbTx pgx.Tx, policy itypes.PluginPolicySync) error
}

type PricingRepository interface {
	GetPricingByPluginId(ctx context.Context, pluginId types.PluginID) ([]types.Pricing, error)
	FindPricingById(ctx context.Context, id uuid.UUID) (*types.Pricing, error)
	CreatePricing(ctx context.Context, pricingDto types.PricingCreateDto) (*types.Pricing, error)
	DeletePricingById(ctx context.Context, id uuid.UUID) error
}

type PluginRepository interface {
	FindPlugins(ctx context.Context, filters itypes.PluginFilters, take int, skip int, sort string) (*itypes.PluginsPaginatedList, error)
	FindPluginById(ctx context.Context, dbTx pgx.Tx, id types.PluginID) (*itypes.Plugin, error)
	GetPluginTitlesByIDs(ctx context.Context, ids []string) (map[string]string, error)

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

type TagRepository interface {
	FindTags(ctx context.Context) ([]itypes.Tag, error)
	FindTagById(ctx context.Context, id string) (*itypes.Tag, error)
	FindTagByName(ctx context.Context, name string) (*itypes.Tag, error)
}

type ReviewRepository interface {
	CreateReview(ctx context.Context, dbTx pgx.Tx, reviewDto itypes.ReviewCreateDto, pluginId string) (string, error)
	FindReviews(ctx context.Context, pluginId string, take int, skip int, sort string) (itypes.ReviewsDto, error)
	FindReviewById(ctx context.Context, db pgx.Tx, id string) (*itypes.ReviewDto, error)
}

type RatingRepository interface {
	FindRatingByPluginId(ctx context.Context, dbTx pgx.Tx, pluginId string) ([]itypes.PluginRatingDto, error)
	FindAvgRatingByPluginID(ctx context.Context, pluginID string) (itypes.PluginAvgRatingDto, error)
	CreateRatingForPlugin(ctx context.Context, dbTx pgx.Tx, pluginId string) error
	UpdateRatingForPlugin(ctx context.Context, dbTx pgx.Tx, pluginId string, reviewRating int) error
}

type ApiKeyRepository interface {
	GetAPIKey(ctx context.Context, apiKey string) (*itypes.APIKey, error)
	GetAPIKeyByPluginId(ctx context.Context, pluginId string) (*itypes.APIKey, error)
}

type ControlFlagsRepository interface {
	GetControlFlags(ctx context.Context, k1, k2 string) (map[string]bool, error)
}

type ReportRepository interface {
	UpsertReport(ctx context.Context, pluginID types.PluginID, publicKey, reason string, cooldown time.Duration) error
	GetReport(ctx context.Context, pluginID types.PluginID, publicKey string) (*itypes.PluginReport, error)
	CountReportsInWindow(ctx context.Context, pluginID types.PluginID, window time.Duration) (int, error)
	HasInstallation(ctx context.Context, pluginID types.PluginID, publicKey string) (bool, error)
	CountInstallations(ctx context.Context, pluginID types.PluginID) (int, error)
	IsPluginPaused(ctx context.Context, pluginID types.PluginID) (bool, error)
	PausePlugin(ctx context.Context, pluginID types.PluginID, record itypes.PauseHistoryRecord) error
}
