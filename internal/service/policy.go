package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"github.com/sirupsen/logrus"

	"github.com/vultisig/verifier/internal/storage"
	"github.com/vultisig/verifier/internal/syncer"
	itypes "github.com/vultisig/verifier/internal/types"
	"github.com/vultisig/verifier/types"
)

type Policy interface {
	CreatePolicy(ctx context.Context, policy types.PluginPolicy) (*types.PluginPolicy, error)
	UpdatePolicy(ctx context.Context, policy types.PluginPolicy) (*types.PluginPolicy, error)
	DeletePolicy(ctx context.Context, policyID uuid.UUID, signature string) error
	GetPluginPolicies(ctx context.Context, pluginType, publicKey string) ([]types.PluginPolicy, error)
	GetPluginPolicy(ctx context.Context, policyID uuid.UUID) (types.PluginPolicy, error)
	GetPluginPolicyTransactionHistory(ctx context.Context, policyID uuid.UUID) ([]itypes.TransactionHistory, error)
}

var _ Policy = (*PolicyService)(nil)

type PolicyService struct {
	db     storage.DatabaseStorage
	logger *logrus.Logger
	client *asynq.Client
}

func NewPolicyService(db storage.DatabaseStorage,
	client *asynq.Client) (*PolicyService, error) {
	if db == nil {
		return nil, fmt.Errorf("database storage cannot be nil")
	}
	return &PolicyService{
		db:     db,
		logger: logrus.WithField("service", "policy").Logger,
		client: client,
	}, nil
}

func (s *PolicyService) syncPolicy(syncEntity itypes.PluginPolicySync) error {
	syncEntityJSON, err := json.Marshal(syncEntity)
	if err != nil {
		return fmt.Errorf("failed to marshal sync entity: %w", err)
	}
	ti, err := s.client.Enqueue(
		asynq.NewTask(syncer.TaskKeySyncPolicy, syncEntityJSON),
		asynq.Queue(syncer.QUEUE_NAME),
		asynq.MaxRetry(3),
	)
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}
	s.logger.WithField("task_id", ti.ID).Info("enqueued sync policy task")
	return nil
}
func (s *PolicyService) CreatePolicy(ctx context.Context, policy types.PluginPolicy) (*types.PluginPolicy, error) {
	// Start transaction
	tx, err := s.db.Pool().Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer s.handleRollback(tx, ctx)

	// Insert policy
	newPolicy, err := s.db.InsertPluginPolicyTx(ctx, tx, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to insert policy: %w", err)
	}

	policySync := itypes.PluginPolicySync{
		ID:         uuid.New(),
		PolicyID:   newPolicy.ID,
		SyncType:   itypes.AddPolicy,
		Status:     itypes.NotSynced,
		FailReason: "",
	}
	if err := s.db.AddPluginPolicySync(ctx, tx, policySync); err != nil {
		return nil, fmt.Errorf("failed to add policy sync: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	if err := s.syncPolicy(policySync); err != nil {
		s.logger.WithError(err).Error("failed post sync policy to queue")
	}

	return newPolicy, nil
}

func (s *PolicyService) UpdatePolicy(ctx context.Context, policy types.PluginPolicy) (*types.PluginPolicy, error) {
	// start transaction
	tx, err := s.db.Pool().Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer s.handleRollback(tx, ctx)

	// Update policy with tx
	updatedPolicy, err := s.db.UpdatePluginPolicyTx(ctx, tx, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to update policy: %w", err)
	}

	syncPolicyEntity := itypes.PluginPolicySync{
		ID:         uuid.New(),
		PolicyID:   updatedPolicy.ID,
		SyncType:   itypes.UpdatePolicy,
		Status:     itypes.NotSynced,
		FailReason: "",
	}
	if err := s.db.AddPluginPolicySync(ctx, tx, syncPolicyEntity); err != nil {
		return nil, fmt.Errorf("failed to add policy sync: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	if err := s.syncPolicy(syncPolicyEntity); err != nil {
		s.logger.WithError(err).Error("failed post sync policy to queue")
	}
	return updatedPolicy, nil
}

func (s *PolicyService) handleRollback(tx pgx.Tx, ctx context.Context) {
	if err := tx.Rollback(ctx); err != nil {
		s.logger.WithError(err).Error("failed to rollback transaction")
	}
}

func (s *PolicyService) DeletePolicy(ctx context.Context, policyID uuid.UUID, signature string) error {
	tx, err := s.db.Pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer s.handleRollback(tx, ctx)
	err = s.db.DeletePluginPolicyTx(ctx, tx, policyID)
	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	syncPolicyEntity := itypes.PluginPolicySync{
		ID:         uuid.New(),
		PolicyID:   policyID,
		SyncType:   itypes.RemovePolicy,
		Status:     itypes.NotSynced,
		FailReason: "",
	}
	if err := s.db.AddPluginPolicySync(ctx, tx, syncPolicyEntity); err != nil {
		return fmt.Errorf("failed to add policy sync: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	if err := s.syncPolicy(syncPolicyEntity); err != nil {
		s.logger.WithError(err).Error("failed post sync policy to queue")
	}

	return nil
}

func (s *PolicyService) GetPluginPolicies(ctx context.Context, pluginType, publicKey string) ([]types.PluginPolicy, error) {
	policies, err := s.db.GetAllPluginPolicies(ctx, pluginType, publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get policies: %w", err)
	}
	return policies, nil
}

func (s *PolicyService) GetPluginPolicy(ctx context.Context, policyID uuid.UUID) (types.PluginPolicy, error) {
	policy, err := s.db.GetPluginPolicy(ctx, policyID)
	if err != nil {
		return types.PluginPolicy{}, fmt.Errorf("failed to get policy: %w", err)
	}
	return policy, nil
}

func (s *PolicyService) GetPluginPolicyTransactionHistory(ctx context.Context, policyID uuid.UUID) ([]itypes.TransactionHistory, error) {

	history, err := s.db.GetTransactionHistory(ctx, policyID, "SWAP", 30, 0) // take the last 30 records and skip the first 0
	if err != nil {
		return []itypes.TransactionHistory{}, fmt.Errorf("failed to get policy history: %w", err)
	}

	return history, nil
}
