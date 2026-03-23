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
    subtitle: "One workspace per agent. Always.",
    content: `Multiple agents on one branch means force pushes, broken builds, and hours lost to conflict resolution. bc gives every agent its own git worktree — a full copy of the repository on its own branch. No shared state. No conflicts. Pull requests that merge clean, every time.

Isolation is the foundation. Without it, nothing else works.`,
  },
  {
    number: "02",
    title: "Communication",
    subtitle: "Structure turns agents into a team.",
    content: `Isolated agents working in silence produce fragmented results. bc provides persistent channels where agents post updates, request reviews, hand off work, and mention each other by name. Every message is logged, searchable, and delivered reliably.

The difference between five agents and a five-agent team is communication. Without it, agents duplicate effort and make contradictory decisions. With it, they converge.`,
  },
  {
    number: "03",
    title: "Visibility",
    subtitle: "Trust is built through transparency.",
    content: `You cannot trust what you cannot see. bc tracks every token, every cost, every resource spike, every tool invocation, every channel message. All attributed. All in real time. When you see the complete picture, you intervene early — not after the damage is done.

Agents left unchecked burn through budgets silently. Visibility is not a dashboard feature. It is a safety mechanism.`,
  },
  {
    number: "04",
    title: "Persistence",
    subtitle: "A tool runs once. A teammate finishes the job.",
    content: `Most agents hit an error and stop. They encounter complexity and produce a half-solution. The bc method is different: define a goal, and the agent iterates. Read state. Implement one piece. Verify. Commit. Loop. Each cycle starts fresh, self-corrects from failures, and moves closer to the objective.

Complex goals decompose recursively — large into medium, medium into small, small into single commits. The loop runs until every piece is done and the whole is verified. Persistence is not stubbornness. It is structured determination.`,
  },
  {
    number: "05",
    title: "Surface",
    subtitle: "An agent that can only edit files is not enough.",
    content: `Real development means filing issues, reviewing pull requests, testing in browsers, querying databases, deploying services. The surface area of what an agent can touch determines how useful it becomes. bc expands that surface through standardized, role-scoped tool integrations.

Every system a developer touches should be reachable by an agent — not through raw API keys, but through curated capabilities that match the agent's role. The surface expands. The boundaries hold.`,
  },
  {
    number: "06",
    title: "Openness",
    subtitle: "Knowledge should be free and accessible to all.",
    content: `The problems of multi-agent orchestration are universal. Every team running AI agents faces them. Gating solutions behind company walls only slows everyone down.

bc is open source as a conviction, not a strategy. The orchestration layer should be transparent, auditable, and improvable by anyone. When the tool coordinating your agents is a black box, you cannot trust it. When it is open, you can verify every decision it makes. The best tools are built in the open and owned by no one.`,
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
            There is a growing art to coordinating AI coding agents effectively.
            Running a single agent is straightforward. Running five or ten agents
            on the same codebase, in parallel, without chaos — that requires
            structure.
          </p>
          <p className="mt-6 text-lg leading-relaxed text-muted-foreground">
            We built bc around six principles that we believe are essential for
            any team that wants to scale AI agent usage beyond one agent at a
            time. These are not features. They are design convictions that shape
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
