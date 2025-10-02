package logger

import (
    "fmt"

    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

// New creates a new logger instance
func New(level, format string) (*zap.Logger, error) {
    var config zap.Config

    switch format {
    case "json":
        config = zap.NewProductionConfig()
    case "console":
        config = zap.NewDevelopmentConfig()
        config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
    default:
        config = zap.NewProductionConfig()
    }

    // Set log level
    switch level {
    case "debug":
        config.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
    case "info":
        config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
    case "warn":
        config.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
    case "error":
        config.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
    default:
        config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
    }

    // Configure output paths
    config.OutputPaths = []string{"stdout"}
    config.ErrorOutputPaths = []string{"stderr"}

    // Configure encoder
    config.EncoderConfig.TimeKey = "timestamp"
    config.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
    config.EncoderConfig.MessageKey = "message"
    config.EncoderConfig.LevelKey = "level"
    config.EncoderConfig.CallerKey = "caller"
    config.EncoderConfig.StacktraceKey = "stacktrace"

    logger, err := config.Build()
    if err != nil {
        return nil, fmt.Errorf("failed to build logger: %w", err)
    }

    return logger, nil
}

// NewWithOptions creates a logger with custom options
func NewWithOptions(options ...zap.Option) (*zap.Logger, error) {
    config := zap.NewProductionConfig()
    logger, err := config.Build(options...)
    if err != nil {
        return nil, fmt.Errorf("failed to build logger with options: %w", err)
    }

    return logger, nil
}
