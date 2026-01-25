package store

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
)

// Store manages the SQLite connection and schema.
type Store struct {
	db *sql.DB
}

// NewStore initializes the SQLite database connection.
// It enables WAL mode for concurrency and durability.
func NewStore(dbPath string) (*Store, error) {
	// Open the database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite db: %w", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping sqlite db: %w", err)
	}

	// Enable WAL mode (Write-Ahead Logging)
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Enforce foreign keys (good practice)
	if _, err := db.Exec("PRAGMA foreign_keys=ON;"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	s := &Store{db: db}

	// Initialize schema
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("schema migration failed: %w", err)
	}

	return s, nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// migrate creates the necessary tables if they don't exist.
func (s *Store) migrate() error {
	// Schema for the append-only events table
	// We store the canonical envelope fields as columns for querying,
	// and the full payload as a JSON blob.
	query := `
	CREATE TABLE IF NOT EXISTS events (
		event_id TEXT PRIMARY KEY,
		event_type TEXT NOT NULL,
		schema_version INTEGER NOT NULL,
		ts_event DATETIME NOT NULL,
		ts_ingest DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		
		-- Source metadata
		origin_kind TEXT,
		origin_id TEXT,
		writer_id TEXT,

		-- Dimensions (Mandatory)
		agent_id TEXT NOT NULL,
		identity_id TEXT NOT NULL,
		workload_id TEXT NOT NULL,
		scope_id TEXT NOT NULL,

		-- Correlation
		correlation_id TEXT,
		causation_id TEXT,

		-- Payload
		payload JSON NOT NULL
	);
	
	-- Index for replay by ingestion order (implicit rowid is efficient, but ingest time is explicit)
	CREATE INDEX IF NOT EXISTS idx_events_ts_ingest ON events(ts_ingest);
	
	-- Index for lookup by correlation (common access pattern)
	CREATE INDEX IF NOT EXISTS idx_events_correlation ON events(correlation_id);
	`

	if _, err := s.db.Exec(query); err != nil {
		return fmt.Errorf("failed to create events table: %w", err)
	}

	return nil
}
