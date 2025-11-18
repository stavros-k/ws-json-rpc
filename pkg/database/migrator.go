package database

import (
	"embed"
	"fmt"
	"log/slog"
	"net/url"
	"ws-json-rpc/pkg/utils"

	"github.com/amacneil/dbmate/v2/pkg/dbmate"
	_ "github.com/amacneil/dbmate/v2/pkg/driver/sqlite"
)

type Migrator struct {
	db      *dbmate.DB
	fs      embed.FS // This must contain a migrations directory
	sqlPath string
	l       *slog.Logger
}

// TODO: Set a common set of PRAGMA settings for SQLite connections
// TODO: Test if we can edit db from a db browser while working
func NewMigrator(fs embed.FS, sqlPath string, l *slog.Logger) (*Migrator, error) {
	if sqlPath == "" {
		return nil, fmt.Errorf("sqlPath is required")
	}

	_, err := fs.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	u, err := url.Parse("sqlite:" + sqlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database url: %w", err)
	}
	db := dbmate.New(u)
	db.Strict = true
	db.FS = fs
	db.MigrationsDir = []string{"migrations"}
	db.AutoDumpSchema = false

	l = l.With(slog.String("component", "db-migrator"))
	db.Log = utils.NewSlogWriter(l)

	return &Migrator{
		l:       l,
		db:      db,
		fs:      fs,
		sqlPath: sqlPath,
	}, nil
}

func (m *Migrator) Migrate() error {
	m.l.Info("Migrating database")
	if err := m.db.Migrate(); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	return nil
}

func (m *Migrator) DumpSchema(filePath string) error {
	m.db.SchemaFile = filePath

	m.l.Info("Dumping schema", slog.String("file", filePath))

	if err := m.db.DumpSchema(); err != nil {
		return fmt.Errorf("failed to dump schema: %w", err)
	}

	return nil
}
