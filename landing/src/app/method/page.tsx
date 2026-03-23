import { Nav } from "../_components/Nav";
import { Footer } from "../_components/Footer";

export const metadata = {
  title: "The bc Method - bc",
  description:
    "Six principles for orchestrating AI agent teams. Isolation, communication, visibility, persistence, surface, and openness.",
};

const PRINCIPLES = [
  {
    number: "01",
    title: "Isolation",
    subtitle: "Every agent needs its own workspace.",
    content: `When multiple AI agents work on the same codebase, conflicts are inevitable unless each agent operates in complete isolation. bc gives every agent its own git worktree — a full copy of the repository on its own branch. No shared state. No merge conflicts from parallel edits. Clean pull requests that merge the first time, every time.

This principle was born from failure. An earlier system we built let agents share branches. The result was force pushes overwriting each other's work, broken builds from conflicting changes, and hours spent on manual conflict resolution. The founding lesson was simple: one workspace per agent, always.

Isolation is not just about preventing conflicts. It gives each agent the freedom to experiment, refactor, and iterate without fear of breaking someone else's work. It is the foundation that makes everything else possible.`,
  },
  {
    number: "02",
    title: "Communication",
    subtitle: "Agents coordinate through structure, not chaos.",
    content: `Isolated agents working in silence produce fragmented results. Coordination requires structure. bc provides persistent channels where agents post updates, request reviews, hand off work, and mention each other by name. Every message is logged, searchable, and delivered reliably.

Without structured communication, agents duplicate effort, miss context, and make contradictory decisions. With it, they become a team — passing context forward, building on each other's work, and converging toward a shared goal. The difference between five agents and a five-agent team is communication.

This mirrors how real engineering organizations work. People do not coordinate by reading each other's code diffs. They coordinate through conversation, review, and explicit handoffs. AI agents need the same structure.`,
  },
  {
    number: "03",
    title: "Visibility",
    subtitle: "Trust requires transparency.",
    content: `Running five AI agents in parallel is only useful if you can see what they are doing. Not just their output — their costs, their resource usage, their activity patterns, their tool invocations. Visibility is the mechanism that builds trust between you and your agents.

Every token is tracked. Every cost is attributed to the agent that incurred it. Resource usage — CPU, memory, disk — is monitored in real time. Channel activity shows who is talking to whom. Event logs capture the full timeline. When you can see the complete picture, you can intervene early when something drifts instead of discovering problems after the damage is done.

This principle was born from a concrete problem: agents running unchecked for hours, burning through hundreds of dollars in API tokens before anyone noticed. The teams that scale AI agent usage successfully are the ones that treat visibility as a first-class requirement, not an afterthought.`,
  },
  {
    number: "04",
    title: "Persistence",
    subtitle: "Agents iterate toward goals. They do not give up.",
    content: `Most AI agents run once and stop. They hit an error, they halt. They encounter a complex task, they produce a partial solution and declare it done. This is not how real work gets accomplished.

The bc method is different. Instead of running an agent once and hoping for the best, you define a goal and the agent iterates — reading the current state, implementing one piece, verifying the result, committing if it passes, and looping again. Each iteration starts with fresh context, reads external state rather than relying on stale memory, and self-corrects from previous failures.

This is what separates a tool from a teammate. A tool runs once. A teammate persists until the job is done. Complex tasks decompose recursively — large goals break into medium goals, medium into small, small into single commits. The loop continues until every piece is complete and the whole is verified. Persistence is not stubbornness. It is structured determination.`,
  },
  {
    number: "05",
    title: "Surface",
    subtitle: "The boundary of what agents can do should always expand.",
    content: `An AI agent that can only edit files is fundamentally limited. Real development involves filing issues, reviewing pull requests, testing in browsers, querying databases, deploying services, and communicating across tools. The surface area of what an agent can interact with determines how useful it can be.

bc treats tool integration as a core architectural concern, not a plugin afterthought. Agents connect to external systems through standardized protocols, gaining typed and permissioned access to capabilities that match their role. An engineer agent gets different tools than a manager agent. The surface expands, but within boundaries.

The long-term trajectory is clear: every system a developer touches should be reachable by an agent. Not through raw API keys and ad-hoc scripts, but through curated, role-appropriate integrations that make agents full participants in the development process.`,
  },
  {
    number: "06",
    title: "Openness",
    subtitle: "Knowledge should be free and accessible to all.",
    content: `Gating tools behind company walls only slows progress. The problems of multi-agent orchestration — isolation, coordination, cost control, goal persistence — are universal. Every team running AI agents faces them. Solutions to universal problems should be universally available.

bc is open source not as a marketing strategy, but as a conviction. The orchestration layer should be transparent, auditable, and improvable by anyone who uses it. When the tool that coordinates your AI agents is a black box, you cannot trust it. When it is open, you can verify every decision it makes.

This extends beyond code. Documentation, design decisions, architecture trade-offs, and even the failures that shaped these principles — all of it is public. The best tools are built in the open, shaped by the people who use them, and owned by no one.`,
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
            Practices for orchestrating AI agent teams.
          </p>
        </header>

        {/* Introduction */}
        <div className="prose-section mb-20">
          <p className="text-lg leading-relaxed text-muted-foreground">
            Running a single AI coding agent is straightforward. Running five or
            ten agents on the same codebase, in parallel, without chaos — that
            requires a method.
          </p>
          <p className="mt-6 text-lg leading-relaxed text-muted-foreground">
            We learned this the hard way. Before bc, we built an orchestrator
            where agents shared branches, had no communication structure, and ran
            with unlimited budgets. The result was expensive chaos. These six
            principles emerged from that failure and from every iteration since.
          </p>
          <p className="mt-6 text-lg leading-relaxed text-muted-foreground">
            They are not features. They are design convictions that shape
            everything bc does.
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
            These six principles are not aspirational. They are implemented in bc
            today. Every command, every view, every architectural decision exists
            because it serves one of these principles.
          </p>
          <p className="mt-6 text-lg leading-relaxed text-muted-foreground">
            If you are orchestrating AI agents at scale, we believe this is the
            way to do it.
          </p>
        </div>
      </article>

      <Footer />
    </main>
  );
}
