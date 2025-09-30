package middleware

import (
	"context"
	"log/slog"
	"time"
	"ws-json-rpc/pkg/rpc"
)

func LoggingMiddleware(next rpc.HandlerFunc) rpc.HandlerFunc {
	return func(ctx context.Context, hctx *rpc.HandlerContext, params any) (any, error) {
		start := time.Now()
		hctx.Logger.Debug("request started",
			slog.Time("timestamp", start),
		)

		result, err := next(ctx, hctx, params)

		hctx.Logger.Debug("request completed",
			slog.Duration("duration", time.Since(start)),
			slog.Bool("success", err == nil),
		)

		return result, err
	}
}
