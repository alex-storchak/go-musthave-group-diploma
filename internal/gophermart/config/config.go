package config

import (
	"flag"
	handlers "github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/config"
	accrual "github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/service/accrual/config"
	"os"
)

type Config struct {
	Handlers    *handlers.Config
	Accrual     *accrual.Config
	LogLevel    string
	DatabaseDsn string
}

func GetConfig(args []string) (*Config, error) {
	cfg := Config{
		Handlers: &handlers.Config{
			ServerAddr:    handlers.DefaultServerAddr,
			CompressLevel: handlers.DefaultCompressLevel,
			AuthConfig: handlers.AuthConfig{
				AuthCookieName:     handlers.DefaultAuthCookieName,
				AuthSecretKey:      handlers.DefaultAuthSecretKey,
				AuthExpireDuration: handlers.DefaultAuthExpireDuration,
			},
		},
		Accrual: &accrual.Config{
			Addr:              accrual.DefaultAddr,
			RequestTimeout:    accrual.DefaultRequestTimeout,
			RetryAfterDefault: accrual.DefaultRetryAfter,
		},
		DatabaseDsn: "host=127.127.126.41 port=5432 dbname=shorturl user=shorturl password=shorturl connect_timeout=10 sslmode=prefer",
		LogLevel:    "info",
	}

	if serverAddr := os.Getenv("RUN_ADDRESS"); serverAddr != "" {
		cfg.Handlers.ServerAddr = serverAddr
	}

	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		cfg.LogLevel = envLogLevel
	}

	if databaseDsn := os.Getenv("DATABASE_URI"); databaseDsn != "" {
		cfg.DatabaseDsn = databaseDsn
	}

	if secretKey := os.Getenv("AUTH_SECRET_KEY"); secretKey != "" {
		cfg.Handlers.AuthSecretKey = secretKey
	}

	if accrualAddr := os.Getenv("ACCRUAL_SYSTEM_ADDRESS"); accrualAddr != "" {
		cfg.Accrual.Addr = accrualAddr
	}

	fs := flag.NewFlagSet("myFlagSet", flag.ContinueOnError)
	fs.StringVar(&cfg.Handlers.ServerAddr, "a", cfg.Handlers.ServerAddr, "address of HTTP server")
	fs.StringVar(&cfg.LogLevel, "l", cfg.LogLevel, "log level")
	fs.StringVar(&cfg.DatabaseDsn, "d", cfg.DatabaseDsn, "connection string")
	fs.StringVar(&cfg.Handlers.AuthSecretKey, "s", cfg.Handlers.AuthSecretKey, "secret key")
	fs.StringVar(&cfg.Accrual.Addr, "r", cfg.Accrual.Addr, "address accrual")
	err := fs.Parse(args)
	if err != nil {
		return &Config{}, err
	}

	return &cfg, nil
}
