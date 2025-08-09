package config

type Database struct {
	DSN string `mapstructure:"dsn" json:"dsn,omitempty"`
}

type Redis struct {
	Host     string `mapstructure:"host" json:"host,omitempty"`
	Port     string `mapstructure:"port" json:"port,omitempty"`
	User     string `mapstructure:"user" json:"user,omitempty"`
	Password string `mapstructure:"password" json:"password,omitempty"`
	DB       int    `mapstructure:"db" json:"db,omitempty"`
}

// Verifier config for plugins
type Verifier struct {
	URL         string `mapstructure:"url"`
	Token       string `mapstructure:"token"`
	PartyPrefix string `mapstructure:"party_prefix"`
}
