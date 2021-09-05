package router

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/ninlil/butler/bufferedresponse"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
)

func wrapWriterMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w2 := bufferedresponse.Wrap(w)
		next.ServeHTTP(http.ResponseWriter(w2), r)
		w2.Flush()
	})
}

func (r *Router) panicHandler(next http.Handler) http.Handler {
	var exposedErrors = r.exposedErrors
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				if w2, ok := bufferedresponse.Get(w); ok {
					w2.Reset()
				}
				hlog.FromRequest(r).WithLevel(zerolog.PanicLevel).Caller(2).Msg(fmt.Sprint(err))
				w.WriteHeader(http.StatusInternalServerError)
				if exposedErrors {
					_, _ = w.Write([]byte(fmt.Sprint(err)))
				}
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func accessLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w2, _ := bufferedresponse.Get(w)
		log := hlog.FromRequest(r)
		route := mux.CurrentRoute(r)

		start := time.Now()
		next.ServeHTTP(w, r)
		dur := time.Since(start)

		var e *zerolog.Event
		switch true {
		case w2.Status() < 200:
			e = log.Debug()
		case w2.Status() < 400:
			e = log.Info()
		case w2.Status() < 500:
			e = log.Warn()
		default:
			e = log.Error()
		}

		e.Dur("duration", dur)
		e.Int("status", w2.Status())
		e.Int("size", w2.Size())

		if route != nil {
			e.Str("route", route.GetName())
			// if path, err := route.GetPathTemplate(); err == nil {
			// 	e.Str("path", path)
			// }
		}

		e.Msgf("%s %s", r.Method, r.URL.Path)
	})
	// hlog.FromRequest(r).Info().
	// Str("method", r.Method).
	// Stringer("url", r.URL).
	// Int("status", status).
	// Int("size", size).
	// Dur("duration", duration).
	// Msg("")

	// return nil
}
