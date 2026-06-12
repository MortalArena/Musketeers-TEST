package observability

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger writes structured logs.
type Logger interface {
	Debug(msg string, fields map[string]any)
	Info(msg string, fields map[string]any)
	Warn(msg string, fields map[string]any)
	Error(msg string, err error, fields map[string]any)
	WithFields(fields map[string]any) Logger
	WithField(key string, value any) Logger
}

// ZapLogger is a Zap-backed Logger implementation.
type ZapLogger struct {
	logger *zap.Logger
	sugar  *zap.SugaredLogger
}

func NewZapLogger(level string) (*ZapLogger, error) {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(zapLevel),
		Development:      false,
		Encoding:         "json",
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "timestamp",
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
	}

	logger, err := config.Build()
	if err != nil {
		return nil, err
	}
	return &ZapLogger{logger: logger, sugar: logger.Sugar()}, nil
}

func (l *ZapLogger) Debug(msg string, fields map[string]any) {
	l.sugar.Debugw(msg, convertFields(fields)...)
}

func (l *ZapLogger) Info(msg string, fields map[string]any) {
	l.sugar.Infow(msg, convertFields(fields)...)
}

func (l *ZapLogger) Warn(msg string, fields map[string]any) {
	l.sugar.Warnw(msg, convertFields(fields)...)
}

func (l *ZapLogger) Error(msg string, err error, fields map[string]any) {
	if fields == nil {
		fields = make(map[string]any)
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	l.sugar.Errorw(msg, convertFields(fields)...)
}

func (l *ZapLogger) WithFields(fields map[string]any) Logger {
	return &ZapLogger{
		logger: l.logger.With(convertToZapFields(fields)...),
		sugar:  l.sugar.With(convertFields(fields)...),
	}
}

func (l *ZapLogger) WithField(key string, value any) Logger {
	return l.WithFields(map[string]any{key: value})
}

func convertFields(fields map[string]any) []any {
	if fields == nil {
		return nil
	}
	result := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		result = append(result, k, v)
	}
	return result
}

func convertToZapFields(fields map[string]any) []zap.Field {
	if fields == nil {
		return nil
	}
	result := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		result = append(result, zap.Any(k, v))
	}
	return result
}
