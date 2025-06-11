package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"github.com/sirupsen/logrus"

	"github.com/vultisig/verifier/internal/storage"
	"github.com/vultisig/verifier/internal/syncer"
	"github.com/vultisig/verifier/internal/tasks"
	itypes "github.com/vultisig/verifier/internal/types"
	"github.com/vultisig/verifier/types"
)

type Policy interface {
	CreatePolicy(ctx context.Context, policy types.PluginPolicy) (*types.PluginPolicy, error)
	UpdatePolicy(ctx context.Context, policy types.PluginPolicy) (*types.PluginPolicy, error)
	DeletePolicy(ctx context.Context, policyID uuid.UUID, pluginID types.PluginID, signature string) error
	GetPluginPolicies(ctx context.Context, publicKey string, pluginID types.PluginID, take int, skip int) (itypes.PluginPolicyPaginatedList, error)
	GetPluginPolicy(ctx context.Context, policyID uuid.UUID) (types.PluginPolicy, error)
	GetPluginPolicyTransactionHistory(ctx context.Context, policyID string, take int, skip int) (itypes.TransactionHistoryPaginatedList, error)
	PluginPolicyGetFeeInfo(ctx context.Context, policyID string) (itypes.FeeHistoryDto, error)
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
func (s *PolicyService) CreatePolicy(ctx context.Context, policy types.PluginPolicy) (*types.PluginPolicy, error) {
	// Start transaction
	tx, err := s.db.Pool().Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	policy.PopulateBilling()
	//TODO garry, do we need to validate the policy here?

	// Insert policy
	newPolicy, err := s.db.InsertPluginPolicyTx(ctx, tx, policy)
	if err != nil {
		s.handleRollback(tx, ctx)
		return nil, fmt.Errorf("failed to insert policy: %w", err)
	}

	//TODO handle updates sync cases with billing info
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
		s.handleRollback(tx, ctx)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	if err := s.syncPolicy(policySync); err != nil {
		s.logger.WithError(err).Error("failed post sync policy to queue")
	}

	for _, billingPolicy := range newPolicy.Billing {
		if billingPolicy.Type == string(types.BILLING_TYPE_ONCE) {
			bid, err := billingPolicy.ID.MarshalBinary()
			//TODO garry. Need to handle this error properly with a rollback if needed.
			if err != nil {
				s.logger.WithError(err).Error("failed to marshal billing policy ID")
			}
			s.client.Enqueue(
				asynq.NewTask(tasks.TypeOneTimeFeeRecord, bid),
				asynq.MaxRetry(0),
				asynq.Timeout(2*time.Minute),
				asynq.Retention(5*time.Minute),
				asynq.Queue(tasks.QUEUE_NAME),
			)
		}
	}

	//TODO garry, potentially we don't send this to a job but rather handle it in the same transaction. This reduces the risk of a committed

	return newPolicy, nil
}

func (s *PolicyService) UpdatePolicy(ctx context.Context, policy types.PluginPolicy) (*types.PluginPolicy, error) {
	// start transaction
	tx, err := s.db.Pool().Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer s.handleRollback(tx, ctx)

	// TODO garry do we need to validate the policy here?

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
	if err := s.syncPolicy(syncPolicyEntity); err != nil {
		s.logger.WithError(err).Error("failed post sync policy to queue")
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
	err = s.db.DeletePluginPolicyTx(ctx, tx, policyID)
	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

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

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	if err := s.syncPolicy(syncPolicyEntity); err != nil {
		s.logger.WithError(err).Error("failed post sync policy to queue")
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

func (s *PolicyService) GetPluginPolicyTransactionHistory(ctx context.Context, policyID string, take int, skip int) (itypes.TransactionHistoryPaginatedList, error) {
	policyUUID, err := uuid.Parse(policyID)
	if err != nil {
		return itypes.TransactionHistoryPaginatedList{}, fmt.Errorf("invalid policy_id: %s", policyID)
	}

	history, totalCount, err := s.db.GetTransactionHistory(ctx, policyUUID, "SWAP", take, skip)
	if err != nil {
		return itypes.TransactionHistoryPaginatedList{}, fmt.Errorf("failed to get policy history: %w", err)
	}

	return itypes.TransactionHistoryPaginatedList{
		History:    history,
		TotalCount: int(totalCount),
	}, nil
}

func (s *PolicyService) PluginPolicyGetFeeInfo(ctx context.Context, policyID string) (itypes.FeeHistoryDto, error) {
	policyUUID, err := uuid.Parse(policyID)
	history := itypes.FeeHistoryDto{}

	if err != nil {
		return history, fmt.Errorf("invalid policy_id: %s", policyID)
	}

	fees, err := s.db.GetAllFeesByPolicyId(ctx, policyUUID)
	if err != nil {
		return history, fmt.Errorf("failed to get fees: %w", err)
	}

	totalFeesIncurred := 0
	feesPendingCollection := 0

	ifees := make([]itypes.FeeDto, 0, len(fees))
	for _, fee := range fees {
		collected := true
		if fee.CollectedAt == nil {
			collected = false
		}
		collectedDt := ""
		if collected {
			collectedDt = fee.CollectedAt.Format(time.RFC3339)
		}
		ifee := itypes.FeeDto{
			Amount:      fee.Amount,
			Collected:   collected,
			CollectedAt: collectedDt,
			ChargedAt:   fee.ChargedAt.Format(time.RFC3339),
		}
		totalFeesIncurred += fee.Amount
		if !collected {
			feesPendingCollection += fee.Amount
		}
		ifees = append(ifees, ifee)
	}

	history = itypes.FeeHistoryDto{
		Fees:                  ifees,
		PolicyId:              policyUUID,
		TotalFeesIncurred:     totalFeesIncurred,
		FeesPendingCollection: feesPendingCollection,
	}

	return history, nil
}
