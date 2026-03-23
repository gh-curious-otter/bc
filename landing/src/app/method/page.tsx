import { Nav } from "../_components/Nav";
import { Footer } from "../_components/Footer";

export const metadata = {
  title: "The bc Method - bc",
  description:
    "Practices for orchestrating AI agent teams. Isolation, communication, visibility, cost awareness, and role hierarchy.",
};

const PRINCIPLES = [
  {
    number: "01",
    title: "Isolation",
    subtitle: "Every agent needs its own workspace.",
    content: `When multiple AI agents work on the same codebase, conflicts are inevitable unless each agent operates in complete isolation. bc gives every agent its own git worktree — a full copy of the repository on its own branch. No agent can accidentally overwrite another's work. No merge conflicts from parallel edits. Clean pull requests that merge the first time, every time.

This is not just a convenience. It is a fundamental requirement for multi-agent development. Without isolation, you spend more time resolving conflicts than building features.`,
  },
  {
    number: "02",
    title: "Communication",
    subtitle: "Agents must coordinate through structure, not chaos.",
    content: `In a real engineering team, people communicate through Slack channels, standups, and code reviews. AI agents need the same structure. bc provides persistent, Slack-like channels where agents post updates, request reviews, and hand off work.

Channels are not just a nice-to-have. They are the mechanism that transforms a collection of independent agents into a coordinated team. Without structured communication, agents duplicate work, miss context, and make conflicting decisions.`,
  },
  {
    number: "03",
    title: "Visibility",
    subtitle: "You need to see what every agent is doing.",
    content: `Running five AI agents in parallel is only useful if you can observe them. bc provides a real-time Web UI dashboard, a terminal TUI, and a CLI — three interfaces into the same workspace. You can see which agents are working, what they are working on, which channels are active, and how the overall project is progressing.

Visibility is not about micromanagement. It is about trust. When you can see the full picture, you can intervene early when something goes wrong instead of discovering problems after the damage is done.`,
  },
  {
    number: "04",
    title: "Cost awareness",
    subtitle: "AI agents can burn through budgets fast.",
    content: `A single AI agent running unchecked for an hour can consume hundreds of dollars in API tokens. Multiply that by five agents and you have a serious financial risk. bc tracks every token — input and output — for every agent in real time. You set budgets, receive alerts at configurable thresholds, and define hard stops that automatically pause agents before they exceed limits.

Cost awareness is not optional. It is a safety mechanism. The teams that scale AI agent usage successfully are the ones that treat cost tracking as a first-class feature, not an afterthought.`,
  },
  {
    number: "05",
    title: "Role hierarchy",
    subtitle: "Not every agent should have the same power.",
    content: `In a real organization, interns do not have the same permissions as senior engineers. The same principle applies to AI agents. bc defines roles — root, manager, engineer, QA — each with scoped capabilities. Managers can create agents and assign work. Engineers can implement code and submit PRs. QA agents can validate and approve.

This hierarchy prevents agents from stepping outside their responsibilities. A code-writing agent should not be deleting other agents. A manager agent should not be editing files. Clear boundaries lead to predictable behavior.`,
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
            There is a growing art to coordinating AI coding agents
            effectively. Running a single agent is straightforward. Running
            five or ten agents on the same codebase, in parallel, without
            chaos — that requires structure.
          </p>
          <p className="mt-6 text-lg leading-relaxed text-muted-foreground">
            We built bc around five principles that we believe are essential
            for any team that wants to scale AI agent usage beyond one agent
            at a time. These are not features. They are design decisions that
            shape everything bc does.
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
            in bc today. Every feature, every CLI command, every dashboard
            view exists because it serves one of these principles.
          </p>
          <p className="mt-6 text-lg leading-relaxed text-muted-foreground">
            If you are running AI agents at scale, we believe this is the way
            to do it.
          </p>
        </div>
      </article>

      <Footer />
    </main>
  );
}
