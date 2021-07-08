package workers

import (
	"context"
	"time"
)

type workerState int

const (
	statePending workerState = iota
	stateRunning
	stateDone
	statePanic
)

// WorkerFunc is how a handler should look like
type WorkerFunc func(ctx context.Context)

// Worker contains data regarding a specific worker
type Worker struct {
	state     workerState
	name      string
	handler   WorkerFunc
	ctx       context.Context
	started   time.Time
	ended     time.Time
	realPanic bool
}

// WorkerOption is a handler to supply options to a worker
type WorkerOption func(*Worker)

// WithRealPanic makes a real 'panic' instead of a handled one is a worker actually does panic
func WithRealPanic() WorkerOption {
	return func(w *Worker) {
		w.realPanic = true
	}
}

// WithKV sets a key-value on the context, use context.Get to get it from within your WorkerFunc
func WithKV(key, val interface{}) WorkerOption {
	return func(w *Worker) {
		w.ctx = context.WithValue(w.ctx, key, val)
	}
}
