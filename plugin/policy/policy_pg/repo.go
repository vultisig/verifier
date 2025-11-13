package policy_pg

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vultisig/verifier/plugin/postgres"
	"github.com/vultisig/verifier/plugin/storage"

	"github.com/vultisig/verifier/types"
)

func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{
		tx: postgres.NewTxHandler(pool),
	}
}

type Repo struct {
	tx *postgres.TxHandler
}

func (r *Repo) Tx() storage.Tx {
	return r.tx
}

func (r *Repo) GetPluginPolicy(ctx context.Context, id uuid.UUID) (*types.PluginPolicy, error) {
	var policy types.PluginPolicy
	query := `
        SELECT id, public_key, plugin_id, plugin_version, policy_version, signature, active,  recipe
        FROM plugin_policies 
        WHERE id = $1`

	err := r.tx.Pool().QueryRow(ctx, query, id).Scan(
		&policy.ID,
		&policy.PublicKey,
		&policy.PluginID,
		&policy.PluginVersion,
		&policy.PolicyVersion,
		&policy.Signature,
		&policy.Active,
		&policy.Recipe,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}
	return &policy, nil
}

func (r *Repo) GetAllPluginPolicies(ctx context.Context, publicKey string, pluginID types.PluginID, onlyActive bool) ([]types.PluginPolicy, error) {
	query := `
  	SELECT id, public_key,  plugin_id, plugin_version, policy_version, signature, active, recipe
		FROM plugin_policies
		WHERE public_key = $1
		AND plugin_id = $2`

	if onlyActive {
		query += ` AND active = true`
	}

	rows, err := r.tx.Pool().Query(ctx, query, publicKey, pluginID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var policies []types.PluginPolicy
	for rows.Next() {
		var policy types.PluginPolicy
		err := rows.Scan(
			&policy.ID,
			&policy.PublicKey,
			&policy.PluginID,
			&policy.PluginVersion,
			&policy.PolicyVersion,
			&policy.Signature,
			&policy.Active,
			&policy.Recipe,
		)
		if err != nil {
			return nil, err
		}
		policies = append(policies, policy)
	}

	return policies, nil
}

func (r *Repo) InsertPluginPolicy(ctx context.Context, policy types.PluginPolicy) (*types.PluginPolicy, error) {
	query := `
  	INSERT INTO plugin_policies (
      id, public_key, plugin_id, plugin_version, policy_version, signature, active, recipe
    ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    RETURNING id, public_key,  plugin_id, plugin_version, policy_version, signature, active, recipe
	`

	var insertedPolicy types.PluginPolicy
	err := r.tx.Try(ctx).QueryRow(ctx, query,
		policy.ID,
		policy.PublicKey,
		policy.PluginID,
		policy.PluginVersion,
		policy.PolicyVersion,
		policy.Signature,
		policy.Active,
		policy.Recipe,
	).Scan(
		&insertedPolicy.ID,
		&insertedPolicy.PublicKey,
		&insertedPolicy.PluginID,
		&insertedPolicy.PluginVersion,
		&insertedPolicy.PolicyVersion,
		&insertedPolicy.Signature,
		&insertedPolicy.Active,
		&insertedPolicy.Recipe,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert policy: %w", err)
	}

	return &insertedPolicy, nil
}

func (r *Repo) UpdatePluginPolicy(ctx context.Context, policy types.PluginPolicy) (*types.PluginPolicy, error) {
	query := `
		UPDATE plugin_policies 
		SET plugin_version = $2,
		    policy_version = $3,
			signature = $4,
			active = $5,
			recipe = $6
		WHERE id = $1
		RETURNING id, public_key, plugin_id, plugin_version, policy_version, signature, active, recipe
	`

	var updatedPolicy types.PluginPolicy
	err := r.tx.Try(ctx).QueryRow(ctx, query,
		policy.ID,
		policy.PluginVersion,
		policy.PolicyVersion,
		policy.Signature,
		policy.Active,
		policy.Recipe,
	).Scan(
		&updatedPolicy.ID,
		&updatedPolicy.PublicKey,
		&updatedPolicy.PluginID,
		&updatedPolicy.PluginVersion,
		&updatedPolicy.PolicyVersion,
		&updatedPolicy.Signature,
		&updatedPolicy.Active,
		&updatedPolicy.Recipe,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("policy not found with ID: %s", policy.ID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update policy: %w", err)
	}

	return &updatedPolicy, nil
}

func (r *Repo) DeletePluginPolicy(ctx context.Context, id uuid.UUID) error {
	_, err := r.tx.Try(ctx).Exec(ctx, `
	UPDATE plugin_policies
	SET deleted = true
	WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	return nil
}
