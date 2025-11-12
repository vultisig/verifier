package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
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
	GetPluginPolicies(ctx context.Context, publicKey string, pluginID types.PluginID, take int, skip int, includeInactive bool) (*itypes.PluginPolicyPaginatedList, error)
	GetPluginPolicy(ctx context.Context, policyID uuid.UUID) (*types.PluginPolicy, error)
	GetPluginInstallationsCount(ctx context.Context, pluginID types.PluginID) (itypes.PluginTotalCount, error)
	DeleteAllPolicies(ctx context.Context, pluginID types.PluginID, publicKey string) error
}

var _ Policy = (*PolicyService)(nil)

type PolicyService struct {
	db     storage.DatabaseStorage
	logger *logrus.Logger
	syncer *syncer.Syncer
}

func NewPolicyService(db storage.DatabaseStorage, syncer *syncer.Syncer) (*PolicyService, error) {
	if db == nil {
		return nil, fmt.Errorf("database storage cannot be nil")
	}
	return &PolicyService{
		db:     db,
		logger: logrus.WithField("service", "policy").Logger,
		syncer: syncer,
	}, nil
}

// This loops through the billing policies and checks if the pricing is valid for the billing policy.
func compareBillingPricing(pricing *types.Pricing, billing *types.BillingPolicy) bool {
	if pricing == nil && billing == nil {
		return true
	}
	if pricing == nil || billing == nil {
		return false
	}
	sameType := pricing.Type == billing.Type
	sameFrequency := true
	if pricing.Type == types.PricingTypeRecurring && pricing.Frequency != nil {
		sameFrequency = *pricing.Frequency == *billing.Frequency
	}
	sameAmount := pricing.Amount == billing.Amount
	//metrics to be added later for support. For now the only pricing db entry supported in fixed.

	return sameType && sameFrequency && sameAmount
}

func (s *PolicyService) validateBillingInformation(ctx context.Context, policy types.PluginPolicy) error {
	var err error
	tx, err := s.db.Pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	//If there is an error, then rollback the tx at the end
	defer func() {
		if err != nil {
			s.handleRollback(tx)
		}
	}()

	pluginData, err := s.db.FindPluginById(ctx, tx, policy.PluginID)
	if err != nil {
		return fmt.Errorf("failed to find plugin: %w", err)
	}

	err = policy.ParseBillingFromRecipe()
	if err != nil {
		return fmt.Errorf("failed to parse billing from recipe: %w", err)
	}

	if len(policy.Billing) != len(pluginData.Pricing) {
		return fmt.Errorf("billing policies count (%d) does not match plugin pricing count (%d)", len(policy.Billing), len(pluginData.Pricing))
	}

	// For each billing policy, check for a matching pricing entry
	usedPricing := make([]bool, len(pluginData.Pricing))
	for i, billing := range policy.Billing {
		found := false
		for j, pricing := range pluginData.Pricing {
			if usedPricing[j] {
				continue
			}
			if compareBillingPricing(&pricing, &billing) {
				usedPricing[j] = true
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("no matching plugin pricing found for billing policy at index %d", i)
		}
	}

	return nil
}

func (s *PolicyService) isFeePolicyInstalled(ctx context.Context, publicKey string) (bool, error) {
	pluginData, err := s.db.GetPluginPolicies(ctx, publicKey, []types.PluginID{types.PluginVultisigFees_feee}, false)
	if err != nil {
		return false, fmt.Errorf("failed to get plugin policy: %w", err)
	}
	return len(pluginData) > 0, nil
}

func (s *PolicyService) CreatePolicy(ctx context.Context, policy types.PluginPolicy) (*types.PluginPolicy, error) {
	// Start transaction
	tx, err := s.db.Pool().Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer s.handleRollback(tx)

	// Populate billing information
	if err := policy.ParseBillingFromRecipe(); err != nil {
		return nil, fmt.Errorf("failed to populate billing: %w", err)
	}

	// Compare and contrast the billing information (signed by user) with the pricing information (defined in the pricings table and connected to the plugin definition)
	if err := s.validateBillingInformation(ctx, policy); err != nil {
		return nil, fmt.Errorf("failed to validate billing information: %w", err)
	}

	//// If a non plugin policy, then we need to validate the fee information
	//if policy.PluginID != types.PluginVultisigFees_feee && len(policy.Billing) > 0 {
	//	isFeePolicyInstalled, err := s.isFeePolicyInstalled(ctx, policy.PublicKey)
	//	if err != nil {
	//		return nil, fmt.Errorf("failed to check if fee policy is installed: %w", err)
	//	}
	//	if !isFeePolicyInstalled {
	//		return nil, fmt.Errorf("fee policy is not installed")
	//	}
	//}

	// Insert policy
	newPolicy, err := s.db.InsertPluginPolicyTx(ctx, tx, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to insert policy: %w", err)
	}

	//// Create one-time fee records within the transaction
	//for _, billingPolicy := range newPolicy.Billing {
	//	if billingPolicy.Type == types.PricingTypeOnce {
	//		// Create fee record for one-time billing
	//		fee, err := s.db.InsertFee(ctx, tx, types.Fee{
	//			PluginPolicyBillingID: billingPolicy.ID,
	//			Amount:                billingPolicy.Amount,
	//		})
	//		if err != nil {
	//			return nil, fmt.Errorf("failed to insert fee for billing policy %s: %w", billingPolicy.ID, err)
	//		}
	//
	//		s.logger.WithFields(logrus.Fields{
	//			"plugin_policy_id":         newPolicy.ID,
	//			"plugin_policy_billing_id": billingPolicy.ID,
	//			"fee_id":                   fee.ID,
	//			"amount":                   fee.Amount,
	//		}).Info("Inserted one-time fee record")
	//	}
	//}

	// Sync policy synchronously - if this fails, the entire operation fails
	if err := s.syncer.CreatePolicySync(ctx, policy); err != nil {
		s.logger.WithError(err).Error("failed to sync policy with plugin server")
		return nil, fmt.Errorf("failed to sync policy with plugin server: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return newPolicy, nil
}

func (s *PolicyService) UpdatePolicy(ctx context.Context, policy types.PluginPolicy) (*types.PluginPolicy, error) {
	// start transaction
	tx, err := s.db.Pool().Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer s.handleRollback(tx)

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

	// Sync policy synchronously - if this fails, the entire operation fails
	if err := s.syncer.CreatePolicyAsync(ctx, syncPolicyEntity); err != nil {
		s.logger.WithError(err).Error("failed to sync policy with plugin server")
		return nil, fmt.Errorf("failed to sync policy with plugin server: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	return updatedPolicy, nil
}

func (s *PolicyService) handleRollback(tx pgx.Tx) {
	ctx := context.Background()
	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		s.logger.WithError(err).Error("failed to rollback transaction")
	}
}

func (s *PolicyService) DeletePolicy(ctx context.Context, policyID uuid.UUID, pluginID types.PluginID, signature string) error {
	tx, err := s.db.Pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer s.handleRollback(tx)

	// Check if policy exists
	policy, err := s.db.GetPluginPolicy(ctx, policyID)
	if err != nil {
		return fmt.Errorf("failed to get policy: %w", err)
	}

	// Sync policy synchronously - if this fails, the entire operation fails
	if err := s.syncer.DeletePolicySync(ctx, *policy); err != nil {
		s.logger.WithError(err).Error("failed to sync policy deletion with plugin server")
		return fmt.Errorf("failed to sync policy deletion with plugin server: %w", err)
	}

	err = s.db.DeletePluginPolicyTx(ctx, tx, policyID)
	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (s *PolicyService) GetPluginPolicies(ctx context.Context, publicKey string, pluginID types.PluginID, take int, skip int, includeInactive bool) (*itypes.PluginPolicyPaginatedList, error) {
	policies, err := s.db.GetAllPluginPolicies(ctx, publicKey, pluginID, take, skip, includeInactive)
	if err != nil {
		return nil, fmt.Errorf("failed to get policies: %w", err)
	}
	return policies, nil
}

func (s *PolicyService) GetPluginPolicy(ctx context.Context, policyID uuid.UUID) (*types.PluginPolicy, error) {
	policy, err := s.db.GetPluginPolicy(ctx, policyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}
	return policy, nil
}

func (s *PolicyService) GetPluginInstallationsCount(ctx context.Context, pluginID types.PluginID) (itypes.PluginTotalCount, error) {
	count, err := s.db.GetPluginInstallationsCount(ctx, pluginID)
	if err != nil {
		return itypes.PluginTotalCount{}, fmt.Errorf("failed to get plugin installations count: %w", err)
	}
	return count, nil
}

func (s *PolicyService) DeleteAllPolicies(ctx context.Context, pluginID types.PluginID, publicKey string) error {
	tx, err := s.db.Pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer s.handleRollback(tx)

	err = s.db.DeleteAllPolicies(ctx, tx, pluginID, publicKey)
	if err != nil {
		return fmt.Errorf("failed to delete all policies: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
