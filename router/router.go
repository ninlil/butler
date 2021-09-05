package router

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	"github.com/ninlil/butler/log"
	"github.com/ninlil/butler/runtime"
)

// Router constants
const (
	All = "*"
)

// Misc types we reuse a lot
var (
	tResponseWriter = reflect.TypeOf(new(http.ResponseWriter)).Elem()
	tRequest        = reflect.TypeOf(new(http.Request))
	tContext        = reflect.TypeOf(new(context.Context)).Elem()
	// tError          = reflect.TypeOf(new(error)).Elem()
	tTime = reflect.TypeOf(time.Now())
	tDur  = reflect.TypeOf(time.Second)
	// tInt            = reflect.TypeOf(int(0))
)

type errHandlerNotAFunc Route

func (e errHandlerNotAFunc) Error() string {
	return fmt.Sprintf("error: handler for %s '%s' is not a function", e.Method, e.Path)
}

// Route contains the definition of a REST-API with method, path, handler and more
type Route struct {
	Name    string
	Method  string
	Path    string
	Handler interface{}
	fnType  reflect.Type
	fnValue reflect.Value
}

// Router is the handler which serves your routes
type Router struct {
	// options
	strictSlash   bool
	port          int
	healthPath    string
	readyPath     string
	prefix        string
	exposedErrors bool

	// runtime
	router *mux.Router
	routes []*Route
	server *http.Server
}

func (rt *Route) init() error {
	if rt.Handler == nil {
		return nil
	}
	rt.fnType = reflect.TypeOf(rt.Handler)
	rt.fnValue = reflect.ValueOf(rt.Handler)

	if rt.fnType.Kind() != reflect.Func {
		return errHandlerNotAFunc(*rt)
	}
	return nil
}

var defaultRouter *Router

// Serve starts the default router with the supplied routes on the specified port
func Serve(routes []Route, opts ...Option) error {
	var err error
	defaultRouter, err = New(routes, opts...)
	if err != nil {
		return err
	}
	go defaultRouter.goServe()
	return nil
}

// Shutdown does a graceful shutdown on the default router
func Shutdown() {
	defaultRouter.Shutdown()
}

// New creates a custom Router using the supplied routes
func New(routes []Route, opts ...Option) (*Router, error) {
	router := &Router{
		strictSlash: true,
		port:        10000,
		routes:      make([]*Route, 0, len(routes)),
		healthPath:  "/healthz",
		readyPath:   "/readyz",
	}

	for _, opt := range opts {
		if err := opt(router); err != nil {
			return nil, err
		}
	}

	router.router = mux.NewRouter().StrictSlash(router.strictSlash)

	for i := range routes {
		route := routes[i]
		if err := route.init(); err != nil {
			log.Fatal().Msg(err.Error())
		}
		router.routes = append(router.routes, &route)
	}

	return router, nil
}

func (r *Router) goServe() {
	if err := r.Serve(); err != nil && err != http.ErrServerClosed {
		log.Error().Msgf("router.Serve-error: %v", err)
	}
}

// Serve starts the http-server on the router
func (r *Router) Serve() error {
	if r.server != nil {
		return ErrRouterAlreadyRunning
	}
	var haveReady bool
	var haveHealty bool

	for _, route := range r.routes {
		// log.Trace().Msgf("adding %s %s", route.Method, route.Path)
		h := r.router.NewRoute().Name(route.Name)

		if route.Method != All {
			h = h.Methods(route.Method)
		}

		if route.Path == "/" && r.prefix != "" {
			route.Path = r.prefix
		} else {
			route.Path = r.prefix + route.Path
		}
		if route.Path != "" {
			h = h.Path(route.Path)
		}
		// log.Trace().Msgf("%s -> %s", route.Name, route.Path)

		switch route.Path {
		case r.readyPath:
			haveReady = true
		case r.healthPath:
			haveHealty = true
		}

		chain := alice.New().Append(wrapWriterMW)

		chain = chain.Append(log.NewHandler())
		chain = chain.Append(IDHandler())
		chain = chain.Append(accessLogger)
		// chain = chain.Append(hlog.RemoteAddrHandler("ip"))
		// chain = chain.Append(hlog.UserAgentHandler("user_agent"))
		// chain = chain.Append(hlog.RefererHandler("referer"))
		// chain = chain.Append(hlog.RequestIDHandler("req_id", "Request-Id"))

		fn, err := route.wrapHandler()
		if err != nil {
			log.Fatal().Msg(err.Error())
		}

		h.Handler(chain.Append(r.panicHandler).ThenFunc(fn))
	}

	if !haveHealty && r.healthPath != "" {
		// log.Trace().Msg("adding /healtyz")
		r.router.NewRoute().Name("healthyz").Methods("GET").Path(r.healthPath).HandlerFunc(healthyProbe)
	}
	if !haveReady && r.readyPath != "" {
		// log.Trace().Msg("adding /readyz")
		r.router.NewRoute().Name("readyz").Methods("GET").Path(r.readyPath).HandlerFunc(readyProbe)
	}

	runtime.OnClose("router", r.Shutdown)

	log.Info().Msgf("listening to port %s:%d%s", "", r.port, r.prefix)

	r.server = &http.Server{Addr: fmt.Sprintf(":%d", r.port), Handler: r.router}
	return r.server.ListenAndServe()
}

// Shutdown does a graceful shutdown of the router
func (r *Router) Shutdown() {
	if r.server == nil {
		return
	}
	log.Warn().Msgf("router-shutdown initiated...")

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	if err := r.server.Shutdown(ctx); err != nil && err != http.ErrServerClosed {
		log.Error().Msgf("router-shutdown-error: %v", err)
	}
	r.server = nil
	log.Warn().Msgf("router-shutdown complete")
}

func (rt *Route) wrapHandler() (http.HandlerFunc, error) {
	if h, ok := rt.Handler.(http.HandlerFunc); ok {
		return h, nil
	}

	return func(w http.ResponseWriter, r *http.Request) {
		rt.wrap(w, r)
	}, nil
}

func (rt *Route) writeError(err error, w http.ResponseWriter, r *http.Request, code int) {
	if code == 0 {
		code = http.StatusBadRequest
	}
	var result struct {
		Error interface{} `json:"error"`
	}

	if fe, ok := err.(FieldError); ok {
		result.Error = fe
	} else {
		result.Error = err.Error()
	}
	rt.writeResponse(w, r, code, result)
}

func (rt *Route) wrap(w http.ResponseWriter, r *http.Request) {
	log := log.FromCtx(r.Context())

	args, err := rt.createArgs(w, r)
	if err != nil {
		log.Error().Msg(err.Error())
		rt.writeError(err, w, r, 0)
		return
	}

	// log.Trace().Msgf("wrap - calling...")
	results := rt.fnValue.Call(args)
	// log.Trace().Msgf("wrap - result: %d values", len(results))

	var status int
	var data interface{}

	for _, res := range results {
		if res.CanInterface() {
			o := res.Interface()
			switch v := o.(type) {
			case error:
				err = v
				// log.Debug().Msgf("#%d: error  = %v", i, err)
			case int:
				status = v
				// log.Debug().Msgf("#%d: status = %+v", i, status)
			default:
				if v != nil { // this stop nil-error to be parsed as data
					data = v
				}
				// log.Debug().Msgf("#%d: data   = %+v", i, data)
			}
		}
	}

	if err != nil {
		rt.writeError(err, w, r, status)
		return
	}

	rt.writeResponse(w, r, status, data)
}
