"use client";

import Image from "next/image";
import Link from "next/link";
import { Nav } from "../_components/Nav";
import { Footer } from "../_components/Footer";
import { RevealSection } from "../_components/TerminalComponents";
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

/* ── Reusable screenshot frame ─────────────────────────────────────── */

function ScreenshotFrame({
  src,
  alt,
}: {
  src: string;
  alt: string;
}) {
  return (
    <div className="overflow-hidden rounded-xl border border-border shadow-2xl dark:border-[rgba(210,180,140,0.06)]">
      <Image
        src={src}
        alt={alt}
        width={1280}
        height={800}
        className="w-full h-auto"
        loading="lazy"
      />
    </div>
  );
}

/* ── CLI command display ───────────────────────────────────────────── */

function CLICommands({ commands }: { commands: string[] }) {
  return (
    <div className="mt-8 space-y-2 font-mono text-[13px] text-muted-foreground">
      {commands.map((cmd) => (
        <div key={cmd}>
          <span className="text-[var(--terminal-prompt)]">$ </span>
          {cmd}
        </div>
      ))}
    </div>
  );
}

/* ── Feature section wrapper (alternating layout) ──────────────────── */

function FeatureSection({
  id,
  label,
  title,
  description,
  commands,
  screenshot,
  screenshotAlt,
  docsLink,
  imageFirst = false,
}: {
  id: string;
  label: string;
  title: string;
  description: string;
  commands: string[];
  screenshot: string;
  screenshotAlt: string;
  docsLink?: string;
  imageFirst?: boolean;
}) {
  const textContent = (
    <div className={imageFirst ? "order-1 lg:order-2" : ""}>
      <span className="font-mono text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
        {label}
      </span>
      <h2 className="mt-3 text-3xl font-bold tracking-tight sm:text-4xl">
        {title}
      </h2>
      <p className="mt-4 text-muted-foreground leading-relaxed">
        {description}
        {docsLink && (
          <>
            {" "}
            <Link
              href={docsLink}
              className="text-foreground underline underline-offset-4 hover:text-primary transition-colors"
            >
              See docs &rarr;
            </Link>
          </>
        )}
      </p>
      <CLICommands commands={commands} />
    </div>
  );

  const imageContent = (
    <div className={imageFirst ? "order-2 lg:order-1" : ""}>
      <ScreenshotFrame src={screenshot} alt={screenshotAlt} />
    </div>
  );

  return (
    <RevealSection
      className="py-24 lg:py-32 border-t border-border/50"
      id={id}
    >
      <div className="grid items-start gap-10 lg:grid-cols-2 lg:gap-20">
        {imageFirst ? (
          <>
            {imageContent}
            {textContent}
          </>
        ) : (
          <>
            {textContent}
            {imageContent}
          </>
        )}
      </div>
    </RevealSection>
  );
}

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
            Agents, channels, roles, cost tracking, secrets, cron jobs, and a
            real-time Web UI dashboard — everything you need to run parallel AI
            agents from your terminal.
          </p>
        </motion.div>

        {/* Hero dashboard screenshot */}
        <motion.div
          variants={fadeUp}
          custom={1}
          className="mx-auto mt-16 max-w-5xl"
        >
          <ScreenshotFrame
            src="/screenshots/dashboard-01-home.png"
            alt="bc Web UI dashboard showing workspace overview with agent status, channels, and system health"
          />
        </motion.div>
      </motion.section>

      <div className="mx-auto max-w-7xl px-6">
        {/* ═══════════════════ AGENT LIFECYCLE ═══════════════════ */}
        <FeatureSection
          id="agents"
          label="Agent Lifecycle"
          title="Create, command, observe, stop."
          description="Create agents with roles and tools, send them work, peek at their output in real-time, and manage the full lifecycle from spawn to cleanup. Each agent runs in its own tmux session with an isolated git worktree."
          commands={[
            'bc agent create eng-01 --role engineer --tool claude',
            'bc agent send eng-01 "Build the auth module"',
            'bc agent peek eng-01',
            'bc agent list',
          ]}
          screenshot="/screenshots/dashboard-02-agents.png"
          screenshotAlt="bc Agents view showing a table of agents with their roles, tools, states, and tasks"
          docsLink="/docs#commands"
        />

        {/* ═══════════════════ COMMUNICATION ═══════════════════ */}
        <FeatureSection
          id="channels"
          label="Communication"
          title="Structured coordination via channels."
          description="Channels like #engineering, #merge, and #ops keep your agents organized. Agents @mention each other, hand off work, and coordinate autonomously. Every message is logged and searchable."
          commands={[
            'bc channel create deploys',
            'bc channel send engineering "@eng-01 review PR #42"',
            'bc channel history engineering --last 20',
          ]}
          screenshot="/screenshots/dashboard-03-channels.png"
          screenshotAlt="bc Channels view showing inter-agent communication with message history and reactions"
          docsLink="/docs#commands"
          imageFirst
        />

        {/* ═══════════════════ COST TRACKING ═══════════════════ */}
        <FeatureSection
          id="costs"
          label="Cost Tracking"
          title="Track spending across every agent."
          description="See total cost, token usage, and per-agent breakdowns in real-time. Set budgets with alerts at configurable thresholds, and enable hard stops to automatically pause agents that exceed their budget."
          commands={[
            'bc cost show',
            'bc cost budget set 50.00 --agent eng-01 --alert-at 0.8',
            'bc cost usage',
          ]}
          screenshot="/screenshots/dashboard-04-costs.png"
          screenshotAlt="bc Costs dashboard showing total spend, daily cost trends, and per-agent cost breakdown with bar chart"
          docsLink="/docs#commands"
        />

        {/* ═══════════════════ ROLES & PERMISSIONS ═══════════════════ */}
        <FeatureSection
          id="roles"
          label="Roles & Permissions"
          title="RBAC for your agent team."
          description="Define roles with custom prompts and scoped capabilities. Engineers can implement tasks but can't create agents. Managers can assign work but can't modify config. Roles are defined as markdown files in .bc/roles/."
          commands={[
            'bc role list',
            'bc role show engineer',
          ]}
          screenshot="/screenshots/dashboard-05-roles.png"
          screenshotAlt="bc Roles view showing configured roles with their capabilities and hierarchy"
          imageFirst
        />

        {/* ═══════════════════ TOOL MANAGEMENT ═══════════════════ */}
        <FeatureSection
          id="tools"
          label="Tool Management"
          title="Add any AI tool. Mix and match."
          description="Register AI coding tools and assign different tools to different agents — Claude Code for complex features, Cursor for UI work, Aider for quick fixes. bc manages the tool lifecycle including install, upgrade, and status checks."
          commands={[
            'bc tool list',
            'bc tool add mytool --command "mytool --yes"',
            'bc tool status claude',
          ]}
          screenshot="/screenshots/dashboard-06-tools.png"
          screenshotAlt="bc Tools view showing registered AI tools with their versions, installation status, and commands"
          docsLink="/docs#commands"
        />

        {/* ═══════════════════ MCP INTEGRATION ═══════════════════ */}
        <FeatureSection
          id="mcp"
          label="MCP Integration"
          title="Connect tools natively via MCP."
          description="Configure Model Context Protocol servers that your agents connect to automatically. Supports stdio and SSE transport. Attach MCP servers to roles so every agent of that type gets the same capabilities."
          commands={[
            'bc mcp add github --command npx --args "@modelcontextprotocol/server-github"',
            'bc mcp list',
            'bc mcp enable github',
          ]}
          screenshot="/screenshots/dashboard-07-mcp.png"
          screenshotAlt="bc MCP view showing configured MCP servers with transport type and connection status"
          imageFirst
        />

        {/* ═══════════════════ CRON JOBS ═══════════════════ */}
        <FeatureSection
          id="cron"
          label="Scheduled Tasks"
          title="Cron-powered automation."
          description="Schedule recurring tasks with familiar cron syntax. Send prompts to agents on a schedule, run shell commands, and view execution history with full log retention."
          commands={[
            'bc cron add daily-lint --schedule "0 9 * * *" --agent qa-01 --prompt "Run make lint"',
            'bc cron list',
            'bc cron logs daily-lint --last 10',
          ]}
          screenshot="/screenshots/dashboard-08-cron.png"
          screenshotAlt="bc Cron Jobs view showing scheduled jobs with their schedules, run counts, and last execution times"
        />

        {/* ═══════════════════ SECRETS ═══════════════════ */}
        <FeatureSection
          id="secrets"
          label="Secrets Management"
          title="Encrypted credentials, zero plaintext."
          description="Store API keys, tokens, and credentials encrypted at rest using macOS Keychain, Linux libsecret, or AES-256-GCM fallback. Reference secrets in configs — no plaintext in your repo, ever."
          commands={[
            'bc secret set OPENAI_KEY',
            'bc secret list',
            'bc secret get GITHUB_TOKEN',
          ]}
          screenshot="/screenshots/dashboard-09-secrets.png"
          screenshotAlt="bc Secrets view showing stored secrets with encryption backend and creation dates"
          imageFirst
        />

        {/* ═══════════════════ STATS ═══════════════════ */}
        <FeatureSection
          id="stats"
          label="System Stats"
          title="Full visibility into your workspace."
          description="See agent counts, total cost, CPU/memory/disk usage, uptime, goroutines, and channel activity at a glance. The stats dashboard gives you a real-time system overview of your entire workspace."
          commands={[
            'bc stats',
            'bc status',
          ]}
          screenshot="/screenshots/dashboard-10-stats-loaded.png"
          screenshotAlt="bc Stats dashboard showing agent summary, system overview, resource usage bars, and runtime metrics"
        />

        {/* ═══════════════════ LOGS ═══════════════════ */}
        <FeatureSection
          id="logs"
          label="Centralized Logs"
          title="Every event, searchable and filterable."
          description="All agent activity, channel messages, cost events, and system events stream into a unified log. Filter by agent, level, or time range. Debug issues across your entire agent team from one place."
          commands={[
            'bc logs',
            'bc agent logs eng-01',
          ]}
          screenshot="/screenshots/dashboard-11-logs.png"
          screenshotAlt="bc Logs view showing a filterable stream of workspace events from agents and system"
          imageFirst
        />

        {/* ═══════════════════ DAEMONS ═══════════════════ */}
        <FeatureSection
          id="daemons"
          label="Daemon Processes"
          title="Long-running services, managed."
          description="Run background processes alongside your agents — databases, dev servers, watchers. bc manages their lifecycle, restarts them on failure, and shows their status in the dashboard."
          commands={[
            'bc status',
            'bc doctor',
          ]}
          screenshot="/screenshots/dashboard-13-daemons.png"
          screenshotAlt="bc Daemons view showing running background processes with their status and uptime"
        />

        {/* ═══════════════════ DOCTOR ═══════════════════ */}
        <FeatureSection
          id="doctor"
          label="System Health"
          title="One command to check everything."
          description="bc doctor checks 8 categories — workspace, database, agents, tools, MCP, secrets, git, and daemon. Found an issue? bc doctor fix auto-repairs what it can."
          commands={[
            'bc doctor',
            'bc doctor check tools',
            'bc doctor fix',
          ]}
          screenshot="/screenshots/dashboard-14-doctor.png"
          screenshotAlt="bc Doctor view showing system health checks across workspace, tools, agents, and configuration"
          imageFirst
        />

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
                desc: "Shell scripts break at scale. bc gives you structured communication, persistent memory, cost tracking, and a Web UI dashboard out of the box.",
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
