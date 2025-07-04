package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"

	tx_indexer_config "github.com/vultisig/verifier/tx_indexer/pkg/config"
	"github.com/vultisig/verifier/vault_config"
)

type WorkerConfig struct {
	VaultService vault_config.Config       `mapstructure:"vault_service" json:"vault_service,omitempty"`
	Redis        RedisConfig               `mapstructure:"redis" json:"redis,omitempty"`
	BlockStorage vault_config.BlockStorage `mapstructure:"block_storage" json:"block_storage,omitempty"`
	Database     DatabaseConfig            `mapstructure:"database" json:"database,omitempty"`
	Plugin       struct {
		PluginConfigs map[string]map[string]interface{} `mapstructure:"plugin_configs" json:"plugin_configs,omitempty"`
	} `mapstructure:"plugin" json:"plugin,omitempty"`
	Datadog DatadogConfig `mapstructure:"datadog" json:"datadog"`
}

type VerifierConfig struct {
	Server struct {
		Host      string `mapstructure:"host" json:"host,omitempty"`
		Port      int64  `mapstructure:"port" json:"port,omitempty"`
		JWTSecret string `mapstructure:"jwt_secret" json:"jwt_secret,omitempty"`
	} `mapstructure:"server" json:"server"`
	Database         DatabaseConfig            `mapstructure:"database" json:"database,omitempty"`
	Redis            RedisConfig               `mapstructure:"redis" json:"redis,omitempty"`
	BlockStorage     vault_config.BlockStorage `mapstructure:"block_storage" json:"block_storage,omitempty"`
	Datadog          DatadogConfig             `mapstructure:"datadog" json:"datadog"`
	EncryptionSecret string                    `mapstructure:"encryption_secret" json:"encryption_secret,omitempty"`
	Auth             struct {
		NonceExpiryMinutes int `mapstructure:"nonce_expiry_minutes" json:"nonce_expiry_minutes,omitempty"`
		// could be disabled for autotests / local,
		// pointer so it must be explicitly set to false, no value considered as enabled
		Enabled *bool `mapstructure:"enabled" json:"enabled,omitempty"`
	} `mapstructure:"auth" json:"auth"`
}

type DatadogConfig struct {
	Host string `mapstructure:"host" json:"host,omitempty"`
	Port string `mapstructure:"port" json:"port,omitempty"`
}

type DatabaseConfig struct {
	DSN string `mapstructure:"dsn" json:"dsn,omitempty"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host" json:"host,omitempty"`
	Port     string `mapstructure:"port" json:"port,omitempty"`
	User     string `mapstructure:"user" json:"user,omitempty"`
	Password string `mapstructure:"password" json:"password,omitempty"`
	DB       int    `mapstructure:"db" json:"db,omitempty"`
}

func GetConfigure() (*WorkerConfig, error) {
	configName := os.Getenv("VS_WORKER_CONFIG_NAME")
	if configName == "" {
		configName = "config"
	}
	return ReadConfig(configName)
}

func ReadConfig(configName string) (*WorkerConfig, error) {
	viper.SetConfigName(configName)
	viper.AddConfigPath(".")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	viper.SetDefault("VaultService.VaultsFilePath", "vaults")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("fail to reading config file, %w", err)
	}
	var cfg WorkerConfig
	err := viper.Unmarshal(&cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to decode into struct, %w", err)
	}
	return &cfg, nil
}

func ReadVerifierConfig() (*VerifierConfig, error) {
	configName := os.Getenv("VS_VERIFIER_CONFIG_NAME")
	if configName == "" {
		configName = "config"
	}
	viper.SetConfigName(configName)
	viper.AddConfigPath(".")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Set default values
	viper.SetDefault("auth.nonce_expiry_minutes", 15)

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("fail to reading config file, %w", err)
	}
	var cfg VerifierConfig
	err := viper.Unmarshal(&cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to decode into struct, %w", err)
	}
	return &cfg, nil
}

func ReadTxIndexerConfig() (*tx_indexer_config.Config, error) {
	configName := os.Getenv("VS_TX_INDEXER_CONFIG_NAME")
	if configName == "" {
		configName = "config"
	}
	viper.SetConfigName(configName)
	viper.AddConfigPath(".")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("fail to reading config file, %w", err)
	}
	var cfg tx_indexer_config.Config
	err := viper.Unmarshal(&cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to decode into struct, %w", err)
	}
	return &cfg, nil
}
