package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

var log zerolog.Logger

func Init(level, format string) {
	zerolog.TimeFieldFormat = time.RFC3339

	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}

	if format == "console" {
		log = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).
			Level(lvl).
			With().
			Timestamp().
			Caller().
			Logger()
	} else {
		log = zerolog.New(os.Stdout).
			Level(lvl).
			With().
			Timestamp().
			Logger()
	}
}

func Get() *zerolog.Logger {
	return &log
}

func Debug() *zerolog.Event {
	return log.Debug()
}

func Info() *zerolog.Event {
	return log.Info()
}

func Warn() *zerolog.Event {
	return log.Warn()
}

func Error() *zerolog.Event {
	return log.Error()
}

func Fatal() *zerolog.Event {
	return log.Fatal()
}

func With() zerolog.Context {
	return log.With()
}

func WithRequestID(requestID string) zerolog.Logger {
	return log.With().Str("request_id", requestID).Logger()
}

func WithUserID(userID string) zerolog.Logger {
	return log.With().Str("user_id", userID).Logger()
}
