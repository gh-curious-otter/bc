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

/* ── Command group definitions for search ── */
const COMMAND_GROUPS = [
  {
    id: "cmd-workspace",
    title: "Workspace",
    alias: "bc ws",
    keywords: "workspace init up down status config logs list discover",
  },
  {
    id: "cmd-agents",
    title: "Agent Management",
    alias: "bc ag",
    keywords:
      "agent create list show attach peek send broadcast stop start delete rename health cost logs",
  },
  {
    id: "cmd-channels",
    title: "Channels",
    alias: "bc ch",
    keywords:
      "channel create send history react list show status add remove delete",
  },
  {
    id: "cmd-tools",
    title: "Tool Management",
    alias: "bc tl",
    keywords: "tool add setup list show status upgrade edit delete run",
  },
  {
    id: "cmd-secrets",
    title: "Secrets",
    alias: "bc sec",
    keywords: "secret create list show edit delete encrypted keychain",
  },
  {
    id: "cmd-costs",
    title: "Cost Tracking",
    alias: "bc co",
    keywords: "cost show usage budget set delete alert hard-stop",
  },
  {
    id: "cmd-cron",
    title: "Cron Jobs",
    alias: "bc cr",
    keywords: "cron add list show run edit delete enable disable logs schedule",
  },
  {
    id: "cmd-roles",
    title: "Roles & Permissions",
    alias: "bc rl",
    keywords:
      "role create list show edit delete clone diff rename validate permissions",
  },
  {
    id: "cmd-mcp",
    title: "MCP Servers",
    alias: "bc mcp",
    keywords: "mcp add list show edit delete status server",
  },
  {
    id: "cmd-doctor",
    title: "Health Checks",
    alias: "bc dr",
    keywords: "doctor check fix health diagnostics",
  },
  {
    id: "cmd-daemon",
    title: "Daemon",
    alias: "bcd",
    keywords: "daemon start stop status logs server",
  },
  {
    id: "cmd-misc",
    title: "Other Commands",
    alias: "",
    keywords: "completion home version shell bash zsh fish",
  },
];

/* ═══════════════════════════════════════════════════════════════════ */

export default function Docs() {
  const [search, setSearch] = useState("");

  const filteredGroups = useMemo(() => {
    if (!search.trim()) return COMMAND_GROUPS.map((g) => g.id);
    const q = search.toLowerCase();
    return COMMAND_GROUPS.filter(
      (g) =>
        g.title.toLowerCase().includes(q) ||
        g.alias.toLowerCase().includes(q) ||
        g.keywords.toLowerCase().includes(q),
    ).map((g) => g.id);
  }, [search]);

  const isVisible = (id: string) => filteredGroups.includes(id);

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
              {filteredGroups.length}{" "}
              {filteredGroups.length === 1 ? "group" : "groups"} matching &quot;
              {search}&quot;
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
              { href: "#presets", label: "Presets" },
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
                  command="bc init --preset solo"
                  lines={[
                    {
                      text: "Initializing bc workspace...",
                      color: "text-terminal-muted",
                    },
                    {
                      text: "Created .bc/config.toml",
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
                        text: "  ✓ Daemon running on localhost:9374",
                        color: "text-terminal-success",
                      },
                    ]}
                  />
                </div>
                <div className="mt-4">
                  <CommandOutput
                    command="bc up"
                    delay={0.6}
                    lines={[
                      {
                        text: "Starting agents...",
                        color: "text-terminal-muted",
                      },
                      {
                        text: "  ✓ eng-01  engineer  working",
                        color: "text-terminal-success",
                      },
                      { text: "1 agent active.", color: "text-terminal-muted" },
                    ]}
                  />
                </div>
                <div className="mt-4">
                  <CommandOutput
                    command="bc home"
                    delay={0.9}
                    lines={[
                      {
                        text: "Opening Web UI at localhost:9374...",
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
              <div className="grid gap-4 sm:grid-cols-3">
                <CodeBlock
                  title="Homebrew"
                  code="brew install bcinfra1/tap/bc"
                />
                <CodeBlock
                  title="Go Install"
                  code="go install github.com/bcinfra1/bc@latest"
                />
                <CodeBlock
                  title="Binary"
                  code={`curl -fsSL https://bc-infra.com/install | sh`}
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
                  desc="AI instances with roles, tools, and states. Lifecycle: create → start → work → stop → delete."
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
              <CommandSection
                id="cmd-workspace"
                title="Workspace"
                alias="bc ws"
                count={10}
                icon={Layers}
                visible={isVisible("cmd-workspace")}
                defaultOpen={!!search}
              >
                <CodeBlock
                  code={`bc ws init [dir] [--quick] [--preset solo|small-team|full-team]
bc ws up [--agent claude|cursor|codex]
bc ws down [--force]
bc ws status [--activity]
bc ws config show|get|set|validate|edit
bc ws logs [--agent X] [--type Y] [--since 1h] [--tail N]
bc ws list
bc ws discover`}
                />
              </CommandSection>

              <CommandSection
                id="cmd-agents"
                title="Agent Management"
                alias="bc ag"
                count={14}
                icon={Users}
                visible={isVisible("cmd-agents")}
                defaultOpen={!!search}
              >
                <CodeBlock
                  code={`bc ag create [name] --role <role> [--tool X] [--env file]
bc ag list [--role <role>]
bc ag show <agent>
bc ag attach <agent>               # Attach to tmux session
bc ag peek <agent> [--lines N] [--follow]
bc ag send <agent> <message> [--preview]
bc ag broadcast <message>
bc ag send-to-role <role> <message>
bc ag stop|start|delete <agent>
bc ag health [agent] [--detect-stuck]
bc ag cost <agent>
bc ag logs <agent>`}
                />
              </CommandSection>

              <CommandSection
                id="cmd-channels"
                title="Channels"
                alias="bc ch"
                count={12}
                icon={MessageSquare}
                visible={isVisible("cmd-channels")}
                defaultOpen={!!search}
              >
                <CodeBlock
                  code={`bc ch create <name> [--desc "..."]
bc ch send <channel> <message>     # @mention support
bc ch history <channel> [--limit N] [--since 1h] [--agent X]
bc ch react <channel> <msg-index> <emoji>
bc ch list|show|status
bc ch add|remove <channel> <member>
bc ch delete <channel>`}
                />
                <p className="mt-3 text-sm text-muted-foreground">
                  <strong>Default channels:</strong> #eng, #pr, #standup, #leads
                </p>
              </CommandSection>

              <CommandSection
                id="cmd-tools"
                title="Tool Management"
                alias="bc tl"
                count={8}
                icon={Wrench}
                visible={isVisible("cmd-tools")}
                defaultOpen={!!search}
              >
                <CodeBlock
                  code={`bc tl add|setup <tool>
bc tl list
bc tl show|status <tool>
bc tl upgrade|edit|delete <tool>
bc tl run <tool>`}
                />
                <p className="mt-3 text-sm text-muted-foreground">
                  <strong>Supported tools:</strong> Claude Code, Cursor, Codex,
                  Gemini, Aider, OpenCode, OpenClaw, Custom
                </p>
              </CommandSection>

              <CommandSection
                id="cmd-secrets"
                title="Secrets"
                alias="bc sec"
                count={5}
                icon={Key}
                visible={isVisible("cmd-secrets")}
                defaultOpen={!!search}
              >
                <CodeBlock
                  code={`bc sec create <name> [--from-env] [--from-file]
bc sec list
bc sec show|edit|delete <name>`}
                />
                <p className="mt-3 text-sm text-muted-foreground">
                  Encrypted at rest (macOS Keychain / Linux libsecret /
                  AES-256-GCM fallback). Referenced via{" "}
                  <code className="rounded bg-muted px-1.5 py-0.5 text-xs font-mono">
                    {"${secret:NAME}"}
                  </code>{" "}
                  in configs.
                </p>
              </CommandSection>

              <CommandSection
                id="cmd-costs"
                title="Cost Tracking"
                alias="bc co"
                count={5}
                icon={DollarSign}
                visible={isVisible("cmd-costs")}
                defaultOpen={!!search}
              >
                <CodeBlock
                  code={`bc co show [agent]
bc co usage [--monthly] [--session]
bc co budget show|set|delete [--agent X] [--period daily|weekly|monthly] [--alert-at 0.8] [--hard-stop]`}
                />
              </CommandSection>

              <CommandSection
                id="cmd-cron"
                title="Cron Jobs"
                alias="bc cr"
                count={9}
                icon={Clock}
                visible={isVisible("cmd-cron")}
                defaultOpen={!!search}
              >
                <CodeBlock
                  code={`bc cr add <name> --schedule '<cron>' --cmd '<command>' [--prompt "..."]
bc cr list
bc cr show <name>
bc cr run <name>                   # Manual trigger
bc cr edit|delete|enable|disable <name>
bc cr logs <name>`}
                />
              </CommandSection>

              <CommandSection
                id="cmd-roles"
                title="Roles & Permissions"
                alias="bc rl"
                count={14}
                icon={Shield}
                visible={isVisible("cmd-roles")}
                defaultOpen={!!search}
              >
                <CodeBlock
                  code={`bc rl create --name <name> --prompt "..." [--prompt-file <file>]
bc rl list|show|edit|delete <role>
bc rl clone <source> <target>
bc rl diff <role1> <role2>
bc rl rename <old> <new>
bc rl validate

# Permissions
bc rl permissions show|list|set|add|remove <role> [<perm>...]`}
                />
                <p className="mt-3 text-sm text-muted-foreground">
                  <strong>11 permissions:</strong> can_create_agents,
                  can_stop_agents, can_delete_agents, can_restart_agents,
                  can_send_commands, can_view_logs, can_modify_config,
                  can_modify_roles, can_create_channels, can_delete_channels,
                  can_send_messages
                </p>
                <p className="mt-1 text-sm text-muted-foreground">
                  <strong>Hierarchy:</strong> PM (L0) → Manager (L1) → Engineer
                  (L2)
                </p>
              </CommandSection>

              <CommandSection
                id="cmd-mcp"
                title="MCP Servers"
                alias="bc mcp"
                count={6}
                icon={Hash}
                visible={isVisible("cmd-mcp")}
                defaultOpen={!!search}
              >
                <CodeBlock
                  code={`bc mcp add <name>
bc mcp list
bc mcp show|edit|delete|status <name>`}
                />
                <p className="mt-3 text-sm text-muted-foreground">
                  MCP server configs referenced by roles. Supports stdio and SSE
                  transport.
                </p>
              </CommandSection>

              <CommandSection
                id="cmd-doctor"
                title="Health Checks"
                alias="bc dr"
                count={3}
                icon={Stethoscope}
                visible={isVisible("cmd-doctor")}
                defaultOpen={!!search}
              >
                <CodeBlock
                  code={`bc dr                              # Full health check
bc dr check <category>             # Check specific category
bc dr fix [--dry-run]              # Auto-fix issues`}
                />
                <p className="mt-3 text-sm text-muted-foreground">
                  <strong>8 categories:</strong> workspace, database, agents,
                  tools, MCP, secrets, git, daemon
                </p>
              </CommandSection>

              <CommandSection
                id="cmd-daemon"
                title="Daemon"
                alias="bcd"
                count={4}
                icon={Server}
                visible={isVisible("cmd-daemon")}
                defaultOpen={!!search}
              >
                <CodeBlock
                  code={`bc daemon start [-d]               # Start bcd server
bc daemon stop                     # Graceful shutdown
bc daemon status                   # Health check
bc daemon logs                     # Show daemon logs`}
                />
                <p className="mt-3 text-sm text-muted-foreground">
                  Shortcut:{" "}
                  <code className="rounded bg-muted px-1.5 py-0.5 text-xs font-mono">
                    bcd
                  </code>{" "}
                  is an alias for{" "}
                  <code className="rounded bg-muted px-1.5 py-0.5 text-xs font-mono">
                    bc daemon start
                  </code>
                </p>
              </CommandSection>

              <CommandSection
                id="cmd-misc"
                title="Other Commands"
                count={3}
                icon={Terminal}
                visible={isVisible("cmd-misc")}
                defaultOpen={!!search}
              >
                <CodeBlock
                  code={`bc completion [bash|zsh|fish|powershell]
bc home                            # Open Web UI in browser
bc version`}
                />
              </CommandSection>
            </div>

            {search && filteredGroups.length === 0 && (
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
                {[
                  {
                    cmd: "bc workspace",
                    alias: "bc ws",
                    example: "bc ws status",
                  },
                  { cmd: "bc agent", alias: "bc ag", example: "bc ag list" },
                  {
                    cmd: "bc channel",
                    alias: "bc ch",
                    example: 'bc ch send #eng "hello"',
                  },
                  { cmd: "bc tool", alias: "bc tl", example: "bc tl list" },
                  {
                    cmd: "bc secret",
                    alias: "bc sec",
                    example: "bc sec create API_KEY",
                  },
                  {
                    cmd: "bc cost",
                    alias: "bc co",
                    example: "bc co usage --monthly",
                  },
                  { cmd: "bc cron", alias: "bc cr", example: "bc cr list" },
                  {
                    cmd: "bc role",
                    alias: "bc rl",
                    example: "bc rl show engineer",
                  },
                  { cmd: "bc doctor", alias: "bc dr", example: "bc dr fix" },
                  { cmd: "bc daemon", alias: "bcd", example: "bcd" },
                ].map((row) => (
                  <div
                    key={row.cmd}
                    className="grid grid-cols-3 border-t border-border px-5 py-3 text-sm"
                  >
                    <div className="font-mono text-[12px] text-foreground">
                      {row.cmd}
                    </div>
                    <div className="font-mono text-[12px] text-primary">
                      {row.alias}
                    </div>
                    <div className="font-mono text-[12px] text-muted-foreground">
                      {row.example}
                    </div>
                  </div>
                ))}
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
                  bc ws config
                </code>{" "}
                or edit{" "}
                <code className="rounded bg-muted px-1.5 py-0.5 text-sm font-mono">
                  .bc/config.toml
                </code>{" "}
                directly.
              </p>
              <CodeBlock
                title=".bc/config.toml"
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

          {/* ═══════════════════ PRESETS ═══════════════════ */}
          {!search && (
            <RevealSection id="presets">
              <h2 className="text-2xl font-bold tracking-tight mb-6">
                Init Presets
              </h2>
              <div className="grid gap-4 sm:grid-cols-3">
                {[
                  {
                    name: "solo",
                    desc: "Just me and my agents",
                    agents: "1 engineer",
                    cmd: "bc init --preset solo",
                  },
                  {
                    name: "small-team",
                    desc: "A few engineers with a lead",
                    agents: "1 manager + 2-3 engineers",
                    cmd: "bc init --preset small-team",
                  },
                  {
                    name: "full-team",
                    desc: "PM, managers, engineers, QA",
                    agents: "PM + managers + engineers + QA + UX",
                    cmd: "bc init --preset full-team",
                  },
                ].map((p) => (
                  <div
                    key={p.name}
                    className="rounded-xl border border-border bg-card p-5"
                  >
                    <div className="font-mono text-sm font-bold text-foreground mb-1">
                      {p.name}
                    </div>
                    <p className="text-sm text-muted-foreground mb-3">
                      {p.desc}
                    </p>
                    <div className="text-[11px] text-muted-foreground/60 mb-3">
                      Creates: {p.agents}
                    </div>
                    <code className="block rounded-lg bg-muted px-3 py-2 text-[12px] font-mono text-muted-foreground">
                      {p.cmd}
                    </code>
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
                  href="https://github.com/bcinfra1/bc"
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
