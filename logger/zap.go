package logger

import (
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

func utcPlus7TimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	loc, _ := time.LoadLocation("Asia/Bangkok")
	t = t.In(loc)
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}

func zapLoggerProduction() (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
	cfg.EncoderConfig.EncodeTime = utcPlus7TimeEncoder
	cfg.DisableStacktrace = true
	return cfg.Build()
}

func zapLoggerDevelopment() (*zap.Logger, error) {
	cfg := zap.NewDevelopmentConfig()
	cfg.Encoding = "console"
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	cfg.EncoderConfig.EncodeTime = utcPlus7TimeEncoder
	cfg.DisableStacktrace = true
	return cfg.Build()
}

// InitLogger initializes the global logger with appropriate configuration
func InitLogger() error {
	var err error

	// Check environment
	env := os.Getenv("ENV")

	if env == "production" {
		// Production logger - JSON format, INFO level and above
		Logger, err = zapLoggerProduction()
	} else {
		// Development logger - console format, DEBUG level and above
		Logger, err = zapLoggerDevelopment()
	}

	if err != nil {
		return err
	}

	// Replace global logger
	zap.ReplaceGlobals(Logger)

	return nil
}

// Sync flushes any buffered log entries
func Sync() {
	if Logger != nil {
		Logger.Sync()
	}
}

// GetLogger returns the global logger instance
func GetLogger() *zap.Logger {
	if Logger == nil {
		// Fallback to a no-op logger if not initialized
		Logger = zap.NewNop()
	}
	return Logger
}

// Sugar returns a sugared logger for easier use
func Sugar() *zap.SugaredLogger {
	return GetLogger().Sugar()
}

// Info logs an info message
func Info(msg string, fields ...zap.Field) {
	GetLogger().Info(msg, fields...)
}

// Debug logs a debug message
func Debug(msg string, fields ...zap.Field) {
	GetLogger().Debug(msg, fields...)
}

// Error logs an error message
func Error(msg string, fields ...zap.Field) {
	GetLogger().Error(msg, fields...)
}

// Warn logs a warning message
func Warn(msg string, fields ...zap.Field) {
	GetLogger().Warn(msg, fields...)
}

// Fatal logs a fatal message and exits
func Fatal(msg string, fields ...zap.Field) {
	GetLogger().Fatal(msg, fields...)
}

// With creates a child logger with the given fields
func With(fields ...zap.Field) *zap.Logger {
	return GetLogger().With(fields...)
}
