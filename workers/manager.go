package workers

import (
	"context"

	"github.com/ninlil/butler/log"
	"github.com/rs/zerolog"
)

type ctxWorkerType struct{}

var ctxWorker ctxWorkerType

// DoneBehavior -
type DoneBehavior int

// OnDone determines what action should be taken once all workers are done
var OnDone DoneBehavior

// OnDone enums
const (
	ExitOnDone DoneBehavior = iota
	ContinueOnDone
	ReadyOnDone
)

// New creates a new workers with the specified options
func New(name string, h WorkerFunc, opts ...WorkerOption) {
	w := &Worker{
		state:   statePending,
		name:    name,
		handler: h,
	}

	w.ctx = context.WithValue(context.Background(), ctxWorker, w)
	w.ctx = log.WithContext(w.ctx)
	log := log.FromCtx(w.ctx)
	log.UpdateContext(func(c zerolog.Context) zerolog.Context {
		return c.Str("worker", name)
	})
	for _, opt := range opts {
		opt(w)
	}

	if w = driver.add(w); w == nil {
		log.Panic().Msg("Couldn't create worker, maybe name is already used")
	}
}

// Get the current Worker from a context
func Get(ctx context.Context) *Worker {
	v := ctx.Value(ctxWorker)
	if w, ok := v.(*Worker); ok {
		return w
	}
	return nil
}
