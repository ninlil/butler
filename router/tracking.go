package router

import (
	"context"
	"net/http"

	"github.com/rs/xid"
	"github.com/rs/zerolog"
)

// The following code is copied and adapted from github.com/rs/zerolog/hlog
// Need to handle incoming Request-ID and/or Correlation-Id differently

const (
	requestID     = "X-Request-Id"
	correlationID = "X-Correlation-Id"
)

// Handle the "Correlation-Id" header...
type corrIDKey struct{}

// CorrIDFromRequest returns the unique id associated to the request if any.
func CorrIDFromRequest(r *http.Request) (id string) {
	if r == nil {
		return
	}
	if cid := r.Header.Get(correlationID); cid != "" {
		return cid
	}
	return CorrIDFromCtx(r.Context())
}

// CorrIDFromCtx returns the unique id associated to the context if any.
func CorrIDFromCtx(ctx context.Context) (id string) {
	if id, ok := ctx.Value(corrIDKey{}).(string); ok {
		return id
	}
	return
}

// CtxWithCorrID adds the given correlation-id to the context
func CtxWithCorrID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, corrIDKey{}, id)
}

// Handle the "Request-Id" header...
type reqIDKey struct{}

// ReqIDFromRequest returns the unique id associated to the request if any.
func ReqIDFromRequest(r *http.Request) (id string) {
	if r == nil {
		return
	}
	if rid := r.Header.Get(requestID); rid != "" {
		return rid
	}
	return ReqIDFromCtx(r.Context())
}

// ReqIDFromCtx returns the unique id associated to the context if any.
func ReqIDFromCtx(ctx context.Context) (id string) {
	if id, ok := ctx.Value(reqIDKey{}).(string); ok {
		return id
	}
	return
}

// CtxWithReqID adds the given request-id to the context
func CtxWithReqID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, reqIDKey{}, id)
}

// IDHandler returns a handler setting a unique id to the request which can
// be gathered using IDFromRequest(req). This generated id is added as a field to the
// logger using the passed fieldKey as field name. The id is also added as a response
// header if the headerName is not empty.
//
// The generated id is a URL safe base64 encoded mongo object-id-like unique id.
// Mongo unique id generation algorithm has been selected as a trade-off between
// size and ease of use: UUID is less space efficient and snowflake requires machine
// configuration.
func IDHandler( /*fieldKey, headerName string*/ ) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := zerolog.Ctx(ctx)

			// Request-Id
			rid := ReqIDFromRequest(r)
			if rid == "" {
				rid = xid.New().String()
			}
			ctx = CtxWithReqID(ctx, rid)
			w.Header().Set(requestID, rid)

			// Correlation-Id
			cid := CorrIDFromRequest(r)
			if cid == "" {
				cid = xid.New().String()
			}
			ctx = CtxWithCorrID(ctx, cid)
			w.Header().Set(correlationID, cid)

			r = r.WithContext(ctx)

			log.UpdateContext(func(c zerolog.Context) zerolog.Context {
				// return c.Str(fieldKey, id)
				return c.Str("req_id", rid).Str("corr_id", cid)
			})

			next.ServeHTTP(w, r)
		})
	}
}
