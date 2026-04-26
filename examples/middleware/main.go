package main

import (
	"context"
	"net/http"
	"time"

	"github.com/ninlil/butler"
	"github.com/ninlil/butler/log"
	"github.com/ninlil/butler/router"
)

// --- routes ---

var routes = []router.Route{
	{Name: "public", Method: "GET", Path: "/public", Handler: publicHandler},
	{Name: "protected", Method: "GET", Path: "/protected", Handler: protectedHandler},
}

func main() {
	defer butler.Cleanup(nil)

	err := router.Serve(routes,
		router.WithPort(10000),
		router.WithMiddleware(
			timingMiddleware, // measures and logs wall-clock time per request
			apiKeyMiddleware, // rejects requests missing the correct X-API-Key header
		),
	)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	butler.Run()
}

// --- handlers ---

func publicHandler() string {
	return "this route is public — but the middleware still ran"
}

type protectedArgs struct {
	User string `json:"x-api-user" from:"header"`
}

func protectedHandler(args *protectedArgs) string {
	return "hello, " + args.User
}

// --- middlewares ---

// timingMiddleware measures wall-clock time and adds it to the zerolog context so the
// access-logger (already in the chain) can pick it up.  It also demonstrates that
// log.FromCtx works here because butler's log-handler runs earlier in the chain.
func timingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.FromCtx(r.Context()).Debug().
			Dur("handler_ns", time.Since(start)).
			Msg("timing")
	})
}

// apiKeyMiddleware is a minimal API-key gate.  Any middleware with the standard
// func(http.Handler) http.Handler signature works here — including chi middleware,
// gorilla handlers, etc.
//
// Try:
//
//	curl -i http://localhost:10000/protected
//	curl -i -H "X-API-Key: secret" -H "X-API-User: alice" http://localhost:10000/protected
const validKey = "secret"

func apiKeyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != validKey {
			log.FromCtx(r.Context()).Warn().Msg("rejected: missing or invalid X-API-Key")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		// Stash the validated user in the context so downstream handlers
		// (or further middlewares) can read it without re-parsing the header.
		ctx := context.WithValue(r.Context(), ctxKeyUser{}, r.Header.Get("X-API-User"))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type ctxKeyUser struct{}
