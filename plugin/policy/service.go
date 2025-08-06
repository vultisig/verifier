package policy

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vultisig/verifier/plugin/scheduler"
	"github.com/vultisig/verifier/types"
)

var _ Policy = (*Service)(nil)

type Policy interface {
	CreatePolicy(ctx context.Context, policy types.PluginPolicy) (*types.PluginPolicy, error)
	UpdatePolicy(ctx context.Context, policy types.PluginPolicy) (*types.PluginPolicy, error)
	DeletePolicy(ctx context.Context, policyID uuid.UUID, signature string) error
	GetPluginPolicies(
		ctx context.Context,
		pluginID types.PluginID,
		publicKey string,
		onlyActive bool,
	) ([]types.PluginPolicy, error)
	GetPluginPolicy(ctx context.Context, policyID uuid.UUID) (*types.PluginPolicy, error)
}

type Service[T any] struct {
	policy    Storage[T]
	scheduler scheduler.Service
	logger    *logrus.Logger
}

func NewService[T any](
	policy Storage[T],
	scheduler scheduler.Service,
	logger *logrus.Logger,
) (*Service[T], error) {
	return &Service[T]{
		policy:    policy,
		scheduler: scheduler,
		logger:    logger,
	}, nil
}

func (s *Service[T]) CreatePolicy(c context.Context, policy types.PluginPolicy) (*types.PluginPolicy, error) {
	ctx, err := s.policy.Tx().Begin(c)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = s.policy.Tx().Rollback(ctx)
	}()

	// Insert policy
	newPolicy, err := s.policy.InsertPluginPolicy(ctx, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to insert policy: %w", err)
	}

	err = s.scheduler.Create(ctx, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to create policy in scheduler: %w", err)
	}

	if err := s.policy.Tx().Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return newPolicy, nil
}

func (s *Service[T]) UpdatePolicy(c context.Context, policy types.PluginPolicy) (*types.PluginPolicy, error) {
	ctx, err := s.policy.Tx().Begin(c)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = s.policy.Tx().Rollback(ctx)
	}()

	oldPolicy, err := s.policy.GetPluginPolicy(ctx, policy.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plugin policy: %w", err)
	}

	err = s.scheduler.Update(ctx, *oldPolicy, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to update policy in scheduler: %w", err)
	}

	// Update policy with tx
	updatedPolicy, err := s.policy.UpdatePluginPolicy(ctx, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to update policy: %w", err)
	}

	if err := s.policy.Tx().Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return updatedPolicy, nil
}

func (s *Service[T]) DeletePolicy(c context.Context, policyID uuid.UUID, signature string) error {
	ctx, err := s.policy.Tx().Begin(c)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = s.policy.Tx().Rollback(ctx)
	}()

	err = s.policy.DeletePluginPolicy(ctx, policyID)
	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	err = s.scheduler.Delete(ctx, policyID)
	if err != nil {
		return fmt.Errorf("failed to delete policy from scheduler: %w", err)
	}

	if err := s.policy.Tx().Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (s *Service[T]) GetPluginPolicies(
	ctx context.Context,
	pluginID types.PluginID,
	publicKey string,
	onlyActive bool,
) ([]types.PluginPolicy, error) {
	return s.policy.GetAllPluginPolicies(ctx, publicKey, pluginID, onlyActive)
}

func (s *Service[T]) GetPluginPolicy(ctx context.Context, policyID uuid.UUID) (*types.PluginPolicy, error) {
	return s.policy.GetPluginPolicy(ctx, policyID)
}
