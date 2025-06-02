package postgres

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/vultisig/verifier/internal/storage"
	"github.com/vultisig/verifier/internal/types"
	"time"
)

func (p *PostgresBackend) createTx(ctx context.Context, tx types.Tx) error {
	_, err := p.pool.Exec(ctx, `INSERT INTO tx_indexer (
                        id,
                        plugin_id,
                        tx_hash,
                        chain_id,
                        policy_id,
                        from_public_key,
                        proposed_tx_object,
                        status,
                        status_onchain,
                        lost,
                        broadcasted_at,
                        created_at,
                        updated_at
) VALUES (
          $1,
          $2,
          $3,
          $4,
          $5,
          $6,
          $7,
          $8,
          $9,
          $10,
          $11,
          $12,
          $13
)`, tx.ID,
		tx.PluginID,
		tx.TxHash,
		tx.ChainID,
		tx.PolicyID,
		tx.FromPublicKey,
		tx.ProposedTxObject,
		tx.Status,
		tx.StatusOnChain,
		tx.Lost,
		tx.BroadcastedAt,
		tx.CreatedAt,
		tx.UpdatedAt)
	if err != nil {
		return fmt.Errorf("p.pool.Exec: %w", err)
	}
	return nil
}

func (p *PostgresBackend) SetStatus(c context.Context, id uuid.UUID, status types.TxStatus) error {
	ctx, cancel := context.WithTimeout(c, defaultTimeout)
	defer cancel()

	_, err := p.pool.Exec(
		ctx,
		`UPDATE tx_indexer SET status = $1::tx_indexer_status,
                                   updated_at = now()
                               WHERE id = $2`,
		status,
		id,
	)
	if err != nil {
		return fmt.Errorf("p.pool.Exec: %w", err)
	}
	return nil
}

func (p *PostgresBackend) SetLost(c context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(c, defaultTimeout)
	defer cancel()

	_, err := p.pool.Exec(
		ctx,
		`UPDATE tx_indexer SET lost = $1,
                                   updated_at = now()
                               WHERE id = $2`,
		true,
		id,
	)
	if err != nil {
		return fmt.Errorf("p.pool.Exec: %w", err)
	}
	return nil
}

func (p *PostgresBackend) SetSignedAndBroadcasted(c context.Context, id uuid.UUID, txHash string) error {
	ctx, cancel := context.WithTimeout(c, defaultTimeout)
	defer cancel()

	_, err := p.pool.Exec(
		ctx,
		`UPDATE tx_indexer SET status = $1::tx_indexer_status,
                                   status_onchain = $2::tx_indexer_status_onchain,
                                   tx_hash = $3,
                                   broadcasted_at = now(),
                                   updated_at = now()
                               WHERE id = $4`,
		types.TxSigned,
		types.TxOnChainPending,
		txHash,
		id,
	)
	if err != nil {
		return fmt.Errorf("p.pool.Exec: %w", err)
	}
	return nil
}

func (p *PostgresBackend) SetOnChainStatus(c context.Context, id uuid.UUID, status types.TxOnChainStatus) error {
	ctx, cancel := context.WithTimeout(c, defaultTimeout)
	defer cancel()

	_, err := p.pool.Exec(
		ctx,
		`UPDATE tx_indexer SET status_onchain = $1::tx_indexer_status_onchain,
                                   updated_at = now()
                               WHERE id = $2`,
		status,
		id,
	)
	if err != nil {
		return fmt.Errorf("p.pool.Exec: %w", err)
	}
	return nil
}

func (p *PostgresBackend) GetPendingTxs(ctx context.Context) <-chan storage.RowsStream[types.Tx] {
	return storage.GetRowsStream[types.Tx](
		ctx,
		p.pool,
		types.TxFromRow,
		`SELECT * FROM tx_indexer WHERE status_onchain = $1 AND lost = $2`,
		types.TxOnChainPending,
		false,
	)
}

func (p *PostgresBackend) GetTxByID(c context.Context, id uuid.UUID) (types.Tx, error) {
	ctx, cancel := context.WithTimeout(c, defaultTimeout)
	defer cancel()

	rows, err := p.pool.Query(ctx, `SELECT * FROM tx_indexer WHERE id = $1 LIMIT 1`, id)
	if err != nil {
		return types.Tx{}, fmt.Errorf("p.pool.Query: %w", err)
	}
	if !rows.Next() {
		return types.Tx{}, fmt.Errorf("rows.Next: %w", err)
	}

	tx, err := types.TxFromRow(rows)
	if err != nil {
		return types.Tx{}, fmt.Errorf("types.TxFromRow: %w", err)
	}
	return tx, nil
}

func (p *PostgresBackend) CreateTx(c context.Context, req types.CreateTxDto) (types.Tx, error) {
	ctx, cancel := context.WithTimeout(c, defaultTimeout)
	defer cancel()

	now := time.Now()
	id, err := uuid.NewRandom()
	if err != nil {
		return types.Tx{}, fmt.Errorf("uuid.NewRandom: %w", err)
	}

	tx := types.Tx{
		ID:               id,
		PluginID:         req.PluginID,
		TxHash:           nil,
		ChainID:          int(req.ChainID),
		PolicyID:         req.PolicyID,
		FromPublicKey:    req.FromPublicKey,
		ProposedTxObject: req.ProposedTxObject,
		Status:           types.TxProposed,
		StatusOnChain:    nil,
		Lost:             false,
		BroadcastedAt:    nil,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	err = p.createTx(ctx, tx)
	if err != nil {
		return types.Tx{}, fmt.Errorf("p.pool.Exec: %w", err)
	}
	return tx, nil
}
