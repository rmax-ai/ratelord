package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
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
		payload JSON NOT NULL,

		-- Epoch (Cluster term)
		epoch INTEGER DEFAULT 0
	);
	
	-- Index for replay by ingestion order (implicit rowid is efficient, but ingest time is explicit)
	CREATE INDEX IF NOT EXISTS idx_events_ts_ingest ON events(ts_ingest);
	
	-- Index for querying by event time
	CREATE INDEX IF NOT EXISTS idx_events_ts_event ON events(ts_event);
	
	-- Index for filtering by event type
	CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type);
	
	-- Index for filtering by identity
	CREATE INDEX IF NOT EXISTS idx_events_identity ON events(identity_id);
	
	-- Index for filtering by scope
	CREATE INDEX IF NOT EXISTS idx_events_scope ON events(scope_id);
	
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

	-- Webhook Configurations
	CREATE TABLE IF NOT EXISTS webhook_configs (
		webhook_id TEXT PRIMARY KEY,
		url TEXT NOT NULL,
		secret TEXT NOT NULL,
		events TEXT NOT NULL, -- Stored as JSON array
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		active BOOLEAN NOT NULL DEFAULT 1
	);

	-- Snapshots table for state persistence and faster recovery
	CREATE TABLE IF NOT EXISTS snapshots (
		snapshot_id TEXT PRIMARY KEY,
		schema_version INTEGER NOT NULL DEFAULT 1,
		ts_snapshot DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		last_event_id TEXT NOT NULL,
		payload JSON NOT NULL,

		FOREIGN KEY(last_event_id) REFERENCES events(event_id)
	);

	-- Index to quickly find the most recent snapshot for recovery
	CREATE INDEX IF NOT EXISTS idx_snapshots_ts ON snapshots(ts_snapshot DESC);

	-- Leases table for distributed locking (Leader Election)
	CREATE TABLE IF NOT EXISTS leases (
		name TEXT PRIMARY KEY,
		holder_id TEXT NOT NULL,
		expires_at DATETIME NOT NULL,
		version INTEGER NOT NULL DEFAULT 1,
		epoch INTEGER NOT NULL DEFAULT 0
	);
	`

	if _, err := s.db.Exec(query); err != nil {
		return fmt.Errorf("failed to create events table: %w", err)
	}

	// Migrations for existing tables
	// Ignore errors (duplicate column)
	s.db.Exec("ALTER TABLE events ADD COLUMN epoch INTEGER DEFAULT 0;")
	s.db.Exec("ALTER TABLE leases ADD COLUMN epoch INTEGER DEFAULT 0;")

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
		payload,
		epoch
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
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
		evt.Epoch,
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
		payload,
		epoch
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
			&evt.Epoch,
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
		payload,
		epoch
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
			&evt.Epoch,
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

// QueryEvents retrieves events based on the provided filter.
func (s *Store) QueryEvents(ctx context.Context, filter EventFilter) ([]*Event, error) {
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
			payload,
			epoch
		FROM events
		WHERE 1=1
	`

	args := []interface{}{}

	if !filter.From.IsZero() {
		query += " AND ts_event >= ?"
		args = append(args, filter.From)
	}

	if !filter.To.IsZero() {
		query += " AND ts_event < ?"
		args = append(args, filter.To)
	}

	if len(filter.EventTypes) > 0 {
		placeholders := make([]string, len(filter.EventTypes))
		for i := range placeholders {
			placeholders[i] = "?"
		}
		query += fmt.Sprintf(" AND event_type IN (%s)", strings.Join(placeholders, ","))
		for _, et := range filter.EventTypes {
			args = append(args, et)
		}
	}

	if filter.IdentityID != "" {
		query += " AND identity_id = ?"
		args = append(args, filter.IdentityID)
	}

	if filter.ScopeID != "" {
		query += " AND scope_id = ?"
		args = append(args, filter.ScopeID)
	}

	query += " ORDER BY ts_event ASC"

	limit := filter.Limit
	if limit == 0 {
		limit = 1000
	}
	query += " LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
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
			&evt.Epoch,
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

// RegisterWebhook creates a new webhook configuration.
func (s *Store) RegisterWebhook(ctx context.Context, cfg *WebhookConfig) error {
	query := `
	INSERT INTO webhook_configs (webhook_id, url, secret, events, created_at, active)
	VALUES (?, ?, ?, ?, ?, ?);
	`
	eventsJSON, err := json.Marshal(cfg.Events)
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	_, err = s.db.ExecContext(ctx, query,
		cfg.WebhookID,
		cfg.URL,
		cfg.Secret,
		string(eventsJSON),
		cfg.CreatedAt,
		cfg.Active,
	)
	if err != nil {
		return fmt.Errorf("failed to register webhook: %w", err)
	}
	return nil
}

// ListWebhooks retrieves all registered webhooks.
func (s *Store) ListWebhooks(ctx context.Context) ([]*WebhookConfig, error) {
	query := `SELECT webhook_id, url, secret, events, created_at, active FROM webhook_configs`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list webhooks: %w", err)
	}
	defer rows.Close()

	var webhooks []*WebhookConfig
	for rows.Next() {
		var w WebhookConfig
		var eventsJSON string
		if err := rows.Scan(&w.WebhookID, &w.URL, &w.Secret, &eventsJSON, &w.CreatedAt, &w.Active); err != nil {
			return nil, fmt.Errorf("failed to scan webhook: %w", err)
		}
		if err := json.Unmarshal([]byte(eventsJSON), &w.Events); err != nil {
			return nil, fmt.Errorf("failed to unmarshal events: %w", err)
		}
		webhooks = append(webhooks, &w)
	}
	return webhooks, nil
}

// DeleteWebhook removes a webhook configuration.
func (s *Store) DeleteWebhook(ctx context.Context, webhookID string) error {
	query := `DELETE FROM webhook_configs WHERE webhook_id = ?`
	_, err := s.db.ExecContext(ctx, query, webhookID)
	if err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}
	return nil
}

// SaveSnapshot persists a state snapshot.
func (s *Store) SaveSnapshot(ctx context.Context, snap *Snapshot) error {
	query := `
	INSERT INTO snapshots (snapshot_id, schema_version, ts_snapshot, last_event_id, payload)
	VALUES (?, ?, ?, ?, ?);
	`
	_, err := s.db.ExecContext(ctx, query,
		snap.SnapshotID,
		snap.SchemaVersion,
		snap.TsSnapshot,
		snap.LastEventID,
		snap.Payload,
	)
	if err != nil {
		return fmt.Errorf("failed to save snapshot: %w", err)
	}
	return nil
}

// GetLatestSnapshot retrieves the most recent snapshot.
// Returns nil, nil if no snapshot exists.
func (s *Store) GetLatestSnapshot(ctx context.Context) (*Snapshot, error) {
	query := `
		SELECT snapshot_id, schema_version, ts_snapshot, last_event_id, payload
		FROM snapshots
		ORDER BY ts_snapshot DESC
		LIMIT 1;
		`
	row := s.db.QueryRowContext(ctx, query)
	var snap Snapshot
	err := row.Scan(
		&snap.SnapshotID,
		&snap.SchemaVersion,
		&snap.TsSnapshot,
		&snap.LastEventID,
		&snap.Payload,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest snapshot: %w", err)
	}
	return &snap, nil
}

// GetLatestSnapshotTime retrieves the timestamp of the most recent snapshot.
// Returns zero time if no snapshot exists.
func (s *Store) GetLatestSnapshotTime(ctx context.Context) (time.Time, error) {
	query := `
		SELECT ts_snapshot
		FROM snapshots
		ORDER BY ts_snapshot DESC
		LIMIT 1;
		`
	var ts time.Time
	err := s.db.QueryRowContext(ctx, query).Scan(&ts)
	if err != nil {
		if err == sql.ErrNoRows {
			return time.Time{}, nil
		}
		return time.Time{}, fmt.Errorf("failed to get latest snapshot time: %w", err)
	}
	return ts, nil
}

// GetEvent retrieves a single event by ID.
func (s *Store) GetEvent(ctx context.Context, eventID EventID) (*Event, error) {
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
		payload,
		epoch
	FROM events
	WHERE event_id = ?;
	`

	row := s.db.QueryRowContext(ctx, query, eventID)
	var evt Event
	var payload []byte

	err := row.Scan(
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
		&evt.Epoch,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get event %s: %w", eventID, err)
	}

	evt.Payload = json.RawMessage(payload)
	return &evt, nil
}

// PruneEvents deletes events older than the retention duration.
// includeType: if set, only delete this type.
// excludeTypes: if set, do NOT delete these types (ignored if includeType is set).
func (s *Store) PruneEvents(ctx context.Context, retention time.Duration, includeType string, excludeTypes []string) (int64, error) {
	// 1. Get the latest snapshot
	snap, err := s.GetLatestSnapshot(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get latest snapshot for safety check: %w", err)
	}
	if snap == nil {
		return 0, fmt.Errorf("cannot prune: no snapshots found (create a snapshot first)")
	}

	// 2. Get boundary
	snapEvent, err := s.GetEvent(ctx, EventID(snap.LastEventID))
	if err != nil {
		return 0, fmt.Errorf("failed to get snapshot boundary event: %w", err)
	}
	if snapEvent == nil {
		return 0, fmt.Errorf("snapshot boundary event %s not found", snap.LastEventID)
	}

	cutoffTime := time.Now().Add(-retention)
	if snapEvent.TsIngest.Before(cutoffTime) {
		cutoffTime = snapEvent.TsIngest
	}

	query := `DELETE FROM events WHERE ts_ingest < ?`
	args := []interface{}{cutoffTime}

	if includeType != "" {
		query += " AND event_type = ?"
		args = append(args, includeType)
	} else if len(excludeTypes) > 0 {
		placeholders := make([]string, len(excludeTypes))
		for i, t := range excludeTypes {
			placeholders[i] = "?"
			args = append(args, t)
		}
		query += fmt.Sprintf(" AND event_type NOT IN (%s)", strings.Join(placeholders, ","))
	}

	res, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to prune events: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}

	return rows, nil
}

// ReadCandidateEvents fetches events older than a specific time, limited by batch size.
// Used for archiving old events to cold storage.
func (s *Store) ReadCandidateEvents(ctx context.Context, before time.Time, limit int) ([]*Event, error) {
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
			payload,
			epoch
		FROM events
		WHERE ts_ingest < ?
		ORDER BY ts_ingest ASC
		LIMIT ?;
		`

	rows, err := s.db.QueryContext(ctx, query, before, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query candidate events: %w", err)
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
			&evt.Epoch,
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

// DeleteEvents deletes a specific set of events by ID.
// Used after archiving events to cold storage.
func (s *Store) DeleteEvents(ctx context.Context, eventIDs []string) error {
	if len(eventIDs) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	placeholders := make([]string, len(eventIDs))
	args := make([]interface{}, len(eventIDs))
	for i, id := range eventIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf("DELETE FROM events WHERE event_id IN (%s)", strings.Join(placeholders, ","))

	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete events: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
