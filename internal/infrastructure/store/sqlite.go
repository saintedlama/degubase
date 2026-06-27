package store

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.up.sql
var migrationsFS embed.FS

type Store struct {
	db *sql.DB
}

// DB returns the underlying database connection.
func (s *Store) DB() *sql.DB { return s.db }

// Timestamp policy (see migration 020): application code must always set
// created_at and updated_at explicitly. The DB-level DEFAULT (strftime(...))
// expressions that exist in early migrations are kept for backward compatibility
// only — SQLite does not support ALTER COLUMN DROP DEFAULT. No new columns
// should rely on DB defaults for timestamps.

func New(dsn string) (*Store, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version    TEXT PRIMARY KEY,
		applied_at TEXT NOT NULL
	)`); err != nil {
		return err
	}

	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".up.sql") {
			continue
		}
		var count int
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, e.Name()).Scan(&count)
		if count > 0 {
			continue
		}
		data, err := migrationsFS.ReadFile("migrations/" + e.Name())
		if err != nil {
			return err
		}
		if _, err := s.db.Exec(string(data)); err != nil {
			return fmt.Errorf("apply migration %s: %w", e.Name(), err)
		}
		if _, err := s.db.Exec(`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`, e.Name(), time.Now().UTC().Format(time.RFC3339)); err != nil {
			return err
		}
	}
	return nil
}
