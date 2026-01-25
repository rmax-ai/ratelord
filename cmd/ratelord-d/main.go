package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// M1.3: Emit system_started log on boot (structured)
	fmt.Println(`{"level":"info","msg":"system_started","component":"ratelord-d"}`)

	// M1.2: Handle SIGINT/SIGTERM for graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received
	sig := <-sigs
	fmt.Printf(`{"level":"info","msg":"shutdown_initiated","signal":"%s"}`+"\n", sig)

	// TODO: Clean up resources (DB, listeners)

	fmt.Println(`{"level":"info","msg":"shutdown_complete"}`)
}
