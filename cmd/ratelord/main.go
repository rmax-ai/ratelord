package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

// IdentityRegistration matches the payload for POST /v1/identities

// IdentityRegistration matches the payload for POST /v1/identities
type IdentityRegistration struct {
	IdentityID string                 `json:"identity_id"`
	Kind       string                 `json:"kind"`
	Metadata   map[string]interface{} `json:"metadata"`
}

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: ratelord identity add <name> <kind>")
		os.Exit(1)
	}

	cmd := os.Args[1]
	subCmd := os.Args[2]
	name := os.Args[3]

	if cmd != "identity" || subCmd != "add" {
		fmt.Println("Usage: ratelord identity add <name> <kind>")
		os.Exit(1)
	}

	kind := "user"
	if len(os.Args) > 4 {
		kind = os.Args[4]
	}

	payload := IdentityRegistration{
		IdentityID: name,
		Kind:       kind,
		Metadata: map[string]interface{}{
			"source": "cli",
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("Error encoding request: %v\n", err)
		os.Exit(1)
	}

	resp, err := http.Post("http://127.0.0.1:8090/v1/identities", "application/json", bytes.NewBuffer(data))
	if err != nil {
		fmt.Printf("Error contacting daemon: %v\n", err)
		fmt.Println("Is ratelord-d running?")
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		os.Exit(1)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error: Server returned %s\n%s\n", resp.Status, string(body))
		os.Exit(1)
	}

	fmt.Println(string(body))
}
