package vault

import "context"

// SafetyManager defines the interface for safety enforcement during keygen and keysign operations.
// Implementations can check control flags to block operations globally or per-plugin.
type SafetyManager interface {
	EnforceKeygen(ctx context.Context, pluginID string) error
	EnforceKeysign(ctx context.Context, pluginID string) error
}

// NoOpSafetyManager is a no-op implementation of SafetyManager.
// Use this when safety checks are not required (e.g., in plugins).
type NoOpSafetyManager struct{}

func (n *NoOpSafetyManager) EnforceKeygen(ctx context.Context, pluginID string) error {
	return nil
}

func (n *NoOpSafetyManager) EnforceKeysign(ctx context.Context, pluginID string) error {
	return nil
}
