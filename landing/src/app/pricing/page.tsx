import Link from "next/link";
import { Nav } from "../_components/Nav";
import { Footer } from "../_components/Footer";
import { Check } from "lucide-react";

export const metadata = {
  title: "Pricing - bc",
  description:
    "bc is free and open source. Run it locally with all features. No login required.",
};

const FREE_FEATURES = [
  "Unlimited agents",
  "All 7 AI tool providers",
  "CLI, TUI, and Web UI",
  "Git worktree isolation",
  "Channel communication",
  "Cost tracking and budgets",
  "Persistent memory",
  "Role-based hierarchy",
  "Cron scheduling",
  "MCP integration",
  "Secrets management",
  "Event logs and stats",
];

const CLOUD_FEATURES = [
  "Everything in Free",
  "Hosted dashboard",
  "Team management",
  "Usage analytics",
  "Cloud-synced memory",
  "Webhooks and integrations",
];

const ENTERPRISE_FEATURES = [
  "Everything in Cloud",
  "SSO / SAML",
  "Audit logs",
  "Dedicated support",
  "Custom SLA",
  "On-premise deployment",
];

export default function PricingPage() {
  return (
    <main className="min-h-screen selection:bg-primary/20 selection:text-foreground">
      <Nav />

      <div className="mx-auto max-w-5xl px-4 sm:px-6 py-16 sm:py-24">
        <div className="text-center mb-16">
          <span className="font-mono text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
            Pricing
          </span>
          <h1 className="mt-3 text-4xl font-bold tracking-tight sm:text-6xl">
            Open source. Free forever.
          </h1>
          <p className="mt-4 text-lg text-muted-foreground max-w-xl mx-auto">
            bc runs on your machine. No login, no signup, no cloud dependency.
            Every feature included.
          </p>
        </div>

        <div className="grid gap-8 lg:grid-cols-3">
          {/* Free Tier */}
          <div className="rounded-xl border-2 border-primary bg-card p-8 relative">
            <div className="absolute -top-3 left-6 px-3 py-0.5 rounded-full bg-primary text-primary-foreground text-xs font-semibold">
              Recommended
            </div>
            <div className="mb-6">
              <h2 className="text-xl font-bold">Open Source</h2>
              <div className="mt-4 flex items-baseline gap-1">
                <span className="text-5xl font-bold tracking-tight">₹0</span>
                <span className="text-muted-foreground text-sm">/forever</span>
              </div>
              <p className="mt-3 text-sm text-muted-foreground">
                Runs locally on your machine. No account required.
              </p>
            </div>
            <Link
              href="https://github.com/bcinfra1/bc"
              className="block w-full rounded-lg bg-primary py-2.5 text-center text-sm font-semibold text-primary-foreground transition-all hover:opacity-90 active:scale-[0.98]"
            >
              Get Started
            </Link>
            <ul className="mt-8 space-y-3">
              {FREE_FEATURES.map((f) => (
                <li
                  key={f}
                  className="flex items-start gap-2.5 text-sm text-muted-foreground"
                >
                  <Check
                    className="h-4 w-4 text-success shrink-0 mt-0.5"
                    aria-hidden="true"
                  />
                  {f}
                </li>
              ))}
            </ul>
          </div>

          {/* Cloud Tier - Coming Soon */}
          <div className="rounded-xl border border-border bg-card p-8 relative overflow-hidden">
            <div className="absolute inset-0 bg-background/60 backdrop-blur-[2px] z-10 flex items-center justify-center">
              <span className="px-4 py-2 rounded-full border border-border bg-card text-sm font-semibold text-muted-foreground">
                Coming Soon
              </span>
            </div>
            <div className="mb-6 opacity-40">
              <h2 className="text-xl font-bold">Cloud</h2>
              <div className="mt-4 flex items-baseline gap-1">
                <span className="text-5xl font-bold tracking-tight">—</span>
              </div>
              <p className="mt-3 text-sm text-muted-foreground">
                For teams that want hosted infrastructure.
              </p>
            </div>
            <div className="opacity-40">
              <button
                disabled
                className="block w-full rounded-lg border border-border py-2.5 text-center text-sm font-semibold text-muted-foreground cursor-not-allowed"
              >
                Join Waitlist
              </button>
              <ul className="mt-8 space-y-3">
                {CLOUD_FEATURES.map((f) => (
                  <li
                    key={f}
                    className="flex items-start gap-2.5 text-sm text-muted-foreground"
                  >
                    <Check
                      className="h-4 w-4 text-muted-foreground/40 shrink-0 mt-0.5"
                      aria-hidden="true"
                    />
                    {f}
                  </li>
                ))}
              </ul>
            </div>
          </div>

          {/* Enterprise Tier - Coming Soon */}
          <div className="rounded-xl border border-border bg-card p-8 relative overflow-hidden">
            <div className="absolute inset-0 bg-background/60 backdrop-blur-[2px] z-10 flex items-center justify-center">
              <span className="px-4 py-2 rounded-full border border-border bg-card text-sm font-semibold text-muted-foreground">
                Coming Soon
              </span>
            </div>
            <div className="mb-6 opacity-40">
              <h2 className="text-xl font-bold">Enterprise</h2>
              <div className="mt-4 flex items-baseline gap-1">
                <span className="text-5xl font-bold tracking-tight">—</span>
              </div>
              <p className="mt-3 text-sm text-muted-foreground">
                For organizations with compliance requirements.
              </p>
            </div>
            <div className="opacity-40">
              <button
                disabled
                className="block w-full rounded-lg border border-border py-2.5 text-center text-sm font-semibold text-muted-foreground cursor-not-allowed"
              >
                Contact Sales
              </button>
              <ul className="mt-8 space-y-3">
                {ENTERPRISE_FEATURES.map((f) => (
                  <li
                    key={f}
                    className="flex items-start gap-2.5 text-sm text-muted-foreground"
                  >
                    <Check
                      className="h-4 w-4 text-muted-foreground/40 shrink-0 mt-0.5"
                      aria-hidden="true"
                    />
                    {f}
                  </li>
                ))}
              </ul>
            </div>
          </div>
        </div>

        {/* FAQ-like section */}
        <div className="mt-24 text-center">
          <h2 className="text-2xl font-bold tracking-tight">
            Frequently asked questions
          </h2>
          <div className="mt-12 grid gap-8 sm:grid-cols-2 text-left max-w-3xl mx-auto">
            <div>
              <h3 className="font-semibold mb-2">
                Is bc really free?
              </h3>
              <p className="text-sm text-muted-foreground leading-relaxed">
                Yes. bc is open source and runs entirely on your local machine.
                All features are included with no restrictions. You only pay for
                the AI API tokens from your chosen providers (Claude, Gemini,
                etc.).
              </p>
            </div>
            <div>
              <h3 className="font-semibold mb-2">
                Do I need to create an account?
              </h3>
              <p className="text-sm text-muted-foreground leading-relaxed">
                No. bc is a local CLI tool. Install it, run{" "}
                <code className="text-xs bg-muted px-1.5 py-0.5 rounded">
                  bc init
                </code>
                , and start orchestrating. No signup, no login, no telemetry.
              </p>
            </div>
            <div>
              <h3 className="font-semibold mb-2">
                What about Cloud and Enterprise?
              </h3>
              <p className="text-sm text-muted-foreground leading-relaxed">
                We are working on hosted versions for teams who want managed
                infrastructure. Join the waitlist to be notified when they
                launch, or email us at{" "}
                <a href="mailto:skitzo@bc-infra.com" className="text-primary hover:underline">
                  skitzo@bc-infra.com
                </a>.
              </p>
            </div>
            <div>
              <h3 className="font-semibold mb-2">
                What AI providers are supported?
              </h3>
              <p className="text-sm text-muted-foreground leading-relaxed">
                bc supports Claude Code, Cursor, Gemini, Codex, Aider,
                OpenCode, and OpenClaw. You bring your own API keys.
              </p>
            </div>
          </div>
        </div>
      </div>

      <Footer />
    </main>
  );
}
