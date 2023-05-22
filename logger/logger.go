package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

type Logger struct {
	zerolog.Logger
}

func New(level string, env string) (Logger, error) {

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

	var output io.Writer

	if env == "prod" || env == "production" {
		output = zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.RFC3339,
		}
	} else {
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
			FormatCaller: func(i interface{}) string {
				return filepath.Base(fmt.Sprintf("%s", i))
			},
		}
	}

	var gitRevision string
	buildInfo, ok := debug.ReadBuildInfo()
	if ok {
		for _, v := range buildInfo.Settings {
			if v.Key == "vcs.revision" {
				gitRevision = v.Value
				break
			}
		}
	}

	zerologger := zerolog.New(output).
		With().
		Timestamp().
		Int("pid", os.Getpid()).
		Str("git_revision", gitRevision).
		Str("go_version", buildInfo.GoVersion).
		Logger()

	logger := Logger{zerologger}
	logger.Debug().Msg("Created new base logger")
	return logger, nil
}

func (l Logger) WithService(name string) Logger {
	serviceLogger := l.Logger.With().Str("service", name).Logger()
	serviceLogger.Debug().Msg("Created new service logger")
	return Logger{
		serviceLogger,
	}
}
