package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"

	"github.com/vultisig/verifier/vault"
)

type Config struct {
	VaultServiceConfig vault.Config             `mapstructure:"vault_service_config" json:"vault_service_config,omitempty"`
	Redis              RedisConfig              `mapstructure:"redis" json:"redis,omitempty"`
	BlockStorageConfig vault.BlockStorageConfig `mapstructure:"block_storage_config" json:"block_storage_config,omitempty"`
	Database           struct {
		DSN string `mapstructure:"dsn" json:"dsn,omitempty"`
	} `mapstructure:"database" json:"database,omitempty"`
	Plugin struct {
		PluginConfigs map[string]map[string]interface{} `mapstructure:"plugin_configs" json:"plugin_configs,omitempty"`
	} `mapstructure:"plugin" json:"plugin,omitempty"`

	Datadog struct {
		Host string `mapstructure:"host" json:"host,omitempty"`
		Port string `mapstructure:"port" json:"port,omitempty"`
	} `mapstructure:"datadog" json:"datadog"`
}

type VerifierConfig struct {
	Server struct {
		Host         string `mapstructure:"host" json:"host,omitempty"`
		Port         int64  `mapstructure:"port" json:"port,omitempty"`
		VerifierHost string `mapstructure:"verifier_host" json:"verifier_host,omitempty"`
		Database     struct {
			DSN string `mapstructure:"dsn" json:"dsn,omitempty"`
		} `mapstructure:"database" json:"database,omitempty"`
		JWTSecret string `mapstructure:"jwt_secret" json:"jwt_secret,omitempty"`
	} `mapstructure:"server" json:"server"`
	Redis              RedisConfig              `mapstructure:"redis" json:"redis,omitempty"`
	BlockStorageConfig vault.BlockStorageConfig `mapstructure:"block_storage_config" json:"block_storage_config,omitempty"`
	Datadog            struct {
		Host string `mapstructure:"host" json:"host,omitempty"`
		Port string `mapstructure:"port" json:"port,omitempty"`
	} `mapstructure:"datadog" json:"datadog"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host" json:"host,omitempty"`
	Port     string `mapstructure:"port" json:"port,omitempty"`
	User     string `mapstructure:"user" json:"user,omitempty"`
	Password string `mapstructure:"password" json:"password,omitempty"`
	DB       int    `mapstructure:"db" json:"db,omitempty"`
}

func GetConfigure() (*Config, error) {
	configName := os.Getenv("VS_CONFIG_NAME")
	if configName == "" {
		configName = "config"
	}
	return ReadConfig(configName)
}

func ReadConfig(configName string) (*Config, error) {
	viper.SetConfigName(configName)
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	viper.SetDefault("VaultServiceConfig.VaultsFilePath", "vaults")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("fail to reading config file, %w", err)
	}
	var cfg Config
	err := viper.Unmarshal(&cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to decode into struct, %w", err)
	}
	return &cfg, nil
}

func ReadVerifierConfig() (*VerifierConfig, error) {
	configName := os.Getenv("VERIFIER_CONFIG_NAME")
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
