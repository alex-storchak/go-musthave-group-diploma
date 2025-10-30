package config

import (
	"flag"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/config"
	"os"
)

const (
	DefaultAccrualAddr = "localhost:8081"
	DefaultDatabaseDsn = "host=127.127.126.41 port=5432 dbname=shorturl user=shorturl password=shorturl connect_timeout=10 sslmode=prefer"
	DefaultLogLevel    = "info"
)

type Config struct {
	Handlers    *config.Config
	LogLevel    string
	DatabaseDsn string
	AccrualAddr string
}

func GetConfig(args []string) (*Config, error) {
	cfg := Config{
		Handlers: &config.Config{
			ServerAddr:    config.DefaultServerAddr,
			CompressLevel: config.DefaultCompressLevel,
			AuthConfig: config.AuthConfig{
				AuthCookieName:     config.DefaultAuthCookieName,
				AuthSecretKey:      config.DefaultAuthSecretKey,
				AuthExpireDuration: config.DefaultAuthExpireDuration,
			},
		},
		AccrualAddr: DefaultAccrualAddr,
		DatabaseDsn: DefaultDatabaseDsn,
		LogLevel:    DefaultLogLevel,
	}

	if serverAddr, ok := os.LookupEnv("RUN_ADDRESS"); ok && serverAddr != "" {
		cfg.Handlers.ServerAddr = serverAddr
	}

	if envLogLevel, ok := os.LookupEnv("LOG_LEVEL"); ok && envLogLevel != "" {
		cfg.LogLevel = envLogLevel
	}

	if databaseDsn, ok := os.LookupEnv("DATABASE_URI"); ok && databaseDsn != "" {
		cfg.DatabaseDsn = databaseDsn
	}

	if accrualAddr, ok := os.LookupEnv("ACCRUAL_SYSTEM_ADDRESS"); ok && accrualAddr != "" {
		cfg.AccrualAddr = accrualAddr
	}

	if secretKey, ok := os.LookupEnv("AUTH_SECRET_KEY"); ok && secretKey != "" {
		cfg.Handlers.AuthSecretKey = secretKey
	}

	fs := flag.NewFlagSet("myFlagSet", flag.ContinueOnError)
	fs.StringVar(&cfg.Handlers.ServerAddr, "a", cfg.Handlers.ServerAddr, "address of HTTP server")
	fs.StringVar(&cfg.LogLevel, "l", cfg.LogLevel, "log level")
	fs.StringVar(&cfg.DatabaseDsn, "d", cfg.DatabaseDsn, "database connection string")
	fs.StringVar(&cfg.AccrualAddr, "r", cfg.AccrualAddr, "address of the accrual system")
	fs.StringVar(&cfg.Handlers.AuthSecretKey, "s", cfg.Handlers.AuthSecretKey, "auth secret key")
	err := fs.Parse(args)
	if err != nil {
		return &Config{}, err
	}

	return &cfg, nil
}
