"use client";

import { Nav } from "../_components/Nav";
import { Footer } from "../_components/Footer";
import {
  TerminalWindow,
  CommandOutput,
  RevealSection,
} from "../_components/TerminalComponents";
import { useState, useMemo } from "react";
import Link from "next/link";
import type { CommandGroup, CliCommand } from "@/lib/cli-docs";
import {
  GitBranch,
  MessageSquare,
  Brain,
  DollarSign,
  Users,
  Layers,
  ChevronDown,
  Copy,
  Check,
  Search,
  Shield,
  Clock,
  Terminal,
  Server,
  Wrench,
  Key,
  Stethoscope,
  Hash,
  Settings,
  FolderTree,
} from "lucide-react";

/* ── Copy button ── */
function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);
  return (
    <button
      onClick={() => {
        navigator.clipboard.writeText(text);
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
      }}
      className="absolute right-3 top-3 rounded-md border border-white/10 bg-white/5 p-1.5 text-terminal-muted hover:text-terminal-text transition-colors"
      aria-label="Copy to clipboard"
    >
      {copied ? (
        <Check
          className="h-3.5 w-3.5 text-terminal-success"
          aria-hidden="true"
        />
      ) : (
        <Copy className="h-3.5 w-3.5" aria-hidden="true" />
      )}
    </button>
  );
}

/* ── Code block with copy ── */
function CodeBlock({ code, title }: { code: string; title?: string }) {
  return (
    <div className="relative overflow-hidden rounded-xl border border-border bg-terminal-bg">
      {title && (
        <div className="border-b border-white/[0.06] bg-terminal-header px-4 py-2 text-[10px] font-bold uppercase tracking-[0.2em] text-terminal-muted">
          {title}
        </div>
      )}
      <div className="p-4 font-mono text-[13px] leading-relaxed text-terminal-text overflow-x-auto">
        <CopyButton text={code} />
        <pre>
          <code>{code}</code>
        </pre>
      </div>
    </div>
  );
}

/* ── Icon map for command groups ── */
const GROUP_ICONS: Record<
  string,
  React.ComponentType<{ className?: string }>
> = {
  agent: Users,
  workspace: Layers,
  channel: MessageSquare,
  tool: Wrench,
  secret: Key,
  cost: DollarSign,
  cron: Clock,
  role: Shield,
  mcp: Hash,
  doctor: Stethoscope,
  daemon: Server,
  config: Settings,
  env: FolderTree,
};

/* ── Collapsible command section ── */
function CommandSection({
  id,
  title,
  alias,
  count,
  icon: Icon,
  children,
  defaultOpen = false,
  visible = true,
}: {
  id?: string;
  title: string;
  alias?: string;
  count: number;
  icon?: React.ComponentType<{ className?: string }>;
  children: React.ReactNode;
  defaultOpen?: boolean;
  visible?: boolean;
}) {
  const [open, setOpen] = useState(defaultOpen);
  if (!visible) return null;
  return (
    <div id={id} className="border border-border rounded-xl overflow-hidden">
      <button
        onClick={() => setOpen(!open)}
        aria-expanded={open}
        className="flex w-full items-center justify-between px-5 py-4 text-left hover:bg-accent/50 transition-colors"
      >
        <div className="flex items-center gap-3">
          {Icon && (
            <Icon className="h-4 w-4 text-primary/60" aria-hidden="true" />
          )}
          <span className="font-semibold text-sm">{title}</span>
          {alias && (
            <span className="rounded bg-primary/10 px-1.5 py-0.5 text-[10px] font-mono font-bold text-primary">
              {alias}
            </span>
          )}
          <span className="rounded-full bg-muted px-2 py-0.5 text-[10px] font-bold text-muted-foreground">
            {count}
          </span>
        </div>
        <ChevronDown
          className={`h-4 w-4 text-muted-foreground transition-transform ${open ? "rotate-180" : ""}`}
          aria-hidden="true"
        />
      </button>
      {open && (
        <div className="border-t border-border px-5 py-4 bg-accent/20">
          {children}
        </div>
      )}
    </div>
  );
}

/* ── Concept card ── */
function ConceptCard({
  icon: Icon,
  title,
  desc,
}: {
  icon: React.ComponentType<{ className?: string }>;
  title: string;
  desc: string;
}) {
  return (
    <div className="rounded-xl border border-border bg-card p-5 transition-colors hover:border-primary/20">
      <Icon className="h-5 w-5 text-primary/60 mb-3" aria-hidden="true" />
      <h3 className="font-semibold text-sm mb-1.5">{title}</h3>
      <p className="text-sm text-muted-foreground leading-relaxed">{desc}</p>
    </div>
  );
}

/* ── Render a command group's subcommands ── */
function CommandGroupContent({ group }: { group: CommandGroup }) {
  // Build a usage summary from subcommands
  const lines = group.commands.map((cmd) => {
    const shortName = cmd.name;
    const paddedName = shortName.padEnd(40);
    return `${paddedName} # ${cmd.description}`;
  });
  const code = lines.join("\n");

  return (
    <div className="space-y-3">
      <CodeBlock code={code} />
      {group.commands.length > 0 && (
        <details className="text-sm">
          <summary className="cursor-pointer text-muted-foreground hover:text-foreground transition-colors font-medium py-1">
            Show detailed flags and options
          </summary>
          <div className="mt-3 space-y-4 pl-2 border-l-2 border-border">
            {group.commands.map((cmd) => (
              <SubcommandDetail key={cmd.name} cmd={cmd} />
            ))}
          </div>
        </details>
      )}
    </div>
  );
}

function SubcommandDetail({ cmd }: { cmd: CliCommand }) {
  // Filter out just the -h/--help line
  const meaningfulOptions = cmd.options
    .split("\n")
    .filter((l) => !l.match(/^\s*-h,\s+--help/) && l.trim())
    .join("\n")
    .trim();

  if (!meaningfulOptions) return null;

  return (
    <div>
      <h4 className="font-mono text-xs font-bold text-foreground mb-1">
        {cmd.name}
      </h4>
      {cmd.description && (
        <p className="text-xs text-muted-foreground mb-2">{cmd.description}</p>
      )}
      {meaningfulOptions && (
        <pre className="text-[11px] font-mono text-muted-foreground bg-muted/50 rounded-lg px-3 py-2 overflow-x-auto">
          <code>{meaningfulOptions}</code>
        </pre>
      )}
    </div>
  );
}

/* ── Standalone commands section ── */
function StandaloneCommandsContent({
  commands,
}: {
  commands: CliCommand[];
}) {
  const lines = commands.map((cmd) => {
    const paddedName = cmd.name.padEnd(40);
    return `${paddedName} # ${cmd.description}`;
  });
  return <CodeBlock code={lines.join("\n")} />;
}

/* ═══════════════════════════════════════════════════════════════════ */

export default function DocsContent({
  groups,
  standalone,
}: {
  groups: CommandGroup[];
  standalone: CliCommand[];
}) {
  const [search, setSearch] = useState("");

  // Build searchable index
  const searchableGroups = useMemo(() => {
    return groups.map((g) => ({
      ...g,
      keywords: [
        g.name,
        g.alias,
        g.description,
        ...g.commands.map((c) => c.name),
        ...g.commands.map((c) => c.description),
      ]
        .join(" ")
        .toLowerCase(),
    }));
  }, [groups]);

  const filteredGroupIds = useMemo(() => {
    if (!search.trim()) return searchableGroups.map((g) => g.id);
    const q = search.toLowerCase();
    return searchableGroups
      .filter((g) => g.keywords.includes(q))
      .map((g) => g.id);
  }, [search, searchableGroups]);

  const isVisible = (id: string) => filteredGroupIds.includes(id);

  return (
    <main className="min-h-screen selection:bg-primary/20 selection:text-foreground overflow-x-hidden">
      <div className="pointer-events-none fixed inset-0 bg-[radial-gradient(ellipse_80%_60%_at_50%_-20%,rgba(234,88,12,0.04),transparent)] dark:bg-[radial-gradient(ellipse_80%_60%_at_50%_-20%,rgba(234,88,12,0.08),transparent)]" />

      <Nav />

      <div className="mx-auto max-w-5xl px-6 py-16 lg:py-24">
        {/* ═══════════════════ HEADER ═══════════════════ */}
        <div className="mb-12">
          <span className="font-mono text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
            Documentation
          </span>
          <h1 className="mt-3 text-4xl font-bold tracking-tight sm:text-6xl">
            Documentation
          </h1>
          <p className="mt-4 max-w-2xl text-lg text-muted-foreground leading-relaxed">
            Everything you need to orchestrate AI agents from your terminal.
            CLI-first with a Web UI companion at localhost:9374.
          </p>
        </div>

        {/* ═══════════════════ SEARCH ═══════════════════ */}
        <div className="sticky top-0 z-20 -mx-6 px-6 py-4 bg-background/80 backdrop-blur-xl border-b border-border/50 mb-12">
          <div className="relative max-w-xl">
            <Search
              className="absolute left-4 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground"
              aria-hidden="true"
            />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search commands, features, or aliases..."
              className="h-12 w-full rounded-xl border border-border bg-card pl-11 pr-4 text-sm font-mono outline-none transition-all placeholder:text-muted-foreground/50 focus:border-primary/50 focus:ring-2 focus:ring-primary/10"
            />
            {search && (
              <button
                onClick={() => setSearch("")}
                className="absolute right-4 top-1/2 -translate-y-1/2 text-xs text-muted-foreground hover:text-foreground"
              >
                Clear
              </button>
            )}
          </div>
          {search && (
            <p className="mt-2 text-xs text-muted-foreground">
              {filteredGroupIds.length}{" "}
              {filteredGroupIds.length === 1 ? "group" : "groups"} matching
              &quot;{search}&quot;
            </p>
          )}
        </div>

        {/* ═══════════════════ NAV PILLS ═══════════════════ */}
        {!search && (
          <div className="flex flex-wrap gap-2 mb-12">
            {[
              { href: "#quickstart", label: "Quick Start" },
              { href: "#commands", label: "Commands" },
              { href: "#aliases", label: "Aliases" },
              { href: "#config", label: "Configuration" },
              { href: "#env", label: "Env Vars" },
            ].map((link) => (
              <a
                key={link.href}
                href={link.href}
                className="rounded-lg border border-border bg-card px-3 py-1.5 text-xs font-mono font-semibold text-muted-foreground hover:text-foreground hover:border-primary/30 transition-colors"
              >
                {link.label}
              </a>
            ))}
          </div>
        )}

        <div className="space-y-20">
          {/* ═══════════════════ QUICK START ═══════════════════ */}
          {!search && (
            <RevealSection id="quickstart">
              <h2 className="text-2xl font-bold tracking-tight mb-6">
                Quick Start
              </h2>
              <TerminalWindow title="quickstart">
                <CommandOutput
                  command="bc init"
                  lines={[
                    {
                      text: "Initializing bc workspace...",
                      color: "text-terminal-muted",
                    },
                    {
                      text: "Created .bc/settings.toml",
                      color: "text-terminal-muted",
                    },
                    { text: "Ready.", color: "text-terminal-success" },
                  ]}
                />
                <div className="mt-4">
                  <CommandOutput
                    command="bc daemon start"
                    delay={0.3}
                    lines={[
                      {
                        text: "Starting bcd server...",
                        color: "text-terminal-muted",
                      },
                      {
                        text: "  \u2713 Daemon running on localhost:9374",
                        color: "text-terminal-success",
                      },
                    ]}
                  />
                </div>
                <div className="mt-4">
                  <CommandOutput
                    command='bc agent create eng-01 --role engineer'
                    delay={0.6}
                    lines={[
                      {
                        text: "Creating agent eng-01...",
                        color: "text-terminal-muted",
                      },
                      {
                        text: "  \u2713 eng-01  engineer  created",
                        color: "text-terminal-success",
                      },
                    ]}
                  />
                </div>
                <div className="mt-4">
                  <CommandOutput
                    command="bc agent start eng-01"
                    delay={0.9}
                    lines={[
                      {
                        text: "Starting agent eng-01...",
                        color: "text-terminal-muted",
                      },
                      {
                        text: "  \u2713 eng-01  engineer  working",
                        color: "text-terminal-success",
                      },
                      {
                        text: "1 agent active.",
                        color: "text-terminal-muted",
                      },
                    ]}
                  />
                </div>
              </TerminalWindow>
            </RevealSection>
          )}

          {/* ═══════════════════ INSTALLATION ═══════════════════ */}
          {!search && (
            <RevealSection id="installation">
              <h2 className="text-2xl font-bold tracking-tight mb-6">
                Installation
              </h2>
              <div className="grid gap-4 sm:grid-cols-2">
                <CodeBlock
                  title="Go Install"
                  code="go install github.com/rpuneet/bc/cmd/bc@latest"
                />
                <CodeBlock
                  title="Build from Source"
                  code={`git clone https://github.com/rpuneet/bc.git
cd bc && make build`}
                />
              </div>
              <p className="mt-4 text-sm text-muted-foreground">
                Verify installation:{" "}
                <code className="rounded bg-muted px-1.5 py-0.5 text-xs font-mono">
                  bc doctor
                </code>
              </p>
            </RevealSection>
          )}

          {/* ═══════════════════ CORE CONCEPTS ═══════════════════ */}
          {!search && (
            <RevealSection id="concepts">
              <h2 className="text-2xl font-bold tracking-tight mb-6">
                Core Concepts
              </h2>
              <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                <ConceptCard
                  icon={Layers}
                  title="Workspaces"
                  desc="The .bc/ directory stores config, roles, agents, memory, and channel history. Managed via bc ws."
                />
                <ConceptCard
                  icon={Users}
                  title="Agents"
                  desc="AI instances with roles, tools, and states. Lifecycle: create \u2192 start \u2192 work \u2192 stop \u2192 delete."
                />
                <ConceptCard
                  icon={MessageSquare}
                  title="Channels"
                  desc="Structured coordination. Default: #eng, #pr, #standup, #leads. Supports @mentions and reactions."
                />
                <ConceptCard
                  icon={GitBranch}
                  title="Worktrees"
                  desc="Each agent gets an isolated git worktree. Zero merge conflicts by design."
                />
                <ConceptCard
                  icon={Brain}
                  title="Memory"
                  desc="Learnings (permanent) + Experiences (time-stamped). Injected on agent spawn across sessions."
                />
                <ConceptCard
                  icon={Key}
                  title="Secrets"
                  desc="Encrypted credential management. macOS Keychain, Linux libsecret, or AES-256-GCM fallback."
                />
                <ConceptCard
                  icon={DollarSign}
                  title="Cost Tracking"
                  desc="Per-agent token costs, budgets with alerts, and hard stops to prevent runaway spend."
                />
                <ConceptCard
                  icon={Clock}
                  title="Cron Jobs"
                  desc="Schedule recurring tasks with familiar cron syntax. Automate tests, deploys, reports."
                />
                <ConceptCard
                  icon={Server}
                  title="Daemon"
                  desc="Persistent bcd server. Eliminates write contention, enables the Web UI at localhost:9374."
                />
              </div>
            </RevealSection>
          )}

          {/* ═══════════════════ COMMAND REFERENCE ═══════════════════ */}
          <RevealSection id="commands">
            {!search && (
              <h2 className="text-2xl font-bold tracking-tight mb-6">
                Command Reference
              </h2>
            )}
            <div className="space-y-3">
              {groups.map((group) => {
                const Icon = GROUP_ICONS[group.name.toLowerCase()] || Terminal;
                return (
                  <CommandSection
                    key={group.id}
                    id={group.id}
                    title={group.name.charAt(0).toUpperCase() + group.name.slice(1)}
                    alias={group.alias}
                    count={group.commands.length}
                    icon={Icon}
                    visible={isVisible(group.id)}
                    defaultOpen={!!search}
                  >
                    <CommandGroupContent group={group} />
                  </CommandSection>
                );
              })}

              {/* Standalone commands */}
              {standalone.length > 0 && (
                <CommandSection
                  id="cmd-misc"
                  title="Other Commands"
                  count={standalone.length}
                  icon={Terminal}
                  visible={!search || filteredGroupIds.length === 0}
                  defaultOpen={!!search}
                >
                  <StandaloneCommandsContent commands={standalone} />
                </CommandSection>
              )}

              {search && filteredGroupIds.length === 0 && (
                <div className="text-center py-12 text-muted-foreground">
                  <Search className="h-8 w-8 mx-auto mb-3 opacity-30" />
                  <p className="text-sm">
                    No commands matching &quot;{search}&quot;
                  </p>
                  <button
                    onClick={() => setSearch("")}
                    className="mt-2 text-xs text-primary hover:underline"
                  >
                    Clear search
                  </button>
                </div>
              )}
            </div>
          </RevealSection>

          {/* ═══════════════════ ALIASES TABLE ═══════════════════ */}
          {!search && (
            <RevealSection id="aliases">
              <h2 className="text-2xl font-bold tracking-tight mb-6">
                Command Aliases
              </h2>
              <p className="text-muted-foreground mb-4">
                Every command group has a short alias for faster typing.
              </p>
              <div className="overflow-hidden rounded-xl border border-border">
                <div className="grid grid-cols-3 bg-muted px-5 py-3 text-xs font-bold uppercase tracking-widest text-muted-foreground">
                  <div>Command</div>
                  <div>Alias</div>
                  <div>Example</div>
                </div>
                {groups
                  .filter((g) => g.alias)
                  .map((g) => {
                    const fullCmd = `bc ${g.name}`;
                    const example =
                      g.commands.length > 0
                        ? g.commands[0].name
                        : g.alias;
                    return (
                      <div
                        key={g.id}
                        className="grid grid-cols-3 border-t border-border px-5 py-3 text-sm"
                      >
                        <div className="font-mono text-[12px] text-foreground">
                          {fullCmd}
                        </div>
                        <div className="font-mono text-[12px] text-primary">
                          {g.alias}
                        </div>
                        <div className="font-mono text-[12px] text-muted-foreground">
                          {example}
                        </div>
                      </div>
                    );
                  })}
              </div>
            </RevealSection>
          )}

          {/* ═══════════════════ CONFIGURATION ═══════════════════ */}
          {!search && (
            <RevealSection id="config">
              <h2 className="text-2xl font-bold tracking-tight mb-6">
                Configuration
              </h2>
              <p className="text-muted-foreground mb-4">
                Manage config via{" "}
                <code className="rounded bg-muted px-1.5 py-0.5 text-sm font-mono">
                  bc config
                </code>{" "}
                or edit{" "}
                <code className="rounded bg-muted px-1.5 py-0.5 text-sm font-mono">
                  .bc/settings.toml
                </code>{" "}
                directly.
              </p>
              <CodeBlock
                title=".bc/settings.toml"
                code={`[workspace]
name = "my-project"
version = 2

[user]
nickname = "@yourname"

[providers]
default = "claude"

[providers.claude]
command = "claude"
enabled = true

[providers.gemini]
command = "gemini"
enabled = true

[runtime]
backend = "tmux"     # or "docker"`}
              />
            </RevealSection>
          )}

          {/* ═══════════════════ ENV VARS ═══════════════════ */}
          {!search && (
            <RevealSection id="env">
              <h2 className="text-2xl font-bold tracking-tight mb-6">
                Environment Variables
              </h2>
              <p className="text-muted-foreground mb-4">
                Automatically set in each agent&apos;s session:
              </p>
              <div className="overflow-hidden rounded-xl border border-border">
                <div className="grid grid-cols-[180px_1fr] bg-muted px-5 py-3 text-xs font-bold uppercase tracking-widest text-muted-foreground">
                  <div>Variable</div>
                  <div>Description</div>
                </div>
                {[
                  {
                    var: "BC_AGENT_ID",
                    desc: "Unique identifier for the agent",
                  },
                  {
                    var: "BC_AGENT_ROLE",
                    desc: "The agent's assigned role (engineer, manager, etc.)",
                  },
                  {
                    var: "BC_WORKSPACE",
                    desc: "Path to the .bc/ workspace directory",
                  },
                  {
                    var: "BC_AGENT_WORKTREE",
                    desc: "Path to the agent's isolated git worktree",
                  },
                  { var: "BC_BIN", desc: "Path to the bc binary" },
                  {
                    var: "BC_ROOT",
                    desc: "Root directory of the bc workspace",
                  },
                  {
                    var: "NO_COLOR",
                    desc: "Disables color output in agent sessions",
                  },
                ].map((e) => (
                  <div
                    key={e.var}
                    className="grid grid-cols-[180px_1fr] border-t border-border px-5 py-3 text-sm"
                  >
                    <div className="font-mono text-[12px] text-primary">
                      {e.var}
                    </div>
                    <div className="text-muted-foreground">{e.desc}</div>
                  </div>
                ))}
              </div>
            </RevealSection>
          )}

          {/* ═══════════════════ CTA ═══════════════════ */}
          {!search && (
            <RevealSection>
              <div className="text-center pt-8 border-t border-border">
                <p className="text-muted-foreground mb-6">
                  Explore the full CLI reference above, or get started on
                  GitHub.
                </p>
                <Link
                  href="https://github.com/rpuneet/bc"
                  className="inline-flex h-12 items-center justify-center rounded-lg bg-primary px-8 text-sm font-semibold text-primary-foreground shadow-lg transition-all hover:shadow-xl active:scale-[0.97]"
                >
                  Get Started
                </Link>
              </div>
            </RevealSection>
          )}
        </div>
      </div>

      <Footer />
    </main>
  );
}
