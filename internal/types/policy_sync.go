package types

import (
	"github.com/google/uuid"
	ptypes "github.com/vultisig/verifier/types"
)

type PolicySyncStatus int
type PolicySyncType int

const (
	NotSynced PolicySyncStatus = iota
	Synced    PolicySyncStatus = iota + 1
	Failed    PolicySyncStatus = iota + 2
)

const (
	AddPolicy PolicySyncType = iota
	UpdatePolicy
	RemovePolicy
)

type PluginPolicySync struct {
	ID         uuid.UUID        `json:"id" validate:"required"`
	PolicyID   uuid.UUID        `json:"policy_id" validate:"required"`
	PluginID   ptypes.PluginID  `json:"plugin_id" validate:"required"`
	Signature  string           `json:"signature" validate:"required"`
	SyncType   PolicySyncType   `json:"sync_type" validate:"required"`
	Status     PolicySyncStatus `json:"status" validate:"required"`
	FailReason string           `json:"fail_reason"` // when synced is false, this field contains the reason for the failure
}
