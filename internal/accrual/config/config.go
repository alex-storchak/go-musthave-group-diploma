package config

import "time"

type Server struct {
	RunAddress               string        `mapstructure:"run_address"`
	ShutdownWaitSecsDuration time.Duration `mapstructure:"shutdown_wait_secs_duration"`
}

type Log struct {
	Level string `mapstructure:"level"`
}

type DB struct {
	DatabaseURI    string `mapstructure:"database_uri"`
	MigrationsPath string `mapstructure:"migrations_path"`
}

type Config struct {
	Server Server `mapstructure:"server"`
	Log    Log    `mapstructure:"log"`
	DB     DB     `mapstructure:"db"`
}
