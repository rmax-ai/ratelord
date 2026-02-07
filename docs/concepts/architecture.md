# Architecture: How Ratelord Works

Ratelord is a **local-first, predictive rate limiter** designed for AI agents and automated workflows. Unlike traditional rate limiters that just count requests, Ratelord acts as a central "daemon" that forecasts exhaustion and negotiates intents.

## The Core Concept: Daemon Authority

The heart of Ratelord is `ratelord-d`, a daemon that runs locally on your machine.

- **Single Source of Truth**: The daemon is the only component allowed to write state or make decisions.
- **Read-Only Clients**: Your tools (CLI, TUI, Web UI) are just viewers. They display what the daemon knows but cannot change it directly.
- **Intent-Based**: Agents don't just "grab" tokens; they submit an **Intent** ("I want to scan this repo"). The daemon analyzes the risk and approves or denies it.

## Event Sourcing: The Memory

Ratelord doesn't just store "current counts." It uses **Event Sourcing** to record everything that happens as an immutable sequence of events.

1. **The Event Log**: Every provider observation, usage tick, and decision is appended to a log.
2. **Replayability**: The system can always rebuild its current state by replaying the log. This ensures auditability ("Why was I blocked yesterday?") and resilience.
3. **Projections**: For speed, the daemon maintains "projections" (like a current status table) derived from the log.

## Prediction: The Brain

Ratelord operates in the **Time Domain**, not just the Count Domain.

- **Forecasting**: Instead of saying "450/500 requests used," it calculates **Time-to-Exhaustion (TTE)**.
  - *Example: "At current burn rate, you will hit the limit in 12 minutes."*
- **Risk Assessment**: It calculates the probability of hitting a limit before the next reset window.
- **Burn Rates**: It tracks how fast you are consuming resources to model future usage.

## Intent Negotiation: The Workflow

This is the primary way your code interacts with Ratelord.

1. **Ask**: Your agent sends a `POST /v1/intent` request describing the action (Identity, Scope, Workload).
2. **Evaluate**: The daemon checks the Constraint Graph, current budget, and forecasts.
3. **Decide**: The daemon responds with:
   - `approve`: Go ahead.
   - `deny_with_reason`: Stop. (e.g., "Risk too high," "Budget exhausted").
   - `approve_with_modifications`: Go ahead, but slow down (e.g., "Wait 2s first").

## Components

### `ratelord-d` (The Daemon)
- Runs in the background.
- Polls providers (GitHub, OpenAI, etc.) for limits.
- Writes to the Event Log.
- Serves the API.

### Storage (Local SQLite)
- Stores the Event Log and Projections.
- Local-only; no cloud database required.

### Clients (TUI / Web)
- Connect to the daemon to visualize consumption.
- Render forecasts and decision history.
