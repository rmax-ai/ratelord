# CLI Guide

The Ratelord command-line interface (CLI) is your primary tool for managing the daemon, registering identities, and monitoring system status.

## Installation

Ensure `ratelord` is in your `$PATH`. If you built from source:

```bash
go install ./cmd/ratelord
go install ./cmd/ratelord-tui
```

## Running the Daemon

The daemon `ratelord-d` is the core component.

```bash
ratelord-d [flags]
```

**Common Flags:**
*   `--mode`: Operating mode. Defaults to `leader`. Set to `follower` for federation.
*   `--port`: HTTP port (default `8090`).

## Managing Identities

Ratelord requires explicit registration of identities (API keys, tokens, users) before they can be used in intents.

### Registering an Identity

Use the `identity add` command to register a new actor. If no token is provided, one will be generated for you.

```bash
ratelord identity add <identity_id> [flags]
```

**Arguments:**
*   `<identity_id>`: A unique string identifier (e.g., `pat:github:my-token`, `oauth:google:user1`).

**Flags:**
*   `--scope`: The scope this identity belongs to (e.g., `org:acme`).

**Example:**
```bash
# Register and auto-generate a token
ratelord identity add pat:bot-01 --scope org:engineering

# Register with an existing known token (if supported)
ratelord identity add pat:bot-02 --scope org:engineering --token "existing-secret"
```

This event is recorded in the ledger, and the identity becomes immediately available for policy evaluation.

## MCP Integration

Ratelord supports the Model Context Protocol (MCP), allowing AI assistants to directly interact with the daemon.

```bash
ratelord mcp
```

This starts an MCP server over stdio that exposes Ratelord's capabilities (reading status, analyzing trends) to compatible LLM clients.

## Monitoring with the TUI

The Terminal User Interface (TUI) provides a real-time, low-latency dashboard for operators.

### Launching the Dashboard

Run the TUI from any terminal:

```bash
ratelord-tui
```

*Note: The TUI connects to the daemon at `http://localhost:8090` by default.*

### Key Features
*   **Live Stream**: Watch requests and decisions stream in real-time.
*   **Usage Bars**: Visual gauges for critical constraint pools (e.g., GitHub API remaining).
*   **Status Indicators**: Immediate feedback on daemon health and policy reload status.

## Troubleshooting

### Daemon Connection Failed
If the CLI or TUI cannot connect:
1.  Ensure `ratelord-d` is running.
2.  Check the port (default `8090`).
3.  Verify `RATELORD_PORT` environment variable matches.

### Identity Not Found
If an agent receives "Identity not found" errors:
1.  Run `ratelord identity list` (if available) or check the TUI to see registered identities.
2.  Ensure the `IdentityID` in the SDK `Intent` matches exactly what was registered via CLI.
