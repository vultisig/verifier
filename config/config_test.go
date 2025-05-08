package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"
)

func TestConfig(t *testing.T) {
	t.SkipNow()
	cfg := Config{}
	cfg.VaultServiceConfig.Relay.Server = "http://localhost:8080"
	cfg.VaultServiceConfig.QueueEmailTask = false
	cfg.Datadog.Host = "localhost"
	cfg.Datadog.Port = "8125"

	cfg.Redis.Host = "localhost"
	cfg.Redis.Port = "6379"
	cfg.Redis.DB = 0
	cfg.Redis.Password = ""

	cfg.BlockStorageConfig.Host = "http://localhost:9000"
	cfg.BlockStorageConfig.AccessKey = "minioadmin"
	cfg.BlockStorageConfig.SecretKey = "minioadmin"
	cfg.BlockStorageConfig.Bucket = "vultisig-verifier"
	cfg.BlockStorageConfig.Region = "us-east-1"

	result, err := yaml.Marshal(&cfg)
	assert.NoError(t, err)
	t.Logf("%s", result)

	jsonResult, err := json.Marshal(&cfg)
	assert.NoError(t, err)
	t.Logf("%s", jsonResult)
}
