package types

type PolicySyncStatus int

const (
	NotSynced PolicySyncStatus = iota
	Synced    PolicySyncStatus = iota + 1
	Failed    PolicySyncStatus = iota + 2
)

type PluginPolicySync struct {
	PolicyID   string           `json:"policy_id" validate:"required"`
	Status     PolicySyncStatus `json:"status" validate:"required"`
	FailReason string           `json:"fail_reason"` // when synced is false, this field contains the reason for the failure
}
