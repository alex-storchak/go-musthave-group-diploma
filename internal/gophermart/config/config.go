package config

import (
	"flag"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/config"
	"os"
)

type Config struct {
	Handlers *config.Config
	LogLevel string
}

func GetConfig(args []string) (*Config, error) {
	cfg := Config{
		Handlers: &config.Config{
			ServerAddr:  "localhost:8080",
			DatabaseDsn: "host=127.127.126.41 port=5432 dbname=shorturl user=shorturl password=shorturl connect_timeout=10 sslmode=prefer",
		},
		LogLevel: "info",
	}

	if serverAddr := os.Getenv("SERVER_ADDRESS"); serverAddr != "" {
		cfg.Handlers.ServerAddr = serverAddr
	}

	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		cfg.LogLevel = envLogLevel
	}

	if databaseDsn := os.Getenv("DATABASE_DSN"); databaseDsn != "" {
		cfg.Handlers.DatabaseDsn = databaseDsn
	}

	if secretKey := os.Getenv("SECRET_KEY"); secretKey != "" {
		cfg.Handlers.SecretKey = secretKey
	}

	fs := flag.NewFlagSet("myFlagSet", flag.ContinueOnError)
	fs.StringVar(&cfg.Handlers.ServerAddr, "a", cfg.Handlers.ServerAddr, "address of HTTP server")
	fs.StringVar(&cfg.LogLevel, "l", cfg.LogLevel, "log level")
	fs.StringVar(&cfg.Handlers.DatabaseDsn, "d", cfg.Handlers.DatabaseDsn, "connection string")
	fs.StringVar(&cfg.Handlers.SecretKey, "s", cfg.Handlers.SecretKey, "secret key")
	err := fs.Parse(args)
	if err != nil {
		return &Config{}, err
	}

	return &cfg, nil
}
