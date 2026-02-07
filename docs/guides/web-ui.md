# Web UI Guide

The Ratelord Web UI provides a comprehensive visual interface for historical analysis, system monitoring, and deep-dive investigations. While the TUI is for real-time ops, the Web UI is for understanding "why" and "what if".

## Accessing the Dashboard

By default, the Web UI is embedded in the daemon and served at:

**[http://localhost:8090](http://localhost:8090)**

*Note: Ensure `ratelord-d` is running.*

## Core Views

### 1. Dashboard (`/`)
The landing page provides a high-level health check of your constraint graph.
*   **Metric Grid**: Key performance indicators like current global burn rate and active violation counts.
*   **Burn Rate Chart**: A time-series visualization showing usage trends against limits.
*   **Recent Alerts**: A feed of the latest `deny` or `throttle` decisions, allowing you to quickly spot blocked agents.

### 2. History (`/history`)
The History view is the canonical log of all system activity.
*   **Timeline**: A visual scrubber to navigate through time.
*   **Event Log**: A detailed list of every `Intent`, `Decision`, `Poll`, and `System` event.
*   **Filtering**: Drill down by time range, Agent ID, or Scope to isolate specific interactions.
*   **Inspection**: Click any event to view its full JSON payload, including policy trace data.

### 3. Identities (`/identities`)
This explorer visualizes the hierarchical structure of your system.
*   **Hierarchy Tree**: See how Agents, Scopes, and Constraint Pools relate.
    *   **Roots**: Agents (e.g., `crawler-01`)
    *   **Children**: Scopes (e.g., `repo:acme/backend`)
    *   **Leaves**: Pools (e.g., `github-core-api`)
*   **Status**: Quickly identify which identities are active, idle, or nearing exhaustion.
*   **Navigation**: Clicking an identity jumps to the History view filtered for that specific actor.

## Simulation (Experimental)
The `/simulate` view allows "what-if" analysis. You can replay historical windows with modified parameters (e.g., "What if we doubled our rate limit?" or "What if we added 5 more agents?") to forecast outcomes before making changes in production.
