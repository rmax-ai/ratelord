import { Zap, Shield, GitBranch, BarChart3, Lock, Server } from "lucide-react";

export function Features() {
  const features = [
    {
      title: "Local-First Architecture",
      description: "Zero-ops daemon that runs alongside your agents, providing low-latency constraint management without external dependencies.",
      icon: <Server className="h-6 w-6" />,
    },
    {
      title: "Event-Sourced System",
      description: "Complete audit trail of all constraint decisions and budget allocations, fully replayable for debugging and analysis.",
      icon: <GitBranch className="h-6 w-6" />,
    },
    {
      title: "Predictive Modeling",
      description: "Time-to-exhaustion forecasts help agents plan their activities based on real-time resource availability.",
      icon: <BarChart3 className="h-6 w-6" />,
    },
    {
      title: "Intent Negotiation",
      description: "Standardized protocol for agents to request resources and negotiate budgets before execution.",
      icon: <Zap className="h-6 w-6" />,
    },
    {
      title: "Hierarchical Modeling",
      description: "Model complex constraint relationships with nested scopes and resource pools.",
      icon: <Shield className="h-6 w-6" />,
    },
    {
      title: "Provider Agnostic",
      description: "Extensible design that supports any API or resource type, starting with GitHub API integration.",
      icon: <Lock className="h-6 w-6" />,
    },
  ];

  return (
    <section id="features" className="py-20 bg-muted/30">
      <div className="container px-4 md:px-6 mx-auto">
        <div className="flex flex-col items-center justify-center space-y-4 text-center mb-12">
          <h2 className="text-3xl font-bold tracking-tighter md:text-4xl">
            Core Capabilities
          </h2>
          <p className="max-w-[700px] text-muted-foreground">
            Built to solve the "blind agent" problem by making resource constraints a first-class citizen in autonomous systems.
          </p>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
          {features.map((feature, index) => (
            <div key={index} className="flex flex-col space-y-3 p-6 rounded-lg border bg-card text-card-foreground shadow-sm">
              <div className="p-2 w-fit rounded-full bg-primary/10 text-primary">
                {feature.icon}
              </div>
              <h3 className="text-xl font-bold">{feature.title}</h3>
              <p className="text-muted-foreground">{feature.description}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
