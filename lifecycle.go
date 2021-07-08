// Package butler is a go-framework for writing cloud/kubernetes-friendly microservices
package butler

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/ninlil/butler/log"
	"github.com/ninlil/butler/router"
	"github.com/ninlil/butler/runtime"
	"github.com/ninlil/butler/workers"
)

// Cleanup should be called using 'defer butler.Cleanup(myFunc|nil)' in your main package
func Cleanup(h func(ctx context.Context)) {
	if h != nil {
		h(context.Background())
	}
}

var (
	sigs = make(chan os.Signal, 1)
	done = make(chan bool, 1)
)

// Run makes the butler start all pending tasks and wait until done or signalled to stop
//
// Normal stop-signals is SIGINT and SIGTERM allowing for a graceful close/shutdown
func Run() {

	if workers.OnDone == workers.ReadyOnDone {
		router.Ready = false
	}

	workersDone := workers.StartPending()

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		log.Debug().Msgf("!! Signal = %s", sig)

		runtime.Close()
		done <- true
	}()

	if workersDone != nil {
		go func() {
			onDone := <-workersDone
			switch onDone {
			case workers.ExitOnDone:
				sigs <- syscall.SIGQUIT
			case workers.ContinueOnDone:
				// nothing to do
			case workers.ReadyOnDone:
				router.Ready = true
			}
		}()
	}

	log.Debug().Msg("awaiting signal")
	<-done
	log.Debug().Msg("exiting")
}
