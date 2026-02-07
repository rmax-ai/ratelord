# Model Context Protocol (MCP) Integration

Ratelord implements the **Model Context Protocol (MCP)**, allowing LLMs (like Claude, Gemini, and others) to natively discover, query, and respect rate limits and constraints without custom API integration logic.

This integration transforms rate limits from "hidden errors" into **visible context** that the model can reason about before generating code or taking action.

## Starting the MCP Server

The MCP server is built directly into the Ratelord CLI. To start it, run:

```bash
ratelord mcp
```

This will start the MCP server on stdio, which is the standard transport for local MCP integrations. You can configure this command in your MCP client (e.g., Claude Desktop, Zed, VS Code).

### Configuration Example (Claude Desktop)

Add the following to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "ratelord": {
      "command": "ratelord",
      "args": ["mcp"]
    }
  }
}
```

## Resources

Ratelord exposes the following resources to the LLM, giving it direct visibility into the system state:

### `ratelord://events`
**Description**: Recent system events.
**Purpose**: Allows the LLM to see what just happened—intent approvals, denials, policy triggers, or usage spikes. This is critical for "self-correction" if an action was denied.

### `ratelord://usage`
**Description**: Current usage trends and burn rates.
**Purpose**: Provides the model with a high-level view of consumption. It can see which pools are near exhaustion and which are healthy, allowing it to plan complex tasks accordingly.

### `ratelord://config`
**Description**: Active policy configuration.
**Purpose**: Exposes the rules of the road. The LLM can read the active policies (e.g., "Warn if budget < 10%") and understand *why* certain constraints exist.

## Tools

The MCP integration provides tools that allow the LLM to actively interact with the Ratelord daemon:

### `ask_intent`
**Description**: Negotiate permission for an action.
**Purpose**: Before performing a high-cost operation (like "Scan 100 repositories"), the LLM uses this tool to ask Ratelord for permission.
**Parameters**:
- `agent_id`: Who is asking?
- `identity_id`: Which credential will be used?
- `pool_id`: What resource is being consumed?
- `amount`: How much?

**Response**:
- `Approved`: Proceed.
- `Denied`: Stop. The response includes the reason (e.g., "Projected exhaustion in 5m").
- `Modified`: Proceed, but with changes (e.g., "Wait 2s first").

### `check_usage`
**Description**: Query specific usage status.
**Purpose**: Allows the LLM to "look before it leaps" by checking the specific status of a pool or identity without submitting a formal intent.

## Why use MCP?

By using MCP, you shift the burden of rate limit logic from **hard-coded retry loops** to **semantic reasoning**:

1.  **Natural Compliance**: The LLM "sees" the limit just like it sees a file. It naturally avoids actions that violate the visible policy.
2.  **Negotiation**: Instead of just failing, the LLM can ask, get a denial with a reason ("Risk too high"), and then *decide* to change its strategy (e.g., "I will scan only 10 repos instead").
3.  **Zero-Code Integration**: You don't need to write a custom Python/JS wrapper for every new agent. If the agent speaks MCP, it works with Ratelord out of the box.
