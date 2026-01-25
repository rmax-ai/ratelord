package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/rmax/ratelord/pkg/api"
	"github.com/rmax/ratelord/pkg/engine"
	"github.com/rmax/ratelord/pkg/store"
)

func main() {
	// M1.3: Emit system_started log on boot (structured)
	fmt.Println(`{"level":"info","msg":"system_started","component":"ratelord-d"}`)

	// Configuration (M1.4 placeholder: hardcoded DB path for now)
	cwd, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("failed to get cwd: %v", err))
	}
	dbPath := filepath.Join(cwd, "ratelord.db")

	// M2.1: Initialize SQLite Store
	st, err := store.NewStore(dbPath)
	if err != nil {
		fmt.Printf(`{"level":"fatal","msg":"failed_to_init_store","error":"%v"}`+"\n", err)
		os.Exit(1)
	}
	fmt.Printf(`{"level":"info","msg":"store_initialized","path":"%s"}`+"\n", dbPath)

	// M4.2: Initialize Identity Projection
	identityProj := engine.NewIdentityProjection()

	// M5.1: Initialize Usage Projection
	usageProj := engine.NewUsageProjection()

	// Replay events to build projection
	// NOTE: This blocks startup, but safe for small event logs
	events, err := st.ReadEvents(context.Background(), time.Time{}, 10000) // arbitrary large limit, from beginning
	if err == nil {
		// Replay identity events
		if err := identityProj.Replay(events); err != nil {
			fmt.Printf(`{"level":"error","msg":"failed_to_replay_identity_events","error":"%v"}`+"\n", err)
		} else {
			fmt.Printf(`{"level":"info","msg":"identity_projection_replayed","events_count":%d}`+"\n", len(events))
		}
		// Replay usage events
		if err := usageProj.Replay(events); err != nil {
			fmt.Printf(`{"level":"error","msg":"failed_to_replay_usage_events","error":"%v"}`+"\n", err)
		} else {
			fmt.Printf(`{"level":"info","msg":"usage_projection_replayed","events_count":%d}`+"\n", len(events))
		}
	} else {
		fmt.Printf(`{"level":"error","msg":"failed_to_read_events","error":"%v"}`+"\n", err)
	}

	// M5.2: Initialize Policy Engine
	policyEngine := engine.NewPolicyEngine(usageProj)

	// M3.1: Start HTTP Server (in background)
	srv := api.NewServer(st, identityProj, usageProj, policyEngine)
	go func() {
		if err := srv.Start(); err != nil {
			fmt.Printf(`{"level":"error","msg":"server_error","error":"%v"}`+"\n", err)
		}
	}()

	// M1.2: Handle SIGINT/SIGTERM for graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received
	sig := <-sigs
	fmt.Printf(`{"level":"info","msg":"shutdown_initiated","signal":"%s"}`+"\n", sig)

	// Shutdown Server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Stop(ctx); err != nil {
		fmt.Printf(`{"level":"error","msg":"server_shutdown_error","error":"%v"}`+"\n", err)
	} else {
		fmt.Println(`{"level":"info","msg":"server_stopped"}`)
	}

	// Cleanup
	if err := st.Close(); err != nil {
		fmt.Printf(`{"level":"error","msg":"failed_to_close_store","error":"%v"}`+"\n", err)
	} else {
		fmt.Println(`{"level":"info","msg":"store_closed"}`)
	}

	fmt.Println(`{"level":"info","msg":"shutdown_complete"}`)
}
