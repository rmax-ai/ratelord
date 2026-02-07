package store

import (
	"context"
	"testing"
	"time"
)

func TestLeaseAcquire(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	leaseName := "leader"
	holder1 := "node1"
	holder2 := "node2"
	ttl := 1 * time.Second

	// 1. Acquire new lease
	acquired, err := store.Acquire(ctx, leaseName, holder1, ttl)
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}
	if !acquired {
		t.Errorf("expected to acquire new lease")
	}

	// Verify state
	l, err := store.Get(ctx, leaseName)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if l.HolderID != holder1 {
		t.Errorf("expected holder %s, got %s", holder1, l.HolderID)
	}
	if l.Epoch != 1 {
		t.Errorf("expected epoch 1, got %d", l.Epoch)
	}

	// 2. Renew by same holder
	acquired, err = store.Acquire(ctx, leaseName, holder1, ttl)
	if err != nil {
		t.Fatalf("Acquire (renew) failed: %v", err)
	}
	if !acquired {
		t.Errorf("expected to renew lease")
	}

	// Verify epoch same, version inc
	l2, _ := store.Get(ctx, leaseName)
	if l2.Epoch != 1 {
		t.Errorf("expected epoch 1 after renew, got %d", l2.Epoch)
	}
	if l2.Version <= l.Version {
		t.Errorf("expected version increase, got %d -> %d", l.Version, l2.Version)
	}

	// 3. Fail takeover by other holder (lease valid)
	acquired, err = store.Acquire(ctx, leaseName, holder2, ttl)
	if err != nil {
		t.Fatalf("Acquire (steal) failed: %v", err)
	}
	if acquired {
		t.Errorf("should not acquire valid lease held by other")
	}

	// 4. Takeover expired lease
	// Manually expire it
	store.db.Exec("UPDATE leases SET expires_at = ?", time.Now().UTC().Add(-1*time.Minute))

	acquired, err = store.Acquire(ctx, leaseName, holder2, ttl)
	if err != nil {
		t.Fatalf("Acquire (takeover) failed: %v", err)
	}
	if !acquired {
		t.Errorf("expected to takeover expired lease")
	}

	// Verify holder and epoch inc
	l3, _ := store.Get(ctx, leaseName)
	if l3.HolderID != holder2 {
		t.Errorf("expected holder %s, got %s", holder2, l3.HolderID)
	}
	if l3.Epoch != 2 {
		t.Errorf("expected epoch 2, got %d", l3.Epoch)
	}
}

func TestLeaseRenew(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	leaseName := "worker"
	holder := "w1"
	ttl := 1 * time.Second

	// Setup lease
	store.Acquire(ctx, leaseName, holder, ttl)

	// 1. Successful Renew
	if err := store.Renew(ctx, leaseName, holder, ttl); err != nil {
		t.Fatalf("Renew failed: %v", err)
	}

	// 2. Fail Renew (lost/stolen)
	// Another holder takes it (simulate force via DB or expire+takeover)
	store.db.Exec("UPDATE leases SET holder_id = 'w2' WHERE name = ?", leaseName)

	if err := store.Renew(ctx, leaseName, holder, ttl); err == nil {
		t.Errorf("expected error renewing stolen lease, got nil")
	}
}

func TestLeaseRelease(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	leaseName := "lock"
	holder := "h1"

	store.Acquire(ctx, leaseName, holder, 1*time.Second)

	// 1. Release
	if err := store.Release(ctx, leaseName, holder); err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	l, _ := store.Get(ctx, leaseName)
	if l != nil {
		t.Errorf("expected lease to be gone, got %v", l)
	}

	// 2. Release non-existent/not-held (should be no-op/success)
	if err := store.Release(ctx, leaseName, holder); err != nil {
		t.Fatalf("Release (idempotent) failed: %v", err)
	}
}

func TestLeaseGet(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	// 1. Get non-existent
	l, err := store.Get(context.Background(), "missing")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if l != nil {
		t.Errorf("expected nil, got %v", l)
	}
}
