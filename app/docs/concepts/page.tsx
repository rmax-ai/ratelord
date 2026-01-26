export const metadata = {
  title: 'Core Concepts - Ratelord',
  description: 'Understand the fundamental concepts of Ratelord.',
}

export default function Concepts() {
  return (
    <div className="container mx-auto px-4 py-12 max-w-4xl">
      <h1 className="text-4xl font-bold mb-6">Core Concepts</h1>

      <div className="prose prose-slate max-w-none">
        <h2 className="text-2xl font-bold mt-8 mb-4">Constraint Graph</h2>
        <p>
          Ratelord models constraints as a directed graph. Resources are nodes in this graph, and relationships (like "belongs to" or "consumes") are edges. This allows for complex modeling of nested budgets.
        </p>

        <h2 className="text-2xl font-bold mt-8 mb-4">Identities & Scopes</h2>
        <ul className="list-disc pl-5 space-y-2">
          <li><strong>Identity:</strong> A unique actor in the system (e.g., a specific agent script, a user, or a microservice).</li>
          <li><strong>Scope:</strong> A logical grouping of resources. Scopes can be hierarchical (e.g., `org:acme` -&gt; `team:engineering` -&gt; `project:backend`).</li>
        </ul>

        <h2 className="text-2xl font-bold mt-8 mb-4">Resource Pools</h2>
        <p>
          A bucket of available units. Pools can be:
        </p>
        <ul className="list-disc pl-5 space-y-2">
          <li><strong>Renewable:</strong> Refills over time (e.g., API rate limits).</li>
          <li><strong>Depletable:</strong> A fixed budget that runs out (e.g., prepaid token credits).</li>
        </ul>

        <h2 className="text-2xl font-bold mt-8 mb-4">Intent Negotiation</h2>
        <p>
          The core protocol of Ratelord. Agents do not just "take" resources; they negotiate for them.
        </p>
        <ol className="list-decimal pl-5 space-y-2">
          <li><strong>Propose:</strong> Agent asks, "I need 100 tokens to run this query."</li>
          <li><strong>Evaluate:</strong> Ratelord checks available pools, policies, and priorities.</li>
          <li><strong>Grant/Reject/Counter:</strong> Ratelord responds. It might say "Granted", "Rejected (limit reached)", or "Counter: You can have 50 tokens now, or wait 10s for 100."</li>
        </ol>

        <h2 className="text-2xl font-bold mt-8 mb-4">Forecasts</h2>
        <p>
          Ratelord calculates <strong>Time-to-Exhaustion (TTE)</strong> for every pool. This allows agents to make informed decisions: "If I continue at this rate, I will be blocked in 5 minutes. I should slow down now."
        </p>
      </div>
    </div>
  )
}
