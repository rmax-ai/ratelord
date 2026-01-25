# TUI_SPEC: ratelord

This document defines the Terminal User Interface (TUI) for `ratelord`. The TUI is the primary "pane of glass" for operators to observe the system's state, understand constraint pressure, and debug blocked agents.

## 1. Core Philosophy

The TUI must embody the `ratelord` principles: **Predictive, Transparent, and Authoritative**.

### 1.1 "Everything at a Glance"
Constraint management is often a background concern until it becomes a critical failure. The TUI must provide high-density situational awareness immediately upon launch. An operator should answer "Are we healthy?" in < 1 second.

### 1.2 Alert-Centric Visuals
Color and layout must guide attention to risks.
*   **Green**: Nominal. Operating within P99 forecasts.
*   **Yellow**: Elevated risk. Burn rate is high; intent modifications are likely.
*   **Red**: Critical. Exhaustion is imminent or active; intents are being denied.

### 1.3 Read-Only Safe (Primary)
The TUI is primarily an **observer** of the daemon's state. It does not calculate state itself; it renders the views provided by the daemon.
*   It consumes the `/v1/events` stream and `/v1/forecast` endpoints.
*   It does *not* write to the database directly.
*   *Exception*: A debug "Command Mode" allows submitting test intents to verify policy behavior, but this uses the standard public API (`POST /v1/intent`), identical to any other agent.

---

## 2. Technical Stack

To maintain the "single binary, zero-ops" philosophy while delivering a rich, responsive interface:

*   **Language**: Go
*   **Library**: [Bubble Tea](https://github.com/charmbracelet/bubbletea) (ELM architecture)
    *   **Lip Gloss**: For styling and layout.
    *   **Bubbles**: For table, list, and viewport components.
*   **Communication**: HTTP/1.1 (REST + SSE) to `localhost:8090`.

---

## 3. Layout & Views

The TUI is organized into a main **Dashboard View** with modal **Drill-Down Views**.

### 3.1 Global Header (Always Visible)
*   **Left**: `ratelord` logo/text.
*   **Center**: Daemon Status (`Connected` | `Reconnecting...`), Uptime (`14d 2h`), Version (`v0.1.0`).
*   **Right**: Global Risk Score (0-100), rendered as a color-coded gauge.

### 3.2 Dashboard View (Default)
A split-pane layout maximizing information density.

#### Pane A: Pool Posture (Top Half)
A table listing all active constraint pools.
*   **Row Entity**: `Pool` (e.g., `github:rest:core`, `github:graphql`).
*   **Columns**:
    1.  **Name**: Pool ID (e.g., `github:rest`).
    2.  **Scope**: Target (e.g., `org:acme`).
    3.  **Rem**: Remaining Capacity (e.g., `4500/5000`).
    4.  **Burn Rate**: Current consumption (e.g., `12/min`).
    5.  **P99 TTE**: Forecasted time to exhaustion (e.g., `45m`). **Crucial metric.**
    6.  **Status**: Badge (`OK`, `WARN`, `CRIT`).
*   **Interaction**: Arrow keys to highlight a row; `Enter` to drill down.

#### Pane B: Activity Stream (Bottom Left)
A live-tail log of recent events, auto-scrolling.
*   **Format**: `[timestamp] [AGENT_ID] [INTENT] -> [DECISION]`
*   **Coloring**:
    *   `approve`: Green
    *   `modify`: Yellow
    *   `deny`: Red
*   **Example**:
    ```text
    12:01:05 [crawler-01] intent:scan_repo -> APPROVED
    12:01:08 [ci-agent-2] intent:build     -> MODIFIED (wait 5s)
    12:01:12 [dev-script] intent:dump_all  -> DENIED (risk too high)
    ```

#### Pane C: System Vitals (Bottom Right)
Sparklines or mini-charts showing global aggregate metrics.
*   **Total Intents/Min**: Line chart.
*   **Denial Rate**: Percentage gauge.
*   **DB Lag**: Milliseconds.

### 3.3 Pool Detail View (Drill-Down)
Activated by selecting a pool in Pane A. Overlays the dashboard.

*   **Header**: `Pool: github:rest:core [org:acme]`
*   **Forecast Graph**: ASCII/Unicode line chart showing:
    *   **Capacity Line**: The hard limit.
    *   **Historical Usage**: Past 1h.
    *   **Forecast Cone**: P50/P90/P99 projection lines into the future (up to reset).
*   **Attribution Table**: "Who is eating the budget?"
    *   List of Top 5 Agents by consumption in the current window.
*   **Key Controls**:
    *   `Esc`: Back to Dashboard.
    *   `f`: Force refresh forecast.

### 3.4 Command Mode (Interactive Debugging)
Activated by pressing `:`. A command-line input at the bottom of the screen.

*   **Commands**:
    *   `:filter agent <id>`: Filter Activity Stream by Agent ID.
    *   `:filter decision deny`: Show only denied intents.
    *   `:clear`: Reset filters.
    *   `:test <json>`: Open a form to manually submit a raw JSON intent (for testing policies).
    *   `:quit` / `:q`: Exit TUI.

---

## 4. User Stories

### 4.1 "Why is my build throttled?"
1.  Operator sees CI job failing or slowing down.
2.  Opens `ratelord-tui`.
3.  Notices Global Risk is **Yellow**.
4.  Looks at **Pools Table**. Sees `github:rest:core` for `org:acme` has **P99 TTE < 5m** (Critical).
5.  Selects the pool and hits `Enter`.
6.  **Pool Detail View** shows a specific agent (`crawler-experimental`) consuming 80% of the budget.
7.  Operator kills the crawler agent; `ratelord` forecast recovers; builds resume.

### 4.2 "Is this new agent safe to deploy?"
1.  Operator deploys `new-agent-v1` in "dry-run" mode (submitting intents but not acting).
2.  Opens TUI and types `:filter agent new-agent-v1`.
3.  Watches the **Activity Stream**.
4.  Sees a stream of `APPROVED` decisions.
5.  Notices one `MODIFIED (wait 2s)` during a burst.
6.  Concludes the agent is behaving well within the policy; proceeds to full deployment.

---

## 5. Key Bindings (Reference)

| Key | Action |
| :--- | :--- |
| `q` / `Ctrl+c` | Quit |
| `Use Arrows` | Navigate tables/lists |
| `Enter` | Select / Drill down |
| `Esc` | Go back / Clear selection |
| `:` | Enter Command Mode |
| `/` | Quick Filter (Logs) |
| `?` | Show Help Modal |

---

## 6. Implementation Phases

### Phase 1: Read-Only Dashboard
*   Connect to `/v1/health` and `/v1/forecast`.
*   Render static Pools Table and Activity Stream (polling, no SSE yet).
*   Basic styling.

### Phase 2: Live Updates & Drill-Downs
*   Implement SSE client for `/v1/events` (Live Tail).
*   Add Pool Detail View with attribution logic.
*   Implement ASCII graphing for forecasts.

### Phase 3: Interactive & Command Mode
*   Add filtering logic.
*   Add "Test Intent" form.
*   Refine aesthetics (colors, responsive layout).
