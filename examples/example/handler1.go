package main

import (
	"context"

	"github.com/ninlil/butler/log"
)

type handlerArgs struct {
	Sum *sumArgs `from:"body"`
}

type sumArgs struct {
	A float64 `from:"query" json:"a" required:""`
	B float64 `from:"query" json:"b" required:""`
}

type handlerResult struct {
	Sum float64 `json:"sum"`
}

func handler(ctx context.Context, args *handlerArgs) *handlerResult {
	log := log.FromCtx(ctx)
	if args.Sum != nil {
		log.Info().Msgf("handler called with a=%v and b=%v", args.Sum.A, args.Sum.B)
		return &handlerResult{
			Sum: args.Sum.A + args.Sum.B,
		}
	}

	log.Warn().Msg("no data in body")
	return nil
}
