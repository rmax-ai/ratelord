import { Hero } from "@/app/components/hero";
import { Features } from "@/app/components/features";
import { About } from "@/app/components/about";
import Link from "next/link";

export default function Home() {
  return (
    <div className="flex flex-col min-h-screen">
      <Hero />
      <Features />
      <About />

      <section id="docs" className="py-20 bg-background border-t">
        <div className="container px-4 md:px-6 mx-auto text-center">
          <h2 className="text-3xl font-bold tracking-tighter md:text-4xl mb-6">
            Documentation
          </h2>
          <p className="max-w-[700px] mx-auto text-muted-foreground mb-10">
            Comprehensive guides for installing, configuring, and integrating Ratelord into your agentic systems.
          </p>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 text-left max-w-5xl mx-auto">
            <Link href="/docs/getting-started" className="p-6 bg-background rounded-lg border shadow-sm hover:shadow-md transition-shadow group">
              <h3 className="text-xl font-bold mb-2 group-hover:text-primary transition-colors">Getting Started</h3>
              <p className="text-muted-foreground mb-4">Installation instructions, basic setup, and first identity registration.</p>
              <span className="text-primary text-sm font-medium">Read Guide &rarr;</span>
            </Link>
            <Link href="/docs/architecture" className="p-6 bg-background rounded-lg border shadow-sm hover:shadow-md transition-shadow group">
              <h3 className="text-xl font-bold mb-2 group-hover:text-primary transition-colors">Architecture</h3>
              <p className="text-muted-foreground mb-4">Detailed breakdown of components: daemon, storage, clients, and providers.</p>
              <span className="text-primary text-sm font-medium">Explore Architecture &rarr;</span>
            </Link>
            <Link href="/docs/concepts" className="p-6 bg-background rounded-lg border shadow-sm hover:shadow-md transition-shadow group">
              <h3 className="text-xl font-bold mb-2 group-hover:text-primary transition-colors">Core Concepts</h3>
              <p className="text-muted-foreground mb-4">Understand constraint graphs, identities, scopes, pools, forecasts, and policies.</p>
              <span className="text-primary text-sm font-medium">Learn Concepts &rarr;</span>
            </Link>
            <Link href="/docs/api" className="p-6 bg-background rounded-lg border shadow-sm hover:shadow-md transition-shadow group">
              <h3 className="text-xl font-bold mb-2 group-hover:text-primary transition-colors">API Reference</h3>
              <p className="text-muted-foreground mb-4">HTTP endpoints and the intent negotiation protocol specification.</p>
              <span className="text-primary text-sm font-medium">View Reference &rarr;</span>
            </Link>
            <Link href="/docs/configuration" className="p-6 bg-background rounded-lg border shadow-sm hover:shadow-md transition-shadow group">
              <h3 className="text-xl font-bold mb-2 group-hover:text-primary transition-colors">Configuration</h3>
              <p className="text-muted-foreground mb-4">Setting up policy files, providers, and system parameters.</p>
              <span className="text-primary text-sm font-medium">Configure System &rarr;</span>
            </Link>
            <Link href="/docs/troubleshooting" className="p-6 bg-background rounded-lg border shadow-sm hover:shadow-md transition-shadow group">
              <h3 className="text-xl font-bold mb-2 group-hover:text-primary transition-colors">Troubleshooting</h3>
              <p className="text-muted-foreground mb-4">Common issues, log analysis, and debugging strategies.</p>
              <span className="text-primary text-sm font-medium">Get Help &rarr;</span>
            </Link>
          </div>
        </div>
      </section>
    </div>
  );
}
