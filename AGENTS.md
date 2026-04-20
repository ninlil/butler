# Butler – AI Agent Instructions

Butler is a Go framework for building cloud/Kubernetes-friendly microservices. It wraps `go-chi/chi` with automatic parameter binding, structured logging, graceful shutdown, and background workers.

## Build & Lint

```sh
go build ./...
go test ./...
make check   # golangci-lint + gocyclo (threshold: 15)
```

No code generation step is required. Go version: `1.24`.

## Packages

| Package | Role |
|---|---|
| `router/` | HTTP routing, parameter binding, response serialization |
| `workers/` | Background goroutine manager |
| `log/` | Zerolog wrapper with request-context propagation |
| `runtime/` | Shutdown hook registry |
| `bufferedresponse/` | Buffered `ResponseWriter` wrapper |
| root | `butler.Run()` / `butler.Cleanup()` lifecycle entry points |

See [docs/router.md](docs/router.md) and [docs/workers.md](docs/workers.md) for detailed docs.

## Typical `main.go` Pattern

```go
func main() {
    defer butler.Cleanup(nil)

    router.Serve(routes, router.WithPort(10000))
    workers.New("loader", loadData)
    butler.Run()
}
```

- `defer butler.Cleanup(nil)` must be the first statement.
- `butler.Run()` blocks until SIGINT/SIGTERM, then triggers graceful shutdown.

## Defining Routes

```go
var routes = []router.Route{
    {Name: "list",  Method: "GET",  Path: "/items",      Handler: listItems},
    {Name: "get",   Method: "GET",  Path: "/items/{id}", Handler: getItem},
    {Name: "add",   Method: "POST", Path: "/items",      Handler: addItem},
}
```

Path parameters use `{param}` (chi v5 syntax).

## Handler Signatures

Return values and arguments may appear in any order. The framework uses reflection to wire them.

**Accepted arguments** (zero or more, any order):
- `context.Context`
- `http.ResponseWriter`
- `*http.Request`
- Custom `struct` or `*struct` (auto-parsed from request)

**Accepted return values** (zero to three, any order):
- `int` – HTTP status code (0 = automatic: 200 or 204)
- `error` – serialized as `{"error":"…"}` with status 400
- `interface{}` / any concrete type – serialized response body

```go
func getItem(ctx context.Context, args *itemArgs) (*Item, int, error) { … }
func listItems() []Item { … }
func healthCheck() string { return "ok" }
```

## Parameter Binding Tags

Define a separate args struct per handler:

```go
type itemArgs struct {
    ID   int    `json:"id"   from:"path"  min:"1"`
    Name string `json:"name" from:"query" required:""`
    Body *Item  `from:"body"`
}
```

| Tag | Values | Notes |
|---|---|---|
| `from` | `path`, `query`, `header`, `body`, `cookie` | Required for non-path params |
| `json` | param name | Maps request field to struct field |
| `required` | (empty string) | Parameter must be present |
| `min` / `max` | number, duration, string length | Validated before handler runs |
| `default` | any value | Used when parameter is absent |
| `regex` | pattern | Validated before type conversion |

**Supported field types:** `int*`, `float*`, `string`, `bool`, `time.Time`, `time.Duration`, `[]byte`, `[]string`, `map[string]interface{}`, `struct` / `*struct` (body only).

## Logging

```go
func myHandler(ctx context.Context) {
    log := log.FromCtx(ctx)   // logger has req-id and corr-id pre-attached
    log.Info().Str("key", "val").Msg("done")
}
```

Never create a new `zerolog.Logger` directly in handlers — always use `log.FromCtx(ctx)`.

## Workers

```go
workers.New("name", func(ctx context.Context) {
    log := log.FromCtx(ctx)  // logger has "worker=name" field
    // … do work …
})
```

Workers start when `butler.Run()` is called. They may spawn child workers with `workers.New(…)` during execution.

Control post-completion behaviour before calling `butler.Run()`:

```go
workers.OnDone = workers.ExitOnDone      // (default) exit when all workers finish
workers.OnDone = workers.ContinueOnDone  // keep serving HTTP after workers finish
workers.OnDone = workers.ReadyOnDone     // keep serving, but readiness probe fails until done
```

## Router Options

```go
router.Serve(routes,
    router.WithPort(10000),
    router.WithPrefix("/api"),
    router.WithName("main"),
    router.WithExposedErrors(),   // include panic message in response body
    router.Without204(),          // return 200 instead of 204 for nil results
    router.WithHealth("/healthz"),
    router.WithReady("/readyz"),
)
```

## Conventions

- One args struct per handler, defined next to the handler in the same file.
- Validate inputs with struct tags; avoid manual validation for things the framework already handles.
- Return `(nil, http.StatusNotFound, err)` for missing resources; never `panic` for expected error cases.
- Examples live in `examples/`; each subdirectory is a self-contained runnable program.
