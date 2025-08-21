package scheduler_pg

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vultisig/verifier/plugin/postgres"
	"github.com/vultisig/verifier/plugin/scheduler"
	"github.com/vultisig/verifier/plugin/storage"
)

type Repo struct {
	tx *postgres.TxHandler
}

func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{
		tx: postgres.NewTxHandler(pool),
	}
}

func (r *Repo) Tx() storage.Tx {
	return r.tx
}

func (r *Repo) Create(ctx context.Context, policyID uuid.UUID, next time.Time) error {
	_, err := r.tx.Try(ctx).Exec(ctx, `
		INSERT INTO scheduler (policy_id, next_execution)
		VALUES ($1, $2)
	`, policyID, next)
	if err != nil {
		return fmt.Errorf("failed to create scheduler entry: %w", err)
	}
	return nil
}

func (r *Repo) GetByPolicy(ctx context.Context, policyID uuid.UUID) (scheduler.Scheduler, error) {
	var sch scheduler.Scheduler
	err := r.tx.Try(ctx).QueryRow(ctx, `
		SELECT policy_id, next_execution
		FROM scheduler
		WHERE policy_id = $1
		LIMIT 1
	`, policyID).Scan(&sch.PolicyID, &sch.NextExecution)
	if err != nil {
		return scheduler.Scheduler{}, fmt.Errorf("failed to query sch by policy: %w", err)
	}

	return sch, nil
}

func (r *Repo) GetPending(ctx context.Context) ([]scheduler.Scheduler, error) {
	rows, err := r.tx.Try(ctx).Query(ctx, `
		SELECT policy_id, next_execution
		FROM scheduler
		WHERE next_execution <= $1
		ORDER BY next_execution
	`, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to query pending scheduler entries: %w", err)
	}
	defer rows.Close()

	var schs []scheduler.Scheduler
	for rows.Next() {
		var sch scheduler.Scheduler
		if err := rows.Scan(&sch.PolicyID, &sch.NextExecution); err != nil {
			return nil, fmt.Errorf("failed to scan scheduler entry: %w", err)
		}
		schs = append(schs, sch)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate over scheduler entries: %w", err)
	}

	return schs, nil
}

func (r *Repo) SetNext(ctx context.Context, policyID uuid.UUID, next time.Time) error {
	_, err := r.tx.Try(ctx).Exec(ctx, `
		UPDATE scheduler
		SET next_execution = $2
		WHERE policy_id = $1
	`, policyID, next)
	if err != nil {
		return fmt.Errorf("failed to update next execution time: %w", err)
	}
	return nil
}

func (r *Repo) Delete(ctx context.Context, policyID uuid.UUID) error {
	_, err := r.tx.Try(ctx).Exec(ctx, `
		DELETE FROM scheduler
		WHERE policy_id = $1
	`, policyID)
	if err != nil {
		return fmt.Errorf("failed to delete scheduler entry: %w", err)
	}
	return nil
}
