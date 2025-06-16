package storage

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vultisig/verifier/common"
	"github.com/vultisig/verifier/tx_indexer/pkg/conv"
	"github.com/vultisig/verifier/tx_indexer/pkg/rpc"
	"github.com/vultisig/verifier/types"
)

type TxIndexerRepo interface {
	SetStatus(ctx context.Context, id uuid.UUID, status TxStatus) error
	SetLost(ctx context.Context, id uuid.UUID) error
	SetSignedAndBroadcasted(ctx context.Context, id uuid.UUID, txHash string) error
	SetOnChainStatus(ctx context.Context, id uuid.UUID, status rpc.TxOnChainStatus) error
	GetPendingTxs(ctx context.Context) <-chan RowsStream[Tx]
	CreateTx(ctx context.Context, req CreateTxDto) (Tx, error)
	GetTxByID(ctx context.Context, id uuid.UUID) (Tx, error)
	GetTxInTimeRange(
		ctx context.Context,
		pluginID types.PluginID,
		policyID uuid.UUID,
		recipientPublicKey string,
		from, to time.Time,
	) (Tx, error)
}

type TxStatus string

const (
	TxProposed TxStatus = "PROPOSED"
	TxVerified TxStatus = "VERIFIED"
	TxSigned   TxStatus = "SIGNED"
)

var ErrNoTx = errors.New("transaction not found")

type Tx struct {
	ID            uuid.UUID            `json:"id" validate:"required"`
	PluginID      types.PluginID       `json:"plugin_id" validate:"required"`
	TxHash        *string              `json:"tx_hash"`
	ChainID       int                  `json:"chain_id" validate:"required"`
	PolicyID      uuid.UUID            `json:"policy_id" validate:"required"`
	FromPublicKey string               `json:"from_public_key" validate:"required"`
	ToPublicKey   string               `json:"to_public_key" validate:"required"`
	ProposedTxHex string               `json:"proposed_tx_hex" validate:"required"`
	Status        TxStatus             `json:"status" validate:"required"`
	StatusOnChain *rpc.TxOnChainStatus `json:"status_onchain"`
	Lost          bool                 `json:"lost"`
	BroadcastedAt *time.Time           `json:"broadcasted_at"`
	CreatedAt     time.Time            `json:"created_at"  validate:"required"`
	UpdatedAt     time.Time            `json:"updated_at" validate:"required"`
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
	ToPublicKey   string
	ProposedTxHex string
}
