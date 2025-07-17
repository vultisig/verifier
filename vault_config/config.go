package vault_config

type Config struct {
	Relay struct {
		Server string `mapstructure:"server" json:"server"`
		// Encrypt MPC plugin.worker<>relay<>verifier.worker
		// Doesn't need to set secret on verifier.worker side, because it gets it alongside with KeysignRequest
		// by private plugin->verifier API call (put to queue)
		EncryptionSecret string `mapstructure:"encryption_secret" json:"encryption_secret,omitempty"`
	} `mapstructure:"relay" json:"relay,omitempty"`
	LocalPartyPrefix string `mapstructure:"local_party_prefix" json:"local_party_prefix,omitempty"`
	QueueEmailTask   bool   `mapstructure:"queue_email_task" json:"queue_email_task,omitempty"`
	EncryptionSecret string `mapstructure:"encryption_secret" json:"encryption_secret,omitempty"`
	DoSetupMsg       bool   `mapstructure:"do_setup_msg" json:"do_setup_msg,omitempty"`
}

type BlockStorage struct {
	Host      string `mapstructure:"host" json:"host"`
	Region    string `mapstructure:"region" json:"region"`
	AccessKey string `mapstructure:"access_key" json:"access_key"`
	SecretKey string `mapstructure:"secret" json:"secret"`
	Bucket    string `mapstructure:"bucket" json:"bucket"`
}
