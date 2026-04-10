import { useCallback, useEffect, useRef, useState } from "react";
import { useParams, Link, useNavigate } from "react-router-dom";
import { api } from "../api/client";
import type { Agent } from "../api/client";
import { usePolling } from "../hooks/usePolling";
import { useWebSocket } from "../hooks/useWebSocket";
import { StatusBadge } from "../components/StatusBadge";
import { StatsTab as StatsTabComponent } from "../components/StatsTab";
import { WebTerminal } from "../components/WebTerminal";
import { stripAnsi } from "../utils/text";

function RoleBadge({ role }: { role: string }) {
  return (
    <span className="inline-block px-2 py-0.5 rounded text-xs font-medium bg-bc-accent/20 text-bc-accent">
      {role}
    </span>
  );
}

function formatTime(t?: string): string {
  if (!t) return "\u2014";
  try {
    const d = new Date(t);
    if (isNaN(d.getTime())) return "\u2014";
    return d.toLocaleString();
  } catch {
    return "\u2014";
  }
}

function formatRelative(t?: string): string {
  if (!t) return "";
  try {
    const d = new Date(t);
    if (isNaN(d.getTime())) return "";
    const diffMs = Date.now() - d.getTime();
    const diffSec = Math.floor(Math.abs(diffMs) / 1000);
    if (diffSec < 60) return `${String(diffSec)}s ago`;
    const diffMin = Math.floor(diffSec / 60);
    if (diffMin < 60) return `${String(diffMin)}m ago`;
    const diffHr = Math.floor(diffMin / 60);
    if (diffHr < 24) return `${String(diffHr)}h ago`;
    const diffDay = Math.floor(diffHr / 24);
    if (diffDay < 30) return `${String(diffDay)}d ago`;
    return d.toLocaleDateString();
  } catch {
    return "";
  }
}

/* ───────────────────────── Tab types ───────────────────────── */

type Tab = "logs" | "terminal" | "info";

const TABS: { key: Tab; label: string; shortcut: string }[] = [
  { key: "logs", label: "Logs", shortcut: "1" },
  { key: "terminal", label: "Terminal", shortcut: "2" },
  { key: "info", label: "Info", shortcut: "3" },
];

/* ───────────────────────── Tab content ───────────────────────── */

function LogsTab({
  outputLines,
  outputRef,
}: {
  outputLines: string[];
  outputRef: React.RefObject<HTMLPreElement>;
}) {
  return (
    <div className="space-y-2">
      <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide">
        Live Output
      </h2>
      <pre
        ref={outputRef}
        className="rounded-lg border border-bc-border/50 bg-bc-bg p-4 text-xs leading-relaxed overflow-y-auto overflow-x-hidden max-h-[50vh] md:max-h-[70vh] whitespace-pre-wrap break-words text-bc-text/90 shadow-inner w-full min-w-0"
        style={{
          fontFamily:
            "'Space Mono', ui-monospace, SFMono-Regular, Menlo, Consolas, monospace",
        }}
      >
        {outputLines.length > 0 ? (
          outputLines.join("\n")
        ) : (
          <span className="text-bc-muted italic">
            No output yet. Agent may be idle or stopped.
          </span>
        )}
      </pre>
    </div>
  );
}

/* ───────────────────────── Info tab building blocks ───────────────────────── */

function SectionHeader({ children }: { children: React.ReactNode }) {
  return (
    <div className="mb-3 flex items-baseline gap-3">
      <h2 className="text-[10px] font-semibold text-bc-muted uppercase tracking-[0.18em]">
        {children}
      </h2>
      <div className="flex-1 h-px bg-bc-border/40" />
    </div>
  );
}

function AgentPill({
  name,
  onClick,
}: {
  name: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-md border border-bc-border bg-bc-surface hover:border-bc-accent/50 hover:bg-bc-accent/5 transition-colors text-xs font-medium text-bc-text focus:outline-none focus-visible:ring-2 focus-visible:ring-bc-accent"
    >
      <span className="w-1 h-1 rounded-full bg-bc-accent/70" />
      <span
        style={{
          fontFamily:
            "'Space Mono', ui-monospace, SFMono-Regular, Menlo, Consolas, monospace",
        }}
      >
        {name}
      </span>
    </button>
  );
}

interface TimelineEvent {
  key: string;
  label: string;
  timestamp?: string;
  detail?: string;
  active: boolean;
}

function buildTimeline(agent: Agent): TimelineEvent[] {
  const events: TimelineEvent[] = [];
  const isRunning = agent.state !== "stopped" && agent.state !== "error";

  if (agent.created_at) {
    events.push({
      key: "created",
      label: "Created",
      timestamp: agent.created_at,
      active: false,
    });
  }
  if (agent.started_at) {
    events.push({
      key: "started",
      label: "Started",
      timestamp: agent.started_at,
      active: false,
    });
  }
  if (isRunning) {
    // Current running state — skip stale stopped_at from a previous run.
    events.push({
      key: "current",
      label:
        agent.state === "working"
          ? "Working"
          : agent.state === "starting"
          ? "Starting"
          : agent.state === "idle"
          ? "Idle"
          : "Active",
      timestamp: agent.updated_at,
      detail: agent.task,
      active: true,
    });
  } else if (agent.stopped_at) {
    // Stopped state is current — show it as the active event.
    events.push({
      key: "stopped",
      label: agent.state === "error" ? "Errored" : "Stopped",
      timestamp: agent.stopped_at,
      detail: agent.task,
      active: true,
    });
  }
  return events;
}

function InfoTab({ agent }: { agent: Agent }) {
  const navigate = useNavigate();
  const [metaOpen, setMetaOpen] = useState(true);

  const isStopped = agent.state === "stopped" || agent.state === "error";
  const timeline = buildTimeline(agent);
  const lastActivity =
    agent.stopped_at ??
    agent.updated_at ??
    agent.started_at ??
    agent.created_at;

  return (
    <div className="max-w-4xl mx-auto space-y-10">
      {/* ── CURRENT TASK BANNER ── */}
      <section>
        <div
          className={`rounded-lg border p-4 transition-colors ${
            isStopped
              ? "border-bc-border/60 bg-bc-surface/50"
              : "border-bc-accent/30 bg-bc-accent/5"
          }`}
        >
          <div className="flex items-start justify-between gap-4">
            <div className="flex-1 min-w-0">
              <div className="text-[10px] font-semibold text-bc-muted uppercase tracking-[0.18em] mb-1.5">
                {isStopped ? "Last Task" : "Current Task"}
              </div>
              <p className="text-sm text-bc-text break-words leading-relaxed">
                {agent.task ? (
                  agent.task
                ) : (
                  <span className="text-bc-muted italic">no task recorded</span>
                )}
              </p>
            </div>
            {lastActivity && (
              <div className="shrink-0 text-right">
                <div className="text-[10px] font-semibold text-bc-muted uppercase tracking-[0.18em] mb-1.5">
                  {isStopped ? "Last ran" : "Updated"}
                </div>
                <span
                  className="text-sm text-bc-text tabular-nums"
                  title={formatTime(lastActivity)}
                >
                  {formatRelative(lastActivity)}
                </span>
              </div>
            )}
          </div>
        </div>
        {isStopped && (
          <p className="mt-2 ml-1 text-[11px] text-bc-muted/70 italic">
            Agent is not running. Stats below show last known values.
          </p>
        )}
      </section>

      {/* ── STATS ── */}
      <section>
        <SectionHeader>Stats</SectionHeader>
        <StatsTabComponent agent={agent} />
      </section>

      {/* ── HIERARCHY ── */}
      <section>
        <SectionHeader>Hierarchy</SectionHeader>
        <div className="space-y-3">
          <div className="flex items-center gap-3">
            <span className="text-[11px] text-bc-muted w-16 shrink-0 uppercase tracking-wider">
              Parent
            </span>
            {agent.parent_id ? (
              <AgentPill
                name={agent.parent_id}
                onClick={() => {
                  navigate(`/agents/${encodeURIComponent(agent.parent_id ?? "")}`);
                }}
              />
            ) : (
              <span className="text-xs text-bc-muted/40">—</span>
            )}
          </div>
          <div className="flex items-start gap-3">
            <span className="text-[11px] text-bc-muted w-16 shrink-0 pt-1 uppercase tracking-wider">
              Children
            </span>
            <div className="flex flex-wrap gap-1.5">
              {agent.children && agent.children.length > 0 ? (
                agent.children.map((c) => (
                  <AgentPill
                    key={c}
                    name={c}
                    onClick={() => {
                      navigate(`/agents/${encodeURIComponent(c)}`);
                    }}
                  />
                ))
              ) : (
                <span className="text-xs text-bc-muted/40 pt-1">—</span>
              )}
            </div>
          </div>
        </div>
      </section>

      {/* ── ACTIVITY TIMELINE ── */}
      <section>
        <SectionHeader>Activity</SectionHeader>
        {timeline.length === 0 ? (
          <p className="text-xs text-bc-muted/40">No activity recorded</p>
        ) : (
          <ol className="relative ml-1">
            {/* Vertical rail */}
            <span
              aria-hidden
              className="absolute left-[3px] top-2 bottom-2 w-px bg-bc-border/60"
            />
            {timeline.map((evt) => (
              <li key={evt.key} className="relative pl-6 pb-4 last:pb-0">
                {/* Dot */}
                <span
                  aria-hidden
                  className={`absolute left-0 top-[7px] w-[9px] h-[9px] rounded-full border-2 ${
                    evt.active
                      ? "bg-bc-accent border-bc-accent animate-pulse"
                      : "bg-bc-bg border-bc-muted/60"
                  }`}
                />
                <div className="flex items-baseline justify-between gap-3">
                  <span
                    className={`text-sm font-medium ${
                      evt.active ? "text-bc-accent" : "text-bc-text/85"
                    }`}
                  >
                    {evt.label}
                  </span>
                  {evt.timestamp && (
                    <span
                      className="text-[11px] text-bc-muted tabular-nums shrink-0"
                      title={formatTime(evt.timestamp)}
                    >
                      {formatRelative(evt.timestamp)}
                    </span>
                  )}
                </div>
                {evt.detail && (
                  <p className="mt-0.5 text-xs text-bc-muted break-words leading-relaxed">
                    {evt.detail}
                  </p>
                )}
              </li>
            ))}
          </ol>
        )}
      </section>

      {/* ── METADATA (collapsible) ── */}
      <section>
        <button
          type="button"
          onClick={() => {
            setMetaOpen((v) => !v);
          }}
          className="w-full flex items-center gap-3 text-left group"
          aria-expanded={metaOpen}
        >
          <span className="text-[10px] font-semibold text-bc-muted uppercase tracking-[0.18em] group-hover:text-bc-text transition-colors">
            Metadata
          </span>
          <div className="flex-1 h-px bg-bc-border/40" />
          <span className="text-[10px] text-bc-muted tabular-nums">
            {metaOpen ? "−" : "+"}
          </span>
        </button>
        {metaOpen && (
          <dl className="mt-4 grid grid-cols-[6rem_1fr] gap-y-2.5 gap-x-4 text-sm">
            <dt className="text-[11px] text-bc-muted uppercase tracking-wider pt-0.5">
              Role
            </dt>
            <dd>
              <RoleBadge role={agent.role} />
            </dd>

            <dt className="text-[11px] text-bc-muted uppercase tracking-wider pt-0.5">
              Tool
            </dt>
            <dd
              className="text-xs text-bc-text/80"
              style={{
                fontFamily:
                  "'Space Mono', ui-monospace, SFMono-Regular, Menlo, Consolas, monospace",
              }}
            >
              {agent.tool || "—"}
            </dd>

            <dt className="text-[11px] text-bc-muted uppercase tracking-wider pt-0.5">
              Runtime
            </dt>
            <dd
              className="text-xs text-bc-text/80"
              style={{
                fontFamily:
                  "'Space Mono', ui-monospace, SFMono-Regular, Menlo, Consolas, monospace",
              }}
            >
              {agent.runtime_backend || "—"}
            </dd>

            <dt className="text-[11px] text-bc-muted uppercase tracking-wider pt-0.5">
              Session
            </dt>
            <dd
              className="text-xs text-bc-text/80 break-all"
              style={{
                fontFamily:
                  "'Space Mono', ui-monospace, SFMono-Regular, Menlo, Consolas, monospace",
              }}
            >
              {agent.session || "—"}
            </dd>

            {agent.mcp_servers && agent.mcp_servers.length > 0 && (
              <>
                <dt className="text-[11px] text-bc-muted uppercase tracking-wider pt-1">
                  MCP
                </dt>
                <dd>
                  <div className="flex flex-wrap gap-1">
                    {agent.mcp_servers.map((s) => (
                      <span
                        key={s}
                        className="inline-block px-1.5 py-0.5 rounded text-[10px] font-medium bg-bc-accent/10 text-bc-accent"
                      >
                        {s.replace(/^mcp__/, "")}
                      </span>
                    ))}
                  </div>
                </dd>
              </>
            )}

            <dt className="text-[11px] text-bc-muted uppercase tracking-wider pt-0.5">
              Created
            </dt>
            <dd className="text-xs text-bc-text/80 tabular-nums">
              {formatTime(agent.created_at)}
            </dd>

            <dt className="text-[11px] text-bc-muted uppercase tracking-wider pt-0.5">
              Started
            </dt>
            <dd className="text-xs text-bc-text/80 tabular-nums">
              {formatTime(agent.started_at)}
            </dd>
          </dl>
        )}
      </section>
    </div>
  );
}

/* ───────────────────────── Main component ───────────────────────── */

export function AgentDetail() {
  const { name } = useParams<{ name: string }>();
  const [activeTab, setActiveTab] = useState<Tab>("logs");
  const [outputLines, setOutputLines] = useState<string[]>([]);
  const [message, setMessage] = useState("");
  const [sending, setSending] = useState(false);
  const outputRef = useRef<HTMLPreElement>(null);
  const { subscribe } = useWebSocket();

  const agentFetcher = useCallback(async () => {
    if (!name) throw new Error("No agent name");
    return api.getAgent(name);
  }, [name]);

  const {
    data: agent,
    loading,
    error,
    refresh,
  } = usePolling<Agent>(agentFetcher, 3000);

  // Poll peek output every 2 seconds for reliable updates
  useEffect(() => {
    if (!name) return;

    const fetchPeek = () => {
      api
        .getAgentPeek(name, 200)
        .then(({ output }) => {
          if (output) {
            setOutputLines(stripAnsi(output).split("\n"));
          }
        })
        .catch(() => {
          // Peek may fail for stopped agents -- ignore
        });
    };

    fetchPeek();
    const interval = setInterval(fetchPeek, 2000);
    return () => clearInterval(interval);
  }, [name]);

  // Stream live output via SSE
  useEffect(() => {
    if (!name) return;

    const es = new EventSource(
      `/api/agents/${encodeURIComponent(name)}/output`,
    );

    es.onmessage = (e: MessageEvent) => {
      try {
        const parsed = JSON.parse(e.data as string) as { output: string };
        if (parsed.output) {
          const newLines = stripAnsi(parsed.output).split("\n");
          setOutputLines((prev) => [...prev, ...newLines].slice(-500));
        }
      } catch {
        // ignore malformed events
      }
    };

    es.addEventListener("agent.output", ((e: MessageEvent) => {
      try {
        const parsed = JSON.parse(e.data as string) as { output: string };
        if (parsed.output) {
          const newLines = stripAnsi(parsed.output).split("\n");
          setOutputLines((prev) => [...prev, ...newLines].slice(-500));
        }
      } catch {
        // ignore
      }
    }) as EventListener);

    es.onerror = () => {
      // SSE reconnects automatically; no action needed
    };

    return () => {
      es.close();
    };
  }, [name]);

  // Auto-scroll output to bottom
  useEffect(() => {
    if (outputRef.current) {
      outputRef.current.scrollTop = outputRef.current.scrollHeight;
    }
  }, [outputLines]);

  // Refresh on agent state changes
  useEffect(() => {
    return subscribe("agent.state_changed", () => void refresh());
  }, [subscribe, refresh]);

  // Keyboard shortcuts: 1=Logs, 2=Terminal, 3=Info.
  // 4 and 5 also map to Info (muscle memory from the old 5-tab layout).
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      const tag = (e.target as HTMLElement | null)?.tagName;
      if (tag === "INPUT" || tag === "TEXTAREA") return;

      switch (e.key) {
        case "1":
          setActiveTab("logs");
          break;
        case "2":
          setActiveTab("terminal");
          break;
        case "3":
        case "4":
        case "5":
          setActiveTab("info");
          break;
      }
    };
    window.addEventListener("keydown", handler);
    return () => { window.removeEventListener("keydown", handler); };
  }, []);

  const handleSend = async () => {
    if (!name || !message.trim()) return;
    setSending(true);
    try {
      await api.sendToAgent(name, message);
      setMessage("");
    } finally {
      setSending(false);
    }
  };

  if (loading && !agent) {
    return <div className="p-6 text-bc-muted">Loading agent...</div>;
  }
  if (error && !agent) {
    return (
      <div className="p-6 space-y-2">
        <div className="text-bc-error">Error: {error}</div>
        <Link to="/agents" className="text-sm text-bc-accent hover:underline">
          Back to agents
        </Link>
      </div>
    );
  }
  if (!agent) return null;

  return (
    <div className="flex flex-col h-full">
      <div className="flex-1 overflow-y-auto p-6 space-y-4">
        {/* Breadcrumb + Header */}
        <div className="flex items-center gap-4">
          <Link
            to="/agents"
            className="text-bc-muted hover:text-bc-text text-sm"
          >
            &larr; Agents
          </Link>
          <h1 className="text-xl font-bold">{agent.name}</h1>
          <RoleBadge role={agent.role} />
          <StatusBadge status={agent.state} />
        </div>

        {/* Tab bar */}
        <div className="flex flex-wrap gap-1 border-b border-bc-border">
          {TABS.map((tab) => (
            <button
              key={tab.key}
              onClick={() => setActiveTab(tab.key)}
              className={`px-4 py-2 text-sm font-medium transition-colors relative ${
                activeTab === tab.key
                  ? "text-bc-accent"
                  : "text-bc-muted hover:text-bc-text"
              }`}
            >
              {tab.label}
              <span className="ml-1.5 text-[10px] text-bc-muted/60">
                {tab.shortcut}
              </span>
              {activeTab === tab.key && (
                <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-bc-accent" />
              )}
            </button>
          ))}
        </div>

        {/* Tab content */}
        {activeTab === "logs" && (
          <LogsTab outputLines={outputLines} outputRef={outputRef} />
        )}
        {activeTab === "terminal" && (
          <div className="space-y-2">
            <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide">
              Interactive Terminal
            </h2>
            {agent.state === "stopped" || agent.state === "error" ? (
              <div className="rounded border border-bc-border bg-bc-surface p-4 text-bc-muted text-sm">
                Agent is not active. Start the agent to attach to its terminal.
              </div>
            ) : (
              <div className="h-[60vh]">
                <WebTerminal agentName={agent.name} />
              </div>
            )}
          </div>
        )}
        {activeTab === "info" && <InfoTab agent={agent} />}
      </div>

      {/* Message input bar -- always visible at bottom */}
      <div className="shrink-0 border-t border-bc-border p-4">
        <div className="flex gap-2">
          <input
            type="text"
            value={message}
            onChange={(e) => setMessage(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") void handleSend();
            }}
            placeholder="Send message to agent..."
            className="flex-1 bg-bc-bg border border-bc-border rounded px-3 py-1.5 text-sm focus:outline-none focus:border-bc-accent"
          />
          <button
            onClick={() => void handleSend()}
            disabled={sending || !message.trim()}
            className="px-3 py-1.5 bg-bc-accent text-bc-bg rounded text-sm font-medium disabled:opacity-50"
          >
            Send
          </button>
        </div>
      </div>
    </div>
  );
}
