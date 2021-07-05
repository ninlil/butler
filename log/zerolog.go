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

// FromCtx returns the logger from the context
func FromCtx(ctx context.Context) *zerolog.Logger {
	return log.Ctx(ctx)
}

// Trace starts a new log-event with the 'trace'-level
func Trace() *zerolog.Event {
	return log.Trace()
}

// Debug starts a new log-event with the 'bebug'-level
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