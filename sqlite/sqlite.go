package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

// embed the sqlite schema within the binary to create the tables at runtime.
//
//go:embed schema.sql
var Schema string

// Open creates a connection to the sqlite database.
func Open(DSN string) (*sql.DB, error) {
	// Ensure a DSN is set before attempting to open the database.
	if DSN == "" {
		return nil, fmt.Errorf("Open: dsn required")
	}

	// Make the parent directory unless using an in-memory db.
	if DSN != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(DSN), 0700); err != nil {
			return nil, fmt.Errorf("Open os.MkdirAll: %w", err)
		}
	}

	// Connect to the database.
	db, err := sql.Open("sqlite3", DSN)
	if err != nil {
		return nil, fmt.Errorf("Open sql.Open: %w", err)
	}

	// Enable WAL.
	if _, err := db.Exec(`PRAGMA journal_mode = wal;`); err != nil {
		return nil, fmt.Errorf("Open enable wal: %w", err)
	}

	// Enable foreign key checks.
	if _, err := db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		return nil, fmt.Errorf("Open foreign keys pragma: %w", err)
	}

	// Create tables if they don't exist.
	_, err = db.ExecContext(context.Background(), Schema)
	if err != nil {
		return nil, fmt.Errorf("Open db.ExecContext: %w", err)
	}

	return db, nil
}
