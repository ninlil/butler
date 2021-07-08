# butler
Go-framework for cloud/kubernetes-friendly microservices

## Why?
This package is meant to streamline the development of simple microservice without requiring each developer to handle the overhead of parsing http-headers, request- and response-body.

My definition of a successful microservice, specially in kubernetes is this:
- Handles the health-probes
- Separate response-status by 'Data found', 'Data not found/No-data' and 'Route/handler not found'
- Handle different encodings on both request- and response-body
- Consistent tracing/tracking on both request and application logging
- Automatic and consistent error-handling on parameters
- Graceful shutdown (close incoming connections, but keep processing those already received)

With all that in mind (and more) I'm building this framework: **the Butler** - _makes life easier_

## Features

### Router
- Gracefull shutdown of http-server
- Liveness and Readyness-probes for Kubernetes
- Parameter-validation
  - min/max and default-values
  - optional or required
  - from `path`, `query`, `header` or `body`
- Handle the `Accept` & `Content-Type` headers (json, xml)
- Enable handlers to use functional-programming
  - Return the actual result
  - Accept `context.Context` argument
- Wrapped handling of `Request-Id` and `Correlation-Id`
- Automatic log-support with json to pipe/stream and pretty-printed to console/tty
- Automatic `204 'No Content'` on empty result

### Workers
- Easy job/cronjob (run-then-exit) with health-probes
- Startup/initialization-phase

### ...planned for future updates
- Metrics for Prometheus
- More dataformats (yaml, toml)
- Regex-validation of parameters
- Support custom datatypes (ex: UUID)
- More documentation
- ETag-calculation
- Easily detect/handle closed/cancelled requests

## Examples
- [Misc. router-example](examples/example)
- [Worker-demo](examples/workers1)

### HelloWorld
```go
import (
	"github.com/ninlil/butler"
	"github.com/ninlil/butler/log"
	"github.com/ninlil/butler/router"
)

var routes = []router.Route{
	{Name: "hello", Method: "GET", Path: "/", Handler: helloWorld},
}

func main() {
	defer butler.Cleanup(nil)

	err := router.Serve(routes, 10000)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	butler.Run()
}

func helloWorld() string {
	return "Hello World!"
}
```

### API with arguments and return-values
```go
import (
	"context"

	"github.com/ninlil/butler/log"
)

type handlerArgs struct {
	A float64 `from:"query" json:"a" required:""`
	B float64 `from:"query" json:"b" required:""`
}

type handlerResult struct {
	Sum float64 `json:"sum"`
}

func handler(ctx context.Context, args *handlerArgs) *handlerResult {
	log := log.FromCtx(ctx)
	log.Info().Msgf("handler called with a=%v and b=%v", args.B, args.B)
	return &handlerResult{
		Sum: args.A + args.B,
	}
}
```

```
GET http://localhost/handler?a=3&b=0.14
---
HTTP/1.1 200 OK
Content-Length: 12
Content-Type: application/json; charset=utf-8
Correlation-Id: ...tas0
Request-Id: ...targ
Date: Mon, 05 Jul 2021 13:57:17 GMT
Connection: close

{
  "sum": 3.14
}
```
and the log also prints (to console/tty)
```
13:57:17.351 INF handler called with a=0.14 and b=0.14 corr_id=...tas0 req_id=...tas0
```

## External packages
- github.com/gorilla/mux - A powerful HTTP router and URL matcher for building Go web servers
-	github.com/justinas/alice - Painless middleware chaining for Go
-	github.com/rs/xid - xid is a globally unique id generator thought for the web
-	github.com/rs/zerolog - Zero Allocation JSON Logger

