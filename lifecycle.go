// Package butler is a go-framework for writing cloud/kubernetes-friendly microservices
package butler

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ninlil/butler/log"
	"github.com/ninlil/butler/router"
	"github.com/ninlil/butler/runtime"
	"github.com/ninlil/butler/workers"
)

// Cleanup should be called using 'defer butler.Cleanup(myFunc|nil)' in your main package
func Cleanup(h func(ctx context.Context)) {
	runtime.OnClose("butler_close", butlerClose)
	if h != nil {
		h(context.Background())
	}
}

var (
	sigs = make(chan os.Signal, 1)
	done = make(chan bool, 1)
)

// Quit sends a signal to the butler to stop
func Quit() {
	sigs <- syscall.SIGQUIT
}

func butlerClose() {
	log.Trace().Msg("closing butler...")
	select {
	case done <- true:
		//log.Trace().Msg("butler closed - ok")
	case <-time.After(10 * time.Second):
		log.Trace().Msg("butler close - timeout")
	}
}

// Run makes the butler start all pending tasks and wait until done or signalled to stop
//
// Normal stop-signals is SIGINT and SIGTERM allowing for a graceful close/shutdown
func Run() {
	runtime.OnClose("butler_close", butlerClose)

	if workers.OnDone == workers.ReadyOnDone {
		router.Ready = false
	}

	workersDone := workers.StartPending()

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		log.Info().Msgf("!! Signal = %s", sig)

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

	log.Trace().Msg("butler: lifecycle running (awaiting signal)")
	<-done
	log.Trace().Msg("butler: exiting")
}
