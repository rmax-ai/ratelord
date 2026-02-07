package engine

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/rmax-ai/ratelord/pkg/store"
)

type PruneWorker struct {
	store  *store.Store
	config *RetentionConfig
	mu     sync.RWMutex
}

func NewPruneWorker(st *store.Store, cfg *RetentionConfig) *PruneWorker {
	return &PruneWorker{
		store:  st,
		config: cfg,
	}
}

func (w *PruneWorker) UpdateConfig(cfg *RetentionConfig) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.config = cfg
}

func (w *PruneWorker) Run(ctx context.Context) {
	w.mu.RLock()
	disabled := w.config == nil || !w.config.Enabled
	interval := 1 * time.Hour
	if !disabled && w.config.CheckInterval != "" {
		if d, err := time.ParseDuration(w.config.CheckInterval); err == nil {
			interval = d
		}
	}
	w.mu.RUnlock()

	if disabled {
		log.Println("Pruning disabled")
		return
	}
	// ...

	log.Printf("Starting prune worker (interval: %v)", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial run
	w.Prune(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("Prune worker stopping")
			return
		case <-ticker.C:
			w.Prune(ctx)
		}
	}
}

func (w *PruneWorker) Prune(ctx context.Context) {
	w.mu.RLock()
	cfg := w.config
	w.mu.RUnlock()

	if cfg == nil || !cfg.Enabled {
		return
	}

	// Identify exclusions (types handled by specific rules)
	var exclusions []string
	for t := range cfg.ByType {
		exclusions = append(exclusions, t)
	}

	// 1. Prune Default (excluding specifics)
	if cfg.DefaultTTL != "" {
		ttl, err := time.ParseDuration(cfg.DefaultTTL)
		if err == nil {
			deleted, err := w.store.PruneEvents(ctx, ttl, "", exclusions)
			if err != nil {
				// Don't log "no snapshots" error as error on first run, it's expected
				if err.Error() != "cannot prune: no snapshots found (create a snapshot first)" {
					log.Printf("Prune error (default): %v", err)
				}
			} else if deleted > 0 {
				log.Printf("Pruned %d events (default policy > %v)", deleted, ttl)
			}
		}
	}

	// 2. Prune Specifics
	for eventType, ttlStr := range cfg.ByType {
		ttl, err := time.ParseDuration(ttlStr)
		if err != nil {
			continue
		}
		deleted, err := w.store.PruneEvents(ctx, ttl, eventType, nil)
		if err != nil {
			if err.Error() != "cannot prune: no snapshots found (create a snapshot first)" {
				log.Printf("Prune error (%s): %v", eventType, err)
			}
		} else if deleted > 0 {
			log.Printf("Pruned %d %s events older than %v", deleted, eventType, ttl)
		}
	}
}
