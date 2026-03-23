"use client";

import Link from "next/link";
import { Nav } from "../_components/Nav";
import { Footer } from "../_components/Footer";
import { Check, ChevronDown } from "lucide-react";
import { motion } from "framer-motion";
import { useState } from "react";
import { RevealSection } from "../_components/TerminalComponents";

export default function PricingPage() {
  return (
    <main className="min-h-screen selection:bg-primary/20 selection:text-foreground">
      <Nav />

      <div className="mx-auto max-w-5xl px-4 sm:px-6 py-16 sm:py-24">
        <RevealSection className="text-center mb-16">
          <span className="font-mono text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
            Pricing
          </span>
          <h1 className="mt-3 text-4xl font-bold tracking-tight sm:text-6xl lg:text-7xl bg-gradient-to-b from-foreground to-foreground/70 bg-clip-text text-transparent">
            Open source. Free forever.
          </h1>
          <p className="mt-4 text-lg text-muted-foreground max-w-xl mx-auto">
            bc runs on your machine. No login, no signup, no cloud dependency.
            Every feature included.
          </p>
        </RevealSection>

        <div className="grid gap-8 lg:grid-cols-3">
          {/* Free Tier */}
          <RevealSection delay={0.1}>
            <div className="pricing-card-free rounded-xl p-8 relative h-full transition-all duration-300 hover:-translate-y-1 hover:shadow-[0_8px_30px_rgba(234,88,12,0.15)]">
              <div className="absolute -top-3.5 left-6 px-4 py-1 rounded-full bg-primary text-primary-foreground text-xs font-bold tracking-wide shadow-[0_2px_10px_rgba(234,88,12,0.4)]">
                Recommended
              </div>
              <div className="mb-6">
                <h2 className="text-xl font-bold">Open Source</h2>
                <div className="mt-4 flex items-baseline gap-1">
                  <span className="text-6xl sm:text-7xl font-extrabold tracking-tighter bg-gradient-to-br from-primary to-secondary bg-clip-text text-transparent">
                    ₹0
                  </span>
                  <span className="text-muted-foreground text-sm">/forever</span>
                </div>
                <p className="mt-3 text-sm text-muted-foreground">
                  Runs locally on your machine. No account required.
                </p>
              </div>
              <Link
                href="https://github.com/bcinfra1/bc"
                className="block w-full rounded-lg bg-primary py-2.5 text-center text-sm font-semibold text-primary-foreground transition-all hover:opacity-90 active:scale-[0.98] shadow-[var(--btn-shadow)]"
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
          </RevealSection>

          {/* Cloud Tier - Coming Soon */}
          <RevealSection delay={0.2}>
            <div className="rounded-xl border border-border bg-card p-8 relative overflow-hidden h-full transition-all duration-300 hover:-translate-y-1 hover:shadow-[var(--card-shadow)]">
              <div className="absolute inset-0 bg-background/50 backdrop-blur-sm z-10 flex items-center justify-center">
                <span className="px-5 py-2.5 rounded-full border border-primary/20 bg-card/80 backdrop-blur-md text-sm font-bold text-muted-foreground shadow-lg">
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
          </RevealSection>

          {/* Enterprise Tier - Coming Soon */}
          <RevealSection delay={0.3}>
            <div className="rounded-xl border border-border bg-card p-8 relative overflow-hidden h-full transition-all duration-300 hover:-translate-y-1 hover:shadow-[var(--card-shadow)]">
              <div className="absolute inset-0 bg-background/50 backdrop-blur-sm z-10 flex items-center justify-center">
                <span className="px-5 py-2.5 rounded-full border border-primary/20 bg-card/80 backdrop-blur-md text-sm font-bold text-muted-foreground shadow-lg">
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
          </RevealSection>
        </div>

        {/* FAQ Section */}
        <div className="mt-24 text-center">
          <RevealSection>
            <h2 className="text-2xl font-bold tracking-tight">
              Frequently asked questions
            </h2>
          </RevealSection>
          <div className="mt-12 max-w-3xl mx-auto text-left">
            {FAQS.map((faq, i) => (
              <RevealSection key={faq.q} delay={i * 0.05}>
                <FaqItem question={faq.q} answer={faq.a} />
              </RevealSection>
            ))}
          </div>
        </div>
      </div>

      <Footer />
    </main>
  );
}

function FaqItem({ question, answer }: { question: string; answer: React.ReactNode }) {
  const [open, setOpen] = useState(false);

  return (
    <div className="border-b border-border">
      <button
        onClick={() => setOpen(!open)}
        className="flex w-full items-center justify-between py-5 text-left"
        aria-expanded={open}
      >
        <span className="font-bold text-sm">{question}</span>
        <motion.span
          animate={{ rotate: open ? 180 : 0 }}
          transition={{ duration: 0.2 }}
          className="ml-4 text-muted-foreground shrink-0"
        >
          <ChevronDown size={18} aria-hidden="true" />
        </motion.span>
      </button>
      <motion.div
        initial={false}
        animate={{
          height: open ? "auto" : 0,
          opacity: open ? 1 : 0,
        }}
        transition={{ duration: 0.25, ease: "easeInOut" }}
        className="overflow-hidden"
      >
        <p className="pb-5 text-sm text-muted-foreground leading-relaxed">
          {answer}
        </p>
      </motion.div>
    </div>
  );
}

const FREE_FEATURES = [
  "Unlimited agents",
  "All 7 AI tool providers",
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

const FAQS = [
  {
    q: "Is bc really free?",
    a: "Yes. bc is open source. All features included, no restrictions. You only pay for AI API tokens from your providers.",
  },
  {
    q: "Do I need to create an account?",
    a: (
      <>
        No. bc is a local CLI tool. Install it, run{" "}
        <code className="text-xs bg-muted px-1.5 py-0.5 rounded">bc init</code>,
        and start orchestrating. No signup, no login, no telemetry.
      </>
    ),
  },
  {
    q: "What about Cloud and Enterprise?",
    a: (
      <>
        Hosted versions for teams are in development. Join the waitlist or email{" "}
        <a href="mailto:skitzo@bc-infra.com" className="text-primary hover:underline">
          skitzo@bc-infra.com
        </a>
        .
      </>
    ),
  },
  {
    q: "What AI providers are supported?",
    a: "bc supports Claude Code, Cursor, Gemini, Codex, Aider, OpenCode, and OpenClaw. You bring your own API keys.",
  },
];
