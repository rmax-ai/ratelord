import Link from 'next/link';

export function Hero() {
  return (
    <section className="relative overflow-hidden py-24 lg:py-32 bg-background">
      <div className="container px-4 md:px-6 mx-auto">
        <div className="flex flex-col items-center text-center space-y-8">
          <div className="inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 border-transparent bg-primary text-primary-foreground hover:bg-primary/80">
            Active Development - Phase 5
          </div>
          <h1 className="text-4xl font-extrabold tracking-tight lg:text-5xl max-w-3xl">
            Budget-Literate Autonomy for Agentic Systems
          </h1>
          <p className="mx-auto max-w-[700px] text-lg text-muted-foreground">
            Ratelord provides a "sensory organ" and "prefrontal cortex" for resource availability and budget planning, enabling systems to negotiate, forecast, govern, and adapt under constraints.
          </p>
          <div className="flex flex-wrap items-center justify-center gap-4">
            <Link
              href="#docs"
              className="inline-flex h-10 items-center justify-center rounded-md bg-primary px-8 text-sm font-medium text-primary-foreground shadow transition-colors hover:bg-primary/90 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50"
            >
              Get Started
            </Link>
            <Link
              href="https://github.com/rmax-ai/ratelord"
              target="_blank"
              rel="noreferrer"
              className="inline-flex h-10 items-center justify-center rounded-md border border-input bg-background px-8 text-sm font-medium shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50"
            >
              GitHub Repo
            </Link>
          </div>
        </div>
      </div>
    </section>
  );
}
