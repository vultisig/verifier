package postgres

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
                        updated_at,
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

func decodeTxRow(rows pgx.Rows) (types.Tx, error) {
	var tx types.Tx
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
		return types.Tx{}, fmt.Errorf("rows.Scan: %w", err)
	}
	return tx, nil
}

func (p *PostgresBackend) SetStatus(c context.Context, id uuid.UUID, status types.TxStatus) error {
	ctx, cancel := context.WithTimeout(c, defaultTimeout)
	defer cancel()

	_, err := p.pool.Exec(
		ctx,
		`UPDATE tx_indexer SET status = $1 AND updated_at = now() WHERE id = $2`,
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
		`UPDATE tx_indexer SET lost = $1 AND updated_at = now() WHERE id = $2`,
		true,
		id,
	)
	if err != nil {
		return fmt.Errorf("p.pool.Exec: %w", err)
	}
	return nil
}

func (p *PostgresBackend) SetSignedAndBroadcasted(c context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(c, defaultTimeout)
	defer cancel()

	_, err := p.pool.Exec(
		ctx,
		`UPDATE tx_indexer SET status = $1 and status_onchain = $2 and broadcasted_at = now() AND updated_at = now()
                  WHERE id = $3`,
		types.TxSigned,
		types.TxOnChainPending,
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
		`UPDATE tx_indexer SET status_onchain = $1 AND updated_at = now() WHERE id = $2`,
		status,
		id,
	)
	if err != nil {
		return fmt.Errorf("p.pool.Exec: %w", err)
	}
	return nil
}

func (p *PostgresBackend) GetPendingTxs(ctx context.Context) <-chan types.TxErr {
	ch := make(chan types.TxErr)

	go func() {
		defer close(ch)

		rows, err := p.pool.Query(
			ctx,
			`SELECT * FROM tx_indexer WHERE status_onchain = $1 AND lost = $2`,
			types.TxOnChainPending,
			false,
		)
		if err != nil {
			ch <- types.TxErr{Err: fmt.Errorf("p.pool.Query: %w", err)}
			return
		}

		for rows.Next() {
			tx, er := decodeTxRow(rows)
			if er != nil {
				ch <- types.TxErr{Err: fmt.Errorf("decodeTxRow: %w", err)}
				return
			}

			ch <- types.TxErr{
				Tx: tx,
			}
		}
	}()

	return ch
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
		ChainID:          req.ChainID,
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
