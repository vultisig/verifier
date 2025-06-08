package vault_config

type Config struct {
	Relay struct {
		Server string `mapstructure:"server" json:"server"`
	} `mapstructure:"relay" json:"relay,omitempty"`
	QueueEmailTask   bool   `mapstructure:"queue_email_task" json:"queue_email_task,omitempty"`
	EncryptionSecret string `mapstructure:"encryption_secret" json:"encryption_secret,omitempty"`
}

type BlockStorageConfig struct {
	Host      string `mapstructure:"host" json:"host"`
	Region    string `mapstructure:"region" json:"region"`
	AccessKey string `mapstructure:"access_key" json:"access_key"`
	SecretKey string `mapstructure:"secret" json:"secret"`
	Bucket    string `mapstructure:"bucket" json:"bucket"`
}
