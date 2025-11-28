package safety

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/vultisig/verifier/internal/storage"
)

const (
	actionKeygen  = "keygen"
	actionKeysign = "keysign"
)

type Manager struct {
	db     storage.ControlFlagsRepository
	logger *logrus.Logger
}

func NewManager(db storage.ControlFlagsRepository, logger *logrus.Logger) *Manager {
	return &Manager{db: db, logger: logger}
}

func (m *Manager) EnforceKeygen(ctx context.Context, pluginID string) error {
	return m.enforce(ctx, pluginID, actionKeygen)
}

func (m *Manager) EnforceKeysign(ctx context.Context, pluginID string) error {
	return m.enforce(ctx, pluginID, actionKeysign)
}

func (m *Manager) enforce(ctx context.Context, pluginID, action string) error {
	globalKey := "global-" + action      // e.g. "global-keysign"
	pluginKey := pluginID + "-" + action // e.g. "dca-keysign"

	flags, err := m.db.GetControlFlags(ctx, globalKey, pluginKey)
	if err != nil {
		// choose fail-open or fail-closed; this is fail-open with loud log
		m.logger.Error("control flag check failed",
			"plugin", pluginID,
			"action", action,
			"err", err,
		)
		return nil
	}

	// default: missing key => allowed = true
	globalEnabled, ok := flags[globalKey]
	if !ok {
		globalEnabled = true
	}
	pluginEnabled, ok := flags[pluginKey]
	if !ok {
		pluginEnabled = true
	}

	if !globalEnabled {
		m.logger.WithFields(logrus.Fields{
			"key":    globalKey,
			"plugin": pluginID,
			"action": action,
		}).Warn("blocked by global control flag")
		return fmt.Errorf("%s disabled globally", action)
	}

	if !pluginEnabled {
		m.logger.WithFields(logrus.Fields{
			"key":    pluginKey,
			"plugin": pluginID,
			"action": action,
		}).Warn("blocked by plugin control flag")
		return fmt.Errorf("%s disabled for plugin %s", action, pluginID)
	}

	return nil
}
