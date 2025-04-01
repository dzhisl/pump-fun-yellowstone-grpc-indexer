package config

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

func InitLogger() {
	var cfg zap.Config

	cfg = zap.NewDevelopmentConfig()
	cfg.EncoderConfig.StacktraceKey = ""
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	// Parse log level from environment variable
	logLevelStr := strings.ToLower(os.Getenv("LOG_LEVEL"))
	var logLevel zapcore.Level

	switch logLevelStr {
	case "debug":
		logLevel = zapcore.DebugLevel
	case "info":
		logLevel = zapcore.InfoLevel
	case "warn":
		logLevel = zapcore.WarnLevel
	case "error":
		logLevel = zapcore.ErrorLevel
	default:
		logLevel = zapcore.InfoLevel // Default to INFO if the value is invalid or empty
	}
	cfg.Level = zap.NewAtomicLevelAt(logLevel)

	// Build the logger and handle any errors
	var err error
	Logger, err = cfg.Build()
	if err != nil {
		zap.L().Fatal("Error building logger", zap.Error(err))
	}

	// Set the global logger to the newly created instance
	zap.ReplaceGlobals(Logger)
}
