package vault

type Config struct {
	Relay struct {
		Server string `mapstructure:"server" json:"server"`
	} `mapstructure:"relay" json:"relay,omitempty"`
	QueueEmailTask   bool   `mapstructure:"queue_email_task" json:"queue_email_task,omitempty"`
	EncryptionSecret string `mapstructure:"encryption_secret" json:"encryption_secret,omitempty"`
}
