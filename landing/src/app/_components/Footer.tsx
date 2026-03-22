import Link from "next/link";

export function Footer() {
  return (
    <footer className="border-t border-border bg-accent/20">
      <div className="mx-auto max-w-7xl px-6 py-16">
        <div className="grid grid-cols-1 gap-12 md:grid-cols-5 mb-12">
          <div className="col-span-1 md:col-span-2 space-y-4">
            <div className="flex items-center gap-2">
              <span className="font-mono text-xl font-bold tracking-tighter text-foreground">
                bc/&gt;
              </span>
              <div className="h-4 w-2 bg-accent animate-[pulse_1.5s_infinite]" />
            </div>
            <p className="text-sm text-muted-foreground max-w-xs">
              Multi-agent orchestration for AI coding assistants. CLI-first.
              Agent-agnostic. Built for teams that ship.
            </p>
          </div>
          <div className="space-y-4">
            <h2 className="text-xs font-bold uppercase tracking-widest text-primary/40">
              Product
            </h2>
            <nav
              aria-label="Product links"
              className="flex flex-col gap-2 text-sm text-muted-foreground"
            >
              <Link
                href="/"
                className="hover:text-foreground transition-colors"
              >
                Home
              </Link>
              <Link
                href="/product"
                className="hover:text-foreground transition-colors"
              >
                Features
              </Link>
              <Link
                href="/docs"
                className="hover:text-foreground transition-colors"
              >
                Documentation
              </Link>
              <Link
                href="/waitlist"
                className="hover:text-foreground transition-colors"
              >
                Waitlist
              </Link>
            </nav>
          </div>
          <div className="space-y-4">
            <h2 className="text-xs font-bold uppercase tracking-widest text-primary/40">
              Community
            </h2>
            <nav
              aria-label="Community links"
              className="flex flex-col gap-2 text-sm text-muted-foreground"
            >
              <Link
                href="https://github.com/bcinfra1"
                className="hover:text-foreground transition-colors"
                target="_blank"
                rel="noopener noreferrer"
              >
                GitHub
              </Link>
              <Link
                href="https://twitter.com/bc_infra"
                className="hover:text-foreground transition-colors"
                target="_blank"
                rel="noopener noreferrer"
              >
                Twitter / X
              </Link>
              <Link
                href="https://linkedin.com/company/bc-infra"
                className="hover:text-foreground transition-colors"
                target="_blank"
                rel="noopener noreferrer"
              >
                LinkedIn
              </Link>
              <Link
                href="https://discord.gg/bc-infra"
                className="hover:text-foreground transition-colors"
                target="_blank"
                rel="noopener noreferrer"
              >
                Discord
              </Link>
            </nav>
          </div>
          <div className="space-y-4">
            <h2 className="text-xs font-bold uppercase tracking-widest text-primary/40">
              Company
            </h2>
            <nav
              aria-label="Company links"
              className="flex flex-col gap-2 text-sm text-muted-foreground"
            >
              <Link
                href="mailto:puneet@bc-infra.com"
                className="hover:text-foreground transition-colors"
              >
                Contact
              </Link>
              <Link
                href="/privacy"
                className="hover:text-foreground transition-colors"
              >
                Privacy
              </Link>
              <Link
                href="/terms"
                className="hover:text-foreground transition-colors"
              >
                Terms
              </Link>
            </nav>
          </div>
        </div>
        <div className="flex flex-col md:flex-row items-center justify-between gap-4 pt-8 border-t border-border text-xs text-muted-foreground/60">
          <p>&copy; 2026 bc-infra. All rights reserved.</p>
          <div className="flex items-center gap-6">
            <Link
              href="/privacy"
              className="hover:text-foreground transition-colors"
            >
              Privacy Policy
            </Link>
            <Link
              href="/terms"
              className="hover:text-foreground transition-colors"
            >
              Terms of Service
            </Link>
          </div>
        </div>
      </div>
    </footer>
  );
}
