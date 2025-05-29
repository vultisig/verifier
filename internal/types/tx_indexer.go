package types

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vultisig/verifier/common"
	"time"
)

type TxStatus string
type TxOnChainStatus string

const (
	TxProposed       TxStatus        = "PROPOSED"
	TxVerified       TxStatus        = "VERIFIED"
	TxSigned         TxStatus        = "SIGNED"
	TxOnChainPending TxOnChainStatus = "PENDING"
	TxOnChainSuccess TxOnChainStatus = "SUCCESS"
	TxOnChainFail    TxOnChainStatus = "FAIL"
)

type TxErr struct {
	Tx  Tx
	Err error
}

type Tx struct {
	ID               uuid.UUID        `json:"id" validate:"required"`
	PluginID         uuid.UUID        `json:"plugin_id" validate:"required"`
	TxHash           *string          `json:"tx_hash"`
	ChainID          common.Chain     `json:"chain_id" validate:"required"`
	PolicyID         uuid.UUID        `json:"policy_id" validate:"required"`
	FromPublicKey    string           `json:"from_public_key" validate:"required"`
	ProposedTxObject json.RawMessage  `json:"proposed_tx_object" validate:"required"`
	Status           TxStatus         `json:"status" validate:"required"`
	StatusOnChain    *TxOnChainStatus `json:"status_onchain"`
	Lost             bool             `json:"lost"`
	BroadcastedAt    *time.Time       `json:"broadcasted_at"`
	CreatedAt        time.Time        `json:"created_at"  validate:"required"`
	UpdatedAt        time.Time        `json:"updated_at" validate:"required"`
}

func (t Tx) Fields() logrus.Fields {
	return logrus.Fields{
		"id":              t.ID.String(),
		"plugin_id":       t.PluginID.String(),
		"tx_hash":         t.TxHash,
		"chain_id":        t.ChainID.String(),
		"policy_id":       t.PolicyID.String(),
		"from_public_key": t.FromPublicKey,
		"status":          t.Status,
		"status_onchain":  t.StatusOnChain,
		"lost":            t.Lost,
		"broadcasted_at":  t.BroadcastedAt,
		"created_at":      t.CreatedAt,
		"updated_at":      t.UpdatedAt,
	}
}

type CreateTxDto struct {
	PluginID         uuid.UUID
	ChainID          common.Chain
	PolicyID         uuid.UUID
	FromPublicKey    string
	ProposedTxObject json.RawMessage
}
