package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

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

	// M1.2: Handle SIGINT/SIGTERM for graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received
	sig := <-sigs
	fmt.Printf(`{"level":"info","msg":"shutdown_initiated","signal":"%s"}`+"\n", sig)

	// Cleanup
	if err := st.Close(); err != nil {
		fmt.Printf(`{"level":"error","msg":"failed_to_close_store","error":"%v"}`+"\n", err)
	} else {
		fmt.Println(`{"level":"info","msg":"store_closed"}`)
	}

	fmt.Println(`{"level":"info","msg":"shutdown_complete"}`)
}
