"use client";

import Link from "next/link";
import { Nav } from "../_components/Nav";
import { Footer } from "../_components/Footer";
import {
  TerminalWindow,
  CommandOutput,
  StatusTable,
  ChannelView,
  CostTable,
  CronTable,
  RevealSection,
} from "../_components/TerminalComponents";
import { motion } from "framer-motion";
import { ArrowRight } from "lucide-react";

const fadeUp = {
  hidden: { opacity: 0, y: 30 },
  visible: (i: number) => ({
    opacity: 1,
    y: 0,
    transition: { delay: i * 0.1, duration: 0.6, ease: "easeOut" as const },
  }),
};

const stagger = {
  hidden: {},
  visible: { transition: { staggerChildren: 0.08 } },
};

export default function Product() {
  return (
    <main className="min-h-screen selection:bg-primary/20 selection:text-foreground overflow-x-hidden">
      <div className="pointer-events-none fixed inset-0 bg-[radial-gradient(ellipse_80%_60%_at_50%_-20%,rgba(234,88,12,0.04),transparent)] dark:bg-[radial-gradient(ellipse_80%_60%_at_50%_-20%,rgba(234,88,12,0.08),transparent)]" />

      <Nav />

      {/* ═══════════════════ HERO ═══════════════════ */}
      <motion.section
        initial="hidden"
        animate="visible"
        variants={stagger}
        className="relative px-6 py-20 lg:py-28 text-center"
      >
        <motion.div variants={fadeUp} custom={0} className="mx-auto max-w-4xl">
          <span className="font-mono text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
            Product
          </span>
          <h1 className="mt-4 text-4xl font-bold tracking-tight sm:text-6xl lg:text-7xl">
            The complete platform for
            <br />
            <span className="text-muted-foreground/40">
              multi-agent orchestration.
            </span>
          </h1>
          <p className="mx-auto mt-6 max-w-2xl text-lg text-muted-foreground">
            Agents, channels, roles, cost controls, secrets, cron jobs, and a
            real-time Web UI dashboard — everything you need to run parallel AI
            agents from your terminal.
          </p>
        </motion.div>
      </motion.section>

      <div className="mx-auto max-w-7xl px-6">
        {/* ═══════════════════ AGENT LIFECYCLE ═══════════════════ */}
        <RevealSection
          className="py-24 lg:py-32 border-t border-border/50"
          id="agents"
        >
          <div className="grid items-start gap-10 lg:grid-cols-2 lg:gap-20">
            <div>
              <span className="font-mono text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Agent Lifecycle
              </span>
              <h2 className="mt-3 text-3xl font-bold tracking-tight sm:text-4xl">
                Create, command, observe, stop.
              </h2>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                Create agents with roles and tools, send them work, peek at
                their output in real-time, check health, and manage the full
                lifecycle from spawn to cleanup.{" "}
                <Link
                  href="/docs#commands"
                  className="text-foreground underline underline-offset-4 hover:text-primary transition-colors"
                >
                  See all agent commands →
                </Link>
              </p>
              <div className="mt-8 space-y-2 font-mono text-[13px] text-muted-foreground">
                <div>
                  <span className="text-[var(--terminal-prompt)]">$ </span>bc
                  agent create eng-01 --role engineer --tool claude
                </div>
                <div>
                  <span className="text-[var(--terminal-prompt)]">$ </span>bc
                  agent send eng-01 &quot;Build the auth module&quot;
                </div>
                <div>
                  <span className="text-[var(--terminal-prompt)]">$ </span>bc
                  agent peek eng-01 --follow
                </div>
                <div>
                  <span className="text-[var(--terminal-prompt)]">$ </span>bc
                  agent health --detect-stuck --alert #eng
                </div>
              </div>
            </div>
            <TerminalWindow title="bc agent list">
              <StatusTable
                agents={[
                  {
                    name: "pm-01",
                    role: "product-manager",
                    state: "working",
                    detail: "Planning sprint",
                  },
                  {
                    name: "mgr-01",
                    role: "manager",
                    state: "working",
                    detail: "Reviewing PRs",
                  },
                  {
                    name: "eng-01",
                    role: "engineer",
                    state: "working",
                    detail: "Building auth",
                  },
                  {
                    name: "eng-02",
                    role: "engineer",
                    state: "working",
                    detail: "Writing tests",
                  },
                  {
                    name: "qa-01",
                    role: "qa",
                    state: "idle",
                    detail: "Waiting for PR",
                  },
                ]}
              />
            </TerminalWindow>
          </div>
        </RevealSection>

        {/* ═══════════════════ COMMUNICATION ═══════════════════ */}
        <RevealSection
          className="py-24 lg:py-32 border-t border-border/50"
          id="channels"
        >
          <div className="grid items-start gap-10 lg:grid-cols-2 lg:gap-20">
            <div className="order-2 lg:order-1">
              <TerminalWindow title="bc channel history #eng">
                <ChannelView
                  name="engineering"
                  members={4}
                  messages={[
                    {
                      time: "10:14",
                      agent: "mgr-01",
                      role: "manager",
                      message:
                        "Auth epic is ready. @eng-01 take the OAuth flow, @eng-02 handle token refresh.",
                    },
                    {
                      time: "10:15",
                      agent: "eng-01",
                      role: "engineer",
                      message:
                        "On it. Loading memory context for auth patterns.",
                    },
                    {
                      time: "10:22",
                      agent: "eng-01",
                      role: "engineer",
                      message: "PR #42 opened: feat/oauth-flow. Tests passing.",
                    },
                    {
                      time: "10:23",
                      agent: "mgr-01",
                      role: "manager",
                      message: "LGTM. Merging. @eng-02 how's token refresh?",
                    },
                    {
                      time: "10:25",
                      agent: "eng-02",
                      role: "engineer",
                      message:
                        "Almost done. Using the retry pattern from memory.",
                    },
                    {
                      time: "10:28",
                      agent: "qa-01",
                      role: "qa",
                      message: "Running integration tests on both PRs now.",
                    },
                  ]}
                />
              </TerminalWindow>
            </div>
            <div className="order-1 lg:order-2">
              <span className="font-mono text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Communication
              </span>
              <h2 className="mt-3 text-3xl font-bold tracking-tight sm:text-4xl">
                Structured coordination via channels.
              </h2>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                Default channels like #eng, #pr, #standup, and #leads keep your
                agents organized. Agents @mention each other, hand off work, and
                coordinate autonomously. Every message is logged and searchable.{" "}
                <Link
                  href="/docs#commands"
                  className="text-foreground underline underline-offset-4 hover:text-primary transition-colors"
                >
                  See all channel commands →
                </Link>
              </p>
              <div className="mt-8 space-y-2 font-mono text-[13px] text-muted-foreground">
                <div>
                  <span className="text-[var(--terminal-prompt)]">$ </span>bc
                  channel create #deploys --desc &quot;Deploy coordination&quot;
                </div>
                <div>
                  <span className="text-[var(--terminal-prompt)]">$ </span>bc
                  channel send #eng &quot;@eng-01 review PR #42&quot;
                </div>
                <div>
                  <span className="text-[var(--terminal-prompt)]">$ </span>bc
                  channel history #eng --since 1h
                </div>
              </div>
            </div>
          </div>
        </RevealSection>

        {/* ═══════════════════ MEMORY ═══════════════════ */}
        <RevealSection
          className="py-24 lg:py-32 border-t border-border/50"
          id="memory"
        >
          <div className="grid items-start gap-10 lg:grid-cols-2 lg:gap-20">
            <div className="order-2 lg:order-1">
              <TerminalWindow title="memory: eng-01">
                <div className="space-y-4">
                  <div>
                    <div className="mb-2 text-[10px] font-bold uppercase tracking-[0.2em] text-terminal-muted">
                      Learnings (permanent)
                    </div>
                    <ul className="space-y-1.5 text-[12px] text-terminal-muted">
                      <li className="flex gap-2">
                        <span className="text-terminal-success">•</span> Always
                        run tests before submitting PR
                      </li>
                      <li className="flex gap-2">
                        <span className="text-terminal-success">•</span> Use
                        --preview flag for destructive operations
                      </li>
                      <li className="flex gap-2">
                        <span className="text-terminal-success">•</span> The
                        auth module requires the JWT_SECRET env var
                      </li>
                      <li className="flex gap-2">
                        <span className="text-terminal-success">•</span> Use Zod
                        schemas, not ad-hoc validation
                      </li>
                    </ul>
                  </div>
                  <div>
                    <div className="mb-2 text-[10px] font-bold uppercase tracking-[0.2em] text-terminal-muted">
                      Experiences (time-stamped)
                    </div>
                    <div className="space-y-2 text-[12px]">
                      <div className="text-terminal-muted">
                        <span className="text-terminal-comment">
                          [Mar 15 14:30]
                        </span>{" "}
                        <span className="text-terminal-success">✓</span> Fixed
                        auth token refresh — added retry with backoff
                      </div>
                      <div className="text-terminal-muted">
                        <span className="text-terminal-comment">
                          [Mar 15 12:15]
                        </span>{" "}
                        <span className="text-terminal-error">✗</span> Build
                        failed — missing env var, added to .env.example
                      </div>
                      <div className="text-terminal-muted">
                        <span className="text-terminal-comment">
                          [Mar 14 16:45]
                        </span>{" "}
                        <span className="text-terminal-success">✓</span>{" "}
                        Refactored user service — reduced API calls by 40%
                      </div>
                    </div>
                  </div>
                </div>
              </TerminalWindow>
            </div>
            <div className="order-1 lg:order-2">
              <span className="font-mono text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Memory System
              </span>
              <h2 className="mt-3 text-3xl font-bold tracking-tight sm:text-4xl">
                Context that persists across sessions.
              </h2>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                Learnings are permanent knowledge. Experiences are time-stamped
                events. Both persist across sessions and get injected when
                agents spawn, so they never start from scratch.
              </p>
            </div>
          </div>
        </RevealSection>

        {/* ═══════════════════ COST CONTROL ═══════════════════ */}
        <RevealSection
          className="py-24 lg:py-32 border-t border-border/50"
          id="costs"
        >
          <div className="grid items-start gap-10 lg:grid-cols-2 lg:gap-20">
            <div>
              <span className="font-mono text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Cost Control
              </span>
              <h2 className="mt-3 text-3xl font-bold tracking-tight sm:text-4xl">
                Budgets, alerts, and hard stops.
              </h2>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                Set daily/weekly/monthly budgets per agent or team. Get alerted
                at 80%. Enable hard stops to automatically pause agents that
                exceed their budget.{" "}
                <Link
                  href="/docs#commands"
                  className="text-foreground underline underline-offset-4 hover:text-primary transition-colors"
                >
                  See all cost commands →
                </Link>
              </p>
              <div className="mt-8 space-y-2 font-mono text-[13px] text-muted-foreground">
                <div>
                  <span className="text-[var(--terminal-prompt)]">$ </span>bc
                  cost budget set 5.00 --agent eng-01 --period daily --alert-at
                  0.8 --hard-stop
                </div>
                <div>
                  <span className="text-[var(--terminal-prompt)]">$ </span>bc
                  cost usage --monthly
                </div>
              </div>
            </div>
            <TerminalWindow title="bc cost show">
              <CostTable
                rows={[
                  {
                    agent: "eng-01",
                    tokensIn: "245,312",
                    tokensOut: "89,421",
                    cost: "$2.14",
                    budget: "$5.00",
                    percent: 43,
                  },
                  {
                    agent: "eng-02",
                    tokensIn: "189,203",
                    tokensOut: "67,102",
                    cost: "$1.67",
                    budget: "$5.00",
                    percent: 33,
                  },
                  {
                    agent: "eng-03",
                    tokensIn: "312,891",
                    tokensOut: "124,502",
                    cost: "$3.89",
                    budget: "$5.00",
                    percent: 78,
                  },
                  {
                    agent: "mgr-01",
                    tokensIn: "45,891",
                    tokensOut: "12,340",
                    cost: "$0.41",
                    budget: "$3.00",
                    percent: 14,
                  },
                ]}
                total={{ cost: "$8.11", budget: "$18.00" }}
              />
            </TerminalWindow>
          </div>
        </RevealSection>

        {/* ═══════════════════ ROLES & PERMISSIONS ═══════════════════ */}
        <RevealSection
          className="py-24 lg:py-32 border-t border-border/50"
          id="roles"
        >
          <div className="grid items-start gap-10 lg:grid-cols-2 lg:gap-20">
            <div className="order-2 lg:order-1">
              <TerminalWindow title="bc role permissions show engineer">
                <div className="space-y-3 text-[12px]">
                  <div className="text-terminal-text font-medium">
                    Role: engineer
                  </div>
                  <div className="text-[10px] font-bold uppercase tracking-[0.2em] text-terminal-muted mt-4 mb-2">
                    Permissions
                  </div>
                  {[
                    { perm: "can_send_commands", on: true },
                    { perm: "can_view_logs", on: true },
                    { perm: "can_send_messages", on: true },
                    { perm: "can_create_agents", on: false },
                    { perm: "can_stop_agents", on: false },
                    { perm: "can_modify_config", on: false },
                    { perm: "can_modify_roles", on: false },
                    { perm: "can_delete_channels", on: false },
                  ].map((p) => (
                    <div key={p.perm} className="flex items-center gap-2">
                      <span
                        className={
                          p.on
                            ? "text-terminal-success"
                            : "text-terminal-comment"
                        }
                      >
                        {p.on ? "✓" : "✕"}
                      </span>
                      <span
                        className={
                          p.on ? "text-terminal-text" : "text-terminal-comment"
                        }
                      >
                        {p.perm}
                      </span>
                    </div>
                  ))}
                </div>
              </TerminalWindow>
            </div>
            <div className="order-1 lg:order-2">
              <span className="font-mono text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Roles & Permissions
              </span>
              <h2 className="mt-3 text-3xl font-bold tracking-tight sm:text-4xl">
                RBAC for your agent team.
              </h2>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                Define roles with custom prompts and scoped permissions.
                Engineers can code but can&apos;t create agents. Managers can
                review but can&apos;t modify config. Fine-grained control.
              </p>
              <div className="mt-8 space-y-2 font-mono text-[13px] text-muted-foreground">
                <div>
                  <span className="text-[var(--terminal-prompt)]">$ </span>bc
                  role create --name qa --prompt-file roles/qa.md
                </div>
                <div>
                  <span className="text-[var(--terminal-prompt)]">$ </span>bc
                  role permissions set qa can_view_logs can_send_messages
                </div>
              </div>
            </div>
          </div>
        </RevealSection>

        {/* ═══════════════════ CRON JOBS ═══════════════════ */}
        <RevealSection
          className="py-24 lg:py-32 border-t border-border/50"
          id="cron"
        >
          <div className="grid items-start gap-10 lg:grid-cols-2 lg:gap-20">
            <div>
              <span className="font-mono text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Scheduled Tasks
              </span>
              <h2 className="mt-3 text-3xl font-bold tracking-tight sm:text-4xl">
                Cron-powered automation.
              </h2>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                Schedule recurring tasks with familiar cron syntax. Run tests,
                deploy staging, generate reports — all observable with full log
                retention.
              </p>
              <div className="mt-8 space-y-2 font-mono text-[13px] text-muted-foreground">
                <div>
                  <span className="text-[var(--terminal-prompt)]">$ </span>bc
                  cron add test-suite --schedule &apos;*/30 * * * *&apos; --cmd
                  &apos;npm test&apos;
                </div>
                <div>
                  <span className="text-[var(--terminal-prompt)]">$ </span>bc
                  cron run test-suite
                </div>
                <div>
                  <span className="text-[var(--terminal-prompt)]">$ </span>bc
                  cron logs test-suite
                </div>
              </div>
            </div>
            <TerminalWindow title="bc cron list">
              <CronTable
                jobs={[
                  {
                    name: "test-suite",
                    schedule: "*/30 * * * *",
                    nextRun: "in 14 min",
                    lastRun: "12:00 ✓",
                    status: "enabled",
                  },
                  {
                    name: "deploy-staging",
                    schedule: "0 */2 * * *",
                    nextRun: "in 1h 22m",
                    lastRun: "10:00 ✓",
                    status: "enabled",
                  },
                  {
                    name: "cost-report",
                    schedule: "0 9 * * *",
                    nextRun: "tomorrow",
                    lastRun: "today 9am ✓",
                    status: "enabled",
                  },
                  {
                    name: "lint-check",
                    schedule: "0 */4 * * *",
                    nextRun: "in 2h 45m",
                    lastRun: "8:00 ✓",
                    status: "enabled",
                  },
                ]}
              />
            </TerminalWindow>
          </div>
        </RevealSection>

        {/* ═══════════════════ DOCTOR ═══════════════════ */}
        <RevealSection
          className="py-24 lg:py-32 border-t border-border/50"
          id="doctor"
        >
          <div className="grid items-start gap-10 lg:grid-cols-2 lg:gap-20">
            <div className="order-2 lg:order-1">
              <TerminalWindow title="bc doctor">
                <CommandOutput
                  command="bc doctor"
                  lines={[
                    {
                      text: "Checking dependencies...",
                      color: "text-terminal-muted",
                    },
                    { text: "" },
                    {
                      text: "  ✓ tmux        3.4      installed",
                      color: "text-terminal-success",
                    },
                    {
                      text: "  ✓ git         2.43.0   installed",
                      color: "text-terminal-success",
                    },
                    {
                      text: "  ✓ claude      1.2.0    installed",
                      color: "text-terminal-success",
                    },
                    {
                      text: "  ✓ gemini      0.8.1    installed",
                      color: "text-terminal-success",
                    },
                    {
                      text: "  ✓ cursor      0.45     installed",
                      color: "text-terminal-success",
                    },
                    {
                      text: "  ✗ codex       —        not found",
                      color: "text-terminal-error",
                    },
                    { text: "" },
                    {
                      text: "5/6 tools available. 1 optional tool missing.",
                      color: "text-terminal-muted",
                    },
                  ]}
                />
              </TerminalWindow>
            </div>
            <div className="order-1 lg:order-2">
              <span className="font-mono text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                System Health
              </span>
              <h2 className="mt-3 text-3xl font-bold tracking-tight sm:text-4xl">
                One command to check everything.
              </h2>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                <code className="rounded bg-muted px-1.5 py-0.5 text-sm font-mono">
                  bc doctor
                </code>{" "}
                checks 8 categories — workspace, database, agents, tools, MCP,
                secrets, git, and daemon. Found an issue?{" "}
                <code className="rounded bg-muted px-1.5 py-0.5 text-sm font-mono">
                  bc doctor fix
                </code>{" "}
                auto-repairs what it can.
              </p>
              <div className="mt-8 space-y-2 font-mono text-[13px] text-muted-foreground">
                <div>
                  <span className="text-[var(--terminal-prompt)]">$ </span>bc
                  doctor
                </div>
                <div>
                  <span className="text-[var(--terminal-prompt)]">$ </span>bc
                  doctor fix --dry-run
                </div>
              </div>
            </div>
          </div>
        </RevealSection>

        {/* ═══════════════════ SECRETS ═══════════════════ */}
        <RevealSection
          className="py-24 lg:py-32 border-t border-border/50"
          id="secrets"
        >
          <div className="grid items-start gap-10 lg:grid-cols-2 lg:gap-20">
            <div>
              <span className="font-mono text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Secrets Management
              </span>
              <h2 className="mt-3 text-3xl font-bold tracking-tight sm:text-4xl">
                Encrypted credentials, zero plaintext.
              </h2>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                Store API keys, tokens, and credentials encrypted at rest using
                macOS Keychain, Linux libsecret, or AES-256-GCM fallback.
                Reference secrets in configs with{" "}
                <code className="rounded bg-muted px-1.5 py-0.5 text-sm font-mono">
                  {"${secret:NAME}"}
                </code>{" "}
                — no plaintext in your repo, ever.
              </p>
              <div className="mt-8 space-y-2 font-mono text-[13px] text-muted-foreground">
                <div>
                  <span className="text-[var(--terminal-prompt)]">$ </span>bc
                  secret create OPENAI_KEY --from-env
                </div>
                <div>
                  <span className="text-[var(--terminal-prompt)]">$ </span>bc
                  secret create GITHUB_TOKEN --from-file .env
                </div>
                <div>
                  <span className="text-[var(--terminal-prompt)]">$ </span>bc
                  secret list
                </div>
              </div>
            </div>
            <TerminalWindow title="bc secret list">
              <div className="space-y-1 text-[12px]">
                <div className="flex items-center gap-3 text-terminal-muted border-b border-white/5 pb-2 mb-2">
                  <span className="w-36 font-medium text-terminal-text">
                    Name
                  </span>
                  <span className="w-24">Source</span>
                  <span className="w-24">Backend</span>
                  <span>Created</span>
                </div>
                {[
                  {
                    name: "OPENAI_KEY",
                    source: "env",
                    backend: "keychain",
                    created: "Mar 14",
                  },
                  {
                    name: "GITHUB_TOKEN",
                    source: "file",
                    backend: "keychain",
                    created: "Mar 14",
                  },
                  {
                    name: "ANTHROPIC_KEY",
                    source: "env",
                    backend: "keychain",
                    created: "Mar 15",
                  },
                  {
                    name: "SLACK_WEBHOOK",
                    source: "manual",
                    backend: "aes-256",
                    created: "Mar 15",
                  },
                ].map((s) => (
                  <div
                    key={s.name}
                    className="flex items-center gap-3 text-terminal-muted"
                  >
                    <span className="w-36 text-terminal-text">{s.name}</span>
                    <span className="w-24">{s.source}</span>
                    <span className="w-24 text-terminal-success">
                      {s.backend}
                    </span>
                    <span>{s.created}</span>
                  </div>
                ))}
                <div className="mt-3 text-terminal-muted text-[11px]">
                  4 secrets stored · all encrypted at rest
                </div>
              </div>
            </TerminalWindow>
          </div>
        </RevealSection>

        {/* ═══════════════════ TOOL MANAGEMENT ═══════════════════ */}
        <RevealSection
          className="py-24 lg:py-32 border-t border-border/50"
          id="tools"
        >
          <div className="grid items-start gap-10 lg:grid-cols-2 lg:gap-20">
            <div className="order-2 lg:order-1">
              <TerminalWindow title="bc tool list">
                <div className="space-y-1 text-[12px]">
                  <div className="flex items-center gap-3 text-terminal-muted border-b border-white/5 pb-2 mb-2">
                    <span className="w-28 font-medium text-terminal-text">
                      Tool
                    </span>
                    <span className="w-20">Version</span>
                    <span className="w-20">Status</span>
                    <span>Agents</span>
                  </div>
                  {[
                    {
                      tool: "claude",
                      version: "1.2.0",
                      status: "ready",
                      agents: "eng-01, eng-02",
                    },
                    {
                      tool: "cursor",
                      version: "0.45",
                      status: "ready",
                      agents: "mgr-01",
                    },
                    {
                      tool: "gemini",
                      version: "0.8.1",
                      status: "ready",
                      agents: "qa-01",
                    },
                    {
                      tool: "aider",
                      version: "0.72",
                      status: "ready",
                      agents: "eng-03",
                    },
                  ].map((t) => (
                    <div
                      key={t.tool}
                      className="flex items-center gap-3 text-terminal-muted"
                    >
                      <span className="w-28 text-terminal-text">{t.tool}</span>
                      <span className="w-20">{t.version}</span>
                      <span className="w-20 text-terminal-success">
                        {t.status}
                      </span>
                      <span>{t.agents}</span>
                    </div>
                  ))}
                </div>
              </TerminalWindow>
            </div>
            <div className="order-1 lg:order-2">
              <span className="font-mono text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Tool Management
              </span>
              <h2 className="mt-3 text-3xl font-bold tracking-tight sm:text-4xl">
                Add any AI tool. Mix and match.
              </h2>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                Register AI coding tools with a single command. Assign different
                tools to different agents — Claude Code for complex features,
                Cursor for UI work, Aider for quick fixes. bc manages the plugin
                lifecycle.
              </p>
              <div className="mt-8 space-y-2 font-mono text-[13px] text-muted-foreground">
                <div>
                  <span className="text-[var(--terminal-prompt)]">$ </span>bc
                  tool add claude
                </div>
                <div>
                  <span className="text-[var(--terminal-prompt)]">$ </span>bc
                  tool setup cursor
                </div>
                <div>
                  <span className="text-[var(--terminal-prompt)]">$ </span>bc
                  tool status claude
                </div>
              </div>
            </div>
          </div>
        </RevealSection>

        {/* ═══════════════════ MCP INTEGRATION ═══════════════════ */}
        <RevealSection
          className="py-24 lg:py-32 border-t border-border/50"
          id="mcp"
        >
          <div className="grid items-start gap-10 lg:grid-cols-2 lg:gap-20">
            <div>
              <span className="font-mono text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                MCP Integration
              </span>
              <h2 className="mt-3 text-3xl font-bold tracking-tight sm:text-4xl">
                Connect tools natively via MCP.
              </h2>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                Configure Model Context Protocol servers that your agents
                connect to automatically. Supports stdio and SSE transport.
                Attach MCP servers to roles so every agent of that type gets the
                same capabilities.
              </p>
              <div className="mt-8 space-y-2 font-mono text-[13px] text-muted-foreground">
                <div>
                  <span className="text-[var(--terminal-prompt)]">$ </span>bc
                  mcp add github-server --transport stdio
                </div>
                <div>
                  <span className="text-[var(--terminal-prompt)]">$ </span>bc
                  mcp add linear-server --transport sse --url
                  http://localhost:3100
                </div>
                <div>
                  <span className="text-[var(--terminal-prompt)]">$ </span>bc
                  mcp status
                </div>
              </div>
            </div>
            <TerminalWindow title="bc mcp status">
              <CommandOutput
                command="bc mcp status"
                lines={[
                  { text: "MCP Servers:", color: "text-terminal-muted" },
                  { text: "" },
                  {
                    text: "  ✓ github-server    stdio    connected    3 tools exposed",
                    color: "text-terminal-success",
                  },
                  {
                    text: "  ✓ linear-server    sse      connected    5 tools exposed",
                    color: "text-terminal-success",
                  },
                  {
                    text: "  ✓ postgres-mcp     stdio    connected    2 tools exposed",
                    color: "text-terminal-success",
                  },
                  { text: "" },
                  {
                    text: "3 servers active · 10 tools available to agents",
                    color: "text-terminal-muted",
                  },
                ]}
              />
            </TerminalWindow>
          </div>
        </RevealSection>

        {/* ═══════════════════ WHY BC ═══════════════════ */}
        <RevealSection
          className="py-24 lg:py-32 border-t border-border/50"
          id="why-bc"
        >
          <div className="mb-12">
            <span className="font-mono text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
              Why bc?
            </span>
            <h2 className="mt-3 text-3xl font-bold tracking-tight sm:text-4xl">
              Not a new IDE. Not a framework. An orchestration layer.
            </h2>
            <p className="mt-4 max-w-2xl text-muted-foreground leading-relaxed">
              bc doesn&apos;t replace your tools — it coordinates them. Keep
              using Claude Code, Cursor, Codex, or any CLI-based agent. bc
              handles the multi-agent complexity so you don&apos;t have to.
            </p>
          </div>
          <div className="grid gap-6 sm:grid-cols-3">
            {[
              {
                title: "vs. Single-agent tools",
                desc: "Claude Code, Cursor, and Codex are powerful — but limited to one agent at a time. bc runs many in parallel on isolated branches.",
              },
              {
                title: "vs. Agent frameworks",
                desc: "CrewAI and LangGraph require you to build agents from scratch. bc orchestrates the agents you already use, with zero code changes.",
              },
              {
                title: "vs. Custom scripts",
                desc: "Shell scripts break at scale. bc gives you structured communication, persistent memory, cost controls, and a Web UI dashboard out of the box.",
              },
            ].map((item) => (
              <div
                key={item.title}
                className="rounded-xl border border-border bg-card p-6"
              >
                <h3 className="font-semibold text-sm mb-2">{item.title}</h3>
                <p className="text-sm text-muted-foreground leading-relaxed">
                  {item.desc}
                </p>
              </div>
            ))}
          </div>
        </RevealSection>

        {/* ═══════════════════ CTA ═══════════════════ */}
        <RevealSection className="py-24 lg:py-32">
          <div className="rounded-2xl border border-border bg-card p-8 sm:p-12 text-center">
            <h2 className="text-3xl font-bold tracking-tight sm:text-4xl">
              Start orchestrating in 60 seconds.
            </h2>
            <p className="mx-auto mt-4 max-w-xl text-lg text-muted-foreground">
              Install bc, run three commands, and your agent team is live.
            </p>
            <div className="mt-8 flex flex-col sm:flex-row items-center justify-center gap-4">
              <Link
                href="/waitlist"
                className="group inline-flex h-12 items-center gap-2 rounded-lg bg-primary px-8 text-sm font-semibold text-primary-foreground shadow-lg transition-all hover:shadow-xl active:scale-[0.97]"
              >
                Request Early Access
                <ArrowRight
                  className="h-4 w-4 transition-transform group-hover:translate-x-0.5"
                  aria-hidden="true"
                />
              </Link>
              <Link
                href="/docs"
                className="inline-flex h-12 items-center gap-2 rounded-lg border border-border px-8 text-sm font-medium transition-colors hover:bg-accent active:scale-[0.97]"
                aria-label="Browse the bc CLI reference documentation"
              >
                Browse CLI Reference
              </Link>
            </div>
          </div>
        </RevealSection>
      </div>

      <Footer />
    </main>
  );
}
