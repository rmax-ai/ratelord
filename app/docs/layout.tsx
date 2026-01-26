import Link from 'next/link';
import { ChevronRight, Home } from 'lucide-react';

export default function DocsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col min-h-screen">
      <div className="border-b bg-muted/30">
        <div className="container mx-auto px-4 py-4 flex items-center space-x-2 text-sm text-muted-foreground overflow-x-auto whitespace-nowrap">
          <Link href="/" className="hover:text-foreground flex items-center">
            <Home className="h-4 w-4 mr-1" />
            Home
          </Link>
          <ChevronRight className="h-4 w-4" />
          <Link href="/#docs" className="hover:text-foreground">
            Documentation
          </Link>
        </div>
      </div>
      <main className="flex-1">
        {children}
      </main>
    </div>
  );
}
