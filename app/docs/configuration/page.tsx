import { CodeBlock } from '@/app/components/code-block';

export const metadata = {
  title: 'Configuration - Ratelord',
  description: 'Environment variables, policy structure, and provider configs.',
}

export default function Configuration() {
  return (
    <div className="container mx-auto px-4 py-12 max-w-4xl">
      <h1 className="text-4xl font-bold mb-6">Configuration Guide</h1>

      <div className="prose prose-slate max-w-none">
        <p className="lead text-xl text-muted-foreground mb-8">
          Ratelord is driven by environment variables for runtime tuning and a declarative policy file (`policy.yaml` or `.json`) that captures providers, pools, and enforcement rules.
        </p>

        <h2 className="text-2xl font-bold mt-8 mb-4">Environment Variables</h2>
        <p>Use these variables to control networking, storage, policy paths, and clustering behaviour.</p>
        <div className="overflow-x-auto">
          <table className="min-w-full text-left text-sm">
            <thead>
              <tr className="text-xs uppercase text-muted-foreground">
                <th className="px-2 py-2">Variable</th>
                <th className="px-2 py-2">Description</th>
                <th className="px-2 py-2">Default</th>
              </tr>
            </thead>
            <tbody>
              {[
                ['RATELORD_PORT', 'HTTP port the daemon listens on.', '8090'],
                ['RATELORD_DB_PATH', 'Path to SQLite database file (state store).', './ratelord.db'],
                ['RATELORD_LOG_LEVEL', 'Logging verbosity (debug, info, warn, error).', 'info'],
                ['RATELORD_POLICY_PATH', 'Policy file path (.json or .yaml).', './policy.json'],
                ['RATELORD_TLS_CERT', 'TLS certificate file path for HTTPS.', '(disabled)'],
                ['RATELORD_TLS_KEY', 'TLS private key file path.', '(disabled)'],
                ['RATELORD_REDIS_URL', 'Redis connection string for distributed mode.', '(disabled)'],
                ['RATELORD_WEB_DIR', 'Directory containing the Web UI assets.', '(none)'],
                ['RATELORD_MODE', 'Operating mode: leader, follower, or standalone.', 'leader'],
                ['RATELORD_LEADER_URL', 'Leader node URL (followers only).', 'http://localhost:8090'],
                ['RATELORD_FOLLOWER_ID', 'Unique follower ID when in follower mode.', 'hostname'],
                ['RATELORD_ADVERTISED_URL', 'Public URL this node broadcasts.', 'http://localhost:{port}'],
                ['RATELORD_ARCHIVE_ENABLED', 'Enable cold storage archiving of events.', 'false'],
                ['RATELORD_ARCHIVE_RETENTION', 'Retention before archiving.', '720h'],
                ['RATELORD_BLOB_PATH', 'Filesystem path for blob storage.', './blobs'],
              ].map(([name, desc, def]) => (
                <tr key={name} className="border-t border-border">
                  <td className="px-2 py-3 font-mono text-xs text-primary">{name}</td>
                  <td className="px-2 py-3">{desc}</td>
                  <td className="px-2 py-3 font-mono text-xs">{def}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        <h2 className="text-2xl font-bold mt-10 mb-4">Policy Configuration</h2>
        <p>
          The policy file defines providers, pools, and codifies enforcement rules. Run the daemon with `ratelord-d --policy policy.yaml` to load a custom policy.
        </p>
        <p className="text-muted-foreground">
          The document has four primary sections: <strong>providers</strong>, <strong>policies</strong>, optional <strong>pricing</strong>, and optional <strong>retention</strong> rules.
        </p>
        <CodeBlock
          language="yaml"
          code={`policies:
  - id: "github-core-hard-limit"
    scope: "global"
    type: "hard"
    limit: 5000
    rules:
      - name: "deny-exceeded-limit"
        condition: "remaining < 0"
        action: "deny"
        params:
          reason: "hard limit exceeded for GitHub core requests"

providers:
  github:
    - id: "github-main"
      token_env_var: "GITHUB_TOKEN"`}
        />

        <h2 className="text-2xl font-bold mt-10 mb-4">Provider Configuration</h2>
        <p>
          Providers describe how the daemon fetches usage data from third parties. Each provider type (GitHub, OpenAI, etc.) defines connection details and credentials.
        </p>

        <h3 className="text-xl font-bold mt-6 mb-3">GitHub</h3>
        <p>Tracks core, search, and GraphQL rate limits. Use a Personal Access Token stored in an environment variable.</p>
        <CodeBlock
          language="yaml"
          code={`providers:
  github:
    - id: "my-org-github"
      token_env_var: "GITHUB_TOKEN"
      enterprise_url: "https://github.example.com/api/v3"`}
        />

        <h3 className="text-xl font-bold mt-6 mb-3">OpenAI</h3>
        <p>Monitors RPM/TPM quotas and allows custom base URLs for Azure or proxying.</p>
        <CodeBlock
          language="yaml"
          code={`providers:
  openai:
    - id: "openai-prod"
      api_key_env_var: "OPENAI_API_KEY"
      org_id: "org-12345"
      base_url: "https://api.openai.com"`}
        />
        <p className="text-muted-foreground">
          `api_key_env_var` supplies the API key, `org_id` enables organization-level tracking, and `base_url` can point to custom Azure endpoints or local proxies.
        </p>
      </div>
    </div>
  )
}
