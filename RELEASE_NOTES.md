# Release Notes: ratelord v1.0.0

## Overview

ratelord v1.0.0 marks the initial stable release of the local-first, daemon-authoritative rate limit orchestrator. This release establishes the core event-sourcing architecture, policy engine, and predictive capabilities designed to manage API consumption quotas across distributed agents.

## Key Features

### Core Infrastructure
- **Daemon-Authority Model**: `ratelord-d` acts as the single source of truth for rate limit state.
- **Event Sourcing**: Immutable SQLite-based event log for auditability and replayable state.
- **Resumability**: Robust crash recovery and state reconstruction from the event stream.

### Usage & Policy Management
- **Hierarchical Tracking**: Tracks usage across identities, scopes, and shared/isolated pools.
- **Policy Engine**: Dynamic rule evaluation for Approving, Denying, or Shaping traffic.
- **Hot Reloading**: Support for `SIGHUP` to reload policy configurations without downtime.
- **Drift Detection**: Automatic detection and correction of usage drift against external providers.

### Prediction & Forecasting
- **Time-to-Exhaustion**: Forecasts P50/P90/P99 exhaustion times based on usage history.
- **Linear Burn Model**: Initial forecasting model for linear consumption patterns.

### Observability
- **TUI Dashboard**: Terminal User Interface for real-time monitoring of usage, events, and forecasts.
- **Structured Logging**: JSON-formatted logs for easy integration with observability tools.

## Known Limitations
- **TUI**: Currently read-only; administrative actions must be performed via CLI or config files.
- **Providers**: Only Mock Provider is fully implemented; external providers (GitHub, OpenAI) are planned for future releases.

## Getting Started
1. Build the daemon: `make build`
2. Run the daemon: `./bin/ratelord-d`
3. Register an identity: `./bin/ratelord identity add <name> <kind>`
