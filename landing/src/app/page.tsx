import Link from "next/link";
import { Nav } from "./_components/Nav";
import { DashboardScreenshots } from "./_components/DashboardScreenshots";
import { Footer } from "./_components/Footer";
import { AnimatedBackground } from "./_components/AnimatedBackground";
import { HeroSection } from "./_components/HeroSection";
import {
  TerminalWindow,
  CommandOutput,
  ChannelView,
  CostTable,
  TreeDiagram,
  CronTable,
  RevealSection,
} from "./_components/TerminalComponents";
import { ToolMarquee } from "./_components/ToolLogos";
import { CheckCircle2, XCircle } from "lucide-react";
import { ArrowRight } from "lucide-react";

/* \u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550 */

export default function Home() {
  return (
    <main className="min-h-screen selection:bg-primary/20 selection:text-foreground overflow-x-hidden">
      {/* 3D particle background */}
      <AnimatedBackground />
      {/* Gradient overlay */}
      <div className="pointer-events-none fixed inset-0 z-[1] bg-[radial-gradient(ellipse_80%_60%_at_50%_-20%,rgba(234,88,12,0.04),transparent)] dark:bg-[radial-gradient(ellipse_80%_60%_at_50%_-20%,rgba(234,88,12,0.08),transparent)]" />

      <div className="relative z-[2]">
        <Nav />

        {/* \u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550 HERO \u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550 */}
        <HeroSection />

        {/* \u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550 COMPATIBLE AI TOOLS \u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550 */}
        <div className="mt-8 sm:mt-12 lg:mt-16 mx-auto max-w-6xl px-4 sm:px-6">
          <ToolMarquee />
        </div>

        <div className="mx-auto max-w-7xl px-4 sm:px-6">
          {/* \u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550 PROBLEM \u2192 SOLUTION \u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550 */}
          <RevealSection className="py-12 sm:py-16 lg:py-24" id="problem">
            <div className="mb-16">
              <span className="font-mono text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                The problem
              </span>
              <h2 className="mt-3 text-3xl font-bold tracking-tight sm:text-5xl">
                AI agents are powerful alone &mdash;
                <br />
                <span className="text-muted-foreground/50">
                  but chaotic when they work together.
                </span>
              </h2>
            </div>

            <div className="grid gap-8 lg:grid-cols-2 lg:gap-16">
              <div className="rounded-xl border border-destructive/20 bg-card/90 backdrop-blur-sm p-6">
                <h3 className="mb-5 flex items-center gap-2 font-mono text-sm font-bold uppercase tracking-wider text-destructive">
                  <XCircle className="h-4 w-4" aria-hidden="true" />
                  Without bc
                </h3>
                <ul className="space-y-4 text-sm">
                  {[
                    "Only one agent runs at a time, making development serial and painfully slow",
                    "Context is lost between sessions, wasting tokens on re-explaining the same codebase",
                    "Parallel edits on the same branch inevitably cause merge conflicts that block progress",
                    "There is no visibility into what your agents are doing or how much they are spending",
                    "Without budget controls, you discover surprise cost overruns at the end of the month",
                  ].map((t) => (
                    <li key={t} className="flex gap-3 text-muted-foreground">
                      <span className="text-destructive/60 shrink-0">
                        \u2715
                      </span>
                      {t}
                    </li>
                  ))}
                </ul>
              </div>

              <div className="rounded-xl border border-success/20 bg-card/90 backdrop-blur-sm p-6">
                <h3 className="mb-5 flex items-center gap-2 font-mono text-sm font-bold uppercase tracking-wider text-success">
                  <CheckCircle2 className="h-4 w-4" aria-hidden="true" />
                  With bc
                </h3>
                <ul className="space-y-4 text-sm">
                  {[
                    "Run 5 to 10 agents working in parallel, each on its own isolated branch",
                    "Persistent memory is injected on spawn so agents never need repeated context",
                    "Git worktrees give every agent its own branch, ensuring zero merge conflicts",
                    "Real-time channels, a live Web UI dashboard, and agent health monitoring keep you informed",
                    "Per-agent token tracking with budgets and automatic hard stops protect your spending",
                  ].map((t) => (
                    <li key={t} className="flex gap-3 text-muted-foreground">
                      <span className="text-success/60 shrink-0">\u2713</span>
                      {t}
                    </li>
                  ))}
                </ul>
              </div>
            </div>
          </RevealSection>

          {/* \u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550 HOW IT WORKS \u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550 */}
          <RevealSection className="py-12 sm:py-16 lg:py-24" id="how-it-works">
            <div className="mb-16 text-center">
              <span className="font-mono text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                How it works
              </span>
              <h2 className="mt-3 text-3xl font-bold tracking-tight sm:text-5xl">
                Three commands to fully orchestrate your team.
              </h2>
            </div>

            <div className="grid gap-6 md:grid-cols-3 md:gap-8">
              {[
                {
                  step: "01",
                  cmd: "bc init --preset small-team",
                  title: "Initialize workspace",
                  desc: "Creates .bc/ with config, roles, channels, and agent definitions. Choose a preset or configure from scratch.",
                  lines: [
                    {
                      text: "Initializing bc workspace...",
                      color: "text-terminal-muted",
                    },
                    {
                      text: "Created .bc/config.toml",
                      color: "text-terminal-muted",
                    },
                    {
                      text: "Created roles: manager, engineer",
                      color: "text-terminal-muted",
                    },
                    {
                      text: "Channels: #eng, #merge, #ops",
                      color: "text-terminal-muted",
                    },
                    { text: "Ready.", color: "text-terminal-success" },
                  ],
                },
                {
                  step: "02",
                  cmd: "bc daemon start",
                  title: "Start the daemon",
                  desc: "Launches the bcd server with the Web UI dashboard, WebSocket events, and MCP integration.",
                  lines: [
                    {
                      text: "Starting bcd server on :9374...",
                      color: "text-terminal-muted",
                    },
                    {
                      text: "\u2713 Web UI ready at localhost:9374",
                      color: "text-terminal-success",
                    },
                    {
                      text: "\u2713 MCP server connected",
                      color: "text-terminal-success",
                    },
                    {
                      text: "\u2713 WebSocket events streaming",
                      color: "text-terminal-success",
                    },
                  ],
                },
                {
                  step: "03",
                  cmd: "bc agent create eng-01 --role engineer --tool claude",
                  title: "Create agents",
                  desc: "Spawn agents in isolated git worktrees. Each gets a role, tool provider, memory context, and channel access.",
                  lines: [
                    {
                      text: "Created agent eng-01 (engineer, claude)",
                      color: "text-terminal-success",
                    },
                    {
                      text: "Worktree: .bc/worktrees/eng-01",
                      color: "text-terminal-muted",
                    },
                    {
                      text: "Agent started and working.",
                      color: "text-terminal-success",
                    },
                  ],
                },
              ].map((s, i) => (
                <div key={s.step}>
                  <div className="mb-4 font-mono text-xs font-bold uppercase tracking-[0.3em] text-muted-foreground/40">
                    Step {s.step}
                  </div>
                  <TerminalWindow title={`step ${s.step}`}>
                    <CommandOutput
                      command={s.cmd}
                      lines={s.lines}
                      delay={i * 0.2}
                    />
                  </TerminalWindow>
                  <h3 className="mt-5 mb-2 text-lg font-semibold tracking-tight">
                    {s.title}
                  </h3>
                  <p className="text-sm leading-relaxed text-muted-foreground">
                    {s.desc}
                  </p>
                </div>
              ))}
            </div>
          </RevealSection>

          {/* \u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550 FEATURE: ORCHESTRATION \u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550 */}
          <RevealSection className="py-12 sm:py-16 lg:py-24" id="orchestration">
            <div className="grid items-center gap-10 lg:grid-cols-2 lg:gap-20">
              <div>
                <span className="font-mono text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  Multi-Agent Orchestration
                </span>
                <h2 className="mt-3 text-3xl font-bold tracking-tight sm:text-4xl">
                  Structure your agent team like a real engineering org.
                </h2>
                <p className="mt-4 text-muted-foreground leading-relaxed">
                  The Product Manager sets strategic direction and priorities,
                  while Managers break down epics into actionable tasks for the
                  team. Engineers execute the implementation work, and QA agents
                  validate correctness before merging. Each role operates with
                  scoped permissions and distinct memory contexts to prevent
                  conflicts.
                </p>
                <div className="mt-6 font-mono text-sm text-muted-foreground">
                  <span className="text-[var(--terminal-prompt)]">$ </span>
                  bc agent create eng-03 --role engineer --tool claude
                </div>
              </div>
              <TerminalWindow
                title="agent hierarchy"
                ariaLabel="Agent hierarchy tree showing product-manager, managers, engineers, and QA roles"
              >
                <TreeDiagram
                  root={{
                    label: "pm-01",
                    role: "product-manager",
                    state: "working",
                    children: [
                      {
                        label: "mgr-01",
                        role: "manager",
                        state: "working",
                        children: [
                          {
                            label: "eng-01",
                            role: "engineer",
                            state: "working",
                          },
                          {
                            label: "eng-02",
                            role: "engineer",
                            state: "working",
                          },
                          { label: "qa-01", role: "qa", state: "idle" },
                        ],
                      },
                      {
                        label: "mgr-02",
                        role: "manager",
                        state: "idle",
                        children: [
                          {
                            label: "eng-03",
                            role: "engineer",
                            state: "working",
                          },
                          { label: "ux-01", role: "ux", state: "idle" },
                        ],
                      },
                    ],
                  }}
                />
              </TerminalWindow>
            </div>
          </RevealSection>

          {/* \u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550 FEATURE: CHANNELS \u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550 */}
          <RevealSection className="py-12 sm:py-16 lg:py-24" id="channels">
            <div className="grid items-center gap-10 lg:grid-cols-2 lg:gap-20">
              <div className="order-2 lg:order-1">
                <TerminalWindow
                  title="#engineering"
                  ariaLabel="Channel view showing agent-to-agent communication in the engineering channel"
                >
                  <ChannelView
                    name="engineering"
                    members={3}
                    messages={[
                      {
                        time: "10:15",
                        agent: "eng-01",
                        role: "engineer",
                        message: "PR #42 ready for review @mgr-01",
                      },
                      {
                        time: "10:16",
                        agent: "mgr-01",
                        role: "manager",
                        message: "Looking at it now \ud83d\udc40",
                      },
                      {
                        time: "10:18",
                        agent: "mgr-01",
                        role: "manager",
                        message: "LGTM, approved \u2713",
                      },
                      {
                        time: "10:19",
                        agent: "eng-01",
                        role: "engineer",
                        message: "Merged to main \ud83d\ude80",
                      },
                      {
                        time: "10:22",
                        agent: "eng-02",
                        role: "engineer",
                        message: "Starting on the auth module",
                      },
                    ]}
                  />
                </TerminalWindow>
              </div>
              <div className="order-1 lg:order-2">
                <span className="font-mono text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  Channel Communication
                </span>
                <h2 className="mt-3 text-3xl font-bold tracking-tight sm:text-4xl">
                  Agents talk to each other. Not through you.
                </h2>
                <p className="mt-4 text-muted-foreground leading-relaxed">
                  Agents coordinate through Slack-like channels such as #eng,
                  #pr, #standup, and #leads, complete with @mentions and
                  structured handoffs between team members. Every interaction is
                  automatically logged and fully searchable for audit and
                  debugging purposes.
                </p>
                <div className="mt-6 font-mono text-sm text-muted-foreground">
                  <span className="text-[var(--terminal-prompt)]">$ </span>
                  bc channel send #eng &quot;Review PR #42&quot;
                </div>
              </div>
            </div>
          </RevealSection>

          {/* \u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550 FEATURE: WORKTREES \u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550 */}
          <RevealSection className="py-12 sm:py-16 lg:py-24" id="worktrees">
            <div className="grid items-center gap-10 lg:grid-cols-2 lg:gap-20">
              <div>
                <span className="font-mono text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  Git Worktree Isolation
                </span>
                <h2 className="mt-3 text-3xl font-bold tracking-tight sm:text-4xl">
                  Every agent. Its own branch. Zero conflicts.
                </h2>
                <p className="mt-4 text-muted-foreground leading-relaxed">
                  Each agent operates inside its own isolated git worktree,
                  which means no agent can step on another&apos;s changes. The
                  result is clean pull requests that merge without any conflict,
                  every single time.
                </p>
              </div>
              <TerminalWindow
                title="git worktrees"
                ariaLabel="Git branch diagram showing three agents working on isolated worktree branches merged cleanly to main"
              >
                <div className="space-y-1 text-[13px]">
                  <div className="text-terminal-muted mb-3">
                    main
                    \u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500&gt;
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-terminal-comment">
                      {"     \\                    /"}
                    </span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-terminal-command">
                      {
                        "      eng-01/auth \u2500\u2500\u2500\u2500\u2500\u2500\u2500"
                      }
                    </span>
                    <span className="text-terminal-success text-[11px]">
                      (merged clean \u2713)
                    </span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-terminal-comment">
                      {"     \\                  /"}
                    </span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-terminal-prompt">
                      {"      eng-02/api \u2500\u2500\u2500\u2500\u2500\u2500"}
                    </span>
                    <span className="text-terminal-success text-[11px]">
                      (merged clean \u2713)
                    </span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-terminal-comment">
                      {"     \\                /"}
                    </span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-terminal-prompt">
                      {"      eng-03/ui \u2500\u2500\u2500\u2500\u2500"}
                    </span>
                    <span className="text-terminal-success text-[11px]">
                      (merged clean \u2713)
                    </span>
                  </div>
                </div>
              </TerminalWindow>
            </div>
          </RevealSection>

          {/* \u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550 FEATURE: COSTS \u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550 */}
          <RevealSection className="py-12 sm:py-16 lg:py-24" id="costs">
            <div className="grid items-center gap-10 lg:grid-cols-2 lg:gap-20">
              <div className="order-2 lg:order-1">
                <TerminalWindow
                  title="bc cost show"
                  ariaLabel="Cost tracking table showing per-agent token usage, spending, and budget limits"
                >
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
                        agent: "mgr-01",
                        tokensIn: "45,891",
                        tokensOut: "12,340",
                        cost: "$0.41",
                        budget: "$3.00",
                        percent: 14,
                      },
                    ]}
                    total={{ cost: "$4.22", budget: "$13.00" }}
                  />
                </TerminalWindow>
              </div>
              <div className="order-1 lg:order-2">
                <span className="font-mono text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  Cost Visibility
                </span>
                <h2 className="mt-3 text-3xl font-bold tracking-tight sm:text-4xl">
                  Know exactly what every agent costs.
                </h2>
                <p className="mt-4 text-muted-foreground leading-relaxed">
                  Every agent has per-token usage tracking with configurable
                  budget limits, automatic alerts when spending reaches 80%, and
                  hard stops that prevent runaway costs. You will never receive
                  a surprise bill from your AI agents again.
                </p>
                <div className="mt-6 font-mono text-sm text-muted-foreground">
                  <span className="text-[var(--terminal-prompt)]">$ </span>
                  bc cost budget set 5.00 --agent eng-01 --alert-at 0.8
                </div>
              </div>
            </div>
          </RevealSection>

          {/* \u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550 FEATURE: MEMORY \u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550 */}
          <RevealSection className="py-12 sm:py-16 lg:py-24" id="memory">
            <div className="grid items-center gap-10 lg:grid-cols-2 lg:gap-20">
              <div>
                <span className="font-mono text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  Persistent Memory
                </span>
                <h2 className="mt-3 text-3xl font-bold tracking-tight sm:text-4xl">
                  Agents remember. They learn. They get better.
                </h2>
                <p className="mt-4 text-muted-foreground leading-relaxed">
                  Every agent accumulates permanent learnings and time-stamped
                  experiences as it works through tasks. This persistent memory
                  carries across sessions and gets automatically injected
                  whenever an agent spawns, so context is never lost between
                  runs.
                </p>
              </div>
              <TerminalWindow
                title="memory: eng-01"
                ariaLabel="Persistent memory view showing agent learnings and time-stamped experiences"
              >
                <div className="space-y-4">
                  <div>
                    <div className="mb-2 text-[10px] font-bold uppercase tracking-[0.2em] text-terminal-muted">
                      Learnings (permanent)
                    </div>
                    <ul className="space-y-1.5 text-[12px] text-terminal-muted">
                      <li className="flex gap-2">
                        <span className="text-terminal-success shrink-0">
                          \u2022
                        </span>{" "}
                        Always run tests before submitting PR
                      </li>
                      <li className="flex gap-2">
                        <span className="text-terminal-success shrink-0">
                          \u2022
                        </span>{" "}
                        Use --preview flag for destructive operations
                      </li>
                      <li className="flex gap-2">
                        <span className="text-terminal-success shrink-0">
                          \u2022
                        </span>{" "}
                        The auth module requires the JWT_SECRET env var
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
                        <span className="text-terminal-success">\u2713</span>{" "}
                        Fixed auth token refresh \u2014 added retry with backoff
                      </div>
                      <div className="text-terminal-muted">
                        <span className="text-terminal-comment">
                          [Mar 15 12:15]
                        </span>{" "}
                        <span className="text-terminal-error">\u2717</span>{" "}
                        Build failed \u2014 missing env var, added to
                        .env.example
                      </div>
                      <div className="text-terminal-muted">
                        <span className="text-terminal-comment">
                          [Mar 14 16:45]
                        </span>{" "}
                        <span className="text-terminal-success">\u2713</span>{" "}
                        Refactored user service \u2014 reduced API calls by 40%
                      </div>
                    </div>
                  </div>
                </div>
              </TerminalWindow>
            </div>
          </RevealSection>

          {/* \u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550 FEATURE: CRON \u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550 */}
          <RevealSection className="py-12 sm:py-16 lg:py-24" id="cron">
            <div className="grid items-center gap-10 lg:grid-cols-2 lg:gap-20">
              <div className="order-2 lg:order-1">
                <TerminalWindow
                  title="bc cron list"
                  ariaLabel="Scheduled tasks table showing cron-powered test, deploy, and reporting jobs"
                >
                  <CronTable
                    jobs={[
                      {
                        name: "test-suite",
                        schedule: "*/30 * * * *",
                        nextRun: "in 14 min",
                        lastRun: "12:00 \u2713",
                        status: "enabled",
                      },
                      {
                        name: "deploy-staging",
                        schedule: "0 */2 * * *",
                        nextRun: "in 1h 22m",
                        lastRun: "10:00 \u2713",
                        status: "enabled",
                      },
                      {
                        name: "cost-report",
                        schedule: "0 9 * * *",
                        nextRun: "tomorrow",
                        lastRun: "today 9am \u2713",
                        status: "enabled",
                      },
                    ]}
                  />
                </TerminalWindow>
              </div>
              <div className="order-1 lg:order-2">
                <span className="font-mono text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  Scheduled Tasks
                </span>
                <h2 className="mt-3 text-3xl font-bold tracking-tight sm:text-4xl">
                  Automate the boring stuff. On a schedule.
                </h2>
                <p className="mt-4 text-muted-foreground leading-relaxed">
                  Schedule your test suite to run every 30 minutes, deploy to
                  staging every 2 hours, and generate comprehensive cost reports
                  each morning at 9am. All scheduled tasks are cron-powered and
                  fully observable through the Web UI dashboard.
                </p>
                <div className="mt-6 font-mono text-sm text-muted-foreground">
                  <span className="text-[var(--terminal-prompt)]">$ </span>
                  bc cron add test-suite --schedule &apos;*/30 * * * *&apos;
                  --cmd &apos;npm test&apos;
                </div>
              </div>
            </div>
          </RevealSection>

          {/* \u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550 INTERACTIVE DEMO \u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550 */}
          <RevealSection className="py-12 sm:py-16 lg:py-24" id="demo">
            <div className="mb-12 text-center">
              <span className="font-mono text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Web UI Dashboard
              </span>
              <h2 className="mt-3 text-3xl font-bold tracking-tight sm:text-4xl">
                See the real dashboard.
              </h2>
              <p className="mt-4 text-muted-foreground text-lg max-w-xl mx-auto">
                Monitor agents, channels, costs, and more from
                localhost:9374.
              </p>
            </div>
            <div className="mx-auto max-w-5xl">
              <DashboardScreenshots />
            </div>
          </RevealSection>

          {/* \u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550 FINAL CTA \u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550 */}
          <RevealSection className="pb-12 sm:pb-16 lg:pb-24">
            <div className="relative overflow-hidden rounded-2xl border border-border bg-card shadow-[var(--card-shadow)]">
              <div className="absolute inset-0 bg-[radial-gradient(circle_at_30%_50%,rgba(234,88,12,0.04),transparent)] dark:bg-[radial-gradient(circle_at_30%_50%,rgba(234,88,12,0.06),transparent)] pointer-events-none" />
              <div className="relative grid items-center gap-8 p-8 sm:p-12 lg:grid-cols-[1fr_auto] lg:gap-16 lg:p-16">
                <div>
                  <h2 className="text-3xl font-bold tracking-tight sm:text-5xl">
                    Start orchestrating in 60 seconds.
                  </h2>
                  <p className="mt-4 max-w-lg text-lg text-muted-foreground">
                    Go from running a single AI coding agent to coordinating an
                    entire team with just three terminal commands.
                  </p>
                  <div className="mt-8 flex flex-wrap items-center gap-4">
                    <Link
                      href="https://github.com/gh-curious-otter/bc"
                      className="group inline-flex h-12 items-center gap-2 rounded-lg bg-primary px-8 text-sm font-semibold text-primary-foreground shadow-[var(--btn-shadow)] transition-all hover:shadow-xl active:scale-[0.97]"
                      aria-label="Get started with bc on GitHub"
                    >
                      Get Started
                      <ArrowRight
                        className="h-4 w-4 transition-transform group-hover:translate-x-0.5"
                        aria-hidden="true"
                      />
                    </Link>
                    <Link
                      href="/docs"
                      className="inline-flex h-12 items-center gap-2 rounded-lg border border-border px-8 text-sm font-medium transition-colors hover:bg-accent/20 active:scale-[0.97]"
                      aria-label="Explore the bc CLI documentation"
                    >
                      Explore the Docs
                    </Link>
                  </div>
                </div>
                <TerminalWindow
                  title="quickstart"
                  className="min-w-[280px]"
                  ariaLabel="Quick start commands: bc init, bc daemon start, bc agent create"
                >
                  <div className="space-y-1.5 text-[13px]">
                    <div>
                      <span className="text-terminal-prompt">$ </span>bc init
                    </div>
                    <div>
                      <span className="text-terminal-prompt">$ </span>bc daemon
                      start
                    </div>
                    <div>
                      <span className="text-terminal-prompt">$ </span>bc agent
                      create eng-01 --role engineer --tool claude
                    </div>
                    <div className="text-terminal-comment mt-3 text-[12px]">
                      # That&apos;s it. Your agent team is running.
                    </div>
                  </div>
                </TerminalWindow>
              </div>
            </div>
          </RevealSection>
        </div>

        <Footer />
      </div>
    </main>
  );
}
