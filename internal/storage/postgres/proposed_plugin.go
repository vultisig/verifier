package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	itypes "github.com/vultisig/verifier/internal/types"
)

func (p *PostgresBackend) CreateProposedPlugin(ctx context.Context, tx pgx.Tx, params itypes.ProposedPluginCreateParams) (*itypes.ProposedPlugin, error) {
	query := `
		INSERT INTO proposed_plugins (plugin_id, public_key, title, description, server_endpoint, category, supported_chains, pricing_model, contact_email, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING plugin_id, public_key, title, description, server_endpoint, category, supported_chains, pricing_model, contact_email, notes, status, created_at, updated_at
	`

	var record itypes.ProposedPlugin
	err := tx.QueryRow(ctx, query,
		params.PluginID,
		params.PublicKey,
		params.Title,
		params.ShortDescription,
		params.ServerEndpoint,
		string(params.Category),
		params.SupportedChains,
		params.PricingModel,
		params.ContactEmail,
		params.Notes,
	).Scan(
		&record.PluginID,
		&record.PublicKey,
		&record.Title,
		&record.ShortDescription,
		&record.ServerEndpoint,
		&record.Category,
		&record.SupportedChains,
		&record.PricingModel,
		&record.ContactEmail,
		&record.Notes,
		&record.Status,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create proposed plugin: %w", err)
	}

	return &record, nil
}

func (p *PostgresBackend) CreateProposedPluginImage(ctx context.Context, tx pgx.Tx, params itypes.ProposedPluginImageCreateParams) (*itypes.ProposedPluginImage, error) {
	query := `
		INSERT INTO proposed_plugin_images (id, plugin_id, image_type, s3_path, image_order, uploaded_by_public_key, content_type, filename)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, plugin_id, image_type, s3_path, image_order, uploaded_by_public_key, visible, deleted, content_type, filename, created_at, updated_at
	`

	var record itypes.ProposedPluginImage
	err := tx.QueryRow(ctx, query,
		params.ID,
		params.PluginID,
		params.ImageType,
		params.S3Path,
		params.ImageOrder,
		params.UploadedByPublicKey,
		params.ContentType,
		params.Filename,
	).Scan(
		&record.ID,
		&record.PluginID,
		&record.ImageType,
		&record.S3Path,
		&record.ImageOrder,
		&record.UploadedByPublicKey,
		&record.Visible,
		&record.Deleted,
		&record.ContentType,
		&record.Filename,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create proposed plugin image: %w", err)
	}

	return &record, nil
}

func (p *PostgresBackend) GetProposedPluginByOwner(ctx context.Context, publicKey, pluginID string) (*itypes.ProposedPlugin, error) {
	query := `
		SELECT plugin_id, public_key, title, description, server_endpoint, category, supported_chains, pricing_model, contact_email, notes, status, created_at, updated_at
		FROM proposed_plugins
		WHERE public_key = $1 AND plugin_id = $2
	`

	var record itypes.ProposedPlugin
	err := p.pool.QueryRow(ctx, query, publicKey, pluginID).Scan(
		&record.PluginID,
		&record.PublicKey,
		&record.Title,
		&record.ShortDescription,
		&record.ServerEndpoint,
		&record.Category,
		&record.SupportedChains,
		&record.PricingModel,
		&record.ContactEmail,
		&record.Notes,
		&record.Status,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get proposed plugin: %w", err)
	}

	return &record, nil
}

func (p *PostgresBackend) ListProposedPluginsByPublicKey(ctx context.Context, publicKey string) ([]itypes.ProposedPlugin, error) {
	query := `
		SELECT plugin_id, public_key, title, description, server_endpoint, category, supported_chains, pricing_model, contact_email, notes, status, created_at, updated_at
		FROM proposed_plugins
		WHERE public_key = $1
		ORDER BY created_at DESC
	`

	rows, err := p.pool.Query(ctx, query, publicKey)
	if err != nil {
		return nil, fmt.Errorf("list proposed plugins: %w", err)
	}
	defer rows.Close()

	var records []itypes.ProposedPlugin
	for rows.Next() {
		var r itypes.ProposedPlugin
		err := rows.Scan(
			&r.PluginID,
			&r.PublicKey,
			&r.Title,
			&r.ShortDescription,
			&r.ServerEndpoint,
			&r.Category,
			&r.SupportedChains,
			&r.PricingModel,
			&r.ContactEmail,
			&r.Notes,
			&r.Status,
			&r.CreatedAt,
			&r.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan proposed plugin: %w", err)
		}
		records = append(records, r)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate proposed plugins: %w", err)
	}

	return records, nil
}

func (p *PostgresBackend) ListProposedPluginImages(ctx context.Context, pluginID string) ([]itypes.ProposedPluginImage, error) {
	query := `
		SELECT id, plugin_id, image_type, s3_path, image_order, uploaded_by_public_key, visible, deleted, content_type, filename, created_at, updated_at
		FROM proposed_plugin_images
		WHERE plugin_id = $1 AND deleted = false AND visible = true
		ORDER BY image_type, image_order ASC
	`

	rows, err := p.pool.Query(ctx, query, pluginID)
	if err != nil {
		return nil, fmt.Errorf("list proposed plugin images: %w", err)
	}
	defer rows.Close()

	var records []itypes.ProposedPluginImage
	for rows.Next() {
		var r itypes.ProposedPluginImage
		err := rows.Scan(
			&r.ID,
			&r.PluginID,
			&r.ImageType,
			&r.S3Path,
			&r.ImageOrder,
			&r.UploadedByPublicKey,
			&r.Visible,
			&r.Deleted,
			&r.ContentType,
			&r.Filename,
			&r.CreatedAt,
			&r.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan proposed plugin image: %w", err)
		}
		records = append(records, r)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate proposed plugin images: %w", err)
	}

	return records, nil
}

func (p *PostgresBackend) PluginIDExistsInPlugins(ctx context.Context, pluginID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM plugins WHERE id = $1)`
	var exists bool
	err := p.pool.QueryRow(ctx, query, pluginID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check plugin existence: %w", err)
	}
	return exists, nil
}

func (p *PostgresBackend) PluginIDExistsInProposals(ctx context.Context, pluginID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM proposed_plugins WHERE plugin_id = $1 AND status IN ('submitted', 'approved', 'listed'))`
	var exists bool
	err := p.pool.QueryRow(ctx, query, pluginID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check proposed plugin existence: %w", err)
	}
	return exists, nil
}

func (p *PostgresBackend) CountActiveProposalsByPublicKey(ctx context.Context, publicKey string) (int, error) {
	query := `SELECT COUNT(*) FROM proposed_plugins WHERE public_key = $1 AND status IN ('submitted', 'approved')`
	var count int
	err := p.pool.QueryRow(ctx, query, publicKey).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count active proposals: %w", err)
	}
	return count, nil
}

func (p *PostgresBackend) IsProposedPluginApproved(ctx context.Context, pluginID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM proposed_plugins WHERE plugin_id = $1 AND status = 'approved')`
	var exists bool
	err := p.pool.QueryRow(ctx, query, pluginID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check proposed plugin approved: %w", err)
	}
	return exists, nil
}

func (p *PostgresBackend) GetProposedPlugin(ctx context.Context, pluginID string) (*itypes.ProposedPlugin, error) {
	query := `
		SELECT plugin_id, public_key, title, description, server_endpoint, category, supported_chains, pricing_model, contact_email, notes, status, created_at, updated_at
		FROM proposed_plugins
		WHERE plugin_id = $1
	`

	var record itypes.ProposedPlugin
	err := p.pool.QueryRow(ctx, query, pluginID).Scan(
		&record.PluginID,
		&record.PublicKey,
		&record.Title,
		&record.ShortDescription,
		&record.ServerEndpoint,
		&record.Category,
		&record.SupportedChains,
		&record.PricingModel,
		&record.ContactEmail,
		&record.Notes,
		&record.Status,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get proposed plugin: %w", err)
	}

	return &record, nil
}

func (p *PostgresBackend) ListAllProposedPlugins(ctx context.Context) ([]itypes.ProposedPlugin, error) {
	query := `
		SELECT plugin_id, public_key, title, description, server_endpoint, category, supported_chains, pricing_model, contact_email, notes, status, created_at, updated_at
		FROM proposed_plugins
		ORDER BY created_at DESC
	`

	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list all proposed plugins: %w", err)
	}
	defer rows.Close()

	var records []itypes.ProposedPlugin
	for rows.Next() {
		var r itypes.ProposedPlugin
		err := rows.Scan(
			&r.PluginID,
			&r.PublicKey,
			&r.Title,
			&r.ShortDescription,
			&r.ServerEndpoint,
			&r.Category,
			&r.SupportedChains,
			&r.PricingModel,
			&r.ContactEmail,
			&r.Notes,
			&r.Status,
			&r.CreatedAt,
			&r.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan proposed plugin: %w", err)
		}
		records = append(records, r)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate proposed plugins: %w", err)
	}

	return records, nil
}
