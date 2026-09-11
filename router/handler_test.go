package router

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ninlil/butler/bufferedresponse"
	zlog "github.com/rs/zerolog/log"
)

// buildTestHandler sets up a Router's ServeMux with routes and returns the http.Handler.
// It mirrors the route-setup portion of Serve() without binding a real TCP port.
func buildTestHandler(t *testing.T, routes []Route) http.Handler {
	t.Helper()
	allOpts := []Option{WithPort(9999)}
	r, err := New(routes, allOpts...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	for _, route := range r.routes {
		method := "GET"
		if route.Method != "" {
			method = route.Method
		}
		if route.Path == "/" && r.prefix != "" {
			route.Path = r.prefix
		} else {
			route.Path = r.prefix + route.Path
		}
		handler := r.buildChain(route)
		r.router.Handle(buildPattern(method, route.Path), handler)
	}
	return r.router
}

// buildTestHandlerWithOpts is like buildTestHandler but accepts additional options
// (e.g. WithPrefix). Used for the migration baseline tests.
func buildTestHandlerWithOpts(t *testing.T, routes []Route, extra ...Option) http.Handler {
	t.Helper()
	allOpts := append([]Option{WithPort(9999)}, extra...)
	r, err := New(routes, allOpts...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	for _, route := range r.routes {
		method := "GET"
		if route.Method != "" {
			method = route.Method
		}
		if route.Path == "/" && r.prefix != "" {
			route.Path = r.prefix
		} else {
			route.Path = r.prefix + route.Path
		}
		handler := r.buildChain(route)
		r.router.Handle(buildPattern(method, route.Path), handler)
	}
	return r.router
}

// --- test handlers and args structs ---

// nameArgs / handlerName — single string path param for TestPathParamSingleString.
type nameArgs struct {
	Name string `json:"name" from:"path"`
}

func handlerName(args *nameArgs) string {
	return args.Name
}

// teamUserArgs / handlerTeamUser — two path params for TestPathParamMultiple.
type teamUserArgs struct {
	Team string `json:"team" from:"path"`
	User string `json:"user" from:"path"`
}

func handlerTeamUser(args *teamUserArgs) testItem {
	return testItem{ID: 1, Name: args.Team + "/" + args.User}
}

// wildcardSuffixArgs / handlerWildcardValue — reads the catch-all path segment.
// ServeMux matches "/*" patterns as "/{urlsuffix...}"; the value is bound via json:"urlsuffix".
type wildcardSuffixArgs struct {
	Suffix string `json:"urlsuffix" from:"path"`
}

func handlerWildcardValue(args *wildcardSuffixArgs) string {
	return args.Suffix
}

func handlerNoArgsNoReturn() {}

func handlerReturnStatus() int { return 201 }

type testItem struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func handlerReturnStruct() testItem {
	return testItem{ID: 42, Name: "answer"}
}

func handlerReturnError() error {
	return fmt.Errorf("something went wrong")
}

type pathIDArgs struct {
	ID int `json:"id" from:"path"`
}

func handlerPathParam(args *pathIDArgs) testItem {
	return testItem{ID: args.ID, Name: "found"}
}

type requiredQueryArgs struct {
	Name string `json:"name" from:"query" required:""`
}

func handlerRequiredQuery(args *requiredQueryArgs) string {
	return args.Name
}

type bodyArgs struct {
	Body *testItem `from:"body"`
}

func handlerBody(args *bodyArgs) testItem {
	if args.Body == nil {
		return testItem{}
	}
	return *args.Body
}

type headerArgs struct {
	Token string `json:"X-Token" from:"header"` //nolint:tagliatelle
}

func handlerHeader(args *headerArgs) string {
	return args.Token
}

type regexArgs struct {
	Code string `json:"code" from:"path" regex:"^[0-9]+$"`
}

func handlerRegex(args *regexArgs) string {
	return args.Code
}

type bodyMissingPtrArgs struct {
	Body *testItem `from:"body"`
}

// Returns 204 when Body is nil, 200 when non-nil.
func handlerBodyMissingPtr(args *bodyMissingPtrArgs) int {
	if args.Body == nil {
		return http.StatusNoContent
	}
	return http.StatusOK
}

type bodyMissingStructArgs struct {
	Body testItem `from:"body"`
}

func handlerBodyMissingStruct(args *bodyMissingStructArgs) testItem {
	return args.Body
}

// --- tests ---

func TestHandlerNoArgsNoReturn(t *testing.T) {
	h := buildTestHandler(t, []Route{
		{Name: "noop", Method: "GET", Path: "/noop", Handler: handlerNoArgsNoReturn},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/noop", nil)
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestHandlerReturnStatus(t *testing.T) {
	h := buildTestHandler(t, []Route{
		{Name: "status", Method: "GET", Path: "/status", Handler: handlerReturnStatus},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/status", nil)
	h.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Errorf("status = %d, want 201", w.Code)
	}
	if body := strings.TrimSpace(w.Body.String()); body != "" {
		t.Errorf("body = %q, want empty", body)
	}
}

func TestHandlerReturnStruct(t *testing.T) {
	h := buildTestHandler(t, []Route{
		{Name: "item", Method: "GET", Path: "/item", Handler: handlerReturnStruct},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/item", nil)
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var got testItem
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("body is not valid JSON: %v", err)
	}
	if got.ID != 42 || got.Name != "answer" {
		t.Errorf("got %+v, want {ID:42, Name:answer}", got)
	}
}

func TestHandlerReturnError(t *testing.T) {
	h := buildTestHandler(t, []Route{
		{Name: "err", Method: "GET", Path: "/err", Handler: handlerReturnError},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/err", nil)
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not valid JSON: %v", err)
	}
	if body.Error == "" {
		t.Error("expected non-empty error message in body")
	}
}

func TestHandlerPathParam(t *testing.T) {
	h := buildTestHandler(t, []Route{
		{Name: "get", Method: "GET", Path: "/items/{id}", Handler: handlerPathParam},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/items/7", nil)
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var got testItem
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("body is not valid JSON: %v", err)
	}
	if got.ID != 7 {
		t.Errorf("ID = %d, want 7", got.ID)
	}
}

func TestHandlerRequiredQueryMissing(t *testing.T) {
	h := buildTestHandler(t, []Route{
		{Name: "q", Method: "GET", Path: "/q", Handler: handlerRequiredQuery},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/q", nil) // no ?name=...
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d (missing required query param)", w.Code, http.StatusBadRequest)
	}
}

func TestHandlerBodyBinding(t *testing.T) {
	h := buildTestHandler(t, []Route{
		{Name: "body", Method: "POST", Path: "/body", Handler: handlerBody},
	})
	payload := `{"id":3,"name":"three"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/body", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var got testItem
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("body is not valid JSON: %v", err)
	}
	if got.ID != 3 || got.Name != "three" {
		t.Errorf("got %+v, want {ID:3, Name:three}", got)
	}
}

func TestHandlerHeaderBinding(t *testing.T) {
	h := buildTestHandler(t, []Route{
		{Name: "hdr", Method: "GET", Path: "/hdr", Handler: handlerHeader},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/hdr", nil)
	req.Header.Set("X-Token", "secret")
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := strings.Trim(w.Body.String(), `"`+"\n")
	if body != "secret" {
		t.Errorf("body = %q, want %q", body, "secret")
	}
}

func TestHandlerRegexTag(t *testing.T) {
	h := buildTestHandler(t, []Route{
		{Name: "re", Method: "GET", Path: "/check/{code}", Handler: handlerRegex},
	})

	t.Run("valid value passes", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/check/12345", nil)
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
	})

	t.Run("invalid value returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/check/abc", nil)
		h.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})
}

func TestHandlerBodyMissingPointer(t *testing.T) {
	h := buildTestHandler(t, []Route{
		{Name: "bmp", Method: "POST", Path: "/bmp", Handler: handlerBodyMissingPtr},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/bmp", nil) // no body
	h.ServeHTTP(w, req)
	// expect 204: the handler signals the pointer was nil
	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d (pointer should be nil when body is missing)", w.Code, http.StatusNoContent)
	}
}

func TestHandlerBodyMissingStruct(t *testing.T) {
	h := buildTestHandler(t, []Route{
		{Name: "bms", Method: "POST", Path: "/bms", Handler: handlerBodyMissingStruct},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/bms", nil) // no body
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var got testItem
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("body is not valid JSON: %v", err)
	}
	if got.ID != 0 || got.Name != "" {
		t.Errorf("got %+v, want zero-value testItem{ID:0, Name:\"\"}", got)
	}
}

// =============================================================================
// Router behaviour tests — verify ServeMux routing, path params, wildcards, and
// method matching behave as expected.
// =============================================================================

// --- path-parameter binding ---

func TestPathParamSingleString(t *testing.T) {
	h := buildTestHandler(t, []Route{
		{Name: "user", Method: "GET", Path: "/users/{name}", Handler: handlerName},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/users/alice", nil)
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	name := strings.Trim(w.Body.String(), `"`+"\n")
	if name != "alice" {
		t.Errorf("name = %q, want %q", name, "alice")
	}
}

func TestPathParamMultiple(t *testing.T) {
	h := buildTestHandler(t, []Route{
		{Name: "team-user", Method: "GET", Path: "/teams/{team}/users/{user}", Handler: handlerTeamUser},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/teams/ops/users/alice", nil)
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var got testItem
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("body is not valid JSON: %v", err)
	}
	if got.Name != "ops/alice" {
		t.Errorf("name = %q, want %q", got.Name, "ops/alice")
	}
}

// --- wildcard route matching ---

func TestWildcardRoot(t *testing.T) {
	h := buildTestHandler(t, []Route{
		{Name: "catch-all", Method: "*", Path: "/*", Handler: handlerReturnStatus},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/any/deep/path", nil)
	h.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Errorf("status = %d, want 201 (wildcard route should match any path)", w.Code)
	}
}

func TestWildcardPrefix(t *testing.T) {
	h := buildTestHandler(t, []Route{
		{Name: "api-catch-all", Method: "*", Path: "/api/*", Handler: handlerReturnStatus},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v2/foo", nil)
	h.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Errorf("status = %d, want 201 (prefixed wildcard route should match)", w.Code)
	}
}

func TestWildcardPathValue(t *testing.T) {
	// handlerWildcardValue reads args.Suffix which is bound via json:"urlsuffix" from:"path".
	// ServeMux stores catch-all "/*" segments under the key "urlsuffix" (from "/{urlsuffix...}").
	h := buildTestHandler(t, []Route{
		{Name: "files", Method: "*", Path: "/*", Handler: handlerWildcardValue},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/files/a/b.txt", nil)
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	suffix := strings.Trim(w.Body.String(), `"`+"\n")
	if !strings.Contains(suffix, "files") || !strings.Contains(suffix, "b.txt") {
		t.Errorf("suffix = %q, want it to contain the request path segments", suffix)
	}
}

// --- method routing ---

func TestMethodMatch(t *testing.T) {
	h := buildTestHandler(t, []Route{
		{Name: "match", Method: "GET", Path: "/match", Handler: handlerReturnStatus},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/match", nil)
	h.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Errorf("status = %d, want 201", w.Code)
	}
}

func TestMethodMismatch(t *testing.T) {
	// net/http ServeMux (Go 1.22+) returns 405 Method Not Allowed when a path is registered
	// for a specific method and a request arrives with a different method. This matches chi's
	// behaviour, so no status change was needed during the chi-to-stdlib migration.
	h := buildTestHandler(t, []Route{
		{Name: "get-only", Method: "GET", Path: "/get-only", Handler: handlerNoArgsNoReturn},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/get-only", nil)
	h.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d",
			w.Code, http.StatusMethodNotAllowed)
	}
}

func TestMethodWildcard(t *testing.T) {
	h := buildTestHandler(t, []Route{
		{Name: "all-methods", Method: "*", Path: "/all-methods", Handler: handlerReturnStatus},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/all-methods", nil)
	h.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Errorf("status = %d, want 201 (wildcard method should accept DELETE)", w.Code)
	}
}

func TestMethodWildcardGET(t *testing.T) {
	h := buildTestHandler(t, []Route{
		{Name: "all-methods-get", Method: "*", Path: "/all-methods-get", Handler: handlerReturnStatus},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/all-methods-get", nil)
	h.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Errorf("status = %d, want 201 (wildcard method should accept GET)", w.Code)
	}
}

// --- route prefix combined with path parameters ---

func TestPrefixWithPathParam(t *testing.T) {
	h := buildTestHandlerWithOpts(t, []Route{
		{Name: "prefixed-item", Method: "GET", Path: "/items/{id}", Handler: handlerPathParam},
	}, WithPrefix("/api"))
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/items/7", nil)
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var got testItem
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("body is not valid JSON: %v", err)
	}
	if got.ID != 7 {
		t.Errorf("ID = %d, want 7", got.ID)
	}
}

func TestPrefixWithWildcard(t *testing.T) {
	h := buildTestHandlerWithOpts(t, []Route{
		{Name: "prefixed-catch-all", Method: "*", Path: "/*", Handler: handlerReturnStatus},
	}, WithPrefix("/api"))
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v2/anything", nil)
	h.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Errorf("status = %d, want 201 (wildcard under prefix should match)", w.Code)
	}
}

// =============================================================================
// Streaming route tests — verify Route.Streaming bypasses response buffering,
// uses the pre-handler streaming access-log line, and recovers from panics.
// =============================================================================

func TestStreamingRoute_BypassesBufferedResponse(t *testing.T) {
	var streamingRaw, normalWrapped bool

	streamingHandler := func(w http.ResponseWriter, r *http.Request) {
		_, ok := bufferedresponse.Get(w)
		streamingRaw = !ok
	}
	normalHandler := func(w http.ResponseWriter, r *http.Request) {
		_, ok := bufferedresponse.Get(w)
		normalWrapped = ok
	}

	h := buildTestHandler(t, []Route{
		{Name: "stream", Method: "GET", Path: "/stream", Handler: streamingHandler, Streaming: true},
		{Name: "normal", Method: "GET", Path: "/normal", Handler: normalHandler},
	})

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/stream", nil))
	if !streamingRaw {
		t.Error("expected streaming route handler to receive the raw writer (bufferedresponse.Get ok=false)")
	}

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/normal", nil))
	if !normalWrapped {
		t.Error("expected normal route handler to receive the wrapped writer (bufferedresponse.Get ok=true)")
	}
}

func TestStreamingRoute_FlushesIncrementally(t *testing.T) {
	const chunkDelay = 200 * time.Millisecond
	const perChunkTimeout = 450 * time.Millisecond // > chunkDelay, < total handler runtime (3*chunkDelay)

	handler := func(w http.ResponseWriter, r *http.Request) {
		fl, ok := w.(http.Flusher)
		if !ok {
			return
		}
		for i := range 3 {
			fmt.Fprintf(w, "chunk-%d\n", i)
			fl.Flush()
			time.Sleep(chunkDelay)
		}
	}

	h := buildTestHandler(t, []Route{
		{Name: "stream", Method: "GET", Path: "/stream", Handler: handler, Streaming: true},
	})

	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/stream")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	for i := range 3 {
		type readResult struct {
			line string
			err  error
		}
		ch := make(chan readResult, 1)
		go func() {
			line, err := reader.ReadString('\n')
			ch <- readResult{line, err}
		}()

		select {
		case res := <-ch:
			if res.err != nil {
				t.Fatalf("chunk %d: read error: %v", i, res.err)
			}
			want := fmt.Sprintf("chunk-%d\n", i)
			if res.line != want {
				t.Errorf("chunk %d = %q, want %q", i, res.line, want)
			}
		case <-time.After(perChunkTimeout):
			t.Fatalf("chunk %d: timed out waiting for incremental flush", i)
		}
	}
}

func TestStreamingRoute_AccessLogSingleLineBeforeHandler(t *testing.T) {
	orgLogger := zlog.Logger
	buf := new(bytes.Buffer)
	zlog.Logger = zlog.Output(buf)
	defer func() { zlog.Logger = orgLogger }()

	// the middleware chain runs synchronously, so by the time the handler executes,
	// the streaming access-log line has already been written to buf.
	var loggedBeforeHandler bool
	handler := func(w http.ResponseWriter, r *http.Request) {
		loggedBeforeHandler = strings.Contains(buf.String(), `"streaming":true`)
	}

	h := buildTestHandler(t, []Route{
		{Name: "stream", Method: "GET", Path: "/stream", Handler: handler, Streaming: true},
	})
	buf.Reset() // discard setup-time trace logs so only the request's log line remains

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/stream", nil))

	if !loggedBeforeHandler {
		t.Error("expected streaming access-log line to be written before the handler runs")
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected exactly 1 log line, got %d: %q", len(lines), buf.String())
	}
}

func TestNonStreamingRoute_AccessLogUnchanged(t *testing.T) {
	orgLogger := zlog.Logger
	buf := new(bytes.Buffer)
	zlog.Logger = zlog.Output(buf)
	defer func() { zlog.Logger = orgLogger }()

	h := buildTestHandler(t, []Route{
		{Name: "normal", Method: "GET", Path: "/normal", Handler: handlerReturnStatus},
	})
	buf.Reset() // discard setup-time trace logs so only the request's log line remains

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/normal", nil))

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected exactly 1 log line, got %d: %q", len(lines), buf.String())
	}
	line := lines[0]
	for _, field := range []string{`"status":`, `"size":`, `"duration":`} {
		if !strings.Contains(line, field) {
			t.Errorf("expected log line to contain %s, got %q", field, line)
		}
	}
	if strings.Contains(line, `"streaming"`) {
		t.Errorf("expected no streaming field in non-streaming access log, got %q", line)
	}
}

func TestStreamingRoute_PanicRecovery(t *testing.T) {
	t.Run("panics before writing any output", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			panic("boom")
		}
		h := buildTestHandler(t, []Route{
			{Name: "stream-panic-early", Method: "GET", Path: "/stream-panic-early", Handler: handler, Streaming: true},
		})

		w := httptest.NewRecorder()
		// the panic must be recovered inside panicHandler; nothing should escape ServeHTTP.
		h.ServeHTTP(w, httptest.NewRequest("GET", "/stream-panic-early", nil))

		// no output was sent yet, so panicHandler's WriteHeader(500) actually takes effect.
		if w.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
		}
	})

	t.Run("panics after writing partial output", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("partial"))
			panic("boom")
		}
		h := buildTestHandler(t, []Route{
			{Name: "stream-panic-partial", Method: "GET", Path: "/stream-panic-partial", Handler: handler, Streaming: true},
		})

		w := httptest.NewRecorder()
		// the panic must be recovered inside panicHandler; nothing should escape ServeHTTP.
		h.ServeHTTP(w, httptest.NewRequest("GET", "/stream-panic-partial", nil))

		if w.Body.String() != "partial" {
			t.Errorf("body = %q, want %q (partial output already flushed before the panic)", w.Body.String(), "partial")
		}
		// status was already committed by the first Write before the panic, so it stays 200;
		// panicHandler's later WriteHeader(500) is a graceful no-op, matching real net/http semantics.
		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
	})
}
