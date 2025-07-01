package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ddritzenhoff/statsd/sqlite/gen"
)

// embed the sqlite schema within the binary to create the tables at runtime.
//
//go:embed schema.sql
var Schema string

// DB represents the database connection.
type DB struct {
	db    *sql.DB
	query *gen.Queries

	// Datasource name
	dsn string

	// Returns the current time. Defaults to time.now().
	// Can be mocked for tests.
	now func() time.Time
}

// NewDB returns a new instance of DB associated with the given datasource name.
func NewDB(dsn string) *DB {
	return &DB{
		dsn: dsn,
		now: time.Now,
	}
}

// Open opens the database connection
func (db *DB) Open() (err error) {
	if db.dsn == "" {
		return fmt.Errorf("dsn required")
	}

	// Make the parent directory unless using an in-memory db.
	if !strings.Contains(db.dsn, ":memory:") {
		if err := os.MkdirAll(filepath.Dir(db.dsn), 0700); err != nil {
			return err
		}
	}

	if db.db, err = sql.Open("sqlite3", db.dsn); err != nil {
		return err
	}

	// verify data source name is valid.
	if err := db.db.Ping(); err != nil {
		return fmt.Errorf("ping: %w", err)
	}

	db.query = gen.New(db.db)

	// Enable WAL as it allows multiple readers to operate while data is being written.
	if _, err := db.db.Exec(`PRAGMA journal_mode = wal;`); err != nil {
		return fmt.Errorf("enable wal: %w", err)
	}

	// Enable foreign key checks.
	if _, err := db.db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		return fmt.Errorf("foreign keys pragma: %w", err)
	}

	// Create tables if they don't exist.
	if _, err := db.db.Exec(Schema); err != nil {
		return fmt.Errorf("create tables: %w", err)
	}

	return nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	// close connection.
	if db.db != nil {
		return db.db.Close()
	}
	return nil
}

// BeginTx starts a transaction and returns a wrapper Tx type. This type
// provides a reference to the database and a fixed timestamp at the start of
// the transaction. The timestamp allows us to mock time during tests as well.
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	tx, err := db.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Return wrapper Tx that includes the transaction start time.
	return &Tx{
		Tx:  tx,
		db:  db,
		now: db.now().UTC().Truncate(time.Second),
	}, nil
}

// Tx wraps the SQL Tx object to provide a timestamp at the start of the transaction.
type Tx struct {
	*sql.Tx
	db  *DB
	now time.Time
}
