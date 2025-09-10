package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"time"
	"ws-json-rpc/pkg/ws"
)

func LoggingMiddleware() func(ws.HandlerFunc) ws.HandlerFunc {
	return func(next ws.HandlerFunc) ws.HandlerFunc {
		return func(ctx context.Context, params any) (any, error) {
			cc, ok := ws.ClientContextFromContext(ctx)
			if !ok {
				return nil, fmt.Errorf("no client found")
			}

			start := time.Now()
			cc.Logger.Debug("request started",
				slog.Time("timestamp", start),
			)

			result, err := next(ctx, params)

			cc.Logger.Debug("request completed",
				slog.Duration("duration", time.Since(start)),
				slog.Bool("success", err == nil),
			)

			return result, err
		}
	}
}
