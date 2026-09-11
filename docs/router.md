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

## Shutdown

The router is implemented with a graceful shutdown method, allowing all running handlers to complete (within 2 minutes) before the server is terminated. New connections are not accepted during this phase.

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
