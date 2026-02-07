import { CodeBlock } from '@/app/components/code-block';

export const metadata = {
  title: 'API Reference - Ratelord',
  description: 'Intent negotiation protocol, health endpoints, and observability APIs.',
}

export default function ApiReference() {
  return (
    <div className="container mx-auto px-4 py-12 max-w-4xl">
      <h1 className="text-4xl font-bold mb-6">API Reference</h1>

      <div className="prose prose-slate max-w-none">
        <p className="lead text-xl text-muted-foreground mb-8">
          The Ratelord daemon exposes a JSON-over-HTTP API. Clients must authenticate with an Identity Token for all requests. The API runs locally (default <code>http://localhost:8090</code>) and is designed to fail safe: if you cannot reach the daemon, assume your intent will be denied.
        </p>

        <h2 className="text-2xl font-bold mt-8 mb-4">Base URL</h2>
        <div className="bg-muted p-4 rounded-md my-4 font-mono text-sm px-4 py-2">
          <code>http://localhost:8090/v1</code>
        </div>

        <h2 className="text-2xl font-bold mt-10 mb-4">Core Concepts</h2>
        <ul className="list-disc pl-5 space-y-2">
          <li><strong>Local-only:</strong> The API binds to the loopback interface by default for single-machine coordination.</li>
          <li><strong>Fail-safe:</strong> Clients should treat timeouts as a denial and back off safely.</li>
          <li><strong>Model:</strong> Intent negotiation is the central workflow—agents request resources, and the daemon evaluates policies before granting them.</li>
        </ul>

        <h2 className="text-2xl font-bold mt-10 mb-4">Endpoints</h2>
        <div className="space-y-10">
          <div className="border rounded-lg p-6">
            <h3 className="text-xl font-bold mb-2 flex items-center gap-2">
              <span className="bg-green-100 text-green-800 text-xs px-2 py-1 rounded font-mono">POST</span>
              /v1/intent
            </h3>
            <p className="mb-4">Submit an intent describing the resource you want to consume. The daemon replies with approval, denial, or a counter offer.</p>
            <h4 className="font-bold text-sm mb-2">Request Body</h4>
            <CodeBlock
              language="json"
              code={`{
  "agent_id": "crawler-01",
  "identity_id": "pat:user",
  "workload_id": "repo_scan",
  "scope_id": "repo:owner/project",
  "urgency": "normal"
}`}
            />
            <h4 className="font-bold text-sm mt-4 mb-1">Sample Response</h4>
            <CodeBlock
              language="json"
              code={`{
  "decision": "approve",
  "intent_id": "evt_12345",
  "evaluation": {
    "risk_summary": "Low risk (P99 TTE > 1h)"
  }
}`}
            />
          </div>

          <div className="border rounded-lg p-6">
            <h3 className="text-xl font-bold mb-2 flex items-center gap-2">
              <span className="bg-blue-100 text-blue-800 text-xs px-2 py-1 rounded font-mono">GET</span>
              /v1/health
            </h3>
            <p className="mb-3">Check whether the daemon is ready, degraded, or still initializing.</p>
            <CodeBlock
              language="json"
              code={`{
  "status": "ok",
  "version": "0.1.0",
  "uptime_seconds": 3600
}`}
            />
          </div>

          <div className="border rounded-lg p-6">
            <h3 className="text-xl font-bold mb-2 flex items-center gap-2">
              <span className="bg-blue-100 text-blue-800 text-xs px-2 py-1 rounded font-mono">GET</span>
              /v1/graph
            </h3>
            <p className="mb-3">Returns the current constraint graph (pools, scopes, and identities plus their relationships).</p>
          </div>

          <div className="border rounded-lg p-6">
            <h3 className="text-xl font-bold mb-2">Analytics &amp; Reporting</h3>
            <p className="mb-3">Use the trends and reports endpoints for dashboards or auditors.</p>
            <ul className="list-disc pl-5 space-y-2 text-sm">
              <li><strong>GET /v1/trends:</strong> Aggregated usage over a time window (from, to, bucket, provider_id).</li>
              <li><strong>GET /v1/reports:</strong> CSV exports for usage, access logs, or events (type, from, to).</li>
            </ul>
          </div>

          <div className="border rounded-lg p-6">
            <h3 className="text-xl font-bold mb-2">Federation &amp; Clustering</h3>
            <ul className="list-disc pl-5 space-y-2 text-sm">
              <li><strong>GET /v1/cluster/nodes:</strong> Returns topology of known leaders and followers.</li>
              <li><strong>POST /v1/federation/grant:</strong> Followers request grants from the leader (body: node_id, resource, amount).</li>
            </ul>
          </div>

          <div className="border rounded-lg p-6">
            <h3 className="text-xl font-bold mb-2">Integrations</h3>
            <p className="mb-3">Register webhooks to receive streaming events.</p>
            <CodeBlock
              language="json"
              code={`{
  "url": "https://hooks.slack.com/...",
  "events": ["intent.denied", "system.alert"]
}`}
            />
          </div>

          <div className="border rounded-lg p-6">
            <h3 className="text-xl font-bold mb-2">Administration</h3>
            <p className="mb-3">Prune old events and manage the event ledger.</p>
            <CodeBlock
              language="json"
              code={`{
  "before": "2023-01-01T00:00:00Z"
}`}
            />
          </div>

          <div className="border rounded-lg p-6">
            <h3 className="text-xl font-bold mb-2">Event Streaming</h3>
            <p className="mb-3">Stream recent events as SSE or poll the latest batch.</p>
            <ul className="list-disc pl-5 space-y-1 text-sm">
              <li><strong>GET /v1/events:</strong> Parameters: limit (default 50), stream=true for SSE.</li>
            </ul>
          </div>

          <div className="border rounded-lg p-6">
            <h3 className="text-xl font-bold mb-2">Identity Management</h3>
            <p className="mb-3">Register, inspect, and delete identities.</p>
            <CodeBlock
              language="json"
              code={`{
  "identity_id": "pat:github:rmax",
  "kind": "user",
  "metadata": { "name": "RMax" }
}`}
            />
            <p className="text-sm text-muted-foreground">DELETE <code>/v1/identities/&lt;id&gt;</code> removes an identity and scrubs its history for compliance.</p>
          </div>

          <div className="border rounded-lg p-6">
            <h3 className="text-xl font-bold mb-2">Observability</h3>
            <p className="mb-3">Prometheus-style metrics expose pool usage and forecast data.</p>
            <ul className="list-disc pl-5 space-y-1 text-sm">
              <li><strong>GET /metrics:</strong> Returns metrics such as `ratelord_usage`, `ratelord_limit`, `ratelord_intent_total`, `ratelord_forecast_seconds`.</li>
            </ul>
          </div>

          <div className="border rounded-lg p-6">
            <h3 className="text-xl font-bold mb-2">Debugging</h3>
            <p className="mb-3">Inject synthetic provider data for testing flows.</p>
            <CodeBlock
              language="json"
              code={`{
  "provider_id": "github-mock",
  "pool_id": "core",
  "usage": 4500,
  "limit": 5000
}`}
            />
          </div>
        </div>

        <h2 className="text-2xl font-bold mt-12 mb-4">Error Handling</h2>
        <p className="mb-3">The API uses standard HTTP status codes. A denial from the policy engine still returns 200.</p>
        <ul className="list-disc pl-5 space-y-2 text-sm">
          <li><strong>200 OK:</strong> Request processed (including `deny_with_reason`).</li>
          <li><strong>400 Bad Request:</strong> Invalid payload or missing fields.</li>
          <li><strong>503 Service Unavailable:</strong> Daemon starting, shutting down, or overloaded.</li>
        </ul>
      </div>
    </div>
  )
}
