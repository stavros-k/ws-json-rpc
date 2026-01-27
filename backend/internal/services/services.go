package services

import (
	"database/sql"
	"log/slog"
	sqlitegen "ws-json-rpc/backend/internal/database/sqlite/gen"
)

type Services struct {
	l    *slog.Logger
	Core *CoreService
}

func NewServices(l *slog.Logger, db *sql.DB, queries *sqlitegen.Queries) *Services {
	return &Services{
		l:    l.With(slog.String("module", "services")),
		Core: NewCoreService(l, db),
	}
}
