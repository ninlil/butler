// Package butler is a go-framework for writing cloud/kubernetes-friendly microservices
package butler

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/ninlil/butler/log"
)

var (
	cleanups map[string]func()
)

// OnClose registers a function to be called on butler-close/shutdown
func OnClose(name string, h func()) {
	if cleanups == nil {
		cleanups = make(map[string]func())
	}
	cleanups[name] = h
}

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
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		log.Debug().Msgf("!! Signal = %s", sig)

		for name, cleanup := range cleanups {
			log.Debug().Msgf("-- cleanup %s", name)
			cleanup()
		}
		done <- true
	}()

	log.Debug().Msg("awaiting signal")
	<-done
	log.Debug().Msg("exiting")
}
