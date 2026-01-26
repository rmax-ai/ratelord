# Web UI Specification

## 1. Philosophy & Purpose

The Web UI is the **analytical** interface for `ratelord`. While the TUI handles real-time operations, the Web UI focuses on **historical analysis**, **scenario simulation**, and **broad-scale monitoring**.

**Core Mission**: Answer "Why did this happen?" and "What if we change X?" using deep temporal context.

**Key Distinctions**:
*   **Time Horizon**: Historical (days/weeks) vs. TUI's Real-time (minutes).
*   **Interaction**: Deep drill-down and simulation vs. TUI's observation.
*   **Write Access**: Read-only (except local simulation configs).

## 2. Technology Stack

We enforce a strict, modern stack to ensure maintainability and performance.

*   **Language**: TypeScript
*   **Framework**: React 18+
*   **Build Tool**: Vite
*   **Styling**: Tailwind CSS (Utility-first)
*   **Routing**: React Router v6
*   **State/Fetching**: TanStack Query (React Query)
*   **Visualization**: Recharts (for time-series), React Flow (for node graphs - optional but recommended for Identity Explorer)
*   **Icons**: Lucide React

## 3. Architecture & Build Pipeline

The Web UI is a Single Page Application (SPA) designed to be embedded directly into the `ratelord-d` binary.

### 3.1 Build Process
1.  **Source**: `web/` directory in the repo root.
2.  **Build**: `npm run build` generates static assets (HTML, CSS, JS) into `web/dist`.
3.  **Embedding**:
    *   Go server uses `//go:embed` to bundle `web/dist` into the binary.
    *   A generic HTTP handler serves `index.html` for all non-API routes (SPA fallback) to support client-side routing.

### 3.2 Directory Structure
```
web/
├── src/
│   ├── components/      # Shared atomic components (Button, Card, Badge)
│   ├── features/        # Feature-specific modules (dashboard, simulation)
│   │   ├── components/  # Feature-scoped components
│   │   ├── hooks/       # Feature-scoped data hooks
│   │   └── types.ts     # Feature-scoped types
│   ├── hooks/           # Global hooks (useTheme, useEventStream)
│   ├── layouts/         # Page layouts (AppShell)
│   ├── lib/             # Utilities (api client, formatters)
│   ├── pages/           # Route entry points
│   ├── App.tsx          # Root component + Providers
│   └── main.tsx         # Entry point
├── index.html
├── tailwind.config.js
├── tsconfig.json
└── vite.config.ts
```

## 4. Routing & Navigation

The URL is the source of truth for navigation state to enable deep-linking.

| Route | View | Description |
| :--- | :--- | :--- |
| `/` | **Dashboard** | High-level metrics, health summary, and recent alerts. |
| `/history` | **History** | Event log explorer with time-range filtering. |
| `/identities` | **Explorer** | Hierarchical view of Agents, Scopes, and Pools. |
| `/simulate` | **Scenario Lab** | "What-if" analysis sandbox. |
| `/settings` | **Settings** | Local UI preferences (theme, refresh rate). |

**Query Parameters**:
*   All views involving time must respect `?from=<ts>&to=<ts>` params.
*   Views involving filters must respect `?agent=<id>&scope=<id>`.

## 5. Component Hierarchy

### 5.1 AppShell (Layout)
*   **SidebarNavigation**: Collapsible main nav.
*   **GlobalHeader**: Breadcrumbs, connection status indicator (Daemon: Online/Offline), theme toggle.
*   **MainContent**: Router outlet.

### 5.2 Dashboard (`/`)
*   **MetricGrid**: Grid of `MetricCard` components (Current Usage, Burn Rate, Violations).
*   **BurnRateChart**: Recharts `AreaChart` showing usage vs limits over time.
*   **RecentAlertsFeed**: Condensed list of recent `deny` or `throttle` events.

### 5.3 History (`/history`)
*   **TimeRangePicker**: Specialized control for selecting absolute or relative windows (Last 1h, Last 7d).
*   **EventTimeline**: Visual scrubber/minimap of event density.
*   **EventList**: Virtualized list of raw events.
*   **EventDetailPanel**: Slide-over or modal showing full JSON payload of a selected event.

### 5.4 Scenario Lab (`/simulate`)
*   **SimulationConfig**: Form to set baseline time range and modifiers (multiplier, limit changes).
*   **SimulationRunner**: "Run" button + progress indicator.
*   **ComparisonView**: Split view showing `Actual` vs `Simulated` outcomes.

### 5.5 Identity Explorer (/identities)
*   **Purpose**: Visualize the hierarchical relationship between Agents, Scopes, and Pools.
*   **Data Transformation**: The API (`GET /v1/identities`) returns a flat list of identities. The UI must reconstruct the hierarchy client-side:
    *   **Roots**: Identities with no parent are root nodes (typically Agents).
    *   **Children**: Identities where `parent_id` matches a root become children (Scopes).
    *   **Leaves**: Deepest level nodes are typically Pools.
*   **Visualizer**:
    *   **TreeTable**: Primary view. A collapsible table where rows can be expanded to show children.
    *   **Columns**:
        *   **ID**: The identity identifier (e.g., `agent:foo`).
        *   **Kind**: Icon/Badge indicating Agent, Scope, or Pool.
        *   **Current Usage**: Sparkline or progress bar (if usage data is linked).
        *   **Last Active**: Relative timestamp (e.g., "2m ago").
*   **Interaction**: Clicking a row navigates to `/history` filtered by that identity ID.

## 6. State Management & Data Sync

### 6.1 Server State (TanStack Query)
*   **Pattern**: We treat the Daemon as the source of truth.
*   **Caching**: Aggressive caching for historical data (immutable).
*   **Live Data**:
    *   **Polling**: Default mechanism (e.g., every 5s) for dashboard metrics.
    *   **Invalidation**: `refetchOnWindowFocus` enabled to ensure freshness.

### 6.2 Client State (Zustand/Context)
*   Minimal client-only state:
    *   Theme (Dark/Light)
    *   Sidebar collapsed/expanded
    *   Active simulation configuration (draft)

## 7. API Integration

The UI consumes the REST API defined in `API_SPEC.md`.

*   **Client**: Typed `fetch` wrapper (or `axios`) handling base URL configuration.
*   **Error Handling**: Global toast notifications for API failures.
*   **Types**: TypeScript interfaces generated from or manually synced with Go structs in `DATA_MODEL.md`.

## 8. UX & Design Guidelines

*   **Dark Mode First**: The UI should default to dark mode to match the CLI experience.
*   **Information Density**: Use "sparklines" and compact tables. Avoid excessive whitespace.
*   **Feedback**: Every action (especially simulation runs) must provide immediate visual feedback (loading skeletons, spinners).
*   **Zero Configuration**: The UI must work out-of-the-box when the daemon starts.

## 9. Security

*   **Auth**: If the daemon enforces auth, the UI client must intercept 401s and redirect to a login prompt or generic "Unauthorized" state.
*   **Sanitization**: All user input (filters, simulation params) must be sanitized before sending to API, though the API is the ultimate gatekeeper.
