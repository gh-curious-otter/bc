import { Nav } from "../_components/Nav";
import { Footer } from "../_components/Footer";

const ENTRIES = [
  {
    date: "Now",
    title: "What we're working on",
    items: [
      "Release infrastructure — automated versioning and distribution",
      "Homebrew tap for macOS installation",
      "Docker agent runtime improvements",
      "Thin CLI client migration — all commands via HTTP to bcd",
    ],
  },
  {
    date: "March 2026",
    title: "Landing page revamp",
    items: [
      "Dark theme by default matching the dashboard",
      "Real dashboard screenshots replace all mockups",
      "New /pricing page — free forever, no login required",
      "New /method page — the bc philosophy",
      "Bento grid feature layout on homepage",
      "Competitor-informed design (Linear, Cursor, Warp, Raycast)",
    ],
  },
  {
    date: "March 2026",
    title: "Documentation overhaul",
    items: [
      "Restructured docs following Diataxis framework (tutorials, how-to, reference, explanation)",
      "Auto-generated CLI reference from Cobra commands",
      "Security guide, ADRs, and testing documentation added",
      "Removed all internal issue references from public docs",
    ],
  },
  {
    date: "March 2026",
    title: "Web dashboard parity",
    items: [
      "15 dashboard views: Dashboard, Agents, Channels, Costs, Roles, Tools, MCP, Cron, Secrets, Stats, Logs, Workspace, Daemons, Doctor, Settings",
      "Real-time SSE updates across all views",
      "Responsive sidebar navigation",
      "Command palette (Cmd+K)",
      "Dark and light theme support",
    ],
  },
  {
    date: "March 2026",
    title: "Cron CRUD & cost tracking",
    items: [
      "Create, run, enable/disable, and delete cron jobs from CLI and Web UI",
      "Cost tracking with daily trend charts and per-agent breakdown",
      "Auto-import from Claude Code session files",
    ],
  },
  {
    date: "March 2026",
    title: "Core platform",
    items: [
      "Agent orchestration with 7 AI tool providers",
      "Git worktree isolation per agent",
      "SQLite-backed channels with @mentions and reactions",
      "Encrypted secrets management (AES-256-GCM)",
      "MCP server integration (SSE + stdio)",
      "Role-based agent hierarchy with scoped permissions",
      "TUI (terminal UI) with 13 views",
      "bcd daemon with REST API (44 endpoints)",
    ],
  },
];

export default function Now() {
  return (
    <main className="min-h-screen bg-background">
      <Nav />
      <section className="mx-auto max-w-3xl px-6 pt-24 pb-16 lg:pt-32">
        <div className="mb-16">
          <p className="text-xs font-mono font-bold text-primary uppercase tracking-widest mb-4">
            Now
          </p>
          <h1 className="text-4xl sm:text-5xl font-bold tracking-tight mb-4">
            What&apos;s happening with bc.
          </h1>
          <p className="text-muted-foreground text-lg">
            Current focus, recent updates, and what&apos;s next.
          </p>
        </div>

        <div className="space-y-16">
          {ENTRIES.map((entry) => (
            <article
              key={entry.title}
              className="relative pl-8 border-l-2 border-border"
            >
              <div className="absolute -left-[9px] top-1 h-4 w-4 rounded-full border-2 border-primary bg-background" />
              <time className="text-xs font-mono text-muted-foreground uppercase tracking-widest">
                {entry.date}
              </time>
              <h2 className="text-xl font-bold mt-2 mb-4">{entry.title}</h2>
              <ul className="space-y-2">
                {entry.items.map((item) => (
                  <li
                    key={item}
                    className="text-sm text-muted-foreground leading-relaxed flex gap-2"
                  >
                    <span className="text-primary mt-0.5 flex-shrink-0">
                      +
                    </span>
                    {item}
                  </li>
                ))}
              </ul>
            </article>
          ))}
        </div>
      </section>
      <Footer />
    </main>
  );
}
