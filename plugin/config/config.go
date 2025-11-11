package config

import (
	"fmt"

	"github.com/redis/go-redis/v9"
)

type Database struct {
	DSN string `mapstructure:"dsn" json:"dsn,omitempty"`
}

type Redis struct {
	ConnURI  string `mapstructure:"conn_uri" json:"conn_uri,omitempty"`
	Host     string `mapstructure:"host" json:"host,omitempty"`
	Port     string `mapstructure:"port" json:"port,omitempty"`
	User     string `mapstructure:"user" json:"user,omitempty"`
	Password string `mapstructure:"password" json:"password,omitempty"`
	DB       int    `mapstructure:"db" json:"db,omitempty"`
}

func (r Redis) GetRedisOptions() (*redis.Options, error) {
	if r.ConnURI != "" {
		opts, err := redis.ParseURL(r.ConnURI)
		if err != nil {
			return nil, fmt.Errorf("failed to parse redis URI: %w", err)
		}
		return opts, nil
	}

	if r.Host == "" {
		return nil, fmt.Errorf("redis host is required when conn_uri is not provided")
	}

	return &redis.Options{
		Addr:     r.Host + ":" + r.Port,
		Username: r.User,
		Password: r.Password,
		DB:       r.DB,
	}, nil
}

// Verifier config for plugins
type Verifier struct {
	URL         string `mapstructure:"url"`
	Token       string `mapstructure:"token"`
	PartyPrefix string `mapstructure:"party_prefix"`
}
