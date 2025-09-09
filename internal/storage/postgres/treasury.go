package postgres

import (
	"context"
	"fmt"
	"math/big"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/vultisig/verifier/types"
)

func (p *PostgresBackend) CreateTreasuryLedgerDebitFromFeeBatch(ctx context.Context, tx pgx.Tx, feeBatchId uuid.UUID, amount uint64, developerId uuid.UUID, ref string) (uuid.UUID, error) {
	id := uuid.New()
	query := `INSERT INTO treasury_ledger_fee_batch_collection (
		id,
		developer_id,
		amount, 
		ref, 
		fee_batch_id
	) VALUES ($1, $2, $3, $4, $5)`
	_, err := tx.Exec(ctx, query,
		id,
		developerId,
		amount,
		ref,
		feeBatchId,
	)
	if err != nil {
		return uuid.Nil, err
	}
	return id, err
}

func (p *PostgresBackend) CreateVultisigDuesDebitFromFeeBatch(ctx context.Context, tx pgx.Tx, feeBatchId uuid.UUID, amount uint64) (uuid.UUID, error) {
	id := uuid.New()
	query := `INSERT INTO treasury_ledger_vultisig_debit (
		id,
		amount, 
		fee_batch_id
	) VALUES ($1, $2, $3)`
	_, err := tx.Exec(ctx, query,
		id,
		amount,
		feeBatchId,
	)
	if err != nil {
		return uuid.Nil, err
	}
	return id, err
}

func (p *PostgresBackend) GetUnclaimedTreasuryLedgerRecords(ctx context.Context, developerId uuid.UUID) ([]types.TreasuryLedgerRecord, error) {
	// Get treasury ledger records (both debits and credits) that haven't been claimed in any batch yet
	// Each ledger item can only appear in 0 or 1 batches, so we find items not in treasury_batch_members
	query := `
		SELECT 
			tl.id, 
			tl.amount, 
			tl.type, 
			tl.subtype, 
			tl.ref, 
			tl.created_at
		FROM treasury_ledger tl
		WHERE tl.developer_id = $1
		AND NOT EXISTS (
			SELECT 1 
			FROM treasury_batch_members tbm
			WHERE tbm.treasury_ledger_record_id = tl.id
		)
		ORDER BY tl.created_at ASC`

	rows, err := p.pool.Query(ctx, query, developerId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []types.TreasuryLedgerRecord
	for rows.Next() {
		var record types.TreasuryLedgerRecord
		err := rows.Scan(
			&record.ID,
			&record.Amount,
			&record.Type,
			&record.Subtype,
			&record.Reference,
			&record.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return records, nil
}

func (p *PostgresBackend) createTreasuryLedgerBatch(ctx context.Context, tx pgx.Tx, developerId uuid.UUID, batchId uuid.UUID, amount uint64, ledgerEntries []types.TreasuryLedgerRecord) (*types.TreasuryBatchMembersView, error) {
	ledgerEntryIds := []uuid.UUID{}
	for _, entry := range ledgerEntries {
		if entry.DeveloperID != developerId {
			return nil, fmt.Errorf("developer id mismatch")
		}
	}

	query := `INSERT INTO treasury_batch (id, developer_id, status) VALUES ($1, $2, $3)`
	_, err := tx.Exec(ctx, query, batchId, developerId, types.TreasuryBatchStatusDraft)
	if err != nil {
		return nil, err
	}

	query = `INSERT INTO treasury_batch_members (batch_id, treasury_ledger_record_id) VALUES ($1, $2)`
	for _, entry := range ledgerEntries {
		ledgerEntryIds = append(ledgerEntryIds, entry.ID)
		_, err := tx.Exec(ctx, query, batchId, entry.ID)
		if err != nil {
			return nil, err
		}
	}

	query = `INSERT INTO treasury_ledger_developer_payout (treasury_batch_id, amount, developer_id) VALUES ($1, $2, $3)`
	_, err = tx.Exec(ctx, query, batchId, amount, developerId)
	if err != nil {
		return nil, err
	}

	return &types.TreasuryBatchMembersView{
		BatchID:     batchId,
		RecordCount: uint64(len(ledgerEntries)),
		TotalAmount: amount,
	}, nil
}

func (p *PostgresBackend) CreateTreasuryLedgerBatch(ctx context.Context, tx pgx.Tx, developerId uuid.UUID, txHash string) (*types.TreasuryBatchMembersView, error) {
	records, err := p.GetUnclaimedTreasuryLedgerRecords(ctx, developerId)
	if err != nil {
		return nil, err
	}

	totalAmount := big.NewInt(0)

	for _, record := range records {
		if record.Type == types.TreasuryLedgerTypeDebit {
			totalAmount = totalAmount.Add(totalAmount, big.NewInt(record.Amount))
		} else {
			totalAmount = totalAmount.Sub(totalAmount, big.NewInt(record.Amount))
		}
	}

	if totalAmount.Sign() == 1 {
		batchId := uuid.New()
		return p.createTreasuryLedgerBatch(ctx, tx, developerId, batchId, uint64(totalAmount.Int64()), records)
	} else {
		return nil, nil
	}
}
