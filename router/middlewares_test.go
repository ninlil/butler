package router

import (
	"bytes"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	zlog "github.com/rs/zerolog/log"
)

func TestAccessLoggerScheme(t *testing.T) {
	routes := []Route{
		{Name: "normal", Method: "GET", Path: "/normal", Handler: handlerReturnStatus},
		{Name: "stream", Method: "GET", Path: "/stream", Handler: handlerNoArgsNoReturn, Streaming: true},
	}

	for _, path := range []string{"/normal", "/stream"} {
		for _, tc := range []struct {
			name  string
			state *tls.ConnectionState
			want  string
		}{
			{"tls", &tls.ConnectionState{}, `"scheme":"https"`},
			{"plain", nil, `"scheme":"http"`},
		} {
			t.Run(path+" "+tc.name, func(t *testing.T) {
				orgLogger := zlog.Logger
				buf := new(bytes.Buffer)
				zlog.Logger = zlog.Output(buf)
				defer func() { zlog.Logger = orgLogger }()

				h := buildTestHandler(t, routes)
				buf.Reset()

				req := httptest.NewRequest("GET", path, nil)
				req.TLS = tc.state
				h.ServeHTTP(httptest.NewRecorder(), req)

				if !strings.Contains(buf.String(), tc.want) {
					t.Errorf("log = %q, want it to contain %s", buf.String(), tc.want)
				}
			})
		}
	}
}

func TestRequestScheme(t *testing.T) {
	req := &http.Request{}
	if got := requestScheme(req); got != "http" {
		t.Errorf("requestScheme = %q, want http", got)
	}
	req.TLS = &tls.ConnectionState{}
	if got := requestScheme(req); got != "https" {
		t.Errorf("requestScheme = %q, want https", got)
	}
}
