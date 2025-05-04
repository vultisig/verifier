package storage

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	itypes "github.com/vultisig/verifier/internal/types"
	types "github.com/vultisig/verifier/types"
)

type DatabaseStorage interface {
	Close() error

	FindUserById(ctx context.Context, userId string) (*itypes.User, error)
	FindUserByName(ctx context.Context, username string) (*itypes.UserWithPassword, error)

	GetPluginPolicy(ctx context.Context, id string) (types.PluginPolicy, error)
	GetAllPluginPolicies(ctx context.Context, publicKey string, pluginType string) ([]types.PluginPolicy, error)
	DeletePluginPolicyTx(ctx context.Context, dbTx pgx.Tx, id string) error
	InsertPluginPolicyTx(ctx context.Context, dbTx pgx.Tx, policy types.PluginPolicy) (*types.PluginPolicy, error)
	UpdatePluginPolicyTx(ctx context.Context, dbTx pgx.Tx, policy types.PluginPolicy) (*types.PluginPolicy, error)

	FindPricingById(ctx context.Context, id string) (*itypes.Pricing, error)
	CreatePricing(ctx context.Context, pricingDto itypes.PricingCreateDto) (*itypes.Pricing, error)
	DeletePricingById(ctx context.Context, id string) error

	CreateTimeTriggerTx(ctx context.Context, dbTx pgx.Tx, trigger itypes.TimeTrigger) error
	GetPendingTimeTriggers(ctx context.Context) ([]itypes.TimeTrigger, error)
	UpdateTimeTriggerLastExecution(ctx context.Context, policyID string) error
	UpdateTimeTriggerTx(ctx context.Context, policyID string, trigger itypes.TimeTrigger, dbTx pgx.Tx) error

	DeleteTimeTrigger(ctx context.Context, policyID string) error
	UpdateTriggerStatus(ctx context.Context, policyID string, status itypes.TimeTriggerStatus) error
	GetTriggerStatus(ctx context.Context, policyID string) (itypes.TimeTriggerStatus, error)

	CountTransactions(ctx context.Context, policyID uuid.UUID, status itypes.TransactionStatus, txType string) (int64, error)
	CreateTransactionHistoryTx(ctx context.Context, dbTx pgx.Tx, tx itypes.TransactionHistory) (uuid.UUID, error)
	UpdateTransactionStatusTx(ctx context.Context, dbTx pgx.Tx, txID uuid.UUID, status itypes.TransactionStatus, metadata map[string]interface{}) error
	CreateTransactionHistory(ctx context.Context, tx itypes.TransactionHistory) (uuid.UUID, error)
	UpdateTransactionStatus(ctx context.Context, txID uuid.UUID, status itypes.TransactionStatus, metadata map[string]interface{}) error
	GetTransactionHistory(ctx context.Context, policyID uuid.UUID, transactionType string, take int, skip int) ([]itypes.TransactionHistory, error)
	GetTransactionByHash(ctx context.Context, txHash string) (*itypes.TransactionHistory, error)

	FindPlugins(ctx context.Context, take int, skip int, sort string) (itypes.PlugisDto, error)
	FindPluginById(ctx context.Context, id string) (*itypes.Plugin, error)
	CreatePlugin(ctx context.Context, pluginDto itypes.PluginCreateDto) (*itypes.Plugin, error)
	UpdatePlugin(ctx context.Context, id string, updates itypes.PluginUpdateDto) (*itypes.Plugin, error)
	DeletePluginById(ctx context.Context, id string) error

	Pool() *pgxpool.Pool
}
