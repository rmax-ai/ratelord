package e2e_test

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/rmax-ai/ratelord/pkg/client"
	"github.com/stretchr/testify/assert"
)

func TestEndToEnd(t *testing.T) {
	if os.Getenv("E2E") != "true" {
		t.Skip("Skipping e2e test")
	}

	endpoint := os.Getenv("RATELORD_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:8090"
	}

	c := client.NewClient(endpoint)

	// Poll Ping until success
	var err error
	for i := 0; i < 30; i++ {
		_, err = c.Ping(context.Background())
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		t.Fatal("Failed to ping server after 30 seconds")
	}

	// Call Ask with sample Intent
	intent := client.Intent{
		AgentID:    "test-agent",
		IdentityID: "e2e-user",
		WorkloadID: "test-workload",
		ScopeID:    "test-scope",
	}
	decision, err := c.Ask(context.Background(), intent)
	assert.NoError(t, err)
	assert.True(t, decision.Allowed)

	// Call GetEvents and verify the Intent event is there
	events, err := c.GetEvents(context.Background(), 10)
	assert.NoError(t, err)
	assert.Greater(t, len(events), 0, "Expected at least one event")

	// Call GetGraph and verify it returns nodes
	graph, err := c.GetGraph(context.Background())
	assert.NoError(t, err)
	assert.Greater(t, len(graph.Nodes), 0, "Expected graph to have nodes")

	// Check Web UI is serving
	resp, err := http.Get(endpoint + "/")
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
}
