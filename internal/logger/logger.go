package logger

import (
	"context"
	"os"
	"github.com/charmbracelet/log"
)

type Logger struct {
	*log.Logger
}

func New(level string) *Logger {
	logger := log.NewWithOptions(os.Stdout, log.Options{
		ReportTimestamp: true,
		Level:           getLogLevel(level),
	})

	return &Logger{logger}
}

func getLogLevel(level string) log.Level {
	switch level {
	case "debug":
		return log.DebugLevel
	case "info":
		return log.InfoLevel
	case "warn", "warning":
		return log.WarnLevel
	case "error":
		return log.ErrorLevel
	default:
		return log.InfoLevel
	}
}

func (l *Logger) With(args ...any) *Logger {
	return &Logger{l.Logger.With(args...)}
}

func (l *Logger) WithError(err error) *Logger {
	return &Logger{l.Logger.With("error", err)}
}

func (l *Logger) WithContext(ctx context.Context) *Logger {
	return &Logger{l.Logger.With("request_id", getRequestID(ctx))}
}

func getRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value("request_id").(string); ok {
		return id
	}
	return ""
}
