package types

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/sirupsen/logrus"
	"github.com/vultisig/verifier/common"
	"github.com/vultisig/verifier/internal/conv"
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

type Tx struct {
	ID       uuid.UUID `json:"id" validate:"required"`
	PluginID uuid.UUID `json:"plugin_id" validate:"required"`
	TxHash   *string   `json:"tx_hash"`
	// not common.Chain type to avoid custom JSON marshaling to string
	ChainID          int              `json:"chain_id" validate:"required"`
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

func TxFromRow(rows pgx.Rows) (Tx, error) {
	var tx Tx
	err := rows.Scan(
		&tx.ID,
		&tx.PluginID,
		&tx.TxHash,
		&tx.ChainID,
		&tx.PolicyID,
		&tx.FromPublicKey,
		&tx.ProposedTxObject,
		&tx.Status,
		&tx.StatusOnChain,
		&tx.Lost,
		&tx.BroadcastedAt,
		&tx.CreatedAt,
		&tx.UpdatedAt,
	)
	if err != nil {
		return Tx{}, fmt.Errorf("rows.Scan: %w", err)
	}
	return tx, nil
}

func (t *Tx) Fields() logrus.Fields {
	return logrus.Fields{
		"id":              t.ID.String(),
		"plugin_id":       t.PluginID.String(),
		"tx_hash":         conv.FromPtr(t.TxHash),
		"chain_id":        t.ChainID,
		"chain_id_str":    common.Chain(t.ChainID).String(),
		"policy_id":       t.PolicyID.String(),
		"from_public_key": t.FromPublicKey,
		"status":          t.Status,
		"status_onchain":  conv.FromPtr(t.StatusOnChain),
		"lost":            t.Lost,
		"broadcasted_at":  conv.FromPtr(t.BroadcastedAt).String(),
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
