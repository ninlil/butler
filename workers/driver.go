package workers

import (
	"sync"
	"time"

	"github.com/ninlil/butler/log"
	"github.com/rs/zerolog"
)

type driverType struct {
	list    map[string]*Worker
	wg      sync.WaitGroup
	started bool
}

var driver = new(driverType)

func (d *driverType) add(w *Worker) *Worker {
	if w == nil {
		return nil
	}

	if d.list == nil {
		d.list = make(map[string]*Worker)
	}

	if _, found := d.list[w.name]; found {
		return nil
	}

	d.list[w.name] = w

	if d.started {
		d.wg.Add(1)
		go d.startWorker(w)
	}
	return w
}

// StartPending starts all pending workers
func StartPending() chan DoneBehavior {
	if len(driver.list) == 0 {
		return nil
	}

	done := make(chan DoneBehavior)
	go driver.start(done)
	return done
}

func (d *driverType) start(done chan DoneBehavior) {
	for _, w := range d.list {
		if w.state == statePending {
			d.wg.Add(1)
			go d.startWorker(w)
		}
	}
	d.started = true

	d.wg.Wait()
	log.Debug().Msgf("workers: all done.. signalling %d", OnDone)
	select {
	case done <- OnDone:
	case <-time.After(30 * time.Second):
	}
}

func (d *driverType) startWorker(w *Worker) {
	log := log.FromCtx(w.ctx)

	defer func() {
		w.state = stateDone
		w.ended = time.Now()
		if w.realPanic {
			log.Debug().Msgf("workers: [%s] exit", w.name)
		} else {
			if err := recover(); err != nil {
				w.state = statePanic
				log.WithLevel(zerolog.PanicLevel).Caller(2).Msgf("worker-panic: %v", err)
			} else {
				log.Debug().Msgf("workers: [%s] done", w.name)
			}
		}
		d.wg.Done()
	}()

	w.started = time.Now()
	w.state = stateRunning
	log.Debug().Msgf("workers: [%s] starting...", w.name)
	w.handler(w.ctx)
}
