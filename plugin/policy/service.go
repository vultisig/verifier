package policy

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vultisig/verifier/plugin/progress"
	"github.com/vultisig/verifier/plugin/scheduler"
	"github.com/vultisig/verifier/types"
)

var _ Service = (*Policy)(nil)

type Service interface {
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
	GetProgress(ctx context.Context, policyID uuid.UUID) (*progress.Progress, error)
	GetProgressBatch(ctx context.Context, policyIDs []uuid.UUID) (map[uuid.UUID]*progress.Progress, error)
}

type Policy struct {
	repo      Storage
	scheduler scheduler.Service
	progress  progress.Service
	logger    *logrus.Logger
}

func NewPolicyService(
	repo Storage,
	scheduler scheduler.Service,
	progress progress.Service,
	logger *logrus.Logger,
) (*Policy, error) {
	return &Policy{
		repo:      repo,
		scheduler: scheduler,
		progress:  progress,
		logger:    logger.WithField("pkg", "policy").Logger,
	}, nil
}

func (p *Policy) CreatePolicy(c context.Context, policy types.PluginPolicy) (*types.PluginPolicy, error) {
	ctx, err := p.repo.Tx().Begin(c)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = p.repo.Tx().Rollback(ctx)
	}()

	// Insert policy
	newPolicy, err := p.repo.InsertPluginPolicy(ctx, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to insert policy: %w", err)
	}

	err = p.scheduler.Create(ctx, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to create policy in scheduler: %w", err)
	}

	if err := p.repo.Tx().Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return newPolicy, nil
}

func (p *Policy) UpdatePolicy(c context.Context, policy types.PluginPolicy) (*types.PluginPolicy, error) {
	ctx, err := p.repo.Tx().Begin(c)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = p.repo.Tx().Rollback(ctx)
	}()

	oldPolicy, err := p.repo.GetPluginPolicy(ctx, policy.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plugin policy: %w", err)
	}

	err = p.scheduler.Update(ctx, *oldPolicy, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to update policy in scheduler: %w", err)
	}

	// Update policy with tx
	updatedPolicy, err := p.repo.UpdatePluginPolicy(ctx, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to update policy: %w", err)
	}

	if err := p.repo.Tx().Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return updatedPolicy, nil
}

func (p *Policy) DeletePolicy(c context.Context, policyID uuid.UUID, signature string) error {
	ctx, err := p.repo.Tx().Begin(c)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = p.repo.Tx().Rollback(ctx)
	}()

	err = p.repo.DeletePluginPolicy(ctx, policyID)
	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	err = p.scheduler.Delete(ctx, policyID)
	if err != nil {
		return fmt.Errorf("failed to delete policy from scheduler: %w", err)
	}

	if err := p.repo.Tx().Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (p *Policy) GetPluginPolicies(
	ctx context.Context,
	pluginID types.PluginID,
	publicKey string,
	onlyActive bool,
) ([]types.PluginPolicy, error) {
	return p.repo.GetAllPluginPolicies(ctx, publicKey, pluginID, onlyActive)
}

func (p *Policy) GetPluginPolicy(ctx context.Context, policyID uuid.UUID) (*types.PluginPolicy, error) {
	return p.repo.GetPluginPolicy(ctx, policyID)
}

func (p *Policy) GetProgress(ctx context.Context, policyID uuid.UUID) (*progress.Progress, error) {
	return p.progress.GetProgress(ctx, policyID)
}

func (p *Policy) GetProgressBatch(ctx context.Context, policyIDs []uuid.UUID) (map[uuid.UUID]*progress.Progress, error) {
	return p.progress.GetProgressBatch(ctx, policyIDs)
}
