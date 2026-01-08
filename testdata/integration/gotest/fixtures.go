package gotest

import (
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
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
	ID             string `yaml:"id"`
	Title          string `yaml:"title"`
	Description    string `yaml:"description"`
	ServerEndpoint string `yaml:"server_endpoint"`
	Category       string `yaml:"category"`
	LogoURL        string `yaml:"logo_url"`
	ThumbnailURL   string `yaml:"thumbnail_url"`
	Audited        bool   `yaml:"audited"`
}

type ProposedYAML struct {
	Plugins []PluginConfig `yaml:"plugins"`
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

func LoadPlugins(proposedPath string) ([]PluginConfig, error) {
	data, err := os.ReadFile(proposedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read proposed.yaml: %w", err)
	}

	var proposed ProposedYAML
	err = yaml.Unmarshal(data, &proposed)
	if err != nil {
		return nil, fmt.Errorf("failed to parse proposed.yaml: %w", err)
	}

	return proposed.Plugins, nil
}
