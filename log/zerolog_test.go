package log

import (
	"bytes"
	"testing"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

type testLogger struct {
	buf *bytes.Buffer
}

func newTestLogger() *testLogger {
	return &testLogger{
		buf: new(bytes.Buffer),
	}
}

func (tl *testLogger) Write(p []byte) (n int, err error) {
	return tl.buf.Write(p)
}

func (tl *testLogger) Reset() {
	tl.buf.Truncate(0)
}

func (tl *testLogger) Match(name, t string) bool {
	a := tl.buf.String()
	//fmt.Printf("%s-match? '%s' ?= '%s'\n", name, a, t)
	return a == t
}

func TestWithLevel(t *testing.T) {
	orgLogger := zlog.Logger
	tl := newTestLogger()
	zlog.Logger = zlog.Output(zerolog.ConsoleWriter{
		Out:        tl,
		TimeFormat: " ",
		NoColor:    true,
	})

	tests := []struct {
		name        string
		filterLevel Level
		fn          func() *zerolog.Event
		msg         string
		result      string
		expect      bool
	}{
		{"trace->trace", TraceLevel, Trace, "trace", "  TRC trace\n", true},
		{"debug->trace", TraceLevel, Debug, "debug", "  DBG debug\n", true},
		{"trace->debug-fail", DebugLevel, Trace, "trace", "  TRC trace\n", false},
		{"trace->debug-ok", DebugLevel, Trace, "trace", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tl.Reset()
			WithLevel(tt.filterLevel)
			tt.fn().Msg(tt.msg)
			if tl.Match(tt.name, tt.result) != tt.expect {
				t.Fail()
			}
		})
	}

	zlog.Logger = orgLogger
}

func TestTranslateLevel(t *testing.T) {
	tests := []struct {
		name  string
		input Level
		want  zerolog.Level
	}{
		{"TraceLevel", TraceLevel, zerolog.TraceLevel},
		{"DebugLevel", DebugLevel, zerolog.DebugLevel},
		{"InfoLevel", InfoLevel, zerolog.InfoLevel},
		{"WarnLevel", WarnLevel, zerolog.WarnLevel},
		{"ErrorLevel", ErrorLevel, zerolog.ErrorLevel},
		{"PanicLevel", PanicLevel, zerolog.PanicLevel},
		{"FatalLevel", FatalLevel, zerolog.FatalLevel},
		{"above-trace clamps", Level(9), zerolog.TraceLevel},
		{"below-fatal clamps", Level(-1), zerolog.FatalLevel},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := translateLevel(tt.input)
			if got != tt.want {
				t.Errorf("translateLevel(%d) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
