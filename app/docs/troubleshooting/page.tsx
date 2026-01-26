export const metadata = {
  title: 'Troubleshooting - Ratelord',
  description: 'Common issues and debugging strategies.',
}

export default function Troubleshooting() {
  return (
    <div className="container mx-auto px-4 py-12 max-w-4xl">
      <h1 className="text-4xl font-bold mb-6">Troubleshooting</h1>
      
      <div className="prose prose-slate dark:prose-invert max-w-none">
        <h2 className="text-2xl font-bold mt-8 mb-4">Common Issues</h2>
        
        <h3 className="text-xl font-bold mt-6 mb-3">Daemon Fails to Start</h3>
        <p>Check if the port (8090 by default) is already in use.</p>
        <div className="bg-muted p-4 rounded-md my-4 font-mono text-sm">
          <code>lsof -i :8090</code>
        </div>

        <h3 className="text-xl font-bold mt-6 mb-3">Database Locked</h3>
        <p>
          If using SQLite, ensure no other process is holding a lock on the database file.
        </p>

        <h2 className="text-2xl font-bold mt-8 mb-4">Logs</h2>
        <p>
          Ratelord writes structured JSON logs to stdout. You can pipe these to `jq` for easier reading.
        </p>
        <div className="bg-muted p-4 rounded-md my-4 font-mono text-sm">
          <code>ratelord daemon start | jq</code>
        </div>
      </div>
    </div>
  )
}
