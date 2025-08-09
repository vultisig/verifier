package config

import "time"

type Config struct {
	Database         DatabaseConfig `mapstructure:"database" json:"database,omitempty"`
	Rpc              RpcConfig      `mapstructure:"rpc" json:"rpc,omitempty"`
	Interval         time.Duration  `mapstructure:"interval" json:"interval,omitempty"`
	IterationTimeout time.Duration  `mapstructure:"iteration_timeout" json:"iteration_timeout,omitempty"`
	MarkLostAfter    time.Duration  `mapstructure:"mark_lost_after" json:"mark_lost_after,omitempty"`
	Concurrency      int            `mapstructure:"concurrency" json:"concurrency,omitempty"`
}

type DatabaseConfig struct {
	DSN string `mapstructure:"dsn" json:"dsn,omitempty"`
}

type RpcConfig struct {
	Bitcoin  RpcItem `mapstructure:"bitcoin" json:"bitcoin,omitempty"`
	Ethereum RpcItem `mapstructure:"ethereum" json:"ethereum,omitempty"`
}

type RpcItem struct {
	URL string `mapstructure:"url" json:"url,omitempty"`
}
