package router

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/ninlil/butler/log"
)

type listener struct {
	name   string // "http" | "https"
	addr   string
	server *http.Server
	ln     net.Listener
}

func (r *Router) newListener(name string, port int, tlsCfg *tls.Config, handler http.Handler) *listener {
	addr := fmt.Sprintf(":%d", port)
	return &listener{
		name: name,
		addr: addr,
		server: &http.Server{
			Addr:      addr,
			Handler:   handler,
			TLSConfig: tlsCfg,
			BaseContext: func(net.Listener) context.Context {
				return r.baseCtx
			},
		},
	}
}

func (r *Router) buildListeners() []*listener {
	switch {
	case !r.hasTLS():
		return []*listener{r.newListener("http", r.port, nil, r.router)}
	case r.httpsPort == 0:
		return []*listener{r.newListener("https", r.port, r.tlsConfig, r.router)}
	}

	var plain http.Handler = r.router
	if r.httpsRedirect {
		plain = r.redirectHandler()
	}
	return []*listener{
		r.newListener("http", r.port, nil, plain),
		r.newListener("https", r.httpsPort, r.tlsConfig, r.router),
	}
}

// redirectHandler sends everything except the probes to HTTPS; Location is built from Host and the request URI only.
func (r *Router) redirectHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "" && (req.URL.Path == r.healthPath || req.URL.Path == r.readyPath) {
			r.router.ServeHTTP(w, req)
			return
		}

		host, _, err := net.SplitHostPort(req.Host)
		if err != nil {
			host = strings.TrimSuffix(strings.TrimPrefix(req.Host, "["), "]")
		}
		if host == "" {
			http.Error(w, "missing host", http.StatusBadRequest)
			return
		}

		authority := net.JoinHostPort(host, strconv.Itoa(r.httpsPort))
		if r.httpsPort == 443 {
			authority = host
			if strings.Contains(host, ":") {
				authority = "[" + host + "]"
			}
		}
		http.Redirect(w, req, "https://"+authority+req.URL.RequestURI(), http.StatusPermanentRedirect)
	})
}

// bind opens every listener or none: on failure the ones already opened are closed.
func (r *Router) bind(ls []*listener) error {
	for i, l := range ls {
		ln, err := net.Listen("tcp", l.addr)
		if err != nil {
			for _, opened := range ls[:i] {
				_ = opened.ln.Close()
				opened.ln = nil
			}
			return err
		}
		l.ln = ln
	}
	return nil
}

// run blocks until all listeners stop and returns the first failure other than http.ErrServerClosed.
func (r *Router) run(ls []*listener) error {
	var (
		wg    sync.WaitGroup
		once  sync.Once
		first error
	)
	for _, l := range ls {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var err error
			if l.name == "https" {
				err = l.server.ServeTLS(l.ln, "", "")
			} else {
				err = l.server.Serve(l.ln)
			}
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				once.Do(func() {
					first = err
					r.Shutdown()
				})
			}
		}()
	}
	wg.Wait()
	return first
}

func (r *Router) logListening(ls []*listener) {
	for _, l := range ls {
		log.Info().Msgf("router: listening to %s port %s%s", l.name, l.addr, r.prefix)
	}
}
