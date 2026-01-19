package safety

import (
	"context"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
)

var (
	ErrGloballyDisabled = errors.New("action disabled globally")
	ErrPluginDisabled   = errors.New("action disabled for plugin")
)

type Manager struct {
	db     Storage
	logger *logrus.Logger
}

func NewManager(db Storage, logger *logrus.Logger) *Manager {
	return &Manager{db: db, logger: logger}
}

func (m *Manager) EnforceKeysign(ctx context.Context, pluginID string) error {
	globalKey := GlobalKeysignKey()
	pluginKey := KeysignFlagKey(pluginID)

	flags, err := m.db.GetControlFlags(ctx, globalKey, pluginKey)
	if err != nil {
		m.logger.WithFields(logrus.Fields{
			"plugin": pluginID,
			"err":    err,
		}).Error("control flag check failed")
		return fmt.Errorf("GetControlFlags failed: %w", err)
	}

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
		}).Warn("blocked by global control flag")
		return fmt.Errorf("keysign: %w", ErrGloballyDisabled)
	}

	if !pluginEnabled {
		m.logger.WithFields(logrus.Fields{
			"key":    pluginKey,
			"plugin": pluginID,
		}).Warn("blocked by plugin control flag")
		return fmt.Errorf("keysign %s: %w", pluginID, ErrPluginDisabled)
	}

	return nil
}

func IsDisabledError(err error) bool {
	return errors.Is(err, ErrGloballyDisabled) || errors.Is(err, ErrPluginDisabled)
}
