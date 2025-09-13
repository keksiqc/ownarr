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
	var logLevel log.Level
	switch level {
	case "debug":
		logLevel = log.DebugLevel
	case "info":
		logLevel = log.InfoLevel
	case "warn", "warning":
		logLevel = log.WarnLevel
	case "error":
		logLevel = log.ErrorLevel
	default:
		logLevel = log.InfoLevel
	}

	logger := log.NewWithOptions(os.Stdout, log.Options{
		Level:           logLevel,
		ReportCaller:    false,
		ReportTimestamp: true,
		TimeFormat:      "2006-01-02 15:04:05",
	})

	return &Logger{logger}
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
