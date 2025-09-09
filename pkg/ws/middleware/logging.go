package middleware

import (
	"context"
	"log/slog"
	"time"
	"ws-json-rpc/pkg/ws"
)

func LoggingMiddleware(logger *slog.Logger) func(ws.HandlerFunc) ws.HandlerFunc {
	return func(next ws.HandlerFunc) ws.HandlerFunc {
		return func(ctx context.Context, params any) (any, error) {
			start := time.Now()
			clientID := ws.ClientID(ctx)

			logger.Debug("handler started",
				slog.String("client_id", clientID),
				slog.Time("timestamp", start))

			result, err := next(ctx, params)

			logger.Debug("handler completed",
				slog.String("client_id", clientID),
				slog.Duration("duration", time.Since(start)),
				slog.Bool("success", err == nil))

			return result, err
		}
	}
}
