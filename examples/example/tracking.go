package main

import (
	"context"

	"github.com/ninlil/butler/router"
)

type trackingResult struct {
	ReqID  string `json:"request"`
	CorrID string `json:"correlation"`
}

func tracking(ctx context.Context) *trackingResult {
	return &trackingResult{
		ReqID:  router.ReqIDFromCtx(ctx),
		CorrID: router.CorrIDFromCtx(ctx),
	}
}
