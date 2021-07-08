# butler/workers

This module is when you need to do processing before/without getting a HTTP-call.

Example usage:
- CronJob
- Initial loading of resources / connecting to an external endpoint

Example: [workers1](../examples/workers1/main.go)

## Basic example

```go
func main() {
	defer butler.Cleanup(nil)

  // the router will serve liveness/readiness-probes while running
	err := router.Serve(routes, 10000)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	workers.New("alpha", alpha)

	butler.Run()
}

func alpha(ctx context.Context) {
	log := log.FromCtx(ctx)
  // the log will contain the 'worker=alpha' property

	log.Info().Msg("inside Alpha")  
	time.Sleep(20 * time.Second)
}
```

This will function as a Job/CronJob and exit when `alpha` is done (you can have multiple workers with different names)

### Why should I use `butler` to run my code?
By doing it this way the router will serve the Kubernetes liveness-probes to signal that your process is alive and well.

You can also easily run multiple workers, and can create new (conditional) workers later on.

*** Future updates will also enable metrics to be served while running.

## Example with initial loader

Just set the `workers.OnDone` to either `ContinueOnDone` or `ReadyOnDone` to change the behaviour to not exit when the worker is done.

`ReadyOnDone` will keep the readiness-probe to false until the worker is done.

```go
func main() {
	defer butler.Cleanup(nil)

	err := router.Serve(routes, 10000)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	workers.OnDone = workers.ReadyOnDone // <- changes behaviour

	workers.New("alpha", alpha)

	butler.Run()
}
```

## Worker options

You can create a worker with additional options:
```go
	workers.New("alpha", alpha, workers.WithXxx(), ...)
```

| Option          | Description |
|-----------------|-------------|
| WithRealPanic() | Will change the default (handle a panic) into an actual `panic()` |
| WithKV(k,v)     | Will assign the context.Context a new key/value before starting the worker |

## Running multiple workers

You can run multiple workers in parallel, and even start a new worker from within an existing one.

The code will wait until all started workers are done before exiting (or doing whatever behaviour is set on the `OnDone` property)