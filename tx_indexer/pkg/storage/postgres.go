package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vultisig/verifier/tx_indexer/pkg/rpc"
	"github.com/vultisig/verifier/types"
)

type PostgresTxIndexStore struct {
	pool *pgxpool.Pool
}

const defaultTimeout = 10 * time.Second

func NewPostgresTxIndexStore(c context.Context, dsn string) (*PostgresTxIndexStore, error) {
	ctx, cancel := context.WithTimeout(c, defaultTimeout)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("pgxpool.New: %w", err)
	}

	err = pool.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("pool.Ping: %w", err)
	}

	return &PostgresTxIndexStore{
		pool: pool,
	}, nil
}

func (p *PostgresTxIndexStore) createTx(ctx context.Context, tx Tx) error {
	_, err := p.pool.Exec(ctx, `INSERT INTO tx_indexer (
                        id,
                        plugin_id,
                        tx_hash,
                        chain_id,
                        policy_id,
                        from_public_key,
                        to_public_key,
                        proposed_tx_hex,
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
          $13,
          $14
)`, tx.ID,
		tx.PluginID,
		tx.TxHash,
		tx.ChainID,
		tx.PolicyID,
		tx.FromPublicKey,
		tx.ToPublicKey,
		tx.ProposedTxHex,
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

func (p *PostgresTxIndexStore) SetStatus(c context.Context, id uuid.UUID, status TxStatus) error {
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

func (p *PostgresTxIndexStore) SetLost(c context.Context, id uuid.UUID) error {
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

func (p *PostgresTxIndexStore) SetSignedAndBroadcasted(c context.Context, id uuid.UUID, txHash string) error {
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
		TxSigned,
		rpc.TxOnChainPending,
		txHash,
		id,
	)
	if err != nil {
		return fmt.Errorf("p.pool.Exec: %w", err)
	}
	return nil
}

func (p *PostgresTxIndexStore) SetOnChainStatus(c context.Context, id uuid.UUID, status rpc.TxOnChainStatus) error {
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

func (p *PostgresTxIndexStore) GetPendingTxs(ctx context.Context) <-chan RowsStream[Tx] {
	return GetRowsStream[Tx](
		ctx,
		p.pool,
		TxFromRow,
		`SELECT * FROM tx_indexer WHERE status_onchain = $1 AND lost = $2`,
		rpc.TxOnChainPending,
		false,
	)
}

func (p *PostgresTxIndexStore) GetTxByID(c context.Context, id uuid.UUID) (Tx, error) {
	ctx, cancel := context.WithTimeout(c, defaultTimeout)
	defer cancel()

	rows, err := p.pool.Query(ctx, `SELECT * FROM tx_indexer WHERE id = $1 LIMIT 1`, id)
	if err != nil {
		return Tx{}, fmt.Errorf("p.pool.Query: %w", err)
	}
	if !rows.Next() {
		return Tx{}, ErrNoTx
	}

	tx, err := TxFromRow(rows)
	if err != nil {
		return Tx{}, fmt.Errorf("TxFromRow: %w", err)
	}
	return tx, nil
}

func (p *PostgresTxIndexStore) GetTxInTimeRange(
	c context.Context,
	pluginID types.PluginID,
	policyID uuid.UUID,
	recipientPublicKey string,
	from, to time.Time,
) (Tx, error) {
	ctx, cancel := context.WithTimeout(c, defaultTimeout)
	defer cancel()

	rows, err := p.pool.Query(ctx, `
		SELECT * FROM tx_indexer
		WHERE plugin_id = $1 AND policy_id = $2 AND to_public_key = $3
		  AND created_at >= $4 AND created_at <= $5
		LIMIT 1`, pluginID, policyID, recipientPublicKey, from, to)
	if err != nil {
		return Tx{}, fmt.Errorf("p.pool.Query: %w", err)
	}
	if !rows.Next() {
		return Tx{}, ErrNoTx
	}

	tx, err := TxFromRow(rows)
	if err != nil {
		return Tx{}, fmt.Errorf("TxFromRow: %w", err)
	}
	return tx, nil
}

func (p *PostgresTxIndexStore) CreateTx(c context.Context, req CreateTxDto) (Tx, error) {
	ctx, cancel := context.WithTimeout(c, defaultTimeout)
	defer cancel()

	now := time.Now()
	id, err := uuid.NewRandom()
	if err != nil {
		return Tx{}, fmt.Errorf("uuid.NewRandom: %w", err)
	}

	tx := Tx{
		ID:            id,
		PluginID:      req.PluginID,
		TxHash:        nil,
		ChainID:       int(req.ChainID),
		PolicyID:      req.PolicyID,
		FromPublicKey: req.FromPublicKey,
		ToPublicKey:   req.ToPublicKey,
		ProposedTxHex: req.ProposedTxHex,
		Status:        TxProposed,
		StatusOnChain: nil,
		Lost:          false,
		BroadcastedAt: nil,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	err = p.createTx(ctx, tx)
	if err != nil {
		return Tx{}, fmt.Errorf("p.pool.Exec: %w", err)
	}
	return tx, nil
}

type RowsStream[T any] struct {
	Row T
	Err error
}

// GetRowsStream
// TLDR: fetch rows from db with a non-buffered channel to control concurrency by data-consumer
func GetRowsStream[T any](
	ctx context.Context,
	pool *pgxpool.Pool,
	scanRow func(rows pgx.Rows) (T, error),
	sql string,
	args ...any,
) <-chan RowsStream[T] {
	ch := make(chan RowsStream[T])

	go func() {
		defer close(ch)

		rows, err := pool.Query(
			ctx,
			sql,
			args...,
		)
		if err != nil {
			ch <- RowsStream[T]{Err: fmt.Errorf("p.pool.Query: %w", err)}
			return
		}
		defer rows.Close()

		for rows.Next() {
			item, er := scanRow(rows)
			if er != nil {
				ch <- RowsStream[T]{Err: fmt.Errorf("scanRow: %w", er)}
				return
			}

			ch <- RowsStream[T]{Row: item}
		}
		err = rows.Err()
		if err != nil {
			ch <- RowsStream[T]{Err: fmt.Errorf("rows.Err: %w", err)}
			return
		}
	}()

	return ch
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
		&tx.ToPublicKey,
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
