// =============================================================================
// LOGGER PACKAGE
// Structured logging with zap for high-performance logging
// =============================================================================

package logger

import (
	"context"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps zap logger with convenience methods
type Logger struct {
	*zap.Logger
	sugar *zap.SugaredLogger
}

// Config for logger
type Config struct {
	Level       string // debug, info, warn, error
	Development bool
	Encoding    string // json, console
	OutputPaths []string
}

// contextKey for storing logger in context
type contextKey struct{}

// Global logger instance
var defaultLogger *Logger

// New creates a new logger
func New(cfg *Config) (*Logger, error) {
	// Parse log level
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}
	
	// Encoder config
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	
	if cfg.Development {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	
	// Create encoder
	var encoder zapcore.Encoder
	if cfg.Encoding == "console" || cfg.Development {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}
	
	// Output paths
	outputPaths := cfg.OutputPaths
	if len(outputPaths) == 0 {
		outputPaths = []string{"stdout"}
	}
	
	// Create core
	var cores []zapcore.Core
	for _, path := range outputPaths {
		var writeSyncer zapcore.WriteSyncer
		if path == "stdout" {
			writeSyncer = zapcore.AddSync(os.Stdout)
		} else if path == "stderr" {
			writeSyncer = zapcore.AddSync(os.Stderr)
		} else {
			file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				continue
			}
			writeSyncer = zapcore.AddSync(file)
		}
		cores = append(cores, zapcore.NewCore(encoder, writeSyncer, level))
	}
	
	// Combine cores
	core := zapcore.NewTee(cores...)
	
	// Build logger
	opts := []zap.Option{
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	}
	
	if cfg.Development {
		opts = append(opts, zap.Development())
	}
	
	zapLogger := zap.New(core, opts...)
	
	logger := &Logger{
		Logger: zapLogger,
		sugar:  zapLogger.Sugar(),
	}
	
	return logger, nil
}

// Default returns the default logger, initializing if needed
func Default() *Logger {
	if defaultLogger == nil {
		defaultLogger, _ = New(&Config{
			Level:       "info",
			Development: os.Getenv("ENV") != "production",
			Encoding:    "console",
			OutputPaths: []string{"stdout"},
		})
	}
	return defaultLogger
}

// SetDefault sets the default logger
func SetDefault(l *Logger) {
	defaultLogger = l
}

// WithContext returns a logger with context fields
func (l *Logger) WithContext(ctx context.Context) *Logger {
	// Extract common context values
	fields := []zap.Field{}
	
	if requestID, ok := ctx.Value("request_id").(string); ok {
		fields = append(fields, zap.String("request_id", requestID))
	}
	
	if userID, ok := ctx.Value("user_id").(string); ok {
		fields = append(fields, zap.String("user_id", userID))
	}
	
	return &Logger{
		Logger: l.With(fields...),
		sugar:  l.sugar.With(fields),
	}
}

// WithField adds a single field
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{
		Logger: l.With(zap.Any(key, value)),
		sugar:  l.sugar.With(key, value),
	}
}

// WithFields adds multiple fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}
	return &Logger{
		Logger: l.With(zapFields...),
		sugar:  l.sugar.With(fields),
	}
}

// WithError adds an error field
func (l *Logger) WithError(err error) *Logger {
	return &Logger{
		Logger: l.With(zap.Error(err)),
		sugar:  l.sugar.With("error", err),
	}
}

// WithDuration adds a duration field
func (l *Logger) WithDuration(d time.Duration) *Logger {
	return &Logger{
		Logger: l.With(zap.Duration("duration", d)),
		sugar:  l.sugar.With("duration", d),
	}
}

// =============================================================================
// CONVENIENCE METHODS
// =============================================================================

// Debug logs a debug message
func (l *Logger) Debug(msg string, fields ...zap.Field) {
	l.Logger.Debug(msg, fields...)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.sugar.Debugf(format, args...)
}

// Info logs an info message
func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.Logger.Info(msg, fields...)
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.sugar.Infof(format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields ...zap.Field) {
	l.Logger.Warn(msg, fields...)
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.sugar.Warnf(format, args...)
}

// Error logs an error message
func (l *Logger) Error(msg string, fields ...zap.Field) {
	l.Logger.Error(msg, fields...)
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.sugar.Errorf(format, args...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string, fields ...zap.Field) {
	l.Logger.Fatal(msg, fields...)
}

// Fatalf logs a formatted fatal message and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.sugar.Fatalf(format, args...)
}

// =============================================================================
// CONTEXT HELPERS
// =============================================================================

// ToContext adds logger to context
func ToContext(ctx context.Context, l *Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, l)
}

// FromContext extracts logger from context
func FromContext(ctx context.Context) *Logger {
	if l, ok := ctx.Value(contextKey{}).(*Logger); ok {
		return l
	}
	return Default()
}

// =============================================================================
// PACKAGE-LEVEL FUNCTIONS
// =============================================================================

func Debug(msg string, fields ...zap.Field) {
	Default().Debug(msg, fields...)
}

func Debugf(format string, args ...interface{}) {
	Default().Debugf(format, args...)
}

func Info(msg string, fields ...zap.Field) {
	Default().Info(msg, fields...)
}

func Infof(format string, args ...interface{}) {
	Default().Infof(format, args...)
}

func Warn(msg string, fields ...zap.Field) {
	Default().Warn(msg, fields...)
}

func Warnf(format string, args ...interface{}) {
	Default().Warnf(format, args...)
}

func Error(msg string, fields ...zap.Field) {
	Default().Error(msg, fields...)
}

func Errorf(format string, args ...interface{}) {
	Default().Errorf(format, args...)
}

func Fatal(msg string, fields ...zap.Field) {
	Default().Fatal(msg, fields...)
}

func Fatalf(format string, args ...interface{}) {
	Default().Fatalf(format, args...)
}
