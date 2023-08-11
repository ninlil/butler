package log

import (
	"net/http"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
)

// NewHandler assigns the log to the context of each request
func NewHandler() func(http.Handler) http.Handler {
	return hlog.NewHandler(log.Logger)
}

// AddRequestFields adds the list of key-value pairs to the request context
func AddRequestFields(r *http.Request, keyvals ...interface{}) {
	rlog := log.Ctx(r.Context())
	rlog.UpdateContext(func(c zerolog.Context) zerolog.Context {
		for i := 0; i < len(keyvals)/2; i++ {
			if key, ok := keyvals[i*2].(string); ok {
				c = c.Interface(key, keyvals[i*2+1])
			}
		}
		return c
	})
}
