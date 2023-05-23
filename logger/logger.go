package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

type Logger struct {
	zerolog.Logger
}

func New(level string, env string) (*Logger, error) {

	switch level {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "fatal":
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	case "panic":
		zerolog.SetGlobalLevel(zerolog.PanicLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	var output io.Writer
	if env == "debug" {
		zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
		output = zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.RFC3339,
			FormatLevel: func(i interface{}) string {
				return strings.ToUpper(fmt.Sprintf("[%s]", i))
			},
			FormatMessage: func(i interface{}) string {
				return fmt.Sprintf("| %s |", i)
			},
		}
	} else {
		output = os.Stderr
	}

	zerologger := zerolog.New(output).
		With().
		Caller().
		Timestamp().
		Logger()

	logger := Logger{zerologger}
	logger.Debug().Msg("Created new base logger")
	return &logger, nil
}

func (l Logger) WithService(name string) *Logger {
	serviceLogger := l.Logger.With().Str("service", name).Logger()
	serviceLogger.Debug().Msg("Created new service logger")
	return &Logger{
		serviceLogger,
	}
}

func (l Logger) WithSlackNotifier(url string) *Logger {
	notifier := newSlackNotifier(url)
	slackLogger := l.Hook(notifier)
	slackLogger.Debug().Msg("Added slack notifier to logger")
	return &Logger{
		slackLogger,
	}
}
