export const metadata = {
  title: 'Getting Started - Ratelord',
  description: 'Installation instructions and basic setup for Ratelord.',
}

export default function GettingStarted() {
  return (
    <div className="container mx-auto px-4 py-12 max-w-4xl">
      <h1 className="text-4xl font-bold mb-6">Getting Started with Ratelord</h1>

      <div className="prose prose-slate dark:prose-invert max-w-none">
        <p className="lead text-xl text-muted-foreground mb-8">
          Ratelord is a local-first constraint control plane. This guide will help you install the daemon, configure your first identity, and start managing constraints.
        </p>

        <h2 className="text-2xl font-bold mt-8 mb-4">Installation</h2>
        <p>
          Ratelord is distributed as a single binary. You can download the latest release from GitHub or build from source.
        </p>

        <div className="bg-muted p-4 rounded-md my-4 font-mono text-sm overflow-x-auto">
          <code>
            # Install via Go<br/>
            go install github.com/rmax-ai/ratelord/cmd/ratelord@latest<br/><br/>
            # Verify installation<br/>
            ratelord version
          </code>
        </div>

        <h2 className="text-2xl font-bold mt-8 mb-4">Quick Start</h2>
        <p>
          To start the daemon with default configuration (SQLite storage in ~/.ratelord):
        </p>

        <div className="bg-muted p-4 rounded-md my-4 font-mono text-sm overflow-x-auto">
          <code>
            ratelord daemon start
          </code>
        </div>

        <h3 className="text-xl font-bold mt-6 mb-3">Registering an Identity</h3>
        <p>
          Before any agent can negotiate for resources, it must have an identity.
        </p>

        <div className="bg-muted p-4 rounded-md my-4 font-mono text-sm overflow-x-auto">
          <code>
            ratelord identity create --name "agent-001" --role "scraper"
          </code>
        </div>

        <p>
          This will return an API key that your agent should use in all subsequent requests.
        </p>
      </div>
    </div>
  )
}
