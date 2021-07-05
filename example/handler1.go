package main

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
