package router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/justinas/alice"
	"github.com/ninlil/butler/log"
)

// buildTestHandler sets up a Router's chi mux with routes and returns the http.Handler.
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
		chain := alice.New().Append(wrapWriterMW)
		chain = chain.Append(log.NewHandler())
		chain = chain.Append(IDHandler())
		chain = chain.Append(accessLogger)
		handler := chain.Append(r.panicHandler).ThenFunc(route.wrapHandler())
		if method == "*" {
			r.router.Handle(route.Path, handler)
		} else {
			r.router.Method(method, route.Path, handler)
		}
	}
	return r.router
}

// --- test handlers and args structs ---

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
