package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"

	"github.com/vultisig/verifier/vault"
)

type WorkerConfig struct {
	VaultServiceConfig vault.Config             `mapstructure:"vault_service_config" json:"vault_service_config,omitempty"`
	Redis              RedisConfig              `mapstructure:"redis" json:"redis,omitempty"`
	BlockStorageConfig vault.BlockStorageConfig `mapstructure:"block_storage_config" json:"block_storage_config,omitempty"`
	Database           DatabaseConfig           `mapstructure:"database" json:"database,omitempty"`
	Plugin             struct {
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
	Database           DatabaseConfig           `mapstructure:"database" json:"database,omitempty"`
	Redis              RedisConfig              `mapstructure:"redis" json:"redis,omitempty"`
	BlockStorageConfig vault.BlockStorageConfig `mapstructure:"block_storage_config" json:"block_storage_config,omitempty"`
	Datadog            DatadogConfig            `mapstructure:"datadog" json:"datadog"`
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
	viper.AutomaticEnv()

	viper.SetDefault("VaultServiceConfig.VaultsFilePath", "vaults")

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
	viper.AutomaticEnv()

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
