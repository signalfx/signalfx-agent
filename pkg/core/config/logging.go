package config

import (
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ZapLogger creates a zap logging instance configured similarly to logrus.
func ZapLogger() *zap.Logger {
	var level zapcore.Level
	switch logrus.GetLevel() {
	case logrus.TraceLevel:
		level = zapcore.DebugLevel
	case logrus.DebugLevel:
		level = zapcore.DebugLevel
	case logrus.InfoLevel:
		level = zapcore.InfoLevel
	case logrus.WarnLevel:
		level = zapcore.WarnLevel
	case logrus.ErrorLevel:
		level = zapcore.ErrorLevel
	case logrus.FatalLevel:
		level = zapcore.FatalLevel
	case logrus.PanicLevel:
		level = zapcore.PanicLevel
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(level)

	switch logrus.StandardLogger().Formatter.(type) {
	case *logrus.JSONFormatter:
		cfg.Encoding = "json"
	default:
		cfg.Encoding = "console"
	}

	logger, err := cfg.Build()
	if err != nil {
		logrus.WithError(err).Warn("Failed creating zap logger, zap logs will be missing")
		return zap.NewNop()
	}
	return logger
}
