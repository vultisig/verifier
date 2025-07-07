package service

import (
	"context"
	"encoding/json"
	"errors"
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
	DeletePolicy(ctx context.Context, policyID uuid.UUID, pluginID types.PluginID, signature string) error
	GetPluginPolicies(ctx context.Context, publicKey string, pluginID types.PluginID, take int, skip int) (itypes.PluginPolicyPaginatedList, error)
	GetPluginPolicy(ctx context.Context, policyID uuid.UUID) (types.PluginPolicy, error)
	DeleteAllPolicies(ctx context.Context, pluginID types.PluginID, publicKey string) error
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

	// Populate billing information
	if err := policy.PopulateBilling(); err != nil {
		return nil, fmt.Errorf("failed to populate billing: %w", err)
	}

	// Insert policy
	newPolicy, err := s.db.InsertPluginPolicyTx(ctx, tx, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to insert policy: %w", err)
	}

	// Create one-time fee records within the transaction
	for _, billingPolicy := range newPolicy.Billing {
		if billingPolicy.Type == string(types.BILLING_TYPE_ONCE) {
			// Create fee record for one-time billing
			var feeID uuid.UUID
			var feeAmount int
			err = tx.QueryRow(ctx,
				`INSERT INTO fees (plugin_policy_billing_id, amount) VALUES ($1, $2) RETURNING id, amount`,
				billingPolicy.ID, billingPolicy.Amount,
			).Scan(&feeID, &feeAmount)
			if err != nil {
				return nil, fmt.Errorf("failed to insert one-time fee record for billing policy %s: %w", billingPolicy.ID, err)
			}

			s.logger.WithFields(logrus.Fields{
				"plugin_policy_id":         newPolicy.ID,
				"plugin_policy_billing_id": billingPolicy.ID,
				"fee_id":                   feeID,
				"amount":                   feeAmount,
			}).Info("Inserted one-time fee record")
		}
	}

	// TODO handle updates sync cases with billing info
	policySync := itypes.PluginPolicySync{
		ID:         uuid.New(),
		PolicyID:   newPolicy.ID,
		PluginID:   newPolicy.PluginID,
		Signature:  newPolicy.Signature,
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

	// Sync policy after successful commit
	if err := s.syncPolicy(policySync); err != nil {
		s.logger.WithError(err).Error("failed to enqueue sync policy task")
		// Note: We don't return error here as the policy was successfully created
		// The sync can be retried later through the failed task processor
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
		PluginID:   updatedPolicy.PluginID,
		Signature:  updatedPolicy.Signature,
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

	// Sync policy after successful commit
	if err := s.syncPolicy(syncPolicyEntity); err != nil {
		s.logger.WithError(err).Error("failed to enqueue sync policy task")
		// Note: We don't return error here as the policy was successfully updated
		// The sync can be retried later through the failed task processor
	}
	return updatedPolicy, nil
}

func (s *PolicyService) handleRollback(tx pgx.Tx, ctx context.Context) {
	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		s.logger.WithError(err).Error("failed to rollback transaction")
	}
}

func (s *PolicyService) DeletePolicy(ctx context.Context, policyID uuid.UUID, pluginID types.PluginID, signature string) error {
	tx, err := s.db.Pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer s.handleRollback(tx, ctx)

	syncPolicyEntity := itypes.PluginPolicySync{
		ID:         uuid.New(),
		PolicyID:   policyID,
		PluginID:   pluginID,
		Signature:  signature,
		SyncType:   itypes.RemovePolicy,
		Status:     itypes.NotSynced,
		FailReason: "",
	}
	if err := s.db.AddPluginPolicySync(ctx, tx, syncPolicyEntity); err != nil {
		return fmt.Errorf("failed to add policy sync: %w", err)
	}

	// TODO: use soft delete instead of hard delete (hard delete will remove policy syncs as well)
	err = s.db.DeletePluginPolicyTx(ctx, tx, policyID)
	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Sync policy after successful commit
	if err := s.syncPolicy(syncPolicyEntity); err != nil {
		s.logger.WithError(err).Error("failed to enqueue sync policy task")
		// Note: We don't return error here as the policy was successfully deleted
		// The sync can be retried later through the failed task processor
	}

	return nil
}

func (s *PolicyService) GetPluginPolicies(ctx context.Context, publicKey string, pluginID types.PluginID, take int, skip int) (itypes.PluginPolicyPaginatedList, error) {
	policies, err := s.db.GetAllPluginPolicies(ctx, publicKey, pluginID, take, skip)
	if err != nil {
		return itypes.PluginPolicyPaginatedList{}, fmt.Errorf("failed to get policies: %w", err)
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

func (s *PolicyService) DeleteAllPolicies(ctx context.Context, pluginID types.PluginID, publicKey string) error {
	tx, err := s.db.Pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer s.handleRollback(tx, ctx)

	err = s.db.DeleteAllPolicies(ctx, tx, pluginID, publicKey)
	if err != nil {
		return fmt.Errorf("failed to delete all policies: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
