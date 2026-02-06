package engine

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/rmax-ai/ratelord/pkg/store"
)

// ElectionManager manages distributed leadership election using a lease store.
type ElectionManager struct {
	store     store.LeaseStore
	holderID  string
	leaseName string
	ttl       time.Duration

	onPromote func()
	onDemote  func()

	isLeader bool
	mu       sync.RWMutex

	ticker *time.Ticker
	stopCh chan struct{}
}

// NewElectionManager creates a new ElectionManager instance.
func NewElectionManager(
	store store.LeaseStore,
	holderID string,
	leaseName string,
	ttl time.Duration,
	onPromote func(),
	onDemote func(),
) *ElectionManager {
	return &ElectionManager{
		store:     store,
		holderID:  holderID,
		leaseName: leaseName,
		ttl:       ttl,
		onPromote: onPromote,
		onDemote:  onDemote,
		stopCh:    make(chan struct{}),
	}
}

// Start begins the background election loop.
func (em *ElectionManager) Start(ctx context.Context) {
	em.ticker = time.NewTicker(em.ttl / 2)
	go func() {
		defer em.ticker.Stop()
		for {
			select {
			case <-em.ticker.C:
				em.attemptElection(ctx)
			case <-em.stopCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
	slog.Info("ElectionManager started", "holderID", em.holderID, "leaseName", em.leaseName)
}

// Stop stops the election loop and releases the lease if currently leader.
func (em *ElectionManager) Stop(ctx context.Context) {
	close(em.stopCh)
	em.mu.Lock()
	wasLeader := em.isLeader
	em.mu.Unlock()
	if wasLeader {
		if err := em.store.Release(ctx, em.leaseName, em.holderID); err != nil {
			slog.Error("Failed to release lease on stop", "error", err, "holderID", em.holderID, "leaseName", em.leaseName)
		} else {
			slog.Info("Lease released on stop", "holderID", em.holderID, "leaseName", em.leaseName)
		}
	}
	slog.Info("ElectionManager stopped", "holderID", em.holderID, "leaseName", em.leaseName)
}

// IsLeader returns true if this instance is currently the leader.
func (em *ElectionManager) IsLeader() bool {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.isLeader
}

// attemptElection performs the election logic.
func (em *ElectionManager) attemptElection(ctx context.Context) {
	em.mu.Lock()
	wasLeader := em.isLeader
	em.mu.Unlock()

	var newLeader bool
	var err error

	if wasLeader {
		// Try to renew
		err = em.store.Renew(ctx, em.leaseName, em.holderID, em.ttl)
		if err != nil {
			slog.Warn("Failed to renew lease", "error", err, "holderID", em.holderID, "leaseName", em.leaseName)
			newLeader = false
		} else {
			newLeader = true
			slog.Debug("Lease renewed", "holderID", em.holderID, "leaseName", em.leaseName)
		}
	} else {
		// Try to acquire
		newLeader, err = em.store.Acquire(ctx, em.leaseName, em.holderID, em.ttl)
		if err != nil {
			slog.Warn("Failed to acquire lease", "error", err, "holderID", em.holderID, "leaseName", em.leaseName)
			newLeader = false
		} else if newLeader {
			slog.Info("Lease acquired", "holderID", em.holderID, "leaseName", em.leaseName)
		} else {
			slog.Debug("Lease not acquired", "holderID", em.holderID, "leaseName", em.leaseName)
		}
	}

	em.mu.Lock()
	em.isLeader = newLeader
	em.mu.Unlock()

	// Call callbacks on transition
	if !wasLeader && newLeader {
		if em.onPromote != nil {
			em.onPromote()
		}
		slog.Info("Promoted to leader", "holderID", em.holderID, "leaseName", em.leaseName)
	} else if wasLeader && !newLeader {
		if em.onDemote != nil {
			em.onDemote()
		}
		slog.Info("Demoted from leader", "holderID", em.holderID, "leaseName", em.leaseName)
	}
}
