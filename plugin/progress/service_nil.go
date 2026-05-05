package progress

import (
	"context"

	"github.com/google/uuid"
)

// NilService implements Service for plugins where progress tracking is not required
type NilService struct{}

func NewNilService() *NilService {
	return &NilService{}
}

func (s *NilService) GetProgress(_ context.Context, _ uuid.UUID) (*Progress, error) {
	return nil, nil
}

func (s *NilService) GetProgressBatch(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]*Progress, error) {
	return nil, nil
}
