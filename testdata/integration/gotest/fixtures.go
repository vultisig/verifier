package gotest

import (
	"encoding/json"
	"fmt"
	"os"
)

type FixtureData struct {
	Vault struct {
		PublicKey string `json:"public_key"`
		Name      string `json:"name"`
		CreatedAt string `json:"created_at"`
		VaultB64  string `json:"vault_b64"`
	} `json:"vault"`
	Reshare struct {
		SessionID        string   `json:"session_id"`
		HexEncryptionKey string   `json:"hex_encryption_key"`
		HexChainCode     string   `json:"hex_chain_code"`
		LocalPartyID     string   `json:"local_party_id"`
		OldParties       []string `json:"old_parties"`
		OldResharePrefix string   `json:"old_reshare_prefix"`
		Email            string   `json:"email"`
	} `json:"reshare"`
}

type PluginConfig struct {
	ID             string
	Title          string
	Description    string
	ServerEndpoint string
	Category       string
	LogoURL        string
	ThumbnailURL   string
	Audited        bool
}

func LoadFixture(path string) (*FixtureData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read fixture file: %w", err)
	}

	var fixture FixtureData
	err = json.Unmarshal(data, &fixture)
	if err != nil {
		return nil, fmt.Errorf("failed to parse fixture JSON: %w", err)
	}

	return &fixture, nil
}

func GetTestPlugins() []PluginConfig {
	return []PluginConfig{
		{
			ID:             "vultisig-dca-0000",
			Title:          "DCA (Dollar Cost Averaging)",
			Description:    "Automated recurring swaps and transfers",
			ServerEndpoint: envOrDefault("DCA_PLUGIN_URL", "http://localhost:8082"),
			Category:       "app",
		},
	}
}

func envOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
