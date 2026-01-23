package server

type Config struct {
	Host             string `mapstructure:"host" json:"host,omitempty" envconfig:"SERVER_HOST"`
	Port             int64  `mapstructure:"port" json:"port,omitempty" envconfig:"SERVER_PORT"`
	EncryptionSecret string `mapstructure:"encryption_secret" json:"encryption_secret,omitempty" envconfig:"SERVER_ENCRYPTIONSECRET"`
	TaskQueueName    string `mapstructure:"task_queue_name" json:"task_queue_name,omitempty" envconfig:"SERVER_TASK_QUEUE_NAME"`
}
