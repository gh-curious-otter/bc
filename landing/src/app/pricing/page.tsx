"use client";

import Link from "next/link";
import { useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { Nav } from "../_components/Nav";
import { Footer } from "../_components/Footer";
import {
  Server,
  Cloud,
  Terminal,
  Key,
  Lock,
  Shield,
  Database,
  Container,
  Copy,
  Check,
  ChevronDown,
  Apple,
  GitBranch,
  Cpu,
  ExternalLink,
} from "lucide-react";

const fadeUp = {
  hidden: { opacity: 0, y: 30 },
  visible: (i: number) => ({
    opacity: 1,
    y: 0,
    transition: { delay: i * 0.12, duration: 0.6, ease: "easeOut" as const },
  }),
};

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    void navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <button
      onClick={handleCopy}
      className="shrink-0 p-1 rounded hover:bg-accent/30 transition-colors"
      aria-label="Copy to clipboard"
    >
      {copied ? (
        <Check className="h-3.5 w-3.5 text-success" />
      ) : (
        <Copy className="h-3.5 w-3.5 text-muted-foreground" />
      )}
    </button>
  );
}

interface InstallCardProps {
  icon: React.ComponentType<{ className?: string }>;
  title: string;
  command: string;
}

function InstallCard({ icon: Icon, title, command }: InstallCardProps) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div
      className="group rounded-lg border border-border/60 bg-accent/5 hover:bg-accent/15 transition-all cursor-pointer"
      onClick={() => setExpanded(!expanded)}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") setExpanded(!expanded);
      }}
      role="button"
      tabIndex={0}
      aria-expanded={expanded}
    >
      <div className="flex items-center gap-3 px-4 py-3">
        <Icon className="h-4 w-4 text-muted-foreground shrink-0" />
        <span className="text-sm font-medium">{title}</span>
        <ChevronDown
          className={`h-3.5 w-3.5 text-muted-foreground/50 ml-auto transition-transform ${expanded ? "rotate-180" : ""}`}
        />
      </div>
      <AnimatePresence>
        {expanded && (
          <motion.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.2 }}
            className="overflow-hidden"
          >
            <div className="px-4 pb-3">
              <div className="flex items-center gap-2 rounded-md bg-background/80 border border-border/40 px-3 py-2 font-mono text-xs">
                <code className="text-muted-foreground flex-1 overflow-x-auto whitespace-nowrap">
                  {command}
                </code>
                <CopyButton text={command} />
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}

interface FAQItemProps {
  question: string;
  answer: string;
}

function FAQItem({ question, answer }: FAQItemProps) {
  const [open, setOpen] = useState(false);

  return (
    <div className="border-b border-border/40">
      <button
        onClick={() => setOpen(!open)}
        className="flex w-full items-center justify-between py-5 text-left"
        aria-expanded={open}
      >
        <span className="font-semibold text-sm">{question}</span>
        <ChevronDown
          className={`h-4 w-4 text-muted-foreground shrink-0 transition-transform ${open ? "rotate-180" : ""}`}
        />
      </button>
      <AnimatePresence>
        {open && (
          <motion.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.2 }}
            className="overflow-hidden"
          >
            <p className="pb-5 text-sm text-muted-foreground leading-relaxed">
              {answer}
            </p>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}

export default function PricingPage() {
  return (
    <main className="min-h-screen selection:bg-primary/20 selection:text-foreground">
      <div className="pointer-events-none fixed inset-0 z-[1] bg-[radial-gradient(ellipse_80%_60%_at_50%_-20%,rgba(234,88,12,0.04),transparent)] dark:bg-[radial-gradient(ellipse_80%_60%_at_50%_-20%,rgba(234,88,12,0.08),transparent)]" />

      <div className="relative z-[2]">
        <Nav />

        <div className="mx-auto max-w-5xl px-4 sm:px-6 py-16 sm:py-24">
          <motion.div
            initial="hidden"
            animate="visible"
            className="text-center mb-16"
          >
            <motion.span
              variants={fadeUp}
              custom={0}
              className="font-mono text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground"
            >
              Pricing
            </motion.span>
            <motion.h1
              variants={fadeUp}
              custom={1}
              className="mt-3 text-4xl font-bold tracking-tight sm:text-6xl"
            >
              Open source. Free forever.
            </motion.h1>
            <motion.p
              variants={fadeUp}
              custom={2}
              className="mt-4 text-lg text-muted-foreground max-w-xl mx-auto"
            >
              bc runs on your machine. No login, no signup, no cloud dependency.
              You only pay for your AI API tokens.
            </motion.p>
          </motion.div>

          <motion.div
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true, margin: "-50px" }}
            className="grid gap-8 lg:grid-cols-3"
          >
            {/* Free Tier */}
            <motion.div
              variants={fadeUp}
              custom={0}
              className="rounded-xl border-2 border-primary bg-card p-8 relative shadow-lg shadow-primary/5"
            >
              <div className="absolute -top-3 left-6 px-3 py-0.5 rounded-full bg-primary text-primary-foreground text-xs font-semibold">
                Recommended
              </div>

              <div className="mb-6">
                <h2 className="text-xl font-bold">Free</h2>
                <div className="mt-4 flex items-baseline gap-1">
                  <span className="text-5xl font-bold tracking-tight">
                    &#8377;0
                  </span>
                  <span className="text-muted-foreground text-sm">
                    / forever
                  </span>
                </div>
                <p className="mt-3 text-sm text-muted-foreground">
                  Runs locally on your machine
                </p>
              </div>

              {/* Tech icons */}
              <div className="flex items-center gap-3 mb-6 text-muted-foreground">
                <span title="Docker" className="flex items-center gap-1 text-xs">
                  <Container className="h-4 w-4" aria-hidden="true" />
                </span>
                <span title="tmux" className="flex items-center gap-1 text-xs">
                  <Terminal className="h-4 w-4" aria-hidden="true" />
                </span>
                <span title="PostgreSQL" className="flex items-center gap-1 text-xs">
                  <Database className="h-4 w-4" aria-hidden="true" />
                </span>
                <span title="SQLite" className="flex items-center gap-1 text-xs">
                  <Server className="h-4 w-4" aria-hidden="true" />
                </span>
              </div>

              {/* Install cards */}
              <div className="space-y-2 mb-6">
                <InstallCard
                  icon={Apple}
                  title="macOS / Linux"
                  command="go install github.com/rpuneet/bc/cmd/bc@latest"
                />
                <InstallCard
                  icon={Container}
                  title="Docker"
                  command="docker pull bcinfra/bc:latest"
                />
                <InstallCard
                  icon={GitBranch}
                  title="Build from source"
                  command="git clone https://github.com/rpuneet/bc.git && cd bc && make build"
                />
              </div>

              <Link
                href="https://github.com/gh-curious-otter/bc"
                className="flex items-center justify-center gap-2 w-full rounded-lg bg-primary py-2.5 text-center text-sm font-semibold text-primary-foreground transition-all hover:opacity-90 active:scale-[0.98]"
              >
                Get Started
                <ExternalLink className="h-3.5 w-3.5" aria-hidden="true" />
              </Link>
            </motion.div>

            {/* Cloud Tier */}
            <motion.div
              variants={fadeUp}
              custom={1}
              className="rounded-xl border border-border bg-card p-8 relative"
            >
              <div className="mb-6">
                <h2 className="text-xl font-bold">Cloud</h2>
                <div className="mt-4 flex items-baseline gap-1">
                  <span className="text-5xl font-bold tracking-tight">
                    &#8377;1,000
                  </span>
                  <span className="text-muted-foreground text-sm">
                    / month
                  </span>
                </div>
                <p className="mt-3 text-sm text-muted-foreground">
                  Access your workspace from anywhere
                </p>
              </div>

              <ul className="space-y-3 mb-6">
                {[
                  "SSH into your workspace",
                  "Chat with agents remotely",
                  "Everything in Free, plus:",
                ].map((f) => (
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

              {/* Tech icons */}
              <div className="flex items-center gap-3 mb-6 text-muted-foreground">
                <span title="Kubernetes" className="flex items-center">
                  <Cpu className="h-4 w-4" aria-hidden="true" />
                </span>
                <span title="Cloud" className="flex items-center">
                  <Cloud className="h-4 w-4" aria-hidden="true" />
                </span>
                <span title="Terminal" className="flex items-center">
                  <Terminal className="h-4 w-4" aria-hidden="true" />
                </span>
                <span title="API Keys" className="flex items-center">
                  <Key className="h-4 w-4" aria-hidden="true" />
                </span>
              </div>

              <Link
                href="/waitlist"
                className="block w-full rounded-lg border border-border py-2.5 text-center text-sm font-semibold transition-colors hover:bg-accent/20 active:scale-[0.98]"
              >
                Join Waitlist
              </Link>
            </motion.div>

            {/* Enterprise Tier - Blurred */}
            <motion.div
              variants={fadeUp}
              custom={2}
              className="rounded-xl border border-border/30 bg-card p-8 relative overflow-hidden"
            >
              {/* Content behind blur */}
              <div className="blur-[6px] select-none pointer-events-none">
                <div className="mb-6">
                  <h2 className="text-xl font-bold">Enterprise</h2>
                  <div className="mt-4 flex items-baseline gap-1">
                    <span className="text-5xl font-bold tracking-tight">
                      Custom
                    </span>
                  </div>
                  <p className="mt-3 text-sm text-muted-foreground">
                    For teams at scale
                  </p>
                </div>

                <div className="flex items-center gap-3 mb-6 text-muted-foreground">
                  <Lock className="h-4 w-4" />
                  <Shield className="h-4 w-4" />
                  <Server className="h-4 w-4" />
                  <Key className="h-4 w-4" />
                </div>

                <ul className="space-y-3 mb-6">
                  {["SSO / SAML", "Audit logs", "Dedicated support", "Custom SLA"].map(
                    (f) => (
                      <li
                        key={f}
                        className="flex items-start gap-2.5 text-sm text-muted-foreground"
                      >
                        <Check className="h-4 w-4 shrink-0 mt-0.5" />
                        {f}
                      </li>
                    )
                  )}
                </ul>

                <div className="w-full rounded-lg border border-border py-2.5 text-center text-sm font-semibold">
                  Contact Sales
                </div>
              </div>

              {/* Frosted overlay */}
              <div className="absolute inset-0 bg-background/40 backdrop-blur-sm z-10 flex flex-col items-center justify-center gap-4">
                <span className="px-5 py-2 rounded-full border border-border/60 bg-card/80 text-sm font-semibold text-muted-foreground backdrop-blur-sm">
                  Coming soon
                </span>
                <a
                  href="mailto:skitzo@bc-infra.com"
                  className="text-xs text-muted-foreground/60 hover:text-primary transition-colors"
                >
                  Talk to us &rarr; skitzo@bc-infra.com
                </a>
              </div>
            </motion.div>
          </motion.div>

          {/* FAQ Section */}
          <motion.div
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true, margin: "-50px" }}
            className="mt-24 max-w-2xl mx-auto"
          >
            <motion.h2
              variants={fadeUp}
              custom={0}
              className="text-2xl font-bold tracking-tight text-center mb-8"
            >
              Frequently asked questions
            </motion.h2>
            <motion.div variants={fadeUp} custom={1}>
              <FAQItem
                question="Is bc really free?"
                answer="Yes. bc is open source and runs locally on your machine. All features are included with no restrictions. You only pay for AI API tokens from your chosen providers (Claude, Gemini, etc.)."
              />
              <FAQItem
                question="Do I need an account?"
                answer="No for the Free tier. Install bc, run bc init, and start orchestrating. No signup, no login, no telemetry. The Cloud tier requires a signup for remote access features."
              />
              <FAQItem
                question="What's included in Cloud?"
                answer="SSH access to your workspace, remote agent chat, hosted dashboard, team management features, and cloud-synced memory. Everything in the Free tier is included plus remote access capabilities."
              />
              <FAQItem
                question="What AI providers work with bc?"
                answer="bc supports Claude Code, Cursor, Gemini, Codex, Aider, OpenCode, and OpenClaw. You bring your own API keys and bc orchestrates agents across any combination of providers."
              />
            </motion.div>
          </motion.div>
        </div>

        <Footer />
      </div>
    </main>
  );
}
