package scheduler

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/vultisig/verifier/plugin/storage"
)

type Scheduler struct {
	PolicyID      uuid.UUID `json:"policy_id"`
	NextExecution time.Time `json:"next_execution"`
}

type Storage[T any] interface {
	Tx() storage.Tx
	GetByPolicy(ctx context.Context, policyID uuid.UUID) (Scheduler, error)
	Create(ctx context.Context, policyID uuid.UUID, next time.Time) error
	Delete(ctx context.Context, policyID uuid.UUID) error
	GetPending(ctx context.Context) ([]Scheduler, error)
	SetNext(ctx context.Context, policyID uuid.UUID, next time.Time) error
}
