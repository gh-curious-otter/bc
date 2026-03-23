"use client";

import { useState } from "react";
import Link from "next/link";
import { Nav } from "../_components/Nav";
import { Footer } from "../_components/Footer";
import {
  Copy,
  Check,
  Plus,
  KeyRound,
  Server,
  Wrench,
  Globe,
} from "lucide-react";

function KubernetesIcon({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="currentColor"
      className={className}
      aria-hidden="true"
    >
      <path d="M12 1.5l-9.5 5.5v11l9.5 5.5 9.5-5.5v-11L12 1.5zm0 2.31l6.87 3.97L12 11.74 5.13 7.78 12 3.81zm-7.5 5.5l6.5 3.76v7.52l-6.5-3.76V9.31zm15 0v7.52l-6.5 3.76v-7.52l6.5-3.76z" />
    </svg>
  );
}

interface InstallOption {
  id: string;
  label: string;
  icon: React.ReactNode;
  command: string;
}

const INSTALL_OPTIONS: InstallOption[] = [
  {
    id: "macos",
    label: "macOS / Linux",
    icon: (
      <svg
        viewBox="0 0 24 24"
        fill="currentColor"
        className="h-6 w-6"
        aria-hidden="true"
      >
        <path d="M3 3h8v8H3V3zm2 2v4h4V5H5zm8-2h8v8h-8V3zm2 2v4h4V5h-4zM3 13h8v8H3v-8zm2 2v4h4v-4H5zm11-2h2v3h3v2h-3v3h-2v-3h-3v-2h3v-3z" />
      </svg>
    ),
    command: "go install github.com/rpuneet/bc/cmd/bc@latest",
  },
  {
    id: "docker",
    label: "Docker",
    icon: (
      <svg
        viewBox="0 0 24 24"
        fill="currentColor"
        className="h-6 w-6"
        aria-hidden="true"
      >
        <path d="M13 3v2h-2V3h2zm-4 0v2H7V3h2zM5 3v2H3V3h2zm8 4v2h-2V7h2zm-4 0v2H7V7h2zM5 7v2H3V7h2zm12 0v2h-2V7h2zm-4 4v2h-2v-2h2zm-4 0v2H7v-2h2zM5 11v2H3v-2h2zm12 0v2h-2v-2h2zm4 0c0 3-2.5 5.5-5.5 5.8-.3 1.7-1.2 3.2-2.5 4.2h-1c-1.3-1-2.2-2.5-2.5-4.2C2.5 16.5 0 14 0 11h24z" />
      </svg>
    ),
    command: "docker pull bcinfra/bc:latest",
  },
  {
    id: "source",
    label: "Source",
    icon: (
      <svg
        viewBox="0 0 24 24"
        fill="currentColor"
        className="h-6 w-6"
        aria-hidden="true"
      >
        <path d="M9.4 16.6L4.8 12l4.6-4.6L8 6l-6 6 6 6 1.4-1.4zm5.2 0l4.6-4.6-4.6-4.6L16 6l6 6-6 6-1.4-1.4z" />
      </svg>
    ),
    command: "git clone https://github.com/rpuneet/bc && cd bc && make build",
  },
];

function InstallCard({ option }: { option: InstallOption }) {
  const [copied, setCopied] = useState(false);
  const [expanded, setExpanded] = useState(false);

  function copy() {
    navigator.clipboard.writeText(option.command);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  return (
    <div
      className="group rounded-xl border border-border bg-card/80 backdrop-blur-sm p-6 cursor-pointer transition-all hover:border-primary/30 hover:shadow-lg hover:shadow-primary/5"
      onClick={() => setExpanded(!expanded)}
      role="button"
      tabIndex={0}
      aria-expanded={expanded}
      aria-label={`Install via ${option.label}`}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          setExpanded(!expanded);
        }
      }}
    >
      <div className="flex flex-col items-center gap-3 text-center">
        <div className="text-primary/70">{option.icon}</div>
        <h3 className="font-semibold text-sm">{option.label}</h3>
      </div>
      {expanded && (
        <div className="mt-4 flex items-center gap-2 rounded-lg bg-terminal-bg p-3">
          <code className="flex-1 text-xs font-mono text-terminal-text truncate">
            {option.command}
          </code>
          <button
            onClick={(e) => {
              e.stopPropagation();
              copy();
            }}
            className="shrink-0 p-1.5 rounded hover:bg-accent/20 transition-colors"
            aria-label={`Copy ${option.label} install command`}
          >
            {copied ? (
              <Check className="h-3.5 w-3.5 text-success" />
            ) : (
              <Copy className="h-3.5 w-3.5 text-muted-foreground" />
            )}
          </button>
        </div>
      )}
    </div>
  );
}

export default function InstallPage() {
  return (
    <main className="min-h-screen selection:bg-primary/20 selection:text-foreground">
      <Nav />

      <div className="mx-auto max-w-3xl px-4 sm:px-6 py-16 sm:py-24">
        <div className="text-center mb-16">
          <span className="font-mono text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
            Get Started
          </span>
          <h1 className="mt-3 text-4xl font-bold tracking-tight sm:text-6xl">
            Install bc
          </h1>
          <p className="mt-4 text-lg text-muted-foreground max-w-xl mx-auto">
            Pick your platform. Three commands to your first agent team.
          </p>
        </div>

        {/* Install options */}
        <div className="grid gap-4 sm:grid-cols-3">
          {INSTALL_OPTIONS.map((opt) => (
            <InstallCard key={opt.id} option={opt} />
          ))}
        </div>

        {/* More options link */}
        <div className="mt-6 flex justify-center">
          <Link
            href="/docs#installation"
            className="inline-flex items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground hover:bg-accent/20"
            aria-label="More installation options"
          >
            <Plus className="h-4 w-4" aria-hidden="true" />
            More options
          </Link>
        </div>

        {/* Coming soon teaser */}
        <div className="mt-16 relative overflow-hidden rounded-2xl border border-border/50 bg-card/40 backdrop-blur-md p-8 sm:p-12">
          {/* Frosted blur overlay */}
          <div className="absolute inset-0 bg-gradient-to-br from-primary/[0.03] via-transparent to-info/[0.03] pointer-events-none" />
          <div className="relative flex flex-col items-center text-center gap-6">
            <div className="flex items-center gap-4 text-muted-foreground/30">
              <KeyRound className="h-5 w-5" aria-hidden="true" />
              <KubernetesIcon className="h-5 w-5" />
              <Server className="h-5 w-5" aria-hidden="true" />
              <Globe className="h-5 w-5" aria-hidden="true" />
              <Wrench className="h-5 w-5" aria-hidden="true" />
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground/60">
                Cloud &middot; Enterprise &middot; Teams
              </p>
              <p className="mt-1 text-xs text-muted-foreground/40">
                More coming soon
              </p>
            </div>
            <a
              href="mailto:skitzo@bc-infra.com"
              className="text-xs text-muted-foreground/40 hover:text-primary transition-colors"
            >
              skitzo@bc-infra.com
            </a>
          </div>
        </div>

        {/* FAQ */}
        <div className="mt-24 text-center">
          <h2 className="text-2xl font-bold tracking-tight">
            Frequently asked questions
          </h2>
          <div className="mt-12 grid gap-8 sm:grid-cols-2 text-left max-w-3xl mx-auto">
            <div>
              <h3 className="font-semibold mb-2">
                What are the prerequisites?
              </h3>
              <p className="text-sm text-muted-foreground leading-relaxed">
                Go 1.25+, tmux, and git. For the web dashboard, you also need
                bun. bc runs on macOS and Linux.
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
                What AI providers are supported?
              </h3>
              <p className="text-sm text-muted-foreground leading-relaxed">
                bc supports Claude Code, Cursor, Gemini, Codex, Aider,
                OpenCode, and OpenClaw. You bring your own API keys.
              </p>
            </div>
            <div>
              <h3 className="font-semibold mb-2">
                What about Cloud and Enterprise?
              </h3>
              <p className="text-sm text-muted-foreground leading-relaxed">
                Hosted versions for teams are in development. Email{" "}
                <a
                  href="mailto:skitzo@bc-infra.com"
                  className="text-primary hover:underline"
                >
                  skitzo@bc-infra.com
                </a>{" "}
                for early access.
              </p>
            </div>
          </div>
        </div>
      </div>

      <Footer />
    </main>
  );
}
