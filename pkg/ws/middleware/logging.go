package middleware

import (
	"context"
	"log/slog"
	"time"
	"ws-json-rpc/pkg/ws"
)

func LoggingMiddleware(next ws.HandlerFunc) ws.HandlerFunc {
	return func(ctx context.Context, hctx *ws.HandlerContext, params any) (any, error) {
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
