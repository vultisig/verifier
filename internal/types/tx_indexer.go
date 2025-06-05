package types

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/sirupsen/logrus"
	"github.com/vultisig/mobile-tss-lib/tss"
	"github.com/vultisig/verifier/common"
	"github.com/vultisig/verifier/internal/conv"
	"github.com/vultisig/verifier/types"
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
	ID       uuid.UUID      `json:"id" validate:"required"`
	PluginID types.PluginID `json:"plugin_id" validate:"required"`
	TxHash   *string        `json:"tx_hash"`
	// not common.Chain type to avoid custom JSON marshaling to string
	ChainID       int              `json:"chain_id" validate:"required"`
	PolicyID      uuid.UUID        `json:"policy_id" validate:"required"`
	FromPublicKey string           `json:"from_public_key" validate:"required"`
	ProposedTxHex string           `json:"proposed_tx_hex" validate:"required"`
	Status        TxStatus         `json:"status" validate:"required"`
	StatusOnChain *TxOnChainStatus `json:"status_onchain"`
	Lost          bool             `json:"lost"`
	BroadcastedAt *time.Time       `json:"broadcasted_at"`
	CreatedAt     time.Time        `json:"created_at"  validate:"required"`
	UpdatedAt     time.Time        `json:"updated_at" validate:"required"`
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
		&tx.ProposedTxHex,
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
		"plugin_id":       t.PluginID,
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
	PluginID      types.PluginID
	ChainID       common.Chain
	PolicyID      uuid.UUID
	FromPublicKey string
	ProposedTxHex string
}

type TxIndexerRpc interface {
	GetTxStatus(ctx context.Context, txHash string) (TxOnChainStatus, error)
}

type TxIndexerTss interface {
	// ComputeTxHash
	// we can't use proposedTxObject to compute the hash because it doesn't include the signature,
	// we need to properly decode tx bytes, append signature to it, and compute hash using the particular chain library
	ComputeTxHash(proposedTxHex string, sigs []tss.KeysignResponse) (string, error)
}

var ErrChainNotImplemented = errors.New("chain not implemented")
