package router

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func schemeHandler(w http.ResponseWriter, req *http.Request) {
	if req.TLS != nil {
		_, _ = w.Write([]byte("tls"))
		return
	}
	_, _ = w.Write([]byte("plain"))
}

var helloRoutes = []Route{{Name: "hello", Method: "GET", Path: "/hello", Handler: schemeHandler}}

func testURL(scheme string, port int, path string) string {
	return fmt.Sprintf("%s://127.0.0.1:%d%s", scheme, port, path)
}

func refuses(port int) bool {
	c, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), time.Second)
	if err == nil {
		_ = c.Close()
		return false
	}
	return true
}

func listenerCount(r *Router) int {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return len(r.listeners)
}

func TestServeHTTPOnly(t *testing.T) {
	_, _, pool := writeSelfSignedCert(t)
	p := freePort(t)
	startRouter(t, helloRoutes, WithPort(p))

	if code, body := doGet(t, plainClient(), testURL("http", p, "/hello")); code != 200 || body != "plain" {
		t.Errorf("got %d %q, want 200 plain", code, body)
	}
	if _, err := tlsClient(pool).Get(testURL("https", p, "/hello")); err == nil {
		t.Error("https to a plain port succeeded")
	}
}

func TestServeHTTPSOnly(t *testing.T) {
	certFile, keyFile, pool := writeSelfSignedCert(t)
	p := freePort(t)
	r, _ := startRouter(t, helloRoutes, WithPort(p), WithTLS(certFile, keyFile))

	if code, body := doGet(t, tlsClient(pool), testURL("https", p, "/hello")); code != 200 || body != "tls" {
		t.Errorf("got %d %q, want 200 tls", code, body)
	}
	if resp, err := plainClient().Get(testURL("http", p, "/hello")); err == nil {
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == 200 {
			t.Error("plain http to a TLS port returned 200")
		}
	}
	if n := listenerCount(r); n != 1 {
		t.Errorf("listeners = %d, want 1", n)
	}
}

func TestServeBoth(t *testing.T) {
	certFile, keyFile, pool := writeSelfSignedCert(t)
	ports := freePorts(t, 2)
	startRouter(t, helloRoutes, WithPorts(ports[0], ports[1]), WithTLS(certFile, keyFile))

	assertBoth(t, ports[0], ports[1], tlsClient(pool))
}

func TestServeBothTLSConfig(t *testing.T) {
	certFile, keyFile, pool := writeSelfSignedCert(t)
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		t.Fatal(err)
	}
	ports := freePorts(t, 2)
	startRouter(t, helloRoutes,
		WithPorts(ports[0], ports[1]),
		WithTLSConfig(&tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}))

	assertBoth(t, ports[0], ports[1], tlsClient(pool))
}

func assertBoth(t *testing.T, httpPort, httpsPort int, secure *http.Client) {
	t.Helper()
	plain := plainClient()
	for _, path := range []string{"/hello", "/healthz", "/readyz"} {
		if code, _ := doGet(t, plain, testURL("http", httpPort, path)); code != 200 {
			t.Errorf("http %s = %d, want 200", path, code)
		}
		if code, _ := doGet(t, secure, testURL("https", httpsPort, path)); code != 200 {
			t.Errorf("https %s = %d, want 200", path, code)
		}
	}
	if _, body := doGet(t, plain, testURL("http", httpPort, "/hello")); body != "plain" {
		t.Errorf("http port body = %q, want plain", body)
	}
	if _, body := doGet(t, secure, testURL("https", httpsPort, "/hello")); body != "tls" {
		t.Errorf("https port body = %q, want tls", body)
	}
}

func TestShutdownBoth(t *testing.T) {
	certFile, keyFile, _ := writeSelfSignedCert(t)
	ports := freePorts(t, 2)
	r, res := startRouter(t, helloRoutes, WithPorts(ports[0], ports[1]), WithTLS(certFile, keyFile))

	r.Shutdown()

	if err := res.wait(t, 5*time.Second); err != nil {
		t.Errorf("Serve() = %v, want nil", err)
	}
	for _, p := range ports {
		if !refuses(p) {
			t.Errorf("port %d still accepting connections", p)
		}
	}
}

func TestShutdownCancelsStreaming(t *testing.T) {
	certFile, keyFile, pool := writeSelfSignedCert(t)
	ports := freePorts(t, 2)

	started := make(chan struct{}, 2)
	finished := make(chan struct{}, 2)
	routes := []Route{{Name: "stream", Method: "GET", Path: "/stream", Streaming: true,
		Handler: func(w http.ResponseWriter, req *http.Request) {
			started <- struct{}{}
			<-req.Context().Done()
			finished <- struct{}{}
		}}}
	r, _ := startRouter(t, routes, WithPorts(ports[0], ports[1]), WithTLS(certFile, keyFile))

	go func() { _, _ = plainClient().Get(testURL("http", ports[0], "/stream")) }()
	go func() { _, _ = tlsClient(pool).Get(testURL("https", ports[1], "/stream")) }()
	for range 2 {
		select {
		case <-started:
		case <-time.After(5 * time.Second):
			t.Fatal("streaming handler did not start")
		}
	}

	r.Shutdown()

	for range 2 {
		select {
		case <-finished:
		case <-time.After(5 * time.Second):
			t.Fatal("streaming handler was not cancelled by Shutdown")
		}
	}
}

func TestServeTwiceReturnsAlreadyRunning(t *testing.T) {
	r, _ := startRouter(t, helloRoutes, WithPort(freePort(t)))
	if err := r.Serve(); !errors.Is(err, ErrRouterAlreadyRunning) {
		t.Errorf("second Serve() = %v, want ErrRouterAlreadyRunning", err)
	}
}

func TestServeBindErrorFirstListener(t *testing.T) {
	testBindError(t, 0)
}

func TestServeBindErrorSecondListener(t *testing.T) {
	testBindError(t, 1)
}

func testBindError(t *testing.T, occupy int) {
	t.Helper()
	certFile, keyFile, _ := writeSelfSignedCert(t)
	ports := freePorts(t, 2)

	blocker, err := net.Listen("tcp", fmt.Sprintf(":%d", ports[occupy]))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = blocker.Close() }()

	r := newTestRouter(t, helloRoutes, WithPorts(ports[0], ports[1]), WithTLS(certFile, keyFile))
	if err := r.Serve(); err == nil {
		t.Fatal("Serve() = nil, want bind error")
	}

	other := ports[1-occupy]
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", other))
	if err != nil {
		t.Errorf("port %d left bound after bind failure: %v", other, err)
	} else {
		_ = ln.Close()
	}
	r.Shutdown()
}

func TestShutdownBeforeServe(t *testing.T) {
	r := newTestRouter(t, helloRoutes, WithPort(freePort(t)))
	r.Shutdown()
}

func TestRuntimeListenerFailure(t *testing.T) {
	certFile, keyFile, _ := writeSelfSignedCert(t)
	ports := freePorts(t, 2)
	r := newTestRouter(t, helloRoutes, WithPorts(ports[0], ports[1]), WithTLS(certFile, keyFile))

	r.baseCtx, r.baseCtxClose = context.WithCancel(context.Background())
	ls := r.buildListeners()
	if err := r.bind(ls); err != nil {
		t.Fatal(err)
	}
	r.listeners = ls

	errc := make(chan error, 1)
	go func() { errc <- r.run(ls) }()

	_ = ls[0].ln.Close()

	select {
	case err := <-errc:
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			t.Errorf("run() = %v, want a listener error", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("run() did not return after a listener failed")
	}
	if !refuses(ports[1]) {
		t.Error("sibling listener still accepting connections")
	}
}

func TestShutdownDuringStartup(t *testing.T) {
	for i := range 50 {
		r := newTestRouter(t, helloRoutes, WithPort(freePort(t)))
		res := goServeResult(r)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				r.Shutdown()
				select {
				case <-res.done:
					return
				case <-time.After(time.Millisecond):
				}
			}
		}()

		if err := res.wait(t, 5*time.Second); err != nil {
			t.Fatalf("iteration %d: Serve() = %v", i, err)
		}
		wg.Wait()
	}
}

func noRedirectClient() *http.Client {
	c := plainClient()
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return c
}

func startRedirectRouter(t *testing.T, extra ...Option) (httpPort, httpsPort int) {
	t.Helper()
	certFile, keyFile, _ := writeSelfSignedCert(t)
	ports := freePorts(t, 2)
	opts := append([]Option{WithPorts(ports[0], ports[1]), WithTLS(certFile, keyFile), WithHTTPSRedirect()}, extra...)
	startRouter(t, helloRoutes, opts...)
	return ports[0], ports[1]
}

func TestRedirectPlainToHTTPS(t *testing.T) {
	h, s := startRedirectRouter(t)

	resp, err := noRedirectClient().Get(testURL("http", h, "/hello?x=1"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusPermanentRedirect {
		t.Errorf("status = %d, want 308", resp.StatusCode)
	}
	if got, want := resp.Header.Get("Location"), testURL("https", s, "/hello?x=1"); got != want {
		t.Errorf("Location = %q, want %q", got, want)
	}
}

func TestRedirectProbesNotRedirected(t *testing.T) {
	t.Run("default probes", func(t *testing.T) {
		h, _ := startRedirectRouter(t)
		for _, path := range []string{"/healthz", "/readyz"} {
			if code, _ := doGet(t, noRedirectClient(), testURL("http", h, path)); code != 200 {
				t.Errorf("%s = %d, want 200", path, code)
			}
		}
	})

	t.Run("custom health path", func(t *testing.T) {
		h, _ := startRedirectRouter(t, WithHealth("/custom-health"))
		if code, _ := doGet(t, noRedirectClient(), testURL("http", h, "/custom-health")); code != 200 {
			t.Errorf("/custom-health = %d, want 200", code)
		}
		if code, _ := doGet(t, noRedirectClient(), testURL("http", h, "/healthz")); code != http.StatusPermanentRedirect {
			t.Errorf("/healthz = %d, want 308 once it is no longer the probe", code)
		}
	})
}

func TestRedirectUnknownPathStill308(t *testing.T) {
	h, _ := startRedirectRouter(t)
	if code, _ := doGet(t, noRedirectClient(), testURL("http", h, "/nope")); code != http.StatusPermanentRedirect {
		t.Errorf("status = %d, want 308", code)
	}
}

func TestHTTPSListenerNotRedirected(t *testing.T) {
	certFile, keyFile, pool := writeSelfSignedCert(t)
	ports := freePorts(t, 2)
	startRouter(t, helloRoutes, WithPorts(ports[0], ports[1]), WithTLS(certFile, keyFile), WithHTTPSRedirect())

	if code, body := doGet(t, tlsClient(pool), testURL("https", ports[1], "/hello")); code != 200 || body != "tls" {
		t.Errorf("got %d %q, want 200 tls", code, body)
	}
}

func TestRedirectDisabledByDefault(t *testing.T) {
	certFile, keyFile, _ := writeSelfSignedCert(t)
	ports := freePorts(t, 2)
	startRouter(t, helloRoutes, WithPorts(ports[0], ports[1]), WithTLS(certFile, keyFile))

	if code, body := doGet(t, noRedirectClient(), testURL("http", ports[0], "/hello")); code != 200 || body != "plain" {
		t.Errorf("got %d %q, want 200 plain", code, body)
	}
}

func redirectLocation(r *Router, host, uri string, headers map[string]string) (int, string) {
	req := httptest.NewRequest("GET", uri, nil)
	req.Host = host
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.redirectHandler().ServeHTTP(w, req)
	return w.Code, w.Header().Get("Location")
}

func TestRedirectTargets(t *testing.T) {
	tests := []struct {
		name      string
		httpsPort int
		host      string
		uri       string
		want      string
	}{
		{"default https port omitted", 443, "example.com:8080", "/p?q=1", "https://example.com/p?q=1"},
		{"host without port", 10443, "example.com", "/p", "https://example.com:10443/p"},
		{"ipv6 with port", 10443, "[::1]:8080", "/x", "https://[::1]:10443/x"},
		{"ipv6 without port", 10443, "[::1]", "/x", "https://[::1]:10443/x"},
		{"ipv6 default https port", 443, "[::1]:8080", "/x", "https://[::1]/x"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := &Router{httpsPort: tc.httpsPort, router: http.NewServeMux(), healthPath: "/healthz", readyPath: "/readyz"}
			code, got := redirectLocation(r, tc.host, tc.uri, nil)
			if code != http.StatusPermanentRedirect || got != tc.want {
				t.Errorf("got %d %q, want 308 %q", code, got, tc.want)
			}
		})
	}
}

func TestRedirectIgnoresForwardedHeaders(t *testing.T) {
	r := &Router{httpsPort: 10443, router: http.NewServeMux()}
	_, got := redirectLocation(r, "example.com", "/p", map[string]string{
		"X-Forwarded-Host": "evil.example",
		"X-Forwarded-Port": "1",
		"Forwarded":        "host=evil.example",
	})
	if want := "https://example.com:10443/p"; got != want {
		t.Errorf("Location = %q, want %q", got, want)
	}
}

func TestRedirectMissingHost(t *testing.T) {
	r := &Router{httpsPort: 10443, router: http.NewServeMux()}
	if code, loc := redirectLocation(r, "", "/p", nil); code != http.StatusBadRequest || loc != "" {
		t.Errorf("got %d %q, want 400 and no Location", code, loc)
	}
}
