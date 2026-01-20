package safety

import (
	"context"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/vultisig/verifier/internal/storage"
	psafety "github.com/vultisig/verifier/plugin/safety"
)

var (
	ErrGloballyDisabled = errors.New("action disabled globally")
	ErrPluginDisabled   = errors.New("action disabled for plugin")
	ErrUnknownAction    = errors.New("unknown action")
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
	var globalKey, pluginKey string
	switch action {
	case actionKeysign:
		globalKey = psafety.GlobalKeysignKey()
		pluginKey = psafety.KeysignFlagKey(pluginID)
	case actionKeygen:
		globalKey = psafety.GlobalKeygenKey()
		pluginKey = psafety.KeygenFlagKey(pluginID)
	default:
		return fmt.Errorf("%s: %w", action, ErrUnknownAction)
	}

	flags, err := m.db.GetControlFlags(ctx, globalKey, pluginKey)
	if err != nil {
		m.logger.WithFields(logrus.Fields{
			"plugin": pluginID,
			"action": action,
			"err":    err,
		}).Error("control flag check failed")
		return fmt.Errorf("GetControlFlags failed: %w", err)
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
		return fmt.Errorf("%s: %w", action, ErrGloballyDisabled)
	}

	if !pluginEnabled {
		m.logger.WithFields(logrus.Fields{
			"key":    pluginKey,
			"plugin": pluginID,
			"action": action,
		}).Warn("blocked by plugin control flag")
		return fmt.Errorf("%s %s: %w", action, pluginID, ErrPluginDisabled)
	}

	return nil
}

func IsDisabledError(err error) bool {
	return errors.Is(err, ErrGloballyDisabled) || errors.Is(err, ErrPluginDisabled)
}
