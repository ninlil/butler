# butler/router

Routes are defined as follows:

```go
var routes = []router.Route{
  {Name: "hello", Method: "GET", Path: "/", Handler: helloWorld},
}
```

## Route fields

| Field       | Type          | Description                                                                                       |
|-------------|---------------|-----------------------------------------------------------------------------------------------------|
| `Name`      | `string`      | Name of route, used for logging                                                                   |
| `Method`    | `string`      | HTTP method (GET, POST etc.), `"*"` for any method                                                |
| `Path`      | `string`      | The path/URL, using `net/http.ServeMux` pattern syntax                                            |
| `Handler`   | `interface{}` | Handler function                                                                                   |
| `Streaming` | `bool`        | Opt out of response buffering for long-lived streaming responses (SSE, chunked); default `false`  |

## Handlers

### Input arguments

A handler can accept the following types (in any order):

- `context.Context`
- `http.ResponseWriter`
- `*http.Request`
- Your own custom `*struct` for arguments (see below for details)

### Return values

A handler can return up to 3 different values:

| Type          | Result                                                                                       |
|---------------|----------------------------------------------------------------------------------------------|
| `int`         | HTTP status code                                                                             |
| `error`       | Returns `{"error":"RETURNED_ERROR_TEXT"}` serialized according to the `Accept` header        |
| `interface{}` | `nil` values are treated as "no data"; data is serialized according to the `Accept` header   |

Default status code when status is not used or returned as `0`:

| Status            | Situation                             |
|-------------------|---------------------------------------|
| 200 OK            | Successful call with data in body     |
| 204 No Content    | Successful call, but no data returned |
| 400 Bad Request   | Error returned, with message in body  |

## Getting input

Example handler:

```go
type handlerArgs struct {
  Fieldname datatype `tag:"value" ...`
}

func handler(args *handlerArgs) {
  ...
}
```

Requirement:

- Field names must be _public_ (i.e. uppercase initial character)

### Datatypes

- `int` (any bit size)
- `float` (32 and 64)
- `string`
- `bool`
- `[]byte`
- `time.Time`
- `time.Duration`
- `[]string` (only for `from:"body"`; splits the body into lines)
- `map[string]interface{}` (only for `from:"body"`)
- `struct` or `*struct` (currently only for `from:"body"`)

### Tags

| Tag         | Description                                      | Options                                                         |
|-------------|--------------------------------------------------|-----------------------------------------------------------------|
| `from`      | Source of parameter                              | `"path"`, `"query"`, `"header"`, `"body"`, `"cookie"`, `"form"` |
| `json`      | Name of parameter                                | Required for all but `from:"body"`                              |
| `default`   | Default value if not specified in request        |                                                                 |
| `required`  | If present, the parameter must be in the request |                                                                 |
| `min`/`max` | Numerical min/max limit, or string length        |                                                                 |
| `regex`     | Regexp matching of value before type conversion  |                                                                 |

### Reading the body

The datatype for your body can either be a `struct` type, which will parse the input to your struct.
You can also get the raw data in the following formats:

- `[]byte` — raw data
- `string` — data as a Go string
- `[]string` — a scanner parses multiline text into an array of strings

## Middleware

Standard `func(http.Handler) http.Handler` middleware functions can be added with
`WithMiddleware`. They run for every route, in the order they are registered, after
butler's built-in chain (writer-wrapping → logging → request-ID → access-log →
panic-recovery) and before the route handler.

```go
router.Serve(routes,
    router.WithPort(10000),
    router.WithMiddleware(
        timingMiddleware,
        apiKeyMiddleware,
    ),
)
```

`WithMiddleware` may be called multiple times; middlewares accumulate in order.

Because user middlewares sit inside the panic-recovery layer, a panic in your
middleware is caught and returns `500` rather than crashing the server.

The zerolog logger (and the request/correlation IDs) are already attached to the
`context.Context` by the time your middleware runs, so `log.FromCtx(ctx)` works
without any extra setup.

Any middleware that matches the `func(http.Handler) http.Handler` signature works —
including middleware from chi, gorilla/handlers, or the standard library.

See [examples/middleware](../examples/middleware) for a runnable example.

## Streaming responses (SSE, chunked)

By default, every response is buffered: the handler writes to a `bufferedresponse.ResponseWriter`,
and the real bytes are only sent to the client once the handler returns (or calls `Flush()`).
This buffering breaks Server-Sent Events (SSE) and other responses that need to send data
incrementally over a long-lived connection.

Set `Streaming: true` on a route to opt out of buffering entirely:

```go
var routes = []router.Route{
    {Name: "events", Method: "GET", Path: "/events", Handler: events, Streaming: true},
}
```

Streaming routes should use a raw handler (`func(w http.ResponseWriter, r *http.Request)`) —
any handler works, but since `w` is the real `http.ResponseWriter` (unwrapped), only a raw
handler lets you call `w.(http.Flusher).Flush()` after each write to push data immediately:

```go
func events(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    flusher := w.(http.Flusher)
    for i := 1; i <= 5; i++ {
        fmt.Fprintf(w, "data: message %d\n\n", i)
        flusher.Flush()
        time.Sleep(time.Second)
    }
}
```

Streaming routes also use a different access-log line: instead of the usual single line
logged *after* the request completes (with `status`/`size`/`duration`), a single line with
`streaming: true` is logged *before* the handler runs, since a streaming response has no
final status/size/duration to report.

See [examples/sse](../examples/sse) for a runnable example.

## HTTPS

TLS is enabled through options. The number of listeners is decided by `WithPort` vs `WithPorts`;
a TLS option only decides the protocol.

| Mode         | Options                                          | Listeners                      |
|--------------|--------------------------------------------------|--------------------------------|
| HTTP only    | `WithPort(p)`                                    | plain on `p`                   |
| HTTPS only   | `WithPort(p)` + `WithTLS` / `WithTLSConfig`      | TLS on `p`                     |
| HTTP + HTTPS | `WithPorts(h, s)` + `WithTLS` / `WithTLSConfig`  | plain on `h`, TLS on `s`       |

`WithPorts` without a TLS option makes `router.New` return `ErrorTLSNotConfigured`. Both ports must be
greater than 0 and differ (`ErrorInvalidPort`, `ErrorPortConflict`). `WithPort` and `WithPorts` write the
same settings, so the last one given wins.

```go
// HTTPS only on 8443
router.Serve(routes, router.WithPort(8443), router.WithTLS("tls.crt", "tls.key"))

// HTTP on 10000 and HTTPS on 10443
router.Serve(routes,
    router.WithPorts(10000, 10443), // httpPort, httpsPort
    router.WithTLS("tls.crt", "tls.key"),
)
```

> **Note:** `WithPort` combined with a TLS option means that port is HTTPS. Adding a TLS option to an
> existing `WithPort` service therefore turns its port into HTTPS and breaks Kubernetes probes that use
> `scheme: HTTP`. Use `WithPorts` when migrating, which keeps a plain port for the probes.

The health and ready probes are served on every listener.

### Custom TLS configuration

`WithTLSConfig` takes a full `*tls.Config` (mTLS, cipher policy, `autocert`, ...). It must contain
`Certificates`, `GetCertificate` or `GetConfigForClient`, otherwise `New` returns `ErrorInvalidTLS`.
`WithTLS` and `WithTLSConfig` replace each other; the last one given wins.

```go
cfg := &tls.Config{
    MinVersion: tls.VersionTLS13,
    ClientAuth: tls.RequireAndVerifyClientCert,
    ClientCAs:  pool,
    Certificates: []tls.Certificate{cert},
}
router.Serve(routes, router.WithPorts(10000, 10443), router.WithTLSConfig(cfg))
```

With `WithTLS` the router builds the config itself, with TLS 1.2 as the minimum version. The key pair is
loaded when the router is created, so a missing or invalid file is returned as an error from `router.Serve`
before anything listens. HSTS is not added automatically; add it with `WithMiddleware`.

### Certificate reload

With `WithTLS`, the certificate and key files are re-read when either file's modification time changes.
The check runs at most every 30 seconds, on a TLS handshake. If the new files cannot be read or do not
form a valid pair, an error is logged and the previous certificate stays in use.

### Redirecting HTTP to HTTPS

`WithHTTPSRedirect()` (requires `WithPorts`, otherwise `ErrorRedirectNeedsPorts`) makes the plain listener
answer every request with a `308 Permanent Redirect` to the HTTPS URL, built from the request's `Host`
and URI. The health and ready probe paths are not redirected.

```go
router.Serve(routes,
    router.WithPorts(10000, 10443),
    router.WithTLS("tls.crt", "tls.key"),
    router.WithHTTPSRedirect(),
)
```

### Access log

Access-log lines contain a `scheme` field, `"https"` for TLS requests and `"http"` otherwise.

See [examples/https](../examples/https) for a runnable example.

## Shutdown

The router is implemented with a graceful shutdown method, allowing all running handlers to complete (within 2 minutes) before the server is terminated. New connections are not accepted during this phase. All listeners (HTTP and HTTPS) are shut down together.

As soon as shutdown starts, the context passed to every in-flight request (and returned by
`r.Context()`) is canceled, so long-lived handlers — like the streaming example above — should
watch `r.Context().Done()` in their loop and return promptly instead of holding the connection
open until the 2-minute grace period elapses.

### Manual shutdown

To shut down a router manually, call the `router.Shutdown()` method.

> **Note:** The method is synchronous.
> To call this inside a handler, run it as a goroutine:
>
> ```go
> func shutdownHandler() {
>   go router.Shutdown()
> }
> ```
>
> Otherwise you will block yourself because of the graceful shutdown.

### Router names and runtime lifetime

Router names (`WithName`) must be unique for the lifetime of the process; a name is not released when its router stops, and `router.New` returns `ErrRouterDuplicateName` if it is reused. When the last running router stops, the runtime is closed and later routers are not covered by it. Butler assumes one router set per process; tests starting several routers should use a unique name each and keep one router alive.
