package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Acquire tries to acquire the lease. Returns true if successful.
// If the lease is already held by holderID, it renews it.
func (s *Store) Acquire(ctx context.Context, name, holderID string, ttl time.Duration) (bool, error) {
	now := time.Now().UTC()
	expiry := now.Add(ttl)

	// Optimistic concurrency loop? No, just atomic SQL statements.

	// 1. Try to insert (if it doesn't exist)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO leases (name, holder_id, expires_at, version) 
		VALUES (?, ?, ?, 1)
	`, name, holderID, expiry)

	if err == nil {
		return true, nil
	}

	// If unique constraint violation (already exists), try to take over if expired or if we own it.
	// We do this in a single atomic UPDATE to avoid race conditions.
	res, err := s.db.ExecContext(ctx, `
		UPDATE leases 
		SET holder_id = ?, expires_at = ?, version = version + 1
		WHERE name = ? AND (holder_id = ? OR expires_at < ?)
	`, holderID, expiry, name, holderID, now)

	if err != nil {
		return false, fmt.Errorf("failed to update lease: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to check rows affected: %w", err)
	}

	return rows > 0, nil
}

// Renew updates the expiry of an existing lease held by holderID.
// Returns error if the lease is lost or stolen.
func (s *Store) Renew(ctx context.Context, name, holderID string, ttl time.Duration) error {
	now := time.Now().UTC()
	expiry := now.Add(ttl)

	res, err := s.db.ExecContext(ctx, `
		UPDATE leases 
		SET expires_at = ?, version = version + 1
		WHERE name = ? AND holder_id = ?
	`, expiry, name, holderID)

	if err != nil {
		return fmt.Errorf("failed to renew lease: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("lease lost or stolen")
	}

	return nil
}

// Release releases the lease if held by holderID.
func (s *Store) Release(ctx context.Context, name, holderID string) error {
	// We can delete the row, or just set expires_at to 0 (or past).
	// Deleting is cleaner for "no leader".
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM leases WHERE name = ? AND holder_id = ?
	`, name, holderID)

	if err != nil {
		return fmt.Errorf("failed to release lease: %w", err)
	}

	return nil
}

// Get returns the current lease state.
func (s *Store) Get(ctx context.Context, name string) (*Lease, error) {
	var l Lease
	err := s.db.QueryRowContext(ctx, `
		SELECT name, holder_id, expires_at, version 
		FROM leases WHERE name = ?
	`, name).Scan(&l.Name, &l.HolderID, &l.ExpiresAt, &l.Version)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get lease: %w", err)
	}

	return &l, nil
}
