package postgres

import (
	"context"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	itypes "github.com/vultisig/verifier/internal/types"
)

type ProposedYAML struct {
	Plugins []itypes.Plugin `yaml:"plugins"`
}

func (p *PostgresBackend) SyncPluginsFromYAML(yamlPath string) error {
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return fmt.Errorf("failed to read yaml file: %w", err)
	}

	var proposed ProposedYAML
	err = yaml.Unmarshal(data, &proposed)
	if err != nil {
		return fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	ctx := context.Background()
	for _, plugin := range proposed.Plugins {
		query := `
			INSERT INTO plugins (id, title, description, server_endpoint, category, created_at, updated_at, logo_url, thumbnail_url, images, faqs, features, audited)
			VALUES ($1, $2, $3, $4, $5, NOW(), NOW(), $6, $7, $8, $9, $10, $11)
			ON CONFLICT (id)
			DO UPDATE SET
				title = EXCLUDED.title,
				description = EXCLUDED.description,
				server_endpoint = EXCLUDED.server_endpoint,
				category = EXCLUDED.category,
				updated_at = CASE
					WHEN plugins.title IS DISTINCT FROM EXCLUDED.title
						OR plugins.description IS DISTINCT FROM EXCLUDED.description
						OR plugins.server_endpoint IS DISTINCT FROM EXCLUDED.server_endpoint
						OR plugins.category IS DISTINCT FROM EXCLUDED.category
						OR plugins.logo_url IS DISTINCT FROM EXCLUDED.logo_url
						OR plugins.thumbnail_url IS DISTINCT FROM EXCLUDED.thumbnail_url
						OR plugins.images IS DISTINCT FROM EXCLUDED.images
						OR plugins.faqs IS DISTINCT FROM EXCLUDED.faqs
						OR plugins.features IS DISTINCT FROM EXCLUDED.features
						OR plugins.audited IS DISTINCT FROM EXCLUDED.audited
					THEN NOW()
					ELSE plugins.updated_at
				END,
				logo_url = EXCLUDED.logo_url,
				thumbnail_url = EXCLUDED.thumbnail_url,
				images = EXCLUDED.images,
				faqs = EXCLUDED.faqs,
				features = EXCLUDED.features,
				audited = EXCLUDED.audited
		`

		_, err = p.pool.Exec(ctx, query,
			plugin.ID,
			plugin.Title,
			plugin.Description,
			plugin.ServerEndpoint,
			string(plugin.Category),
			plugin.LogoURL,
			plugin.ThumbnailURL,
			plugin.Images,
			plugin.FAQs,
			plugin.Features,
			plugin.Audited,
		)
		if err != nil {
			return fmt.Errorf("failed to upsert plugin %s: %w", plugin.ID, err)
		}
	}

	return nil
}
