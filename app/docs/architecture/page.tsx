export const metadata = {
  title: 'Architecture - Ratelord',
  description: 'Detailed breakdown of Ratelord components.',
}

export default function Architecture() {
  return (
    <div className="container mx-auto px-4 py-12 max-w-4xl">
      <h1 className="text-4xl font-bold mb-6">System Architecture</h1>

      <div className="prose prose-slate max-w-none">
        <p className="lead text-xl text-muted-foreground mb-8">
          Ratelord is designed as a modular, event-sourced system that sits between your agents and the external APIs they consume.
        </p>

        <h2 className="text-2xl font-bold mt-8 mb-4">Core Components</h2>

        <h3 className="text-xl font-bold mt-6 mb-3">1. The Daemon</h3>
        <p>
          The heart of Ratelord is a lightweight, zero-ops daemon written in Go. It acts as the single source of truth for all constraint states. It is responsible for:
        </p>
        <ul className="list-disc pl-5 space-y-2">
          <li><strong>Ingesting Intents:</strong> Receiving resource requests from agents.</li>
          <li><strong>Policy Enforcement:</strong> Checking requests against defined constraints.</li>
          <li><strong>State Management:</strong> Maintaining the current state of all budgets and rate limits.</li>
          <li><strong>Forecasting:</strong> Predicting when resources will be exhausted or replenished.</li>
        </ul>

        <h3 className="text-xl font-bold mt-6 mb-3">2. Storage Layer</h3>
        <p>
          Ratelord uses an event-sourced architecture.
        </p>
        <ul className="list-disc pl-5 space-y-2">
          <li><strong>Event Log:</strong> An append-only log of all state changes (IntentCreated, BudgetAllocated, LimitHit).</li>
          <li><strong>Snapshots:</strong> Periodic snapshots of the state to ensure fast startup times.</li>
          <li><strong>SQLite (Default):</strong> For local development and single-node deployments, embedded SQLite is used for zero-conf persistence.</li>
        </ul>

        <h3 className="text-xl font-bold mt-6 mb-3">3. Providers</h3>
        <p>
          Providers are adapters that interface with external services to fetch real-time usage data.
        </p>
        <ul className="list-disc pl-5 space-y-2">
          <li><strong>GitHub Provider:</strong> Tracks API rate limits, secondary rate limits, and graphQL node limits.</li>
          <li><strong>Mock Provider:</strong> Used for testing and simulation.</li>
        </ul>

        <h3 className="text-xl font-bold mt-6 mb-3">4. Clients</h3>
        <p>
          Ratelord exposes its functionality through multiple interfaces:
        </p>
        <ul className="list-disc pl-5 space-y-2">
          <li><strong>HTTP API:</strong> For programmatic access by agents.</li>
          <li><strong>TUI (Terminal UI):</strong> For operators to monitor the system in real-time.</li>
          <li><strong>Web UI:</strong> For dashboard-style visualization and configuration.</li>
        </ul>
      </div>
    </div>
  )
}
