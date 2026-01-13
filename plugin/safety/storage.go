package safety

import "context"

type Storage interface {
	GetControlFlags(ctx context.Context, k1, k2 string) (map[string]bool, error)
}
