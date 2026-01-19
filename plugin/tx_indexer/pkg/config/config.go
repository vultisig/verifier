package config

import (
	"time"

	"github.com/vultisig/verifier/internal/logging"
)

type Config struct {
	LogFormat        logging.LogFormat `mapstructure:"log_format" json:"log_format,omitempty"`
	Database         DatabaseConfig    `mapstructure:"database" json:"database,omitempty"`
	Rpc              RpcConfig         `mapstructure:"rpc" json:"rpc,omitempty"`
	Interval         time.Duration     `mapstructure:"interval" json:"interval,omitempty"`
	IterationTimeout time.Duration     `mapstructure:"iteration_timeout" json:"iteration_timeout,omitempty"`
	MarkLostAfter    time.Duration     `mapstructure:"mark_lost_after" json:"mark_lost_after,omitempty"`
	Concurrency      int               `mapstructure:"concurrency" json:"concurrency,omitempty"`
	Metrics          MetricsConfig     `mapstructure:"metrics" json:"metrics,omitempty"`
}

type DatabaseConfig struct {
	DSN string `mapstructure:"dsn" json:"dsn,omitempty"`
}

type RpcConfig struct {
	Bitcoin     RpcItem `mapstructure:"bitcoin" json:"bitcoin,omitempty"`
	Litecoin    RpcItem `mapstructure:"litecoin" json:"litecoin,omitempty"`
	Dogecoin    RpcItem `mapstructure:"dogecoin" json:"dogecoin,omitempty"`
	BitcoinCash RpcItem `mapstructure:"bitcoincash" json:"bitcoincash,omitempty"`
	Solana      RpcItem `mapstructure:"solana" json:"solana,omitempty"`
	XRP         RpcItem `mapstructure:"xrp" json:"xrp,omitempty"`
	Zcash       RpcItem `mapstructure:"zcash" json:"zcash,omitempty"`
	Ethereum    RpcItem `mapstructure:"ethereum" json:"ethereum,omitempty"`
	Avalanche   RpcItem `mapstructure:"avalanche" json:"avalanche,omitempty"`
	BscChain    RpcItem `mapstructure:"bsc" json:"bsc,omitempty"`
	Arbitrum    RpcItem `mapstructure:"arbitrum" json:"arbitrum,omitempty"`
	Base        RpcItem `mapstructure:"base" json:"base,omitempty"`
	Optimism    RpcItem `mapstructure:"optimism" json:"optimism,omitempty"`
	Polygon     RpcItem `mapstructure:"polygon" json:"polygon,omitempty"`
	Blast       RpcItem `mapstructure:"blast" json:"blast,omitempty"`
	Cronos      RpcItem `mapstructure:"cronos" json:"cronos,omitempty"`
	Zksync      RpcItem `mapstructure:"zksync" json:"zksync,omitempty"`
	THORChain   RpcItem `mapstructure:"thorchain" json:"thorchain,omitempty"`
}

type RpcItem struct {
	URL string `mapstructure:"url" json:"url,omitempty"`
}

type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled" json:"enabled,omitempty"`
	Host    string `mapstructure:"host" json:"host,omitempty"`
	Port    int    `mapstructure:"port" json:"port,omitempty"`
}
