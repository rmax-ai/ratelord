package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

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
	
	-- Track background worker progress
	CREATE TABLE IF NOT EXISTS system_state (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Hourly Aggregation
	CREATE TABLE IF NOT EXISTS usage_hourly (
		bucket_ts DATETIME NOT NULL,
		provider_id TEXT NOT NULL,
		pool_id TEXT NOT NULL,
		identity_id TEXT NOT NULL,
		scope_id TEXT NOT NULL,
		
		total_usage INTEGER NOT NULL DEFAULT 0,
		min_usage INTEGER NOT NULL DEFAULT 0,
		max_usage INTEGER NOT NULL DEFAULT 0,
		event_count INTEGER NOT NULL DEFAULT 0,
		
		PRIMARY KEY (bucket_ts, provider_id, pool_id, identity_id, scope_id)
	);

	-- Daily Aggregation
	CREATE TABLE IF NOT EXISTS usage_daily (
		bucket_ts DATETIME NOT NULL,
		provider_id TEXT NOT NULL,
		pool_id TEXT NOT NULL,
		identity_id TEXT NOT NULL,
		scope_id TEXT NOT NULL,
		
		total_usage INTEGER NOT NULL DEFAULT 0,
		min_usage INTEGER NOT NULL DEFAULT 0,
		max_usage INTEGER NOT NULL DEFAULT 0,
		event_count INTEGER NOT NULL DEFAULT 0,
		
		PRIMARY KEY (bucket_ts, provider_id, pool_id, identity_id, scope_id)
	);

	CREATE INDEX IF NOT EXISTS idx_usage_hourly_time ON usage_hourly(bucket_ts);
	CREATE INDEX IF NOT EXISTS idx_usage_daily_time ON usage_daily(bucket_ts);
	`

	if _, err := s.db.Exec(query); err != nil {
		return fmt.Errorf("failed to create events table: %w", err)
	}

	return nil
}

// AppendEvent writes a single event to the database.
// It is an append-only operation.
func (s *Store) AppendEvent(ctx context.Context, evt *Event) error {
	query := `
	INSERT INTO events (
		event_id,
		event_type,
		schema_version,
		ts_event,
		ts_ingest,
		origin_kind,
		origin_id,
		writer_id,
		agent_id,
		identity_id,
		workload_id,
		scope_id,
		correlation_id,
		causation_id,
		payload
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`

	// Ensure ts_ingest is set. If zero, default to current time.
	// Although the DB has a default, setting it explicitly ensures
	// the struct in memory matches what's persisted if we continue to use it.
	tsIngest := evt.TsIngest
	if tsIngest.IsZero() {
		tsIngest = time.Now().UTC()
	}

	// Payload is already []byte (json.RawMessage), so we can store it directly.
	// If it's nil, we should probably store empty JSON object "{}" or null depending on requirements.
	// The schema says JSON NOT NULL, so let's default to "{}" if nil/empty.
	payload := evt.Payload
	if len(payload) == 0 {
		payload = []byte("{}")
	}

	_, err := s.db.ExecContext(ctx, query,
		evt.EventID,
		evt.EventType,
		evt.SchemaVersion,
		evt.TsEvent,
		tsIngest,
		evt.Source.OriginKind,
		evt.Source.OriginID,
		evt.Source.WriterID,
		evt.Dimensions.AgentID,
		evt.Dimensions.IdentityID,
		evt.Dimensions.WorkloadID,
		evt.Dimensions.ScopeID,
		evt.Correlation.CorrelationID,
		evt.Correlation.CausationID,
		payload,
	)

	if err != nil {
		return fmt.Errorf("failed to append event %s: %w", evt.EventID, err)
	}

	return nil
}

// ReadEvents retrieves events ingested after a specific timestamp.
// This is used for replaying the event log to rebuild state.
func (s *Store) ReadEvents(ctx context.Context, since time.Time, limit int) ([]*Event, error) {
	query := `
	SELECT
		event_id,
		event_type,
		schema_version,
		ts_event,
		ts_ingest,
		origin_kind,
		origin_id,
		writer_id,
		agent_id,
		identity_id,
		workload_id,
		scope_id,
		correlation_id,
		causation_id,
		payload
	FROM events
	WHERE ts_ingest > ?
	ORDER BY ts_ingest ASC
	LIMIT ?;
	`

	rows, err := s.db.QueryContext(ctx, query, since, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []*Event

	for rows.Next() {
		var evt Event
		var payload []byte

		err := rows.Scan(
			&evt.EventID,
			&evt.EventType,
			&evt.SchemaVersion,
			&evt.TsEvent,
			&evt.TsIngest,
			&evt.Source.OriginKind,
			&evt.Source.OriginID,
			&evt.Source.WriterID,
			&evt.Dimensions.AgentID,
			&evt.Dimensions.IdentityID,
			&evt.Dimensions.WorkloadID,
			&evt.Dimensions.ScopeID,
			&evt.Correlation.CorrelationID,
			&evt.Correlation.CausationID,
			&payload,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event row: %w", err)
		}

		evt.Payload = json.RawMessage(payload)
		events = append(events, &evt)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return events, nil
}

// ReadRecentEvents retrieves the N most recent events, ordered by ingestion time descending.
// This is primarily used for the diagnostics/dashboard view.
func (s *Store) ReadRecentEvents(ctx context.Context, limit int) ([]*Event, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
	SELECT
		event_id,
		event_type,
		schema_version,
		ts_event,
		ts_ingest,
		origin_kind,
		origin_id,
		writer_id,
		agent_id,
		identity_id,
		workload_id,
		scope_id,
		correlation_id,
		causation_id,
		payload
	FROM events
	ORDER BY ts_ingest DESC
	LIMIT ?;
	`

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent events: %w", err)
	}
	defer rows.Close()

	var events []*Event

	for rows.Next() {
		var evt Event
		var payload []byte

		err := rows.Scan(
			&evt.EventID,
			&evt.EventType,
			&evt.SchemaVersion,
			&evt.TsEvent,
			&evt.TsIngest,
			&evt.Source.OriginKind,
			&evt.Source.OriginID,
			&evt.Source.WriterID,
			&evt.Dimensions.AgentID,
			&evt.Dimensions.IdentityID,
			&evt.Dimensions.WorkloadID,
			&evt.Dimensions.ScopeID,
			&evt.Correlation.CorrelationID,
			&evt.Correlation.CausationID,
			&payload,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event row: %w", err)
		}

		evt.Payload = json.RawMessage(payload)
		events = append(events, &evt)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return events, nil
}

// GetSystemState retrieves the value for a given key from system_state.
// Returns an error if the key is not found.
func (s *Store) GetSystemState(ctx context.Context, key string) (string, error) {
	query := `SELECT value FROM system_state WHERE key = ?;`

	var value string
	err := s.db.QueryRowContext(ctx, query, key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("key not found: %s", key)
		}
		return "", fmt.Errorf("failed to get system state for key %s: %w", key, err)
	}

	return value, nil
}

// SetSystemState upserts the key-value pair into system_state.
func (s *Store) SetSystemState(ctx context.Context, key, value string) error {
	query := `INSERT OR REPLACE INTO system_state (key, value) VALUES (?, ?);`

	_, err := s.db.ExecContext(ctx, query, key, value)
	if err != nil {
		return fmt.Errorf("failed to set system state for key %s: %w", key, err)
	}

	return nil
}

func (s *Store) UpsertUsageStats(ctx context.Context, stats []UsageStat) error {
	if len(stats) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, stat := range stats {
		var table string
		if stat.BucketTs.Minute() == 0 && stat.BucketTs.Second() == 0 {
			if stat.BucketTs.Hour() == 0 {
				table = "usage_daily"
			} else {
				table = "usage_hourly"
			}
		} else {
			return fmt.Errorf("invalid bucket_ts for stat: must be at top of hour or day")
		}

		query := fmt.Sprintf(`
			INSERT INTO %s (
				bucket_ts, provider_id, pool_id, identity_id, scope_id,
				total_usage, min_usage, max_usage, event_count
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (bucket_ts, provider_id, pool_id, identity_id, scope_id)
			DO UPDATE SET
				total_usage = total_usage + excluded.total_usage,
				min_usage = MIN(min_usage, excluded.min_usage),
				max_usage = MAX(max_usage, excluded.max_usage),
				event_count = event_count + excluded.event_count;
		`, table)

		_, err := tx.ExecContext(ctx, query,
			stat.BucketTs, stat.ProviderID, stat.PoolID, stat.IdentityID, stat.ScopeID,
			stat.TotalUsage, stat.MinUsage, stat.MaxUsage, stat.EventCount,
		)
		if err != nil {
			return fmt.Errorf("failed to upsert usage stat: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetUsageStats retrieves usage statistics based on the provided filter.
func (s *Store) GetUsageStats(ctx context.Context, filter UsageFilter) ([]UsageStat, error) {
	var table string
	if filter.Bucket == "day" {
		table = "usage_daily"
	} else {
		table = "usage_hourly"
	}

	query := fmt.Sprintf(`
		SELECT bucket_ts, provider_id, pool_id, identity_id, scope_id,
		       total_usage, min_usage, max_usage, event_count
		FROM %s
		WHERE bucket_ts >= ? AND bucket_ts < ?
	`, table)

	args := []interface{}{filter.From, filter.To}

	if filter.ProviderID != "" {
		query += " AND provider_id = ?"
		args = append(args, filter.ProviderID)
	}
	if filter.PoolID != "" {
		query += " AND pool_id = ?"
		args = append(args, filter.PoolID)
	}
	if filter.IdentityID != "" {
		query += " AND identity_id = ?"
		args = append(args, filter.IdentityID)
	}
	if filter.ScopeID != "" {
		query += " AND scope_id = ?"
		args = append(args, filter.ScopeID)
	}

	query += " ORDER BY bucket_ts ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query usage stats: %w", err)
	}
	defer rows.Close()

	var stats []UsageStat
	for rows.Next() {
		var stat UsageStat
		err := rows.Scan(
			&stat.BucketTs,
			&stat.ProviderID,
			&stat.PoolID,
			&stat.IdentityID,
			&stat.ScopeID,
			&stat.TotalUsage,
			&stat.MinUsage,
			&stat.MaxUsage,
			&stat.EventCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan usage stat row: %w", err)
		}
		stats = append(stats, stat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return stats, nil
}
