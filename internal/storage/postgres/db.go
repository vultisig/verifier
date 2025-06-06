package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/vultisig/verifier/internal/types"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"

	"github.com/vultisig/verifier/internal/storage"
)

var _ storage.DatabaseStorage = (*PostgresBackend)(nil)

type PostgresBackend struct {
	pool *pgxpool.Pool
}

const defaultTimeout = 10 * time.Second

type MigrationOptions struct {
	RunSystemMigrations   bool
	RunVerifierMigrations bool
}

func NewPostgresBackend(dsn string, opts *MigrationOptions) (*PostgresBackend, error) {
	logrus.Info("Connecting to database with DSN: ", dsn)
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	backend := &PostgresBackend{
		pool: pool,
	}

	// Apply default options if not provided
	if opts == nil {
		opts = &MigrationOptions{
			RunSystemMigrations:   true,
			RunVerifierMigrations: true,
		}
	}

	if err := backend.Migrate(opts); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return backend, nil
}

func (p *PostgresBackend) Close() error {
	p.pool.Close()

	return nil
}

func (p *PostgresBackend) Migrate(opts *MigrationOptions) error {
	logrus.Info("Starting database migration...")

	// Run system migrations first (plugin_policies table)
	if opts.RunSystemMigrations {
		systemMgr := NewSystemMigrationManager(p.pool)
		if err := systemMgr.Migrate(); err != nil {
			return fmt.Errorf("failed to run system migrations: %w", err)
		}
	}

	// Run verifier migrations (all other tables)
	if opts.RunVerifierMigrations {
		verifierMgr := NewVerifierMigrationManager(p.pool)
		if err := verifierMgr.Migrate(); err != nil {
			return fmt.Errorf("failed to run verifier migrations: %w", err)
		}
	}

	logrus.Info("Database migration completed successfully")
	return nil
}

func (p *PostgresBackend) CreateTransactionHistoryTx(ctx context.Context, dbTx pgx.Tx, tx types.TransactionHistory) (uuid.UUID, error) {
	query := `
        INSERT INTO transaction_history (
            policy_id, tx_body, tx_hash, status, metadata
        ) VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (tx_hash) DO UPDATE SET
            policy_id = EXCLUDED.policy_id,
            tx_body = EXCLUDED.tx_body,
            status = 'PENDING',
            metadata = EXCLUDED.metadata
		RETURNING id
    `
	var txID uuid.UUID
	err := dbTx.QueryRow(ctx, query,
		tx.PolicyID,
		tx.TxBody,
		tx.TxHash,
		tx.Status,
		tx.Metadata,
	).Scan(&txID)

	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create transaction history: %w", err)
	}

	return txID, nil
}

func (p *PostgresBackend) UpdateTransactionStatusTx(ctx context.Context, dbTx pgx.Tx, txID uuid.UUID, status types.TransactionStatus, metadata map[string]interface{}) error {
	query := `
        UPDATE transaction_history 
        SET status = $1, metadata = metadata || $2::jsonb, updated_at = NOW()
        WHERE id = $3
    `

	_, err := dbTx.Exec(ctx, query, status, metadata, txID)
	return err
}

func (p *PostgresBackend) CreateTransactionHistory(ctx context.Context, tx types.TransactionHistory) (uuid.UUID, error) {
	query := `
        INSERT INTO transaction_history (
            policy_id, tx_body, tx_hash, status, metadata
        ) VALUES ($1, $2, $3, $4, $5)
				RETURNING id
    `
	var txID uuid.UUID
	err := p.pool.QueryRow(ctx, query,
		tx.PolicyID,
		tx.TxBody,
		tx.TxHash,
		tx.Status,
		tx.Metadata,
	).Scan(&txID)

	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create transaction history: %w", err)
	}

	return txID, nil
}

func (p *PostgresBackend) UpdateTransactionStatus(ctx context.Context, txID uuid.UUID, status types.TransactionStatus, metadata map[string]interface{}) error {
	query := `
        UPDATE transaction_history 
        SET status = $1, metadata = metadata || $2::jsonb, updated_at = NOW()
        WHERE id = $3
    `

	_, err := p.pool.Exec(ctx, query, status, metadata, txID)
	return err

}

func (p *PostgresBackend) GetTransactionHistory(ctx context.Context, policyID uuid.UUID, transactionType string, take int, skip int) ([]types.TransactionHistory, int64, error) {
	query := `
        SELECT id, policy_id, tx_body, tx_hash, status, created_at, updated_at, metadata, error_message,
        COUNT(*) OVER() as total_count
        FROM transaction_history
        WHERE policy_id = $1
        AND metadata->>'transaction_type' = $2
        ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
    `

	rows, err := p.pool.Query(ctx, query, policyID, transactionType, take, skip)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var history []types.TransactionHistory
	var totalCount int64
	for rows.Next() {
		var tx types.TransactionHistory
		err := rows.Scan(
			&tx.ID,
			&tx.PolicyID,
			&tx.TxBody,
			&tx.TxHash,
			&tx.Status,
			&tx.CreatedAt,
			&tx.UpdatedAt,
			&tx.Metadata,
			&tx.ErrorMessage,
			&totalCount,
		)
		if err != nil {
			return nil, 0, err
		}
		history = append(history, tx)
	}

	return history, totalCount, nil
}

func (p *PostgresBackend) GetTransactionByHash(ctx context.Context, txHash string) (*types.TransactionHistory, error) {
	query := `
        SELECT 
            id, 
            policy_id, 
            tx_body, 
            tx_hash,
            status, 
            created_at, 
            updated_at, 
            metadata, 
            error_message
        FROM transaction_history
        WHERE tx_hash = $1
    `

	var tx types.TransactionHistory
	err := p.pool.QueryRow(ctx, query, txHash).Scan(
		&tx.ID,
		&tx.PolicyID,
		&tx.TxBody,
		&tx.TxHash,
		&tx.Status,
		&tx.CreatedAt,
		&tx.UpdatedAt,
		&tx.Metadata,
		&tx.ErrorMessage,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("transaction with Tx Hash %s not found", txHash)
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	return &tx, nil
}

func (p *PostgresBackend) CountTransactions(ctx context.Context, policyID uuid.UUID, status types.TransactionStatus, txType string) (int64, error) {
	var count int64
	query := `
		SELECT COUNT(*)
		FROM transaction_history
		WHERE policy_id = $1
		AND status = $2
		AND metadata->>'transaction_type' = $3
	`
	err := p.pool.QueryRow(ctx, query, policyID, status, txType).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count transactions: %w", err)
	}
	return count, nil
}

func (p *PostgresBackend) WithTransaction(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Roll back on error *or* panic.
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	if err := fn(ctx, tx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (p *PostgresBackend) Pool() *pgxpool.Pool {
	return p.pool
}
