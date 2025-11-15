package util

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.Logger

// InitLogger initializes the global logger
func InitLogger(env string) error {
	var err error
	var config zap.Config

	if env == "production" {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	logger, err = config.Build()
	if err != nil {
		return err
	}

	zap.ReplaceGlobals(logger)
	return nil
}

// GetLogger returns the global logger
func GetLogger() *zap.Logger {
	if logger == nil {
		logger, _ = zap.NewDevelopment()
	}
	return logger
}

// SyncLogger flushes any buffered log entries
func SyncLogger() {
	if logger != nil {
		_ = logger.Sync()
	}
}
