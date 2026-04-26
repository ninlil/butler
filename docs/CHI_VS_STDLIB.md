# Chi vs. Go stdlib ServeMux — Migration Research

This document analyzes the feasibility of replacing `go-chi/chi/v5` with the enhanced
`net/http.ServeMux` introduced in Go 1.22.

## Current State

- **Go version**: 1.24 (toolchain 1.24.4)
- **Chi version**: `github.com/go-chi/chi/v5 v5.2.2`
- **Chi usage scope**: limited to two files in `router/`

## Chi API Usage Inventory

| Chi API                  | File                | Active             | Go 1.22+ Equivalent                              |
|--------------------------|---------------------|--------------------|--------------------------------------------------|
| `chi.NewRouter()`        | `router/router.go`  | Yes                | `http.NewServeMux()`                             |
| `chi.RouteContext()`     | `router/reflect.go` | Yes                | `r.PathValue("name")` — breaking change          |
| `mux.Handle(path, h)`    | `router/router.go`  | Yes                | `mux.Handle(pattern, h)` — identical             |
| `mux.Method(m, path, h)` | `router/router.go`  | Yes                | `mux.HandleFunc("M /path", h)` — syntax change   |
| `mux.Get(path, h)`       | `router/router.go`  | Yes                | `mux.HandleFunc("GET /path", h)` — syntax change |
| `mux.Use(...)`           | `router/router.go`  | No (commented out) | N/A                                              |

## Path Parameter Extraction

Chi provides a route context that collects all path parameters in a single pass:

```go
// Current (chi)
params := chi.RouteContext(r.Context()).URLParams
param.vars = make(map[string]string, len(params.Keys))
for i, key := range params.Keys {
    param.vars[key] = params.Values[i]
}
value, found = param.vars[tags.Name]
```

The stdlib replacement fetches one parameter at a time:

```go
// Go 1.22+ net/http
value = r.PathValue(tags.Name)
found = value != ""
```

Since butler caches the extracted map in `param.vars` for the lifetime of a request,
the replacement is clean: drop the chi call and instead call `r.PathValue(tags.Name)`
directly at look-up time, which is equally lazy. The `param.vars` caching layer can
be removed entirely.

## Breaking Changes

### 1. Wildcard path syntax

Chi uses `/*` for catch-all segments; Go 1.22+ ServeMux uses `/{name...}`.
Butler **does** register wildcard routes — two in the `files` example:

```go
{Name: "api-fallback", Method: "*", Path: "/api/*",  Handler: notFound}
{Name: "files",        Method: "*", Path: "/*",       Handler: filesHandler}
```

All occurrences of `/*` in route paths must become `/{urlsuffix...}` before registration.

### 2. Method + path registration syntax

Chi exposes `mux.Method(method, path, handler)` and `mux.Handle(path, handler)` as
separate calls. Go 1.22+ ServeMux encodes the method in the pattern string:

```go
// chi
r.router.Method("POST", "/items", handler)   // specific method
r.router.Handle("/files/{urlsuffix...}", handler) // any method

// stdlib
mux.Handle("POST /items", handler)           // specific method
mux.Handle("/files/{urlsuffix...}", handler)      // any method (no prefix)
```

### Conversion Functions

Both breaking changes can be handled by two small internal helpers at registration time.
No changes to the public `Route` struct or any caller code are needed.

```go
// convertPath translates a chi-style path to a Go 1.22+ ServeMux pattern.
// Chi catch-all /* becomes /{urlsuffix...}; named params {name} are unchanged.
func convertPath(path string) string {
    if strings.HasSuffix(path, "/*") {
        return path[:len(path)-2] + "/{urlsuffix...}"
    }
    return path
}

// buildPattern builds a ServeMux registration pattern from a method and path.
// A method of "*" or "" means any method (no prefix in the pattern).
func buildPattern(method, path string) string {
    p := convertPath(path)
    if method == "*" || method == "" {
        return p
    }
    return method + " " + p
}
```

The registration loop in `router.go` then collapses to a single call:

```go
mux.Handle(buildPattern(method, route.Path), handler)
```

## Path Parameter Syntax

Named segments use the same `{name}` syntax in both chi and Go 1.22+ ServeMux.
The only syntax change required is the catch-all case handled by `convertPath` above.

## Middleware

Butler already does **not** use chi's `Use()` middleware chain. All middleware is
applied per-route via `justinas/alice`, which wraps standard `http.Handler` values and
is completely independent of the router. No changes needed here.

## Behavioral Differences to Consider

| Behavior                      | Chi                    | Go 1.22+ ServeMux                                       |
|-------------------------------|------------------------|---------------------------------------------------------|
| Unknown method on known path  | 405 Method Not Allowed | 404 Not Found (unless method registered with `OPTIONS`) |
| Automatic HEAD for GET routes | Yes                    | Yes (Go 1.22+)                                          |
| Trailing-slash redirects      | Configurable           | Built-in (301)                                          |
| Subtree patterns (`/prefix/`) | Via subrouters         | Native with `/prefix/` pattern                          |

The 405 vs 404 difference is the most user-visible behavioral change. Butler currently
inherits chi's 405 responses; switching to stdlib changes that to 404 for unregistered
methods on a known path. This may or may not matter depending on consumers.

## Dependency Impact

Removing chi eliminates the only non-logging external dependency in the core router:

- Remove: `github.com/go-chi/chi/v5`
- Keep: `github.com/justinas/alice` (middleware chaining)
- Keep: `github.com/rs/xid`, `github.com/rs/zerolog` (logging)

## Migration Scope

All changes are confined to two files:

- `router/router.go` — swap `chi.NewRouter()` and route registration calls
- `router/reflect.go` — replace `chi.RouteContext()` with `r.PathValue()`

No changes are needed in `workers/`, `log/`, `runtime/`, `bufferedresponse/`, or any
user-facing API surface. The public `Route` struct and `Serve()` function signatures
are unaffected.

## Tests to Add Before Migrating

The tests below should be added to `router/` before starting the migration.
They act as a behavioral baseline: all tests must pass before the migration begins,
and must continue to pass unchanged after chi is removed (except where a known
behavioral difference is explicitly noted).

### Path-parameter binding

Add to `router/handler_test.go`.

| Test name | Route path | Request | What it verifies |
|---|---|---|---|
| `TestPathParamSingleString` | `/users/{name}` | `GET /users/alice` | String named param is parsed and bound |
| `TestPathParamMultiple` | `/teams/{team}/users/{user}` | `GET /teams/ops/users/alice` | Two named params in one route are both bound correctly |

`TestHandlerPathParam` already covers the single-integer case; only the new cases above need to be added.

### Wildcard route matching

Add to `router/handler_test.go`.

| Test name | Route | Request | What it verifies |
|---|---|---|---|
| `TestWildcardRoot` | `Method: "*", Path: "/*"` | `GET /any/deep/path` | Request is dispatched to the handler |
| `TestWildcardPrefix` | `Method: "*", Path: "/api/*"` | `GET /api/v2/foo` | Prefix wildcard dispatches correctly |
| `TestWildcardPathValue` | `Method: "*", Path: "/*"` | `GET /files/a/b.txt` | The catch-all segment value is accessible in the handler (via `chi.RouteContext` now; via `r.PathValue("urlsuffix")` after migration) |

### Method routing

Add to `router/handler_test.go`.

| Test name | Registered method | Request method | Expected status | Notes |
|---|---|---|---|---|
| `TestMethodMatch` | `GET` | `GET` | 200 | Sanity check: exact match |
| `TestMethodMismatch` | `GET` | `POST` | 405 | **Chi-specific**: stdlib returns 404 — this test is expected to fail after migration; update expected status to 404 or add a 405 middleware if the behavior must be preserved |
| `TestMethodWildcard` | `*` | `DELETE` | 200 | Wildcard method accepts any HTTP verb |
| `TestMethodWildcardGET` | `*` | `GET` | 200 | Wildcard method accepts GET |

### Route prefix combined with path parameters

Add to `router/handler_test.go`.

| Test name | `WithPrefix` | Route path | Request | What it verifies |
|---|---|---|---|---|
| `TestPrefixWithPathParam` | `/api` | `/items/{id}` | `GET /api/items/7` | Path param resolves correctly when a global prefix is applied |
| `TestPrefixWithWildcard` | `/api` | `/*` | `GET /api/v2/anything` | Wildcard route resolves correctly under a prefix |

### Conversion helpers

Add to a new file `router/convert_test.go`. These are unit tests for the two helper
functions proposed in [Conversion Functions](#conversion-functions) — write the tests
first, then implement the functions.

| Test name | Input | Expected output | Notes |
|---|---|---|---|
| `TestConvertPath_NamedParam` | `"/items/{id}"` | `"/items/{id}"` | Named params are unchanged |
| `TestConvertPath_RootWildcard` | `"/*"` | `"/{urlsuffix...}"` | Root catch-all is rewritten |
| `TestConvertPath_PrefixWildcard` | `"/api/*"` | `"/api/{urlsuffix...}"` | Prefixed catch-all is rewritten |
| `TestConvertPath_Static` | `"/health"` | `"/health"` | Static paths are unchanged |
| `TestBuildPattern_SpecificMethod` | `"GET"`, `"/items"` | `"GET /items"` | Method is prepended |
| `TestBuildPattern_WildcardMethod` | `"*"`, `"/files/*"` | `"/files/{urlsuffix...}"` | No method prefix for wildcard |
| `TestBuildPattern_EmptyMethod` | `""`, `"/health"` | `"/health"` | Empty method treated as wildcard |

## Verdict

Migration is **low risk and low effort**. The two active chi call sites map cleanly to
stdlib equivalents. The path-parameter extraction in `reflect.go` becomes simpler, not
more complex. Removing chi eliminates an external dependency with no loss of
functionality for butler's use cases.
