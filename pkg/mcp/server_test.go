package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestMCPServer_ReadUsage(t *testing.T) {
	// 1. Mock API Server
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/trends" {
			w.Header().Set("Content-Type", "application/json")
			// Return a dummy list of UsageStat
			w.Write([]byte(`[{"provider_id": "test", "pool_id": "pool1", "used": 100, "remaining": 900}]`))
			return
		}
		http.NotFound(w, r)
	})
	ts := httptest.NewServer(apiHandler)
	defer ts.Close()

	// 2. Create MCP Server
	s := NewServer(ts.URL)

	// 3. Test Handler directly
	req := mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: "ratelord://usage",
		},
	}

	result, err := s.handleReadUsage(context.Background(), req)
	if err != nil {
		t.Fatalf("handleReadUsage failed: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 resource content, got %d", len(result))
	}

	content, ok := result[0].(mcp.TextResourceContents)
	if !ok {
		t.Fatalf("Expected TextResourceContents")
	}

	if content.MIMEType != "application/json" {
		t.Errorf("Expected application/json, got %s", content.MIMEType)
	}

	// Basic content check
	var stats []map[string]interface{}
	if err := json.Unmarshal([]byte(content.Text), &stats); err != nil {
		t.Errorf("Failed to parse result JSON: %v", err)
	}
	if len(stats) != 1 {
		t.Errorf("Expected 1 stat item")
	}
}

func TestMCPServer_AskIntent(t *testing.T) {
	// 1. Mock API Server
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/intent" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"decision": "approve", "reason": "ok"}`))
			return
		}
		http.NotFound(w, r)
	})
	ts := httptest.NewServer(apiHandler)
	defer ts.Close()

	// 2. Create MCP Server
	s := NewServer(ts.URL)

	// 3. Test Handler directly
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "ask_intent",
			Arguments: map[string]interface{}{
				"identity_id": "user1",
				"action":      "scan",
				"scope_id":    "repo:foo/bar",
			},
		},
	}

	result, err := s.handleAskIntent(context.Background(), req)
	if err != nil {
		t.Fatalf("handleAskIntent failed: %v", err)
	}

	if result.IsError {
		t.Errorf("Expected success, got error")
	}

	// The result content is a list of Content objects
	if len(result.Content) == 0 {
		t.Errorf("Expected content in result")
	} else {
		text, ok := result.Content[0].(mcp.TextContent)
		if ok {
			if text.Text == "" {
				t.Error("Expected text content")
			}
		}
	}
}
