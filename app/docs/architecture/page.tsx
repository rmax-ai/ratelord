export const metadata = {
  title: 'Architecture - Ratelord',
  description: 'Daemon authority, event sourcing, and intent negotiation overview.',
}

export default function Architecture() {
  return (
    <div className="container mx-auto px-4 py-12 max-w-4xl">
      <h1 className="text-4xl font-bold mb-6">Architecture: How Ratelord Works</h1>

      <div className="prose prose-slate max-w-none">
        <p className="lead text-xl text-muted-foreground mb-8">
          Ratelord is a local-first, predictive control plane that sits between your agents and external APIs. The architecture is organized around a single authoritative daemon, immutable event sourcing, and a negotiation protocol that safely allocates resources.
        </p>

        <h2 className="text-2xl font-bold mt-8 mb-4">Daemon Authority</h2>
        <p>
          The `ratelord-d` daemon is the only process capable of writing state or making decisions. All other clients are read-only observers.
        </p>
        <ul className="list-disc pl-5 space-y-2">
          <li><strong>Single source of truth:</strong> Every policy evaluation, grant, and denial is authored by the daemon.</li>
          <li><strong>Read-only clients:</strong> CLI, TUI, Web UI, and SDKs only display what the daemon reports.</li>
          <li><strong>Intent-based:</strong> Agents describe the action they want to perform; the daemon evaluates risk before allowing it.</li>
        </ul>

        <h2 className="text-2xl font-bold mt-8 mb-4">Event Sourcing & Replayability</h2>
        <p>
          The architecture is backed by an append-only event log. Every observation, decision, and policy change becomes an event, which can always be replayed to rebuild the system state.
        </p>
        <ol className="list-decimal pl-5 space-y-2">
          <li><strong>Event log:</strong> Immutable history of every provider poll, intent negotiation, and policy evaluation.</li>
          <li><strong>Replay:</strong> Restarting the daemon replays the log to reconstruct the current constraint graph.</li>
          <li><strong>Projections:</strong> Materialized views derived from the log keep queries fast (e.g., current usage, pool health).</li>
        </ol>

        <h2 className="text-2xl font-bold mt-8 mb-4">Prediction & Risk Modeling</h2>
        <p>
          Ratelord thinks in time, not just counts. Forecasts and burn rates help agents avoid imminent exhaustion.
        </p>
        <ul className="list-disc pl-5 space-y-2">
          <li><strong>Time-to-Exhaustion (TTE):</strong> Estimates how long before a pool hits zero at the current burn rate.</li>
          <li><strong>Probability scores:</strong> Tracks the risk of exceeding a limit before the next reset window.</li>
          <li><strong>Forecast-aware decisions:</strong> The daemon can counter, shape, or deny intents when the risk is too high.</li>
        </ul>

        <h2 className="text-2xl font-bold mt-8 mb-4">Intent Negotiation Workflow</h2>
        <p>
          Agents never grab capacity; they negotiate it through a deterministic protocol.
        </p>
        <ol className="list-decimal pl-5 space-y-2">
          <li><strong>Propose:</strong> Agent submits the desired action (`POST /v1/intent`).</li>
          <li><strong>Evaluate:</strong> The daemon checks pools, forecasts, policies, and priorities.</li>
          <li><strong>Decide:</strong> Responds with `approve`, `deny_with_reason`, or `approve_with_modifications` (e.g., wait or throttle).</li>
        </ol>

        <h2 className="text-2xl font-bold mt-8 mb-4">Primary Components</h2>
        <h3 className="text-xl font-bold mt-6 mb-3">Ratelord Daemon</h3>
        <ul className="list-disc pl-5 space-y-2">
          <li>Polls providers (GitHub, OpenAI, etc.) for limit data.</li>
          <li>Enforces policies and writes every event to the log.</li>
          <li>Serves the HTTP API that agents and clients call.</li>
        </ul>

        <h3 className="text-xl font-bold mt-6 mb-3">Storage</h3>
        <p>
          Local deployments default to SQLite (with WAL mode). For HA, Redis can back shared state across federated leaders and followers.
        </p>

        <h3 className="text-xl font-bold mt-6 mb-3">Clients & UIs</h3>
        <ul className="list-disc pl-5 space-y-2">
          <li><strong>HTTP API:</strong> Programmatic access via `/v1` endpoints.</li>
          <li><strong>TUI:</strong> Terminal UI for operators to inspect intents and forecasts.</li>
          <li><strong>Web UI:</strong> Dashboards that render the constraint graph, timelines, and health metrics.</li>
        </ul>
      </div>
    </div>
  )
}
