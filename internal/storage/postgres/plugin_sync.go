package postgres

import (
	"context"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	itypes "github.com/vultisig/verifier/internal/types"
	"github.com/vultisig/verifier/types"
)

type PluginYAML struct {
	ID             types.PluginID        `yaml:"id"`
	Title          string                `yaml:"title"`
	Description    string                `yaml:"description"`
	ServerEndpoint string                `yaml:"server_endpoint"`
	Category       itypes.PluginCategory `yaml:"category"`
}

type ProposedYAML struct {
	Plugins []PluginYAML `yaml:"plugins"`
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
			INSERT INTO plugins (id, title, description, server_endpoint, category, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
			ON CONFLICT (id)
			DO UPDATE SET
				title = EXCLUDED.title,
				description = EXCLUDED.description,
				server_endpoint = EXCLUDED.server_endpoint,
				category = EXCLUDED.category,
				updated_at = NOW()
		`

		_, err = p.pool.Exec(ctx, query,
			plugin.ID,
			plugin.Title,
			plugin.Description,
			plugin.ServerEndpoint,
			string(plugin.Category),
		)
		if err != nil {
			return fmt.Errorf("failed to upsert plugin %s: %w", plugin.ID, err)
		}
	}

	return nil
}
