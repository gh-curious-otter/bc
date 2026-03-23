import { Nav } from "../_components/Nav";
import { Footer } from "../_components/Footer";

export const metadata = {
  title: "The bc Method - bc",
  description:
    "Five principles for orchestrating AI agent teams: isolation, communication, visibility, persistence, and agency.",
};

const PRINCIPLES = [
  {
    number: "01",
    title: "Isolation",
    subtitle: "One workspace per agent. Always.",
    content: `bc was born from failure. Its predecessor, an earlier orchestrator called Gas Town, let multiple AI agents work on the same branch. The result was predictable in hindsight: agents force-pushed each other's work, created irreconcilable merge conflicts, and burned through tokens at over a hundred dollars per hour just resolving the chaos they created. The founding lesson was simple and absolute — every agent needs its own workspace.

In bc, every agent gets its own git worktree on its own branch. Not a fork. Not a clone. A worktree — a lightweight, full working copy of the repository that shares the same git history but maintains completely independent state. This means agents cannot touch each other's files. Not by accident, not by design. The isolation is structural, not conventional. There is no rule saying "please don't edit that file." There is a filesystem boundary that makes it impossible.

This approach works uniformly across all seven providers bc supports — Claude, Gemini, Cursor, Codex, Aider, OpenCode, and OpenClaw. Regardless of which AI is doing the work, the isolation model is identical. When an agent finishes its task, it produces a clean pull request from its branch. No conflict resolution. No rebasing someone else's half-finished work. The worktree is created when the agent starts and cleaned up when the agent is done. This is the foundational requirement. Without it, everything else falls apart.`,
  },
  {
    number: "02",
    title: "Communication",
    subtitle: "Structure turns a crowd into a team.",
    content: `A collection of isolated agents is not a team. It is a crowd. What transforms isolated workers into a coordinated unit is the same thing that transforms any group of people into a team: structured communication. Not reading each other's output. Not sharing files. Not hoping context travels by osmosis. Real, persistent, directed messaging.

bc provides typed, persistent channels where agents post updates, request reviews, hand off work, and coordinate decisions. Every message is logged and searchable. Delivery is reliable — messages are written to the agent's terminal, ensuring they enter the agent's context window regardless of what the agent is currently doing. Agents can mention each other, react to messages, and maintain threaded conversations across channel topics. This mirrors how real engineering teams operate: dedicated channels for different concerns, broadcast channels for announcements, direct mentions for urgent handoffs.

The key insight is that ad-hoc coordination does not scale. When two agents work together, they can maybe get by with loose conventions. When ten agents work in parallel across a complex codebase, you need the same communication infrastructure that human teams rely on. Structured messaging with clear channels, reliable delivery, and a searchable history is not overhead. It is the mechanism that makes parallel work possible without parallel chaos.`,
  },
  {
    number: "03",
    title: "Visibility",
    subtitle: "Trust requires transparency.",
    content: `Running multiple AI agents in parallel is an act of trust. You are delegating real work — work that costs real money, touches real code, and affects real deadlines — to systems that operate autonomously. That trust must be earned, and it is earned through transparency. Visibility is not micromanagement. It is the foundation that makes delegation possible.

bc tracks everything by default. Agent status and activity. Token usage broken down by input, output, and cache. Cost per agent, per session, per hour. Resource consumption. Tool invocations. Channel activity. Event logs with timestamps. All of this exists because the alternative — running agents blind and hoping for the best — is how Gas Town's hundred-dollar-per-hour burn rates happened. When you can see that an agent has spent forty dollars in twenty minutes without producing a commit, you intervene. When you can see that an agent is stuck in a loop, retrying the same failing approach, you redirect it. When you can see that costs are tracking toward your budget limit, you make informed decisions about which agents to continue and which to pause.

This principle deliberately absorbs what might otherwise be treated as a separate concern: cost awareness. Cost is not a separate category — it is one dimension of visibility. The teams that scale AI agent usage successfully are the ones that treat comprehensive monitoring as a first-class requirement, not a dashboard they check after the bill arrives. Visibility turns autonomous agents from a liability into an asset by ensuring you always know what is happening, what it costs, and whether it is working.`,
  },
  {
    number: "04",
    title: "Persistence",
    subtitle: "A tool runs once. A teammate persists until the job is done.",
    content: `AI agents give up too easily. They hit an error and stop. They encounter a complex task and produce a half-solution. They lose context halfway through a multi-step implementation and deliver something that compiles but does not work. This is the fundamental limitation of single-shot agent usage — you give it a prompt, it runs once, and you hope the output is correct. For trivial tasks, this works. For anything that matters, it does not.

The bc method is different. Instead of running an agent once and hoping for the best, you define a goal and the agent loops toward it. Each iteration starts fresh: the agent reads external state — the actual codebase, the actual test results, the actual CI output — rather than relying on stale context from a previous run. It implements one piece, verifies it, commits if the tests pass, and repeats. If a step fails, the agent reads the failure, adjusts its approach, and tries again. This is not retry logic. It is goal-oriented iteration with self-correction, where each cycle begins by observing reality rather than remembering what reality used to look like.

This pattern extends naturally to complex tasks through recursive decomposition. A task too large for a single agent session gets broken into smaller pieces — each one achievable in a single focused iteration. An extra-large feature becomes several large tasks. A large task becomes a set of medium tasks. A medium task becomes a handful of small, concrete implementation steps. Each step is a self-contained iteration: read state, implement, verify, commit. bc's cron scheduling enables this automatically — agents can be configured to wake up on a schedule, check for work, execute the next iteration, and go back to sleep. This is what separates a tool from a teammate. A tool runs once. A teammate persists until the job is done.`,
  },
  {
    number: "05",
    title: "Agency",
    subtitle: "A code editor is not enough.",
    content: `An AI agent that can only read and write files is fundamentally limited. It can produce code, but it cannot participate in the development process. It cannot create a GitHub issue when it discovers a prerequisite task. It cannot review a pull request from a teammate. It cannot run a browser test to verify its UI changes render correctly. It cannot query a database to understand the data model. It cannot send a message to another agent asking for clarification. It is, at best, an autocomplete engine with a large context window.

bc transforms agents from code editors into full participants through MCP — the Model Context Protocol. Agents connect to MCP servers that provide typed, permissioned access to external tools. A lead agent might have access to GitHub for creating issues and reviewing pull requests, plus messaging tools for coordinating its team. An engineer agent might get GitHub for pushing code, Playwright for browser testing, and database access for integration verification. A QA agent might get read-only GitHub access and full Playwright access for validation. The tools are curated and scoped to match each agent's role — not raw API keys that grant unlimited access, but structured capabilities with clear boundaries.

This is what makes multi-agent development genuinely useful rather than merely novel. When an agent can create an issue, implement the fix, verify it in a browser, push the branch, open a pull request, and notify the team — all autonomously, all within its role's permissions — it is no longer a tool you operate. It is a teammate you delegate to. The combination of MCP integrations with role-based access control means agents can interact with the full development ecosystem while staying within safe, predictable boundaries. Agency without guardrails is dangerous. Agency with structure is powerful.`,
  },
];

export default function MethodPage() {
  return (
    <main className="min-h-screen selection:bg-primary/20 selection:text-foreground">
      <Nav />

      <article className="mx-auto max-w-3xl px-4 sm:px-6 py-16 sm:py-24">
        {/* Header */}
        <header className="mb-20">
          <span className="font-mono text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
            Philosophy
          </span>
          <h1
            className="mt-4 text-5xl font-bold tracking-tight sm:text-7xl"
            style={{ fontFamily: "Georgia, 'Times New Roman', serif" }}
          >
            The bc Method
          </h1>
          <p className="mt-6 text-xl text-muted-foreground leading-relaxed max-w-2xl">
            Five principles for orchestrating AI agent teams, born from
            building the system that needed them.
          </p>
        </header>

        {/* Introduction */}
        <div className="prose-section mb-20">
          <p className="text-lg leading-relaxed text-muted-foreground">
            Running a single AI agent is straightforward. You give it a task,
            it produces output, you review the result. The challenge begins
            when you want more — five agents, ten agents, working in parallel
            across a real codebase with real deadlines and real costs. That
            requires more than a better prompt. It requires structure.
          </p>
          <p className="mt-6 text-lg leading-relaxed text-muted-foreground">
            These five principles are not theoretical. They emerged from
            building bc and its predecessor, from watching multi-agent
            systems fail in specific, repeatable ways, and from discovering
            what actually works when you scale AI development beyond a single
            session. Each principle exists because ignoring it produced a
            concrete, expensive failure.
          </p>
        </div>

        <div className="w-full h-px bg-border mb-20" />

        {/* Principles */}
        {PRINCIPLES.map((p, i) => (
          <section key={p.number} className={i > 0 ? "mt-20" : ""}>
            <div className="mb-6">
              <span className="font-mono text-xs font-bold uppercase tracking-[0.3em] text-muted-foreground/50">
                Principle {p.number}
              </span>
            </div>
            <h2
              className="text-3xl font-bold tracking-tight sm:text-4xl"
              style={{ fontFamily: "Georgia, 'Times New Roman', serif" }}
            >
              {p.title}
            </h2>
            <p className="mt-2 text-lg text-muted-foreground italic">
              {p.subtitle}
            </p>
            <div className="mt-8 space-y-6">
              {p.content.split("\n\n").map((paragraph, j) => (
                <p
                  key={j}
                  className="text-base leading-relaxed text-muted-foreground"
                >
                  {paragraph}
                </p>
              ))}
            </div>
            {i < PRINCIPLES.length - 1 && (
              <div className="mt-20 w-16 h-px bg-border" />
            )}
          </section>
        ))}

        {/* Closing */}
        <div className="mt-24 pt-16 border-t border-border">
          <p className="text-lg leading-relaxed text-muted-foreground">
            These five principles are not aspirational. They are implemented
            in bc today, and they were each paid for with a failure that made
            them obvious in retrospect. Isolation came from agents destroying
            each other&apos;s work. Communication came from agents duplicating
            effort in silence. Visibility came from surprise bills.
            Persistence came from agents that quit at the first error. Agency
            came from agents that could write code but could not ship it.
          </p>
          <p className="mt-6 text-lg leading-relaxed text-muted-foreground">
            If you are building with AI agents at scale, these are the
            problems you will encounter. This is how we solved them.
          </p>
        </div>
      </article>

      <Footer />
    </main>
  );
}
