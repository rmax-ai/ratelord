package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log.Println("Starting ratelord-d")

	// TODO: Initialize SQLite DB, create tables, log system_started event

	// Signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	log.Printf("Received signal: %v. Shutting down gracefully", sig)

	// TODO: Stop accepting new intents, flush WAL

	log.Println("Shutdown complete")
	os.Exit(0)
}
