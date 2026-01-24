package generate

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"time"
	"ws-json-rpc/backend/internal/database/sqlite"
	"ws-json-rpc/backend/pkg/database"
)

// GenerateDatabaseSchema runs migrations on a temporary database and returns the resulting schema.
// This generates a SQL schema dump from the application's migrations.
func (g *OpenAPICollector) GenerateDatabaseSchema(l *slog.Logger, schemaOutputPath string) (string, error) {
	l.Debug("Generating database schema from migrations")

	// Create a temporary database file
	tempDBPath := fmt.Sprintf("%s/temp-db-%d.sqlite", os.TempDir(), time.Now().Unix())
	defer os.Remove(tempDBPath)

	// Create a migrator for the temporary database
	mig, err := database.NewMigrator(l, sqlite.GetMigrationsFS(), tempDBPath)
	if err != nil {
		return "", fmt.Errorf("failed to create migrator: %w", err)
	}

	// Run migrations
	if err := mig.Migrate(); err != nil {
		return "", fmt.Errorf("failed to migrate database: %w", err)
	}

	// Dump the database schema to the specified output path
	if err = mig.DumpSchema(schemaOutputPath); err != nil {
		return "", fmt.Errorf("failed to dump schema: %w", err)
	}

	// Read the schema file
	schemaBytes, err := os.ReadFile(schemaOutputPath)
	if err != nil {
		return "", fmt.Errorf("failed to read schema file: %w", err)
	}

	l.Info("Database schema generated", slog.String("file", schemaOutputPath))

	return string(bytes.TrimSpace(schemaBytes)), nil
}
