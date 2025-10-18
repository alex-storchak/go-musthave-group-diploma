package config

type Server struct {
	RunAddress string `mapstructure:"run_address"`
}

type Log struct {
	Level string `mapstructure:"level"`
}

type DB struct {
	DatabaseURI string `mapstructure:"database_uri"`
}

type Config struct {
	Server Server `mapstructure:"server"`
	Log    Log    `mapstructure:"log"`
	DB     DB     `mapstructure:"db"`
}
