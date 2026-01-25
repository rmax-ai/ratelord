# Web UI Specification

## 1. Philosophy & Purpose

The Web UI is the secondary, analytical interface for `ratelord`. Unlike the TUI, which focuses on real-time operations and "now," the Web UI is designed for **historical analysis**, **scenario simulation**, and **broad-scale monitoring**.

It answers questions that require deep temporal context or hypothetical reasoning:
*   "Why did we exhaust the search quota last Tuesday?"
*   "What happens if we double our agent count next week?"
*   "How has our burn rate efficiency trended over the last 30 days?"

### 1.1 Key Distinctions
| Feature | TUI (Operational) | Web UI (Analytical) |
| :--- | :--- | :--- |
| **Time Horizon** | Real-time (minutes/hours) | Historical (days/weeks/months) |
| **Interactivity** | Low (observation focused) | High (drilling down, simulation) |
| **Write Access** | Read-only | Read-only (Simulations are local/ephemeral) |
| **Primary Use** | Monitoring & Alerting | Root Cause Analysis & Planning |

## 2. Technical Architecture

### 2.1 Stack & Deployment
*   **Hosting**: The Web UI is a single-page application (SPA) embedded directly within the `ratelord-d` binary.
    *   Served via the daemon's internal HTTP server (e.g., at `http://localhost:8080/ui`).
    *   Zero external dependencies for deployment (no separate `npm run start` required for end-users).
*   **Framework**: Lightweight, component-based architecture (e.g., React, Vue, or Svelte).
    *   Must support client-side routing.
    *   State management should be minimal, relying on the daemon's API as the source of truth.
*   **Visualization**: High-performance charting library (e.g., Recharts, D3, or Vega) capable of rendering dense time-series data.

### 2.2 Data Interaction
*   **API Consumption**: The Web UI consumes the REST API defined in `API_SPEC.md`.
    *   `/v1/history`: For historical trends and event logs.
    *   `/v1/simulate`: For the Scenario Lab (daemon runs logic sandbox).
    *   `/v1/status`: For current high-level dashboard metrics.
*   **Read-Only Nature**: The Web UI does not modify live state (limits, policies, or configurations). All "write" actions are strictly for configuring local simulation parameters or view filters.

## 3. Core Views

### 3.1 Dashboard (The "Long View")
*   **Purpose**: High-level health summary with a focus on trends rather than instantaneous values.
*   **Key Metrics**:
    *   **Burn Rate Efficiency**: 7-day / 30-day trend lines for key quotas.
    *   **Exhaustion Events**: Heatmap of exhaustion events by day/hour.
    *   **Top Consumers**: Aggregated usage by Agent ID and Scope over selected time windows.
*   **Widgets**:
    *   "Risk Forecast": Probability of exhaustion in the next 24h based on current trends.
    *   "Policy Violations": Count of throttles/denials over time.

### 3.2 Time-Travel / History
*   **Purpose**: forensic analysis of past states.
*   **Interaction**:
    *   **Timeline Scrubber**: A visual slider to navigate through the event log.
    *   **Event Inspector**: Detailed view of specific events (e.g., a `policy_trigger` or `intent_denied` event), showing the exact context (identity, scope, pool) at that moment.
    *   **State Reconstruction**: Ability to view the "Constraint Graph" as it existed at a specific timestamp (replayed from the event log).

### 3.3 Scenario Lab (Simulator)
*   **Purpose**: "What-if" analysis using the daemon's prediction engine in a sandboxed environment.
*   **Workflow**:
    1.  **Select Baseline**: Choose a historical period (e.g., "Last Tuesday's traffic") as the base load.
    2.  **Apply Modifiers**:
        *   "Traffic x 2.0"
        *   "Limit Reduced by 50%"
        *   "Add 5 new Agents"
    3.  **Run Simulation**: The daemon processes the baseline + modifiers through the policy and prediction engines without affecting live state.
    4.  **Visualize Outcome**:
        *   "Projected Time-to-Exhaustion"
        *   "Estimated Denial Rate"
        *   "Policy Trigger Heatmap"
*   **Use Cases**: Capacity planning, policy testing, impact analysis of external changes (e.g., GitHub lowering API limits).

### 3.4 Identity & Scope Explorer
*   **Purpose**: Visualizing the hierarchical constraint graph.
*   **Visualization**: Interactive node-link diagram or tree map showing:
    *   Agents -> Identities -> Scopes -> Pools.
    *   Color-coded by usage intensity or risk level.
*   **Drill-down**: Clicking a node reveals detailed history and policy configuration for that specific entity.

## 4. User Experience (UX) Guidelines

*   **Dense Data, Clean Design**: Prioritize information density without clutter. Use sparklines and small multiples for repetitive metrics.
*   **Local-First Performance**: UI should feel instant. Heavy computations (simulations) happen in the daemon, but UI interactions (filtering, zooming) should be client-side and snappy.
*   **Deep Linking**: All views (especially specific time ranges and simulation configurations) must be URL-addressable for sharing context between engineers.
*   **Dark Mode**: Default to dark mode to match developer tooling preferences (and the TUI aesthetic).

## 5. Security & Access
*   **Authentication**: If the daemon is configured with auth (e.g., for shared deployments), the Web UI must handle token management.
*   **Scope Isolation**: In multi-tenant scenarios (future), the UI should only display data the authenticated user is permitted to see.

## 6. Future Extensions
*   **Policy Editor**: A visual builder for `POLICY_ENGINE.md` rules (export-only, to be committed to git).
*   **Report Generation**: PDF/Markdown export of monthly usage reports for management.
