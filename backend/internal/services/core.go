package services

import (
	"database/sql"
	"log/slog"
)

type CoreService struct {
	l  *slog.Logger
	db *sql.DB
}

func NewCoreService(l *slog.Logger, db *sql.DB) *CoreService {
	return &CoreService{
		l:  l.With(slog.String("service", "core")),
		db: db,
	}
}

func (s *CoreService) Ping() bool {
	if err := s.db.Ping(); err != nil {
		s.l.Error("database unreachable", slog.String("error", err.Error()))
		return false
	}
	return true
}
