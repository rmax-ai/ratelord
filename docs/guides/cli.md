# CLI Guide

The Ratelord command-line interface (CLI) is your primary tool for managing the daemon, registering identities, and monitoring system status.

## Installation

Ensure `ratelord` is in your `$PATH`. If you built from source:

```bash
go install ./cmd/ratelord
go install ./cmd/ratelord-tui
```

## Managing Identities

Ratelord requires explicit registration of identities (API keys, tokens, users) before they can be used in intents.

### Registering an Identity

Use the `identity add` command to register a new actor.

```bash
ratelord identity add <identity_id> [flags]
```

**Arguments:**
*   `<identity_id>`: A unique string identifier (e.g., `pat:github:my-token`, `oauth:google:user1`).

**Flags:**
*   `--scope`: The scope this identity belongs to (e.g., `org:acme`).

**Example:**
```bash
ratelord identity add pat:bot-01 --scope org:engineering
```

This event is recorded in the ledger, and the identity becomes immediately available for policy evaluation.

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
