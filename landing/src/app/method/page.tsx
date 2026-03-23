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
    subtitle: "Shared state is the enemy of parallel work.",
    content: `Concurrent agents sharing a branch will destroy each other's work. This is not a tooling problem. It is a physics problem. Parallel writers need separate spaces.

Isolation is the foundation. Without it, every other principle collapses under merge conflicts and broken builds.`,
  },
  {
    number: "02",
    title: "Communication",
    subtitle: "Coordination without structure is noise.",
    content: `Isolated agents working in silence produce fragmented results. Agents need persistent, structured channels — not ad-hoc messages lost to scrollback.

The difference between five agents and a five-agent team is structured coordination. Without it, they duplicate effort and contradict each other. With it, they converge.`,
  },
  {
    number: "03",
    title: "Visibility",
    subtitle: "What you cannot see, you cannot trust.",
    content: `Every token, every cost, every tool call, every decision — attributed and observable in real time. When you see the complete picture, you intervene early. Not after the damage.

Agents left unchecked burn through budgets silently. Visibility is not a convenience. It is a safety mechanism.`,
  },
  {
    number: "04",
    title: "Persistence",
    subtitle: "A tool runs once. A teammate finishes the job.",
    content: `Most agents hit an error and stop. Complex work requires iteration: read state, implement, verify, commit, loop. Each cycle self-corrects from the last.

Hard problems decompose recursively — large into medium, medium into small, small into single commits. The loop runs until the whole is verified. Persistence is not stubbornness. It is structured determination.`,
  },
  {
    number: "05",
    title: "Surface",
    subtitle:
      "An agent's usefulness is bounded by what it can reach.",
    content: `Real development is not just editing files. It is filing issues, reviewing pull requests, testing in browsers, querying databases, deploying services. An agent confined to a text editor solves text-editor problems.

Every system a developer touches should be reachable by an agent. The surface expands. The boundaries hold.`,
  },
  {
    number: "06",
    title: "Openness",
    subtitle: "Universal problems deserve universal solutions.",
    content: `The problems of multi-agent orchestration are universal. Every team running AI agents faces them. Gating solutions behind company walls slows everyone down.

When the tool coordinating your agents is a black box, you cannot trust it. When it is open, you can verify every decision it makes. Open source is a conviction, not a strategy.`,
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
            These are not features. They are design convictions that shape
            everything we build.
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
            These are not features. They are design convictions. Every
            command, every view, every architectural decision exists because
            it serves one of these principles.
          </p>
        </div>
      </article>

      <Footer />
    </main>
  );
}
