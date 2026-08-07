package logger

import (
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Log is the global logger instance.
// Initialized as a no-op logger to prevent nil-pointer panics in code paths
// that use Log before Init() is called (e.g. test CLIs, library imports).
var Log *zap.Logger = zap.NewNop()

// Init initializes the application logger.
func Init(logDir string) error {
	// Create log directory
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	logFile := filepath.Join(logDir, "vhd-opener.log")

	// File config
	fileConfig := zap.NewProductionConfig()
	fileConfig.OutputPaths = []string{logFile}
	fileConfig.ErrorOutputPaths = []string{logFile}
	fileConfig.EncoderConfig.TimeKey = "ts"
	fileConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	fileLogger, err := fileConfig.Build(zap.AddCallerSkip(1))
	if err != nil {
		return err
	}

	// Console config for development
	consoleConfig := zap.NewDevelopmentConfig()
	consoleConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	consoleConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	consoleLogger, err := consoleConfig.Build(zap.AddCallerSkip(1))
	if err != nil {
		fileLogger.Sync()
		return err
	}

	// Tee both loggers
	Log = zap.New(
		zapcore.NewTee(
			fileLogger.Core(),
			consoleLogger.Core(),
		),
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)

	return nil
}

// Sync flushes any buffered log entries.
func Sync() {
	if Log != nil {
		Log.Sync()
	}
}
