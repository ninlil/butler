# butler/router

Routes are defined as follows:

```go
var routes = []router.Route{
  {Name: "hello", Method: "GET", Path: "/", Handler: helloWorld},
}
```

## Route fields

| Field     | Type          | Description                                              |
|-----------|---------------|----------------------------------------------------------|
| `Name`    | `string`      | Name of route, used for logging                          |
| `Method`  | `string`      | HTTP method (GET, POST etc.), `"*"` for any method       |
| `Path`    | `string`      | The path/URL, syntax according to github.com/go-chi/chi  |
| `Handler` | `interface{}` | Handler function                                         |

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
- `struct` or `*struct` (currently only for `from:"body"`)

### Tags

| Tag        | Description                                        | Options                                               |
|------------|----------------------------------------------------|-------------------------------------------------------|
| `from`     | Source of parameter                                | `"path"`, `"query"`, `"header"`, `"body"`, `"cookie"` |
| `json`     | Name of parameter                                  | Required for all but `from:"body"`                    |
| `default`  | Default value if not specified in request          |                                                       |
| `required` | If present, the parameter must be in the request   |                                                       |
| `min`/`max`| Numerical min/max limit, or string length          |                                                       |
| `regex`    | Regexp matching of value before type conversion    |                                                       |

### Reading the body

The datatype for your body can either be a `struct` type, which will parse the input to your struct.
You can also get the raw data in the following formats:

- `[]byte` — raw data
- `string` — data as a Go string
- `[]string` — a scanner parses multiline text into an array of strings

## Shutdown

The router is implemented with a graceful shutdown method, allowing all running handlers to complete (within 2 minutes) before the server is terminated. New connections are not accepted during this phase.

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