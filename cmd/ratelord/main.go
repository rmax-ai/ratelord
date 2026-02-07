package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/rmax-ai/ratelord/pkg/mcp"
)

// IdentityRegistration matches the payload for POST /v1/identities

// IdentityRegistration matches the payload for POST /v1/identities
type IdentityRegistration struct {
	IdentityID string                 `json:"identity_id"`
	Kind       string                 `json:"kind"`
	Metadata   map[string]interface{} `json:"metadata"`
	Token      string                 `json:"token,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "identity":
		handleIdentity(os.Args[2:])
	case "admin":
		handleAdmin(os.Args[2:])
	case "mcp":
		handleMCP(os.Args[2:])
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  ratelord identity add <name> <kind> [token]  Register a new identity")
	fmt.Println("  ratelord identity delete <id>               Delete an identity")
	fmt.Println("  ratelord admin prune <retention>             Prune old events (e.g. 720h)")
	fmt.Println("  ratelord mcp [--url <url>]                   Run MCP server (stdio)")
}

func handleMCP(args []string) {
	apiURL := "http://127.0.0.1:8090"
	for i, arg := range args {
		if arg == "--url" && i+1 < len(args) {
			apiURL = args[i+1]
		}
	}

	srv := mcp.NewServer(apiURL)
	// Log to stderr because stdout is used for MCP protocol
	fmt.Fprintf(os.Stderr, "Starting MCP server connected to %s\n", apiURL)
	if err := srv.Serve(); err != nil {
		fmt.Fprintf(os.Stderr, "MCP Server Error: %v\n", err)
		os.Exit(1)
	}
}

func handleAdmin(args []string) {
	if len(args) < 2 || args[0] != "prune" {
		fmt.Println("Usage: ratelord admin prune <retention>")
		os.Exit(1)
	}
	retention := args[1]

	token := os.Getenv("RATELORD_ADMIN_TOKEN")
	if token == "" {
		fmt.Println("Error: RATELORD_ADMIN_TOKEN env var required")
		os.Exit(1)
	}

	payload := map[string]string{"retention": retention}
	data, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "http://127.0.0.1:8090/v1/admin/prune", bytes.NewBuffer(data))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error contacting daemon: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error: %s\n%s\n", resp.Status, string(body))
		os.Exit(1)
	}

	fmt.Println(string(body))
}

func handleIdentity(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: ratelord identity add <name> <kind> [token] | delete <id>")
		os.Exit(1)
	}

	subcmd := args[0]
	switch subcmd {
	case "add":
		if len(args) < 2 {
			fmt.Println("Usage: ratelord identity add <name> <kind> [token]")
			os.Exit(1)
		}
		name := args[1]

		kind := "user"
		if len(args) > 2 {
			kind = args[2]
		}

		token := os.Getenv("RATELORD_NEW_TOKEN")

		payload := IdentityRegistration{
			IdentityID: name,
			Kind:       kind,
			Metadata: map[string]interface{}{
				"source": "cli",
			},
			Token: token,
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

		var response struct {
			IdentityID string `json:"identity_id"`
			Status     string `json:"status"`
			EventID    string `json:"event_id"`
			Token      string `json:"token,omitempty"`
		}

		if err := json.Unmarshal(body, &response); err != nil {
			fmt.Println(string(body)) // Fallback to raw output
			return
		}

		fmt.Printf("Identity Registered: %s\n", response.IdentityID)
		if response.Token != "" {
			fmt.Printf("Token: %s\n", response.Token)
			fmt.Println("WARNING: Save this token! It will not be shown again.")
		}
	case "delete":
		if len(args) < 2 {
			fmt.Println("Usage: ratelord identity delete <id>")
			os.Exit(1)
		}
		id := args[1]

		token := os.Getenv("RATELORD_ADMIN_TOKEN")
		if token == "" {
			fmt.Println("Error: RATELORD_ADMIN_TOKEN env var required")
			os.Exit(1)
		}

		req, err := http.NewRequest("DELETE", "http://127.0.0.1:8090/v1/identities/"+id, nil)
		if err != nil {
			fmt.Printf("Error creating request: %v\n", err)
			os.Exit(1)
		}
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Printf("Error contacting daemon: %v\n", err)
			fmt.Println("Is ratelord-d running?")
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNoContent {
			fmt.Printf("Identity deleted: %s\n", id)
		} else {
			body, _ := io.ReadAll(resp.Body)
			fmt.Printf("Error: Server returned %s\n%s\n", resp.Status, string(body))
			os.Exit(1)
		}
	default:
		fmt.Println("Usage: ratelord identity add <name> <kind> [token] | delete <id>")
		os.Exit(1)
	}
}
