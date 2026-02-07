import { CodeBlock } from '@/app/components/code-block';

export const metadata = {
  title: 'Getting Started - Ratelord',
  description: 'Installation instructions, quickstart steps, and production links.',
}

export default function GettingStarted() {
  return (
    <div className="container mx-auto px-4 py-12 max-w-4xl">
      <h1 className="text-4xl font-bold mb-6">Installation &amp; Quickstart</h1>

      <div className="prose prose-slate max-w-none">
        <p className="lead text-xl text-muted-foreground mb-8">
          Ratelord is a local-first constraint control plane. Install the daemon, register an identity, and start negotiating intents in just a few commands.
        </p>

        <h2 className="text-2xl font-bold mt-8 mb-4">Install the Binary</h2>
        <p>
          Build the daemon and the helpers from source using Go 1.23+. The same build also produces the HTTP API daemon (`ratelord-d`) and the interactive TUI (`ratelord-tui`).
        </p>

        <CodeBlock
          language="bash"
          code={`go install github.com/rmax-ai/ratelord/cmd/ratelord@latest
go install github.com/rmax-ai/ratelord/cmd/ratelord-d@latest
go install github.com/rmax-ai/ratelord/cmd/ratelord-tui@latest`}
        />

        <h3 className="text-xl font-bold mt-6 mb-3">Verify the install</h3>
        <CodeBlock
          language="bash"
          code={`ratelord version
ratelord-d --version
ratelord-tui --version`}
        />

        <p>
          Pre-built binaries are also published on <a className="text-primary" href="https://github.com/rmax-ai/ratelord/releases" target="_blank" rel="noreferrer">GitHub Releases</a> for macOS, Linux, and Windows.
        </p>

        <h2 className="text-2xl font-bold mt-10 mb-4">Local Quickstart</h2>
        <ol className="list-decimal pl-5 space-y-3">
          <li>
            <strong>Start the daemon:</strong>
            <CodeBlock language="bash" code={`ratelord-d`} />
            It creates `ratelord.db` in the current directory and listens on port 8090 by default.
          </li>
          <li>
            <strong>Launch the UI:</strong>
            <CodeBlock language="bash" code={`ratelord-tui`} />
            Use the terminal UI to inspect pools, intents, and forecasts.
          </li>
          <li>
            <strong>Register an identity:</strong>
            <CodeBlock language="bash" code={`ratelord identity add pat:local user`} />
            Save the returned token; agents must include it when negotiating intents.
          </li>
        </ol>

        <h2 className="text-2xl font-bold mt-10 mb-4">Next Steps</h2>
        <p>
          _Personal_ deployments can stay on SQLite, but production-grade installs should follow the <a className="text-primary" href="/docs/configuration">Configuration Guide</a> and the <a className="text-primary" href="/docs/architecture">Architecture Overview</a> to plan providers, policies, persistence, and HA.
        </p>
        <p>
          When you are ready to run Ratelord in a cluster, read the <a className="text-primary" href="/docs/troubleshooting">Troubleshooting Guide</a> and coordinate your deployment with the <a className="text-primary" href="/docs/api">API Reference</a>.
        </p>
      </div>
    </div>
  )
}
