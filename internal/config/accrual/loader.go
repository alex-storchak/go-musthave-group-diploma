package config

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	ErrEmptyRunAddress  = errors.New("run_address must not be empty")
	ErrEmptyDatabaseURI = errors.New("database_uri must not be empty")
)

func BindFlags(flagset *pflag.FlagSet) error {
	flagset.StringP("server.run_address", "a", "", "address of HTTP server")
	flagset.StringP("log.level", "l", "", "Log level")
	flagset.StringP("db.database_uri", "d", "", "Database URI")

	if err := viper.BindPFlags(flagset); err != nil {
		return fmt.Errorf("bind flags with viper: %w", err)
	}
	return nil
}

func initViper() error {
	viper.SetConfigType("yaml")

	viper.AutomaticEnv()
	if err := viper.BindEnv("server.run_address", "RUN_ADDRESS"); err != nil {
		return fmt.Errorf("bind env key `RUN_ADDRESS`: %w", err)
	}
	if err := viper.BindEnv("db.database_uri", "DATABASE_URI"); err != nil {
		return fmt.Errorf("bind env key `DATABASE_URI`: %w", err)
	}

	viper.AddConfigPath("./configs/accrual")
	viper.SetConfigName("config")

	if err := viper.ReadInConfig(); err != nil {
		var fnfe *viper.ConfigFileNotFoundError
		if !errors.As(err, fnfe) {
			return fmt.Errorf("read config file: %w", err)
		}
	}

	return nil
}

func Load() (*Config, error) {
	fs := pflag.NewFlagSet("accrual", pflag.ContinueOnError)
	if err := BindFlags(fs); err != nil {
		return nil, fmt.Errorf("bind flags: %w", err)
	}
	if err := fs.Parse(os.Args[1:]); err != nil {
		log.Printf("warn: failed to parse flags, falling back to env/file: %v", err)
	}

	if err := initViper(); err != nil {
		return nil, fmt.Errorf("init viper config: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config with viper: %w", err)
	}

	applyDefaults(&cfg)

	if cfg.DB.DatabaseURI == "" {
		return nil, ErrEmptyDatabaseURI
	}
	if cfg.Server.RunAddress == "" {
		return nil, ErrEmptyRunAddress
	}

	return &cfg, nil
}
