package storage

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vultisig/verifier/plugin/tx_indexer/pkg/conv"
	"github.com/vultisig/verifier/plugin/tx_indexer/pkg/rpc"
	"github.com/vultisig/verifier/types"
	"github.com/vultisig/vultisig-go/common"
)

type TxIndexerRepo interface {
	SetStatus(ctx context.Context, id uuid.UUID, status TxStatus) error
	SetLost(ctx context.Context, id uuid.UUID) error
	SetSignedAndBroadcasted(ctx context.Context, id uuid.UUID, txHash string) error
	SetOnChainStatus(ctx context.Context, id uuid.UUID, status rpc.TxOnChainStatus) error
	GetPendingTxs(ctx context.Context) <-chan RowsStream[Tx]
	CreateTx(ctx context.Context, req CreateTxDto) (Tx, error)
	GetTxByID(ctx context.Context, id uuid.UUID) (Tx, error)
	GetTxsInTimeRange(ctx context.Context, policyID uuid.UUID, from, to time.Time) <-chan RowsStream[Tx]
	GetByPolicyID(ctx context.Context, policyID uuid.UUID, skip, take uint32) <-chan RowsStream[Tx]
	CountByPolicyID(ctx context.Context, policyID uuid.UUID) (uint32, error)
	GetByPluginIDAndPublicKey(ctx context.Context, pluginID types.PluginID, publicKey string, skip, take uint32) <-chan RowsStream[Tx]
	CountByPluginIDAndPublicKey(ctx context.Context, pluginID types.PluginID, publicKey string) (uint32, error)
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
	TokenID       string               `json:"token_id" validate:"required"`
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
	TokenID       string
	FromPublicKey string
	ToPublicKey   string
	ProposedTxHex string
}
