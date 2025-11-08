package logger

import (
	"fmt"

	"github.com/alex-storchak/go-musthave-group-diploma/internal/accrual/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(cfg config.Log) (*zap.Logger, error) {
	lvl, err := zap.ParseAtomicLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("parse log level: %w", err)
	}
	zcfg := zap.NewProductionConfig()
	zcfg.Level = lvl
	zcfg.Encoding = "console"
	zcfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	zl, err := zcfg.Build()
	if err != nil {
		return nil, fmt.Errorf("build logger: %w", err)
	}

	return zl, nil
}
