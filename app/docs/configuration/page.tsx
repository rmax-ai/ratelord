import { CodeBlock } from '@/app/components/code-block';

export const metadata = {
  title: 'Configuration - Ratelord',
  description: 'Configuring Ratelord policies and providers.',
}

export default function Configuration() {
  return (
    <div className="container mx-auto px-4 py-12 max-w-4xl">
      <h1 className="text-4xl font-bold mb-6">Configuration</h1>

      <div className="prose prose-slate max-w-none">
        <p className="lead text-xl text-muted-foreground mb-8">
          Ratelord is configured via a YAML file (`ratelord.yaml`) or environment variables.
        </p>

        <h2 className="text-2xl font-bold mt-8 mb-4">Basic Configuration</h2>
        <CodeBlock
          language="yaml"
          code={`server:
  port: 8090
  host: "127.0.0.1"

storage:
  type: "sqlite"
  path: "./data/ratelord.db"

providers:
  github:
    enabled: true
    tokens:
      - "\${GITHUB_TOKEN}"`}
        />

        <h2 className="text-2xl font-bold mt-8 mb-4">Policies</h2>
        <p>
          Policies define how resources are allocated.
        </p>
        <CodeBlock
          language="yaml"
          code={`policies:
  - name: "strict-limit"
    scope: "global"
    rules:
      - resource: "requests"
        limit: 5000
        period: "1h"`}
        />
      </div>
    </div>
  )
}
