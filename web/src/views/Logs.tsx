import { useEffect, useRef, useState } from "react";
import { api } from "../api/client";
import type { Agent } from "../api/client";
import { useWebSocket } from "../hooks/useWebSocket";
import { EmptyState } from "../components/EmptyState";

interface FeedEvent {
  id: string;
  type: "hook" | "state" | "output" | "channel";
  agent: string;
  timestamp: string;
  tool?: string;
  toolType?: "mcp" | "bash" | "internal";
  args?: string;
  duration?: string;
  state?: string;
  message?: string;
  channel?: string;
  raw?: Record<string, unknown>;
}

interface EventGroup {
  agent: string;
  events: FeedEvent[];
  totalDuration: number;
}

let eventCounter = 0;

function parseToolName(name: string): { display: string; type: "mcp" | "bash" | "internal" } {
  if (!name) return { display: "unknown", type: "internal" };
  if (name === "Bash" || name === "bash") return { display: "Bash", type: "bash" };
  if (name.startsWith("mcp__")) {
    const parts = name.split("__");
    const provider = parts[2] ?? parts[1] ?? "mcp";
    const action = parts[parts.length - 1] ?? "call";
    return { display: `${provider}:${action}`, type: "mcp" };
  }
  return { display: name, type: "internal" };
}

function toolIcon(type: "mcp" | "bash" | "internal") {
  if (type === "bash") return "\u{1F4BB}";
  if (type === "mcp") return "\u{1F50C}";
  return "\u2699\uFE0F";
}

function stateColor(state: string) {
  if (state === "working") return "bg-bc-success";
  if (state === "idle") return "bg-yellow-400";
  return "bg-bc-muted";
}

function formatTime(ts: string): string {
  try {
    return new Date(ts).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" });
  } catch {
    return "--:--:--";
  }
}

function formatDuration(ms: number): string {
  if (ms >= 1000) return `${(ms / 1000).toFixed(1)}s`;
  return `${Math.round(ms)}ms`;
}

type FilterType = "all" | "hook" | "state" | "output" | "channel";

export function Logs() {
  const [agents, setAgents] = useState<Agent[]>([]);
  const [agentFilter, setAgentFilter] = useState("");
  const [typeFilter, setTypeFilter] = useState<FilterType>("all");
  const [searchFilter, setSearchFilter] = useState("");
  const [events, setEvents] = useState<FeedEvent[]>([]);
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(new Set());
  const containerRef = useRef<HTMLDivElement>(null);
  const userScrolled = useRef(false);
  const { subscribe } = useWebSocket();

  useEffect(() => {
    api.listAgents().then(setAgents).catch(() => {});
  }, []);

  useEffect(() => {
    const handleEvent = (evt: FeedEvent) => {
      setEvents((prev) => [evt, ...prev].slice(0, 500));
    };

    const unsubs = [
      subscribe("agent.hook", (event) => {
        const d = event.data as Record<string, unknown>;
        const toolName = String(d.tool_name ?? d.name ?? "");
        const parsed = parseToolName(toolName);
        handleEvent({
          id: `evt-${++eventCounter}`,
          type: "hook",
          agent: String(d.agent ?? ""),
          timestamp: event.timestamp || new Date().toISOString(),
          tool: parsed.display,
          toolType: parsed.type,
          args: JSON.stringify(d.args ?? d.input ?? {}).slice(0, 60),
          duration: d.duration_ms ? formatDuration(Number(d.duration_ms)) : "",
          raw: d,
        });
      }),
      subscribe("agent.state_changed", (event) => {
        const d = event.data as Record<string, unknown>;
        handleEvent({
          id: `evt-${++eventCounter}`,
          type: "state",
          agent: String(d.agent ?? ""),
          timestamp: event.timestamp || new Date().toISOString(),
          state: String(d.state ?? d.status ?? "unknown"),
        });
      }),
      subscribe("agent.output", (event) => {
        const d = event.data as Record<string, unknown>;
        const msg = String(d.output ?? d.message ?? d.text ?? "");
        handleEvent({
          id: `evt-${++eventCounter}`,
          type: "output",
          agent: String(d.agent ?? ""),
          timestamp: event.timestamp || new Date().toISOString(),
          message: msg.split("\n")[0]?.slice(0, 120) ?? "",
        });
      }),
      subscribe("channel.message", (event) => {
        const d = event.data as Record<string, unknown>;
        handleEvent({
          id: `evt-${++eventCounter}`,
          type: "channel",
          agent: String(d.sender ?? d.agent ?? ""),
          timestamp: event.timestamp || new Date().toISOString(),
          channel: String(d.channel ?? ""),
          message: String(d.message ?? d.text ?? "").slice(0, 120),
        });
      }),
    ];
    return () => unsubs.forEach((fn) => fn());
  }, [subscribe]);

  // Auto-scroll: scroll to top when new events arrive unless user scrolled down
  useEffect(() => {
    if (!userScrolled.current && containerRef.current) {
      containerRef.current.scrollTop = 0;
    }
  }, [events]);

  const handleScroll = () => {
    if (!containerRef.current) return;
    userScrolled.current = containerRef.current.scrollTop > 50;
  };

  // Filtering
  const filtered = events.filter((e) => {
    if (agentFilter && e.agent !== agentFilter) return false;
    if (typeFilter !== "all" && e.type !== typeFilter) return false;
    if (searchFilter) {
      const q = searchFilter.toLowerCase();
      const haystack = `${e.agent} ${e.tool ?? ""} ${e.args ?? ""} ${e.message ?? ""} ${e.state ?? ""} ${e.channel ?? ""}`.toLowerCase();
      if (!haystack.includes(q)) return false;
    }
    return true;
  });

  // Group consecutive tool calls from same agent
  const grouped: (FeedEvent | EventGroup)[] = [];
  let i = 0;
  while (i < filtered.length) {
    const cur = filtered[i]!;
    if (cur.type === "hook") {
      const batch: FeedEvent[] = [cur];
      while (i + 1 < filtered.length && filtered[i + 1]!.type === "hook" && filtered[i + 1]!.agent === cur.agent) {
        batch.push(filtered[++i]!);
      }
      if (batch.length >= 3) {
        const totalMs = batch.reduce((sum, e) => {
          const raw = e.raw;
          return sum + (raw?.duration_ms ? Number(raw.duration_ms) : 0);
        }, 0);
        grouped.push({ agent: cur.agent, events: batch, totalDuration: totalMs });
      } else {
        grouped.push(...batch);
      }
    } else {
      grouped.push(cur);
    }
    i++;
  }

  const toggleGroup = (key: string) => {
    setExpandedGroups((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  const hasFilters = agentFilter || typeFilter !== "all" || searchFilter;

  const clearFilters = () => {
    setAgentFilter("");
    setTypeFilter("all");
    setSearchFilter("");
  };

  const renderEvent = (e: FeedEvent) => {
    const ts = formatTime(e.timestamp);
    switch (e.type) {
      case "hook":
        return (
          <div className="flex items-center gap-2 min-w-0">
            <span className="text-bc-muted text-xs font-mono shrink-0">{ts}</span>
            <span className="font-medium text-bc-text truncate shrink-0 max-w-[140px]">{e.agent}</span>
            <span className="text-xs shrink-0">{toolIcon(e.toolType!)}</span>
            <span className="text-bc-accent font-mono text-xs truncate">{e.tool}</span>
            {e.args && e.args !== "{}" && (
              <span className="text-bc-muted text-xs font-mono truncate hidden sm:inline">{e.args}</span>
            )}
            {e.duration && (
              <span className="ml-auto text-bc-muted text-xs font-mono shrink-0">{e.duration}</span>
            )}
          </div>
        );
      case "state":
        return (
          <div className="flex items-center gap-2 min-w-0">
            <span className="text-bc-muted text-xs font-mono shrink-0">{ts}</span>
            <span className="font-medium text-bc-text truncate shrink-0 max-w-[140px]">{e.agent}</span>
            <span className="text-bc-muted text-xs">&rarr;</span>
            <span className={`w-2 h-2 rounded-full shrink-0 ${stateColor(e.state ?? "")}`} />
            <span className="text-xs text-bc-text">{e.state}</span>
          </div>
        );
      case "output":
        return (
          <div className="flex items-center gap-2 min-w-0">
            <span className="text-bc-muted text-xs font-mono shrink-0">{ts}</span>
            <span className="font-medium text-bc-text truncate shrink-0 max-w-[140px]">{e.agent}</span>
            <span className="text-xs shrink-0">{"\u{1F4AC}"}</span>
            <span className="text-bc-muted text-xs truncate">{e.message}</span>
          </div>
        );
      case "channel":
        return (
          <div className="flex items-center gap-2 min-w-0">
            <span className="text-bc-muted text-xs font-mono shrink-0">{ts}</span>
            <span className="text-bc-accent text-xs font-mono shrink-0">#{e.channel}</span>
            <span className="font-medium text-bc-text text-xs shrink-0">{e.agent}:</span>
            <span className="text-bc-muted text-xs truncate">{e.message}</span>
          </div>
        );
    }
  };

  return (
    <div className="p-6 flex flex-col h-full">
      {/* Header */}
      <div className="flex items-center gap-3 mb-4">
        <h1 className="text-xl font-bold text-bc-text flex items-center gap-2">
          Live
          <span className="relative flex h-2.5 w-2.5">
            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-red-400 opacity-75" />
            <span className="relative inline-flex rounded-full h-2.5 w-2.5 bg-red-500" />
          </span>
        </h1>
        <span className="text-sm text-bc-muted">Real-time agent activity</span>
        <span className="ml-auto text-xs text-bc-muted">{filtered.length} events</span>
      </div>

      {/* Filter Bar */}
      <div className="flex flex-wrap items-center gap-2 mb-4 sticky top-0 z-10 bg-bc-bg py-2">
        <select
          value={agentFilter}
          onChange={(e) => setAgentFilter(e.target.value)}
          className="text-sm rounded border border-bc-border bg-bc-surface px-2 py-1.5 text-bc-text focus:outline-none focus:ring-1 focus:ring-bc-accent"
        >
          <option value="">All agents</option>
          {agents.map((a) => (
            <option key={a.name} value={a.name}>{a.name}</option>
          ))}
        </select>
        <select
          value={typeFilter}
          onChange={(e) => setTypeFilter(e.target.value as FilterType)}
          className="text-sm rounded border border-bc-border bg-bc-surface px-2 py-1.5 text-bc-text focus:outline-none focus:ring-1 focus:ring-bc-accent"
        >
          <option value="all">All types</option>
          <option value="hook">Tool Calls</option>
          <option value="state">State Changes</option>
          <option value="output">Messages</option>
          <option value="channel">Channels</option>
        </select>
        <input
          type="text"
          value={searchFilter}
          onChange={(e) => setSearchFilter(e.target.value)}
          placeholder="Search events..."
          className="text-sm rounded border border-bc-border bg-bc-surface px-2 py-1.5 text-bc-text placeholder:text-bc-muted focus:outline-none focus:ring-1 focus:ring-bc-accent w-48"
        />
        {hasFilters && (
          <button
            type="button"
            onClick={clearFilters}
            className="text-xs text-bc-muted hover:text-bc-text px-2 py-1.5 rounded border border-bc-border hover:border-bc-accent transition-colors"
          >
            Clear
          </button>
        )}
      </div>

      {/* Activity Feed */}
      <div
        ref={containerRef}
        onScroll={handleScroll}
        className="flex-1 overflow-y-auto min-h-0 space-y-px"
      >
        {filtered.length === 0 ? (
          <EmptyState
            icon="[]"
            title="No activity yet"
            description="Events will stream here in real-time as agents work."
          />
        ) : (
          grouped.map((item) => {
            if ("events" in item) {
              // Grouped tool calls
              const group = item as EventGroup;
              const key = group.events[0]!.id;
              const isOpen = expandedGroups.has(key);
              return (
                <div key={key} className="rounded border border-bc-border/50 bg-bc-surface/30">
                  <button
                    type="button"
                    onClick={() => toggleGroup(key)}
                    className="w-full flex items-center gap-2 px-3 py-2 text-left hover:bg-bc-surface/50 transition-colors"
                  >
                    <span className="text-bc-muted text-xs">{isOpen ? "\u25BC" : "\u25B6"}</span>
                    <span className="font-medium text-bc-text text-sm truncate max-w-[140px]">{group.agent}</span>
                    <span className="text-bc-muted text-xs">
                      {group.events.length} tool calls
                    </span>
                    {group.totalDuration > 0 && (
                      <span className="text-bc-muted text-xs font-mono ml-auto">
                        {formatDuration(group.totalDuration)} total
                      </span>
                    )}
                  </button>
                  {isOpen && (
                    <div className="border-t border-bc-border/30 pl-4">
                      {group.events.map((e, idx) => (
                        <div
                          key={e.id}
                          className="flex items-center gap-2 px-3 py-1.5 text-xs min-w-0"
                        >
                          <span className="text-bc-border shrink-0">
                            {idx === group.events.length - 1 ? "\u2514" : "\u251C"}
                          </span>
                          <span className="shrink-0">{toolIcon(e.toolType!)}</span>
                          <span className="text-bc-accent font-mono truncate">{e.tool}</span>
                          {e.args && e.args !== "{}" && (
                            <span className="text-bc-muted font-mono truncate hidden sm:inline">{e.args}</span>
                          )}
                          {e.duration && (
                            <span className="ml-auto text-bc-muted font-mono shrink-0">{e.duration}</span>
                          )}
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              );
            }
            // Single event
            const e = item as FeedEvent;
            return (
              <div key={e.id} className="px-3 py-2 hover:bg-bc-surface/30 rounded transition-colors">
                {renderEvent(e)}
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}
