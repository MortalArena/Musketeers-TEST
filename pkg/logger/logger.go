package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps zap logger
type Logger struct {
	*zap.Logger
	sugar *zap.SugaredLogger
}

// NewLogger creates a new logger
func NewLogger(level string, development bool) (*Logger, error) {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	config := zap.Config{
		Level:       zap.NewAtomicLevelAt(zapLevel),
		Development: development,
		Encoding:    "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	zapLogger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return &Logger{
		Logger: zapLogger,
		sugar:  zapLogger.Sugar(),
	}, nil
}

// NewProductionLogger creates a production logger
func NewProductionLogger() (*Logger, error) {
	zapLogger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}

	return &Logger{
		Logger: zapLogger,
		sugar:  zapLogger.Sugar(),
	}, nil
}

// NewDevelopmentLogger creates a development logger
func NewDevelopmentLogger() (*Logger, error) {
	zapLogger, err := zap.NewDevelopment()
	if err != nil {
		return nil, err
	}

	return &Logger{
		Logger: zapLogger,
		sugar:  zapLogger.Sugar(),
	}, nil
}

// NewNopLogger creates a no-op logger
func NewNopLogger() *Logger {
	return &Logger{
		Logger: zap.NewNop(),
		sugar:  zap.NewNop().Sugar(),
	}
}

// WithField adds a field to the logger
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{
		Logger: l.Logger.With(zap.Any(key, value)),
		sugar:  l.sugar.With(key, value),
	}
}

// WithFields adds multiple fields to the logger
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}
	return &Logger{
		Logger: l.Logger.With(zapFields...),
		sugar:  l.sugar.With(fields),
	}
}

// WithComponent adds component field
func (l *Logger) WithComponent(component string) *Logger {
	return l.WithField("component", component)
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, fields ...zap.Field) {
	l.Logger.Debug(msg, fields...)
}

// Info logs an info message
func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.Logger.Info(msg, fields...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields ...zap.Field) {
	l.Logger.Warn(msg, fields...)
}

// Error logs an error message
func (l *Logger) Error(msg string, fields ...zap.Field) {
	l.Logger.Error(msg, fields...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string, fields ...zap.Field) {
	l.Logger.Fatal(msg, fields...)
}

// Debugf logs a debug message with formatting
func (l *Logger) Debugf(template string, args ...interface{}) {
	l.sugar.Debugf(template, args...)
}

// Infof logs an info message with formatting
func (l *Logger) Infof(template string, args ...interface{}) {
	l.sugar.Infof(template, args...)
}

// Warnf logs a warning message with formatting
func (l *Logger) Warnf(template string, args ...interface{}) {
	l.sugar.Warnf(template, args...)
}

// Errorf logs an error message with formatting
func (l *Logger) Errorf(template string, args ...interface{}) {
	l.sugar.Errorf(template, args...)
}

// Fatalf logs a fatal message with formatting and exits
func (l *Logger) Fatalf(template string, args ...interface{}) {
	l.sugar.Fatalf(template, args...)
}

// WithError adds error field
func (l *Logger) WithError(err error) *Logger {
	return l.WithField("error", err)
}

// Sync flushes the logger
func (l *Logger) Sync() error {
	return l.Logger.Sync()
}

// Close closes the logger
func (l *Logger) Close() error {
	return l.Sync()
}

// Global logger instance
var globalLogger *Logger

// SetGlobalLogger sets the global logger
func SetGlobalLogger(logger *Logger) {
	globalLogger = logger
}

// GetGlobalLogger returns the global logger
func GetGlobalLogger() *Logger {
	if globalLogger == nil {
		logger, err := NewProductionLogger()
		if err != nil {
			// Fallback to stderr
			logger = NewNopLogger()
		}
		globalLogger = logger
	}
	return globalLogger
}

// InitGlobalLogger initializes the global logger
func InitGlobalLogger(level string, development bool) error {
	logger, err := NewLogger(level, development)
	if err != nil {
		return err
	}
	SetGlobalLogger(logger)
	return nil
}

// StdoutWriter returns a writer that logs to stdout
type StdoutWriter struct {
	logger *Logger
	level  zapcore.Level
}

// NewStdoutWriter creates a new stdout writer
func NewStdoutWriter(logger *Logger, level zapcore.Level) *StdoutWriter {
	return &StdoutWriter{
		logger: logger,
		level:  level,
	}
}

// Write implements io.Writer
func (w *StdoutWriter) Write(p []byte) (n int, err error) {
	msg := string(p)
	switch w.level {
	case zapcore.DebugLevel:
		w.logger.Debug(msg)
	case zapcore.InfoLevel:
		w.logger.Info(msg)
	case zapcore.WarnLevel:
		w.logger.Warn(msg)
	case zapcore.ErrorLevel:
		w.logger.Error(msg)
	default:
		w.logger.Info(msg)
	}
	return len(p), nil
}

// StderrWriter returns a writer that logs to stderr
type StderrWriter struct {
	logger *Logger
	level  zapcore.Level
}

// NewStderrWriter creates a new stderr writer
func NewStderrWriter(logger *Logger, level zapcore.Level) *StderrWriter {
	return &StderrWriter{
		logger: logger,
		level:  level,
	}
}

// Write implements io.Writer
func (w *StderrWriter) Write(p []byte) (n int, err error) {
	msg := string(p)
	switch w.level {
	case zapcore.DebugLevel:
		w.logger.Debug(msg)
	case zapcore.InfoLevel:
		w.logger.Info(msg)
	case zapcore.WarnLevel:
		w.logger.Warn(msg)
	case zapcore.ErrorLevel:
		w.logger.Error(msg)
	default:
		w.logger.Error(msg)
	}
	return len(p), nil
}
