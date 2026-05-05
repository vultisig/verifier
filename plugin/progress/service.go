package progress

import (
	"context"

	"github.com/google/uuid"
)

type Progress struct {
	Kind  string `json:"kind"`
	Value uint32 `json:"value"`
}

type Service interface {
	GetProgress(ctx context.Context, policyID uuid.UUID) (*Progress, error)
	GetProgressBatch(ctx context.Context, policyIDs []uuid.UUID) (map[uuid.UUID]*Progress, error)
}
