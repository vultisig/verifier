package vault

type Config struct {
	Server struct {
		Host     string `mapstructure:"host" json:"host,omitempty"`
		Port     int64  `mapstructure:"port" json:"port,omitempty"`
		Database struct {
			DSN string `mapstructure:"dsn" json:"dsn,omitempty"`
		} `mapstructure:"database" json:"database,omitempty"`
		VaultsFilePath string `mapstructure:"vaults_file_path" json:"vaults_file_path,omitempty"`
	} `mapstructure:"server" json:"server"`
	Redis struct {
		Host     string `mapstructure:"host" json:"host,omitempty"`
		Port     string `mapstructure:"port" json:"port,omitempty"`
		User     string `mapstructure:"user" json:"user,omitempty"`
		Password string `mapstructure:"password" json:"password,omitempty"`
		DB       int    `mapstructure:"db" json:"db,omitempty"`
	} `mapstructure:"redis" json:"redis,omitempty"`

	Relay struct {
		Server string `mapstructure:"server" json:"server"`
	} `mapstructure:"relay" json:"relay,omitempty"`
}
