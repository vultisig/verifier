package safety

import "context"

type ControlFlag struct {
	Key     string `json:"key"`
	Enabled bool   `json:"enabled"`
}

type Storage interface {
	GetControlFlags(ctx context.Context, k1, k2 string) (map[string]bool, error)
	UpsertControlFlags(ctx context.Context, flags []ControlFlag) error
}
