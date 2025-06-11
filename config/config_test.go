package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"
)

func TestConfig(t *testing.T) {
	t.Skip("Skipping config test")
	cfg := WorkerConfig{}
	cfg.VaultServiceConfig.Relay.Server = "http://localhost:8080"
	cfg.VaultServiceConfig.QueueEmailTask = false
	cfg.Datadog.Host = "localhost"
	cfg.Datadog.Port = "8125"

	cfg.Redis.Host = "localhost"
	cfg.Redis.Port = "6379"
	cfg.Redis.DB = 0
	cfg.Redis.Password = ""

	cfg.BlockStorage.Host = "http://localhost:9000"
	cfg.BlockStorage.AccessKey = "minioadmin"
	cfg.BlockStorage.SecretKey = "minioadmin"
	cfg.BlockStorage.Bucket = "vultisig-verifier"
	cfg.BlockStorage.Region = "us-east-1"

	result, err := yaml.Marshal(&cfg)
	assert.NoError(t, err)
	t.Logf("%s", result)

	jsonResult, err := json.Marshal(&cfg)
	assert.NoError(t, err)
	t.Logf("%s", jsonResult)
}
