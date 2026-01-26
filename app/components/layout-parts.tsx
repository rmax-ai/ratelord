import Link from 'next/link';

export function Header() {
  return (
    <header className="sticky top-0 z-50 w-full border-b border-border bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="container mx-auto flex h-16 items-center justify-between px-4 sm:px-8">
        <Link href="/" className="mr-6 flex items-center space-x-2 font-bold text-xl">
          <span>Ratelord</span>
        </Link>
        <nav className="hidden md:flex items-center space-x-6 text-sm font-medium">
          <Link href="#features" className="transition-colors hover:text-foreground/80 text-foreground/60">Features</Link>
          <Link href="#docs" className="transition-colors hover:text-foreground/80 text-foreground/60">Documentation</Link>
          <Link href="#about" className="transition-colors hover:text-foreground/80 text-foreground/60">About</Link>
        </nav>
        <div className="flex items-center space-x-4">
          <Link
            href="https://github.com/rmax-ai/ratelord"
            target="_blank"
            rel="noreferrer"
            className="text-sm font-medium transition-colors hover:text-foreground/80 text-foreground/60"
          >
            GitHub
          </Link>
        </div>
      </div>
    </header>
  );
}

export function Footer() {
  return (
    <footer className="border-t border-border bg-muted/50 py-12">
      <div className="container mx-auto px-4 sm:px-8 flex flex-col md:flex-row justify-between items-center gap-6">
        <div className="flex flex-col gap-2">
          <span className="text-lg font-bold">Ratelord</span>
          <p className="text-sm text-muted-foreground">
            Budget-Literate Autonomy for Agentic Systems.
          </p>
        </div>
        <div className="flex gap-6 text-sm text-muted-foreground">
          <Link href="#" className="hover:underline">Privacy</Link>
          <Link href="#" className="hover:underline">Terms</Link>
          <Link href="https://github.com/rmax-ai/ratelord" className="hover:underline">GitHub</Link>
        </div>
        <div className="text-sm text-muted-foreground">
          &copy; {new Date().getFullYear()} Ratelord Project.
        </div>
      </div>
    </footer>
  );
}
