export const metadata = {
  title: 'API Reference - Ratelord',
  description: 'HTTP endpoints and protocol specification.',
}

export default function ApiReference() {
  return (
    <div className="container mx-auto px-4 py-12 max-w-4xl">
      <h1 className="text-4xl font-bold mb-6">API Reference</h1>
      
      <div className="prose prose-slate dark:prose-invert max-w-none">
        <p className="lead text-xl text-muted-foreground mb-8">
          The Ratelord Daemon exposes a RESTful JSON API. All requests must be authenticated with an Identity Token.
        </p>

        <h2 className="text-2xl font-bold mt-8 mb-4">Base URL</h2>
        <div className="bg-muted p-4 rounded-md my-4 font-mono text-sm">
          <code>http://localhost:8090/v1</code>
        </div>

        <h2 className="text-2xl font-bold mt-8 mb-4">Endpoints</h2>

        <div className="space-y-8">
          <div className="border rounded-lg p-6">
            <h3 className="text-xl font-bold mb-2 flex items-center gap-2">
              <span className="bg-green-100 text-green-800 text-xs px-2 py-1 rounded font-mono">POST</span>
              /intents/negotiate
            </h3>
            <p className="mb-4">Negotiate a budget for a planned action.</p>
            <h4 className="font-bold text-sm mb-2">Request Body</h4>
            <div className="bg-muted p-4 rounded-md font-mono text-sm overflow-x-auto">
{`{
  "identity_id": "agent-123",
  "scope": "github:api",
  "requested_amount": 100,
  "priority": "normal"
}`}
            </div>
          </div>

          <div className="border rounded-lg p-6">
            <h3 className="text-xl font-bold mb-2 flex items-center gap-2">
              <span className="bg-blue-100 text-blue-800 text-xs px-2 py-1 rounded font-mono">GET</span>
              /status
            </h3>
            <p className="mb-4">Get the current status of all resource pools visible to the authenticated identity.</p>
          </div>

          <div className="border rounded-lg p-6">
            <h3 className="text-xl font-bold mb-2 flex items-center gap-2">
              <span className="bg-blue-100 text-blue-800 text-xs px-2 py-1 rounded font-mono">GET</span>
              /forecasts
            </h3>
            <p className="mb-4">Retrieve time-to-exhaustion forecasts for specific scopes.</p>
          </div>
        </div>
      </div>
    </div>
  )
}
