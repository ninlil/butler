package main

import (
	"context"
	"time"

	"github.com/ninlil/butler"
	"github.com/ninlil/butler/log"
	"github.com/ninlil/butler/workers"
)

func main() {
	defer butler.Cleanup(nil)

	workers.New("alpha", alpha)
	//workers.New("alpha", alpha)
	workers.New("beta", beta, workers.WithRealPanic())

	butler.Run()
}

func alpha(ctx context.Context) {
	log := log.FromCtx(ctx)

	log.Info().Msg("inside Alpha")
	time.Sleep(2 * time.Second)

	workers.New("gamma", gamma)

	//panic("alpha")
}

func beta(ctx context.Context) {
	log := log.FromCtx(ctx)

	log.Info().Msg("inside Beta")
	time.Sleep(4 * time.Second)
	//panic("beta")
}

func gamma(ctx context.Context) {
	log := log.FromCtx(ctx)

	log.Info().Msg("inside Gamma")
	time.Sleep(3 * time.Second)
}
