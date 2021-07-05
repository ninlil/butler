package log

import (
	"net/http"

	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
)

// NewHandler assigns the log to the context of each request
func NewHandler() func(http.Handler) http.Handler {
	return hlog.NewHandler(log.Logger)
}
