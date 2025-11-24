package logger

import (
	"log/slog"
	"os"
)

type Logger struct {
	*slog.Logger
}

func New(level, env, app string) (*Logger, error) {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: true,
	}

	var handler slog.Handler
	if env == "debug" {
		handler = slog.NewTextHandler(os.Stderr, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	}

	logger := slog.New(handler).With(slog.String("app", app))
	l := &Logger{logger}
	l.Debug("Created new base logger")
	return l, nil
}

func (l Logger) WithSubsystem(name string) *Logger {
	subsystemLogger := l.With(slog.String("subsystem", name))
	subsystemLogger.Debug("Created new subsystem logger")
	return &Logger{subsystemLogger}
}
