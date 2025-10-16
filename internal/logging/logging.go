package logging

import (
	config "github.com/alex-storchak/go-musthave-group-diploma/internal/config/gophermart"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Initialize(config *config.Config) (*zap.Logger, error) {
	lvl, err := zap.ParseAtomicLevel(config.LogLevel)
	if err != nil {
		return nil, err
	}

	encoderConfig := zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		TimeKey:        "ts",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02T15:04:05.000Z"), // Формат ISO 8601
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = lvl
	cfg.EncoderConfig = encoderConfig
	zl, err := cfg.Build()
	if err != nil {
		return nil, err
	}

	return zl, nil
}
