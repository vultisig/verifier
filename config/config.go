package config

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/spf13/viper"
	"github.com/vultisig/verifier/plugin/config"

	tx_indexer_config "github.com/vultisig/verifier/tx_indexer/pkg/config"
	"github.com/vultisig/verifier/vault_config"
)

type WorkerConfig struct {
	VaultService     vault_config.Config       `mapstructure:"vault_service" json:"vault_service,omitempty"`
	Redis            config.Redis              `mapstructure:"redis" json:"redis,omitempty"`
	BlockStorage     vault_config.BlockStorage `mapstructure:"block_storage" json:"block_storage,omitempty"`
	Database         config.Database           `mapstructure:"database" json:"database,omitempty"`
	Datadog          DatadogConfig             `mapstructure:"datadog" json:"datadog"`
	Fees             FeesConfig                `mapstructure:"fees" json:"fees"`
	ProposedYAMLPath string                    `mapstructure:"proposed_yaml_path" json:"proposed_yaml_path,omitempty"`
}

type VerifierConfig struct {
	Server struct {
		Host      string `mapstructure:"host" json:"host,omitempty"`
		Port      int64  `mapstructure:"port" json:"port,omitempty"`
		JWTSecret string `mapstructure:"jwt_secret" json:"jwt_secret,omitempty"`
	} `mapstructure:"server" json:"server"`
	Database         config.Database           `mapstructure:"database" json:"database,omitempty"`
	Redis            config.Redis              `mapstructure:"redis" json:"redis,omitempty"`
	BlockStorage     vault_config.BlockStorage `mapstructure:"block_storage" json:"block_storage,omitempty"`
	Datadog          DatadogConfig             `mapstructure:"datadog" json:"datadog"`
	EncryptionSecret string                    `mapstructure:"encryption_secret" json:"encryption_secret,omitempty"`
	Auth             struct {
		NonceExpiryMinutes int `mapstructure:"nonce_expiry_minutes" json:"nonce_expiry_minutes,omitempty"`
		// could be disabled for autotests / local,
		// pointer so it must be explicitly set to false, no value considered as enabled
		Enabled *bool `mapstructure:"enabled" json:"enabled,omitempty"`
	} `mapstructure:"auth" json:"auth"`
	Fees             FeesConfig `mapstructure:"fees" json:"fees"`
	ProposedYAMLPath string     `mapstructure:"proposed_yaml_path" json:"proposed_yaml_path,omitempty"`
}

type DatadogConfig struct {
	Host string `mapstructure:"host" json:"host,omitempty"`
	Port string `mapstructure:"port" json:"port,omitempty"`
}

type FeesConfig struct {
	USDCAddress string `mapstructure:"usdc_address" json:"usdc_address,omitempty"`
}

func GetConfigure() (*WorkerConfig, error) {
	configName := os.Getenv("VS_WORKER_CONFIG_NAME")
	if configName == "" {
		configName = "config"
	}
	return ReadConfig(configName)
}

func ReadConfig(configName string) (*WorkerConfig, error) {
	addKeysToViper(viper.GetViper(), reflect.TypeOf(WorkerConfig{}))
	viper.SetConfigName(configName)
	viper.AddConfigPath(".")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	viper.SetDefault("VaultService.VaultsFilePath", "vaults")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("fail to reading config file, %w", err)
		}
		// This is expected for ENV based config
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
	addKeysToViper(viper.GetViper(), reflect.TypeOf(VerifierConfig{}))
	viper.SetConfigName(configName)
	viper.AddConfigPath(".")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Set default values
	viper.SetDefault("auth.nonce_expiry_minutes", 15)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("fail to reading config file, %w", err)
		}
		// This is expected for ENV based config
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
	addKeysToViper(viper.GetViper(), reflect.TypeOf(tx_indexer_config.Config{}))
	viper.SetConfigName(configName)
	viper.AddConfigPath(".")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("fail to reading config file, %w", err)
		}
		// This is expected for ENV based config
	}
	var cfg tx_indexer_config.Config
	err := viper.Unmarshal(&cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to decode into struct, %w", err)
	}
	return &cfg, nil
}

func addKeysToViper(v *viper.Viper, t reflect.Type) {
	keys := getAllKeys(t)
	for _, key := range keys {
		v.SetDefault(key, "")
	}
}

func getAllKeys(t reflect.Type) []string {
	var result []string

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		// Try mapstructure tag first
		tagName := f.Tag.Get("mapstructure")
		if tagName == "" || tagName == "-" {
			// Fallback to JSON tag
			jsonTag := f.Tag.Get("json")
			if jsonTag != "" && jsonTag != "-" {
				// Handle comma-separated options (e.g., "field_name,omitempty")
				tagName = strings.Split(jsonTag, ",")[0]
			}
		} else {
			// Handle comma-separated options in mapstructure tag
			tagName = strings.Split(tagName, ",")[0]
		}

		// Final fallback to field name if no valid tags found
		if tagName == "" || tagName == "-" {
			tagName = f.Name
		}

		n := strings.ToUpper(tagName)

		if reflect.Struct == f.Type.Kind() {
			subKeys := getAllKeys(f.Type)
			for _, k := range subKeys {
				result = append(result, n+"."+k)
			}
		} else {
			result = append(result, n)
		}
	}

	return result
}
