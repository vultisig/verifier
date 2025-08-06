package server

type Config struct {
	Host             string `mapstructure:"host" json:"host,omitempty"`
	Port             int64  `mapstructure:"port" json:"port,omitempty"`
	EncryptionSecret string `mapstructure:"encryption_secret" json:"encryption_secret,omitempty"`
}
