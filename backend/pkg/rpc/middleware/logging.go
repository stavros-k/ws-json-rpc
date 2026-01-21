package middleware

import (
	"context"
	"log/slog"
	"time"
	"ws-json-rpc/backend/pkg/rpc"
)

func LoggingMiddleware(next rpc.HandlerFunc) rpc.HandlerFunc {
	return func(ctx context.Context, hctx *rpc.HandlerContext, params any) (any, error) {
		start := time.Now()
		hctx.Logger = hctx.Logger.With(
			slog.Time("req_start_time", start),
		)

		hctx.Logger.Debug("request started")

		result, err := next(ctx, hctx, params)
		logAttrs := []any{
			slog.Duration("req_duration", time.Since(start)),
			slog.Bool("req_success", err == nil),
		}

		// Update the logger with the final attributes
		hctx.Logger = hctx.Logger.With(logAttrs...)
		hctx.Logger.Info("request finished")

		return result, err
	}
}
