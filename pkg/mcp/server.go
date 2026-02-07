package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rmax-ai/ratelord/pkg/client"
)

// Server adapts ratelord-d to the Model Context Protocol.
type Server struct {
	mcpServer *server.MCPServer
	apiClient *client.Client
}

// NewServer creates a new MCP server instance.
func NewServer(apiURL string) *Server {
	s := &Server{
		mcpServer: server.NewMCPServer(
			"ratelord",
			"1.0.0",
		),
		apiClient: client.NewClient(apiURL),
	}
	s.registerResources()
	s.registerTools()
	s.registerPrompts()
	return s
}

// Serve starts the MCP server on stdio.
func (s *Server) Serve() error {
	return server.ServeStdio(s.mcpServer)
}

// --- Resources ---

func (s *Server) registerResources() {
	// ratelord://events
	s.mcpServer.AddResource(mcp.NewResource(
		"ratelord://events",
		"Ratelord Event Log",
		mcp.WithResourceDescription("Recent system events showing usage, limits, and decisions"),
		mcp.WithMIMEType("application/json"),
	), s.handleReadEvents)

	// ratelord://usage
	s.mcpServer.AddResource(mcp.NewResource(
		"ratelord://usage",
		"Current Usage Trends",
		mcp.WithResourceDescription("Aggregated usage statistics for the last hour"),
		mcp.WithMIMEType("application/json"),
	), s.handleReadUsage)
}

// --- Tools ---

func (s *Server) registerTools() {
	// ask_intent
	s.mcpServer.AddTool(mcp.NewTool(
		"ask_intent",
		mcp.WithDescription("Negotiate permission to perform an action. Returns Approved/Denied."),
		mcp.WithString("identity_id", mcp.Required(), mcp.Description("The identity performing the action")),
		mcp.WithString("action", mcp.Required(), mcp.Description("The type of action/workload (e.g., 'repo_scan')")),
		mcp.WithString("scope_id", mcp.Required(), mcp.Description("The target scope (e.g., 'repo:owner/name')")),
		mcp.WithNumber("cost", mcp.Description("Expected cost (default 1.0)")),
	), s.handleAskIntent)
}

// --- Prompts ---

func (s *Server) registerPrompts() {
	s.mcpServer.AddPrompt(mcp.NewPrompt(
		"ratelord-aware",
		mcp.WithPromptDescription("Provides context about Ratelord concepts (Pools, Identities, Limits)"),
	), s.handleGetPrompt)
}

// --- Handlers ---

func (s *Server) handleReadEvents(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	events, err := s.apiClient.GetEvents(ctx, 50)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch events: %w", err)
	}

	data, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal events: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(data),
		},
	}, nil
}

func (s *Server) handleReadUsage(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Default to last hour
	opts := client.TrendsOptions{
		Bucket: "hour",
		From:   time.Now().Add(-1 * time.Hour),
		To:     time.Now(),
	}

	stats, err := s.apiClient.GetTrends(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch trends: %w", err)
	}

	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal trends: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(data),
		},
	}, nil
}

func (s *Server) handleAskIntent(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	identityID := mcp.ParseString(request, "identity_id", "")
	action := mcp.ParseString(request, "action", "")
	scopeID := mcp.ParseString(request, "scope_id", "")
	cost := mcp.ParseFloat64(request, "cost", 1.0)

	// MCP agent ID
	agentID := "mcp-agent"

	intent := client.Intent{
		AgentID:      agentID,
		IdentityID:   identityID,
		WorkloadID:   action,
		ScopeID:      scopeID,
		ExpectedCost: cost,
	}

	decision, err := s.apiClient.Ask(ctx, intent)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("API error: %v", err)), nil
	}

	// Format result
	resultMsg := fmt.Sprintf("Decision: %s\nReason: %s", decision.Status, decision.Reason)
	return mcp.NewToolResultText(resultMsg), nil
}

func (s *Server) handleGetPrompt(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	name := request.Params.Name
	if name != "ratelord-aware" {
		return nil, fmt.Errorf("prompt not found: %s", name)
	}

	promptText := `You are interacting with Ratelord, a local-first constraint control plane.
    
Concepts:
- Identity: The actor performing an action (e.g., 'ci-bot', 'user-1').
- Scope: The target of the action (e.g., 'repo:owner/name').
- Action/Workload: The type of operation (e.g., 'clone', 'scan').
- Pool: A specific rate limit bucket (e.g., 'github-core', 'openai-rpm').
- Intent: A request to perform an action. You must Ask for an intent before acting.

When the user asks to perform an action that might be rate-limited, use the 'ask_intent' tool.
If the intent is DENIED, you must respect the decision and wait or abort.
`

	return mcp.NewGetPromptResult(
		"ratelord-aware",
		[]mcp.PromptMessage{
			mcp.NewPromptMessage(mcp.RoleUser, mcp.NewTextContent(promptText)),
		},
	), nil
}
