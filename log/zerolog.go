package log

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) != 0 {
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "15:04:05.999",
		})
	} else {
		zerolog.TimeFieldFormat = time.RFC3339Nano
		zerolog.TimestampFieldName = "ts"
	}
}

// Level is a type for log-levels
type Level int8

// Level-values are matched with 'syslog'
const (
	TraceLevel Level = 8
	DebugLevel Level = 7
	InfoLevel  Level = 6
	// Notice = 5 -- not implemented here
	WarnLevel  Level = 4
	ErrorLevel Level = 3
	// Critical = 2 -- not implemented here
	PanicLevel Level = 1
	FatalLevel Level = 0
)

func translateLevel(l Level) zerolog.Level {
	if l > TraceLevel {
		l = TraceLevel
	}
	if l < FatalLevel {
		l = FatalLevel
	}
	switch l {
	case TraceLevel:
		return zerolog.TraceLevel
	case DebugLevel:
		return zerolog.DebugLevel
	case InfoLevel:
		return zerolog.InfoLevel
	case Level(5):
		return zerolog.InfoLevel
	case WarnLevel:
		return zerolog.WarnLevel
	case ErrorLevel:
		return zerolog.ErrorLevel
	case Level(2):
		return zerolog.ErrorLevel
	case PanicLevel:
		return zerolog.PanicLevel
	case FatalLevel:
		return zerolog.FatalLevel
	}
	return zerolog.InfoLevel
}

// WithLevel sets the log-level for the default logger
func WithLevel(l Level) {
	zl := translateLevel(l)
	if log.Logger.GetLevel() == zl {
		return
	}

	log.Logger = log.Logger.Level(zl)
}

// Logger returns a Logger-object
func Logger() zerolog.Logger {
	return log.Logger.With().Logger()
}

// FromCtx returns the logger from the context
func FromCtx(ctx context.Context) *zerolog.Logger {
	return log.Ctx(ctx)
}

// WithContext adds the default logger to the specified context
func WithContext(ctx context.Context) context.Context {
	log := log.Logger.With().Logger()
	return log.WithContext(ctx)
}

// Trace starts a new log-event with the 'trace'-level
func Trace() *zerolog.Event {
	return log.Trace()
}

// Debug starts a new log-event with the 'debug'-level
func Debug() *zerolog.Event {
	return log.Debug()
}

// Info starts a new log-event with the 'info'-level
func Info() *zerolog.Event {
	return log.Info()
}

// Warn starts a new log-event with the 'warn'-level
func Warn() *zerolog.Event {
	return log.Warn()
}

// Error starts a new log-event with the 'error'-level
func Error() *zerolog.Event {
	return log.Error()
}

// Panic starts a new log-event with the 'panic'-level
func Panic() *zerolog.Event {
	return log.Panic()
}

// Fatal starts a new log-event with the 'fatal'-level
func Fatal() *zerolog.Event {
	return log.Fatal()
}
