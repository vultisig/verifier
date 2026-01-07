package server

type Config struct {
	Host             string `mapstructure:"host" json:"host,omitempty"`
	Port             int64  `mapstructure:"port" json:"port,omitempty"`
	EncryptionSecret string `mapstructure:"encryption_secret" json:"encryption_secret,omitempty"`

	// TaskQueueName specifies which asynq queue this plugin server enqueues tasks to.
	// CRITICAL: Each plugin MUST use a unique queue name separate from the verifier's "default_queue".
	// During TSS reshare (plugin install), the verifier worker and plugin worker must be on different
	// queues so each picks up exactly one reshare task. If they share queues, one worker type will
	// pick up both tasks and the 4-party reshare will fail.
	// Example: DCA plugin uses "dca_plugin_queue", verifier uses "default_queue".
	TaskQueueName string `mapstructure:"task_queue_name" json:"task_queue_name,omitempty" envconfig:"TASKQUEUENAME"`
}
