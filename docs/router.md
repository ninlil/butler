# butler/router

Routes are defined as follows:
```go
var routes = []routers.Route{
	{Name: "hello", Method: "GET", Path: "/", Handler: helloWorld},
}
```

## Route-fields
| Field   | Type        | Description                                              |
|---------|-------------|----------------------------------------------------------|
| Name    | string      | Name of route, used for logging                          |
| Method  | string      | HTTP-method (GET, POST a.s.o), "*" for any method        |
| Path    | string      | The path/url, syntax according to github.com/gorilla/mux |
| Handler | interface{} | Handler function                                         |

## Handlers

### Input arguments
A handler can accept the following types (in any order)
- context.Context
- http.ResponseWriter
- *http.Request
- your own custom *struct for arguments (see below for details)

### Return values
A handler can return up to 3 different values:
| Type        | Result           |
|-------------|------------------|
| int         | HTTP Status-code |
| error       | will return json {"error":"RETURNED_ERROR_TEXT"} (according to the `Accept`-header) |
| interface{} | `nil` values are treated as "no data", data will be serialized according to the `Accept`-header |

Default status-code when status is not used or returned as '0'
| Status          | Situation                            |
|-----------------|--------------------------------------|
| 200 Ok          | Succefull call with data in body     |
| 204 No content  | Succefull call, but no data returned |
| 400 Bad Request | Error returned, with message in body |

## Getting input
Example handler
```go
type handlerArgs struct {
  Fieldname datatype `tag:"value" ...`
}

func handler(args *handlerArgs) {
  ...
}
```

Requirement:
- Fieldnames must be _public_ (i.e uppercase initial character)

### Datatypes
- int (any bitsize)
- float (32 and 64)
- string
- bool
- []byte
- time.Time
- struct or *struct (currently only for `from:"body"`)


### Tags
| Tag      | Description          | Options |
|----------|----------------------|---------|
| from     | Source of parameter  | "path", "query", "header" or "body" |
| json     | Name of parameter    | Required for all but `from:"body"` |
| default  | Default value if not specified in request | |
| required | If exists, then the parameter must be specified in the request | |
| min/max  | Numerical min/max-limit, or string-length | |
| regex    | Regexp-matching of value before any type-conversion | |

### Reading the body
The datatype for your body can either be a `type struct` which will parse the input to your struct.
You can also just get the raw data in the following formats:
* `[]byte` - you will get the raw data
* `string` - you will get the data as a Go string
* `[]string` - A scanner will parse multilined text into an array of strings

## Shutdown

The routers is implemented with a 'graceful shutdown method' allowing all running handlers to complete (within 2 minutes) before the server is terminated (new connections are not accepted during this phase)

### Manual shutdown
If you want to shutdown a router manually then you call the `router.Shutdown()` method.

> The method is syncronuous.
> If you want to to this inside a handler you need to run it using a goroutine:
> ```go
> func shutdownHandler() {
>   go router.Shutdown()
> }
> ```
> Otherwise you will block yourself because of the 'graceful shutdown'