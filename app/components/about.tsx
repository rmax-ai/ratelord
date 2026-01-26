export function About() {
  return (
    <section id="about" className="py-20 bg-background">
      <div className="container px-4 md:px-6 mx-auto">
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-12 items-center">
          <div className="space-y-6">
            <h2 className="text-3xl font-bold tracking-tighter md:text-4xl">
              The Vision
            </h2>
            <p className="text-lg text-muted-foreground">
              Autonomous agents today are "blind" to the economic and rate-limiting realities of the APIs they consume. They hit limits unexpectedly, burn through budgets, and fail to plan effectively.
            </p>
            <p className="text-lg text-muted-foreground">
              Ratelord introduces <strong>Budget-Literate Autonomy</strong>. By acting as a sensory organ for constraints, it allows agents to "feel" the pressure of limits before they are hit, enabling sophisticated negotiation and planning strategies.
            </p>
            <div className="space-y-4 pt-4">
              <h3 className="text-xl font-bold">Core Principles</h3>
              <ul className="space-y-2 text-muted-foreground list-disc pl-5">
                <li><strong>Local-First:</strong> No external SaaS dependencies for critical path decisions.</li>
                <li><strong>Daemon Authority:</strong> Centralized truth for resource state.</li>
                <li><strong>Negotiation Mandate:</strong> Agents must ask before they act.</li>
                <li><strong>Prediction:</strong> Reactive is not enough; we must forecast.</li>
              </ul>
            </div>
          </div>
          <div className="rounded-xl bg-muted p-8 border shadow-sm">
            <h3 className="text-xl font-bold mb-4">Current Status</h3>
            <div className="space-y-4">
              <div className="flex items-center justify-between border-b pb-2">
                <span className="font-medium">Phase</span>
                <span className="text-muted-foreground">5: Remediation</span>
              </div>
              <div className="flex items-center justify-between border-b pb-2">
                <span className="font-medium">Core Engine</span>
                <span className="text-muted-foreground">Go + SQLite</span>
              </div>
              <div className="flex items-center justify-between border-b pb-2">
                <span className="font-medium">Interfaces</span>
                <span className="text-muted-foreground">TUI & Web UI</span>
              </div>
              <div className="flex items-center justify-between pt-2">
                <span className="font-medium">Integrations</span>
                <span className="text-muted-foreground">GitHub API (In Progress)</span>
              </div>
            </div>

            <div className="mt-8 p-4 bg-primary/5 rounded-lg border border-primary/10">
              <h4 className="font-bold mb-2">Join the Development</h4>
              <p className="text-sm text-muted-foreground mb-4">
                We are actively looking for contributors to help build the future of agentic constraint management.
              </p>
              <a href="https://github.com/rmax-ai/ratelord" target="_blank" rel="noreferrer" className="text-sm font-medium text-primary hover:underline">
                View Open Issues &rarr;
              </a>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
