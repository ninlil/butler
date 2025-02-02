package router

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
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

type runningData struct {
	routers map[string]*Router
	wg      *sync.WaitGroup
}

var running = new(runningData)

func (run *runningData) addRouter(r *Router) error {

	if _, duplicate := run.routers[r.name]; duplicate {
		return ErrRouterDuplicateName
	}

	var first bool
	if run.routers == nil {
		first = true
		run.routers = make(map[string]*Router)
	}
	if run.wg == nil {
		run.wg = new(sync.WaitGroup)
	}
	run.routers[r.name] = r
	run.wg.Add(1)

	if first {
		go func() {
			log.Trace().Msg("router: waiting for running router(s)")
			run.wg.Wait()
			log.Trace().Msg("router: all routers have stopped.. stopping runtime")
			runtime.Close()
		}()
	}

	return nil
}

func (run *runningData) Done(name string) {
	log.Trace().Msgf("router: [%s] is done", name)
	if run.wg == nil {
		return
	}
	run.wg.Done()
}

func (run *runningData) Wait() {
	if run.wg == nil {
		return
	}
	run.wg.Wait()
}

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
	isRaw   bool // if Handler is a regular http.HandlerFunc, then no wrapping is needed

	router *Router
}

// Router is the handler which serves your routes
type Router struct {
	// options
	name          string
	strictSlash   bool
	port          int
	healthPath    string
	readyPath     string
	prefix        string
	exposedErrors bool
	skip204       bool

	// runtime
	router *chi.Mux
	routes []*Route
	server *http.Server
	mutex  sync.Mutex
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
	_, rt.isRaw = rt.Handler.(func(http.ResponseWriter, *http.Request))

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
	if router.name == "" {
		router.name = "default"
	}

	router.router = chi.NewRouter()

	for i := range routes {
		route := routes[i]
		if err := route.init(); err != nil {
			log.Fatal().Msg(err.Error())
		}
		router.routes = append(router.routes, &route)
	}

	for i := range router.routes {
		router.routes[i].router = router
	}

	return router, nil
}

func (r *Router) goServe() {
	if err := r.Serve(); err != nil && !errors.Is(err, http.ErrServerClosed) {
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

	//r.router.Use(wrapWriterMW, log.NewHandler(), IDHandler(), accessLogger)

	for _, route := range r.routes {
		// log.Trace().Msgf("router: adding %s %s", route.Method, route.Path)
		// h := r.router.NewRoute().Name(route.Name)
		var method = "GET"

		if route.Method != "" {
			method = route.Method
		}

		if route.Path == "/" && r.prefix != "" {
			route.Path = r.prefix
		} else {
			route.Path = r.prefix + route.Path
		}
		// log.Trace().Msgf("router: %s -> %s", route.Name, route.Path)

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

		handler := chain.Append(r.panicHandler).ThenFunc(route.wrapHandler())
		if method == "*" {
			r.router.Handle(route.Path, handler) // allow any method
		} else {
			r.router.Method(method, route.Path, handler)
		}
	}

	if !haveHealty && r.healthPath != "" {
		// log.Trace().Msg("router: adding /healtyz")
		r.router.Get(r.healthPath, healthyProbe)
	}
	if !haveReady && r.readyPath != "" {
		// log.Trace().Msg("router: adding /readyz")
		r.router.Get(r.readyPath, readyProbe)
	}

	if err := running.addRouter(r); err != nil {
		return err
	}
	defer running.Done(r.name)

	runtime.OnClose("router_"+r.name, r.Shutdown)

	log.Info().Msgf("router: listening to port %s:%d%s", "", r.port, r.prefix)

	r.server = &http.Server{Addr: fmt.Sprintf(":%d", r.port), Handler: r.router}
	return r.server.ListenAndServe()
}

// Shutdown does a graceful shutdown of the router
func (r *Router) Shutdown() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.server == nil {
		return
	}
	log.Trace().Msg("router: shutdown initiated...")

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	err := r.server.Shutdown(ctx)
	if err != nil && errors.Is(err, http.ErrServerClosed) {
		log.Error().Msgf("router: shutdown-error: %v", err)
	}
	r.server = nil
	log.Trace().Msg("router: shutdown complete")
}

func (rt *Route) wrapHandler() http.HandlerFunc {
	if rt.isRaw {
		return rt.Handler.(func(http.ResponseWriter, *http.Request))
	}

	return func(w http.ResponseWriter, r *http.Request) {
		rt.wrap(w, r)
	}
}

func (rt *Route) writeError(err error, w http.ResponseWriter, r *http.Request, code int) {
	if code == 0 {
		code = http.StatusBadRequest
	}
	var result struct {
		Error interface{} `json:"error"`
	}
	var fe *FieldError
	if errors.As(err, &fe) {
		result.Error = fe
	} else {
		result.Error = err.Error()
	}
	rt.writeResponse(w, r, code, result)
}

func (rt *Route) wrap(w http.ResponseWriter, r *http.Request) {
	log := log.FromCtx(r.Context())
	defer r.Body.Close()

	args, err := rt.createArgs(w, r)
	if err != nil {
		log.Error().Msg(err.Error())
		rt.writeError(err, w, r, 0)
		return
	}

	// log.Trace().Msgf("router: wrap - calling...")
	results := rt.fnValue.Call(args)
	// log.Trace().Msgf("router: wrap - result: %d values", len(results))

	var status int
	var data interface{}

	for _, res := range results {
		if res.CanInterface() {
			o := res.Interface()
			switch v := o.(type) {
			case error:
				err = v
				// log.Debug().Msgf("router: #%d: error  = %v", i, err)
			case int:
				status = v
				// log.Debug().Msgf("router: #%d: status = %+v", i, status)
			default:
				if v != nil { // this stop nil-error to be parsed as data
					data = v
				}
				// log.Debug().Msgf("router: #%d: data   = %+v", i, data)
			}
		}
	}

	if err != nil {
		rt.writeError(err, w, r, status)
		return
	}

	rt.writeResponse(w, r, status, data)
}
