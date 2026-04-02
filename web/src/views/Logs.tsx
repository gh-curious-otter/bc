import { useCallback, useEffect, useRef, useState, memo } from "react";
import { api } from "../api/client";
import type { Agent } from "../api/client";
import { useWebSocket } from "../hooks/useWebSocket";
import { EmptyState } from "../components/EmptyState";

/* ── Types ─────────────────────────────────────────────────────────── */

interface ToolNode {
  id: string;
  toolName: string;
  args: string;
  fullInput: unknown;
  fullOutput: unknown;
  status: "running" | "completed" | "failed";
  error?: string;
  startTime: number;
  endTime?: number;
  children: ToolNode[];
}

interface AgentActivity {
  name: string;
  state: string;
  task: string;
  tool: string;
  role: string;
  tokens: number;
  nodes: ToolNode[];
  collapsed: boolean;
}

interface HookEvent {
  agent: string;
  event: string;
  tool_name?: string;
  command?: string;
  error?: string;
  task?: string;
  subagent_id?: string;
  subagent_type?: string;
  tool_input?: unknown;
  tool_response?: unknown;
  input_tokens?: number;
  output_tokens?: number;
}

type FilterType = "all" | "tools" | "state";

/* ── Constants ─────────────────────────────────────────────────────── */

const MAX_NODES = 50;
const AUTO_COLLAPSE_MS = 30_000;
const FLUSH_INTERVAL = 150;

/* ── Helpers ───────────────────────────────────────────────────────── */

let _nodeId = 0;
function nextId(): string {
  return `n-${++_nodeId}-${Date.now()}`;
}

function parseToolName(name: string): { display: string; type: "mcp" | "bash" | "internal" } {
  if (!name) return { display: "unknown", type: "internal" };
  if (name === "Bash" || name === "bash") return { display: "Bash", type: "bash" };
  if (name.startsWith("mcp__")) {
    const parts = name.split("__");
    const provider = parts[2] ?? parts[1] ?? "mcp";
    const action = parts[parts.length - 1] ?? "call";
    return { display: provider === action ? action : `${provider}:${action}`, type: "mcp" };
  }
  if (name.includes("__")) {
    const parts = name.split("__");
    const action = parts[parts.length - 1] ?? name;
    return { display: action, type: "mcp" };
  }
  if (["Read", "Write", "Edit", "Glob", "Grep", "Agent", "WebFetch", "WebSearch"].includes(name)) {
    return { display: name, type: "internal" };
  }
  return { display: name, type: "internal" };
}

function toolIcon(name: string): string {
  if (name.startsWith("mcp__playwright")) return "\u{1F3AD}";
  if (name.startsWith("mcp__bc")) return "\u26A1";
  if (name.startsWith("mcp__")) return "\u{1F50C}";
  if (name === "Bash" || name === "BashOutput") return "\u2328";
  if (name === "Read" || name === "Write" || name === "Edit") return "\u{1F4C4}";
  if (name === "Glob" || name === "Grep") return "\u{1F50D}";
  if (name === "Agent") return "\u{1F916}";
  if (name === "WebFetch" || name === "WebSearch") return "\u{1F310}";
  return "\u2699";
}

const SECRET_PATTERNS = [
  /github_pat_[A-Za-z0-9_]{20,}/g,
  /ghp_[A-Za-z0-9]{36,}/g,
  /sk-[A-Za-z0-9]{20,}/g,
  /Bearer\s+[A-Za-z0-9._\-/+=]{20,}/g,
  /(?:password|secret|token|key|auth|credential|api_key)["'=:\s]+["']?[A-Za-z0-9._\-/+=]{8,}["']?/gi,
];

function redactSecrets(text: string): string {
  let result = text;
  for (const pattern of SECRET_PATTERNS) {
    result = result.replace(pattern, "***");
  }
  return result;
}

function redactValue(value: unknown): unknown {
  if (typeof value === "string") return redactSecrets(value);
  if (Array.isArray(value)) return value.map(redactValue);
  if (value && typeof value === "object") {
    const out: Record<string, unknown> = {};
    for (const [k, v] of Object.entries(value)) {
      out[k] = redactValue(v);
    }
    return out;
  }
  return value;
}

function summarizeArgs(evt: HookEvent): string {
  if (evt.command) {
    const s = evt.command.length > 80 ? evt.command.slice(0, 77) + "..." : evt.command;
    return redactSecrets(s);
  }
  if (evt.tool_input && typeof evt.tool_input === "object") {
    const s = JSON.stringify(evt.tool_input);
    return redactSecrets(s.length > 80 ? s.slice(0, 77) + "..." : s);
  }
  return "";
}

function findLastIdx<T>(arr: T[], pred: (v: T) => boolean): number {
  for (let i = arr.length - 1; i >= 0; i--) {
    if (pred(arr[i] as T)) return i;
  }
  return -1;
}

function elapsed(start: number, end?: number): string {
  const ms = (end ?? Date.now()) - start;
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60_000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60_000).toFixed(1)}m`;
}

/* ── State Dots ────────────────────────────────────────────────────── */

function StateDot({ state }: { state: string }) {
  if (state === "working")
    return (
      <span className="relative flex h-2.5 w-2.5">
        <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-blue-400 opacity-75" />
        <span className="relative inline-flex h-2.5 w-2.5 rounded-full bg-blue-500" />
      </span>
    );
  if (state === "stuck")
    return <span className="inline-flex h-2.5 w-2.5 rounded-full bg-amber-500" />;
  if (state === "error" || state === "stopped")
    return <span className="inline-flex h-2.5 w-2.5 rounded-full bg-bc-error/60" />;
  return <span className="inline-flex h-2.5 w-2.5 rounded-full bg-bc-muted/40" />;
}

function ToolDot({ status }: { status: ToolNode["status"] }) {
  if (status === "running")
    return (
      <span className="relative flex h-2 w-2 mt-[5px] shrink-0">
        <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-blue-400 opacity-75" />
        <span className="relative inline-flex h-2 w-2 rounded-full bg-blue-500" />
      </span>
    );
  if (status === "failed")
    return <span className="inline-flex h-2 w-2 mt-[5px] shrink-0 rounded-full bg-bc-error" />;
  return <span className="inline-flex h-2 w-2 mt-[5px] shrink-0 rounded-full bg-bc-success" />;
}

/* ── Elapsed Timer ─────────────────────────────────────────────────── */

function ElapsedTimer({ start }: { start: number }) {
  const [, setTick] = useState(0);
  useEffect(() => {
    const id = setInterval(() => setTick((t) => t + 1), 200);
    return () => clearInterval(id);
  }, []);
  return <>{elapsed(start)}</>;
}

/* ── Tool Node Row ─────────────────────────────────────────────────── */

function ToolNodeRow({ node, depth = 0 }: { node: ToolNode; depth?: number }) {
  const [expanded, setExpanded] = useState(false);
  const indent = depth * 20;

  return (
    <>
      <button
        type="button"
        className="group flex items-start gap-2 py-0.5 px-3 w-full text-left hover:bg-bc-surface-hover cursor-pointer transition-colors focus-visible:ring-2 focus-visible:ring-bc-accent"
        style={{ paddingLeft: `${indent + 12}px` }}
        onClick={() => setExpanded(!expanded)}
        aria-label={`${expanded ? "Collapse" : "Expand"} tool ${node.toolName}`}
      >
        <span className="text-bc-muted text-xs select-none mt-[3px] shrink-0">
          {depth > 0 ? "\u251C\u2500" : ""}
        </span>
        <ToolDot status={node.status} />
        <span className="text-[12px] mr-0.5" aria-hidden="true">{toolIcon(node.toolName)}</span>
        <span className="font-mono text-[13px] text-bc-text font-medium">
          {parseToolName(node.toolName).display}
        </span>
        {node.args && (
          <span className="text-[12px] text-bc-muted truncate max-w-[400px] font-mono">
            {redactSecrets(node.args)}
          </span>
        )}
        <span className="ml-auto text-[11px] text-bc-muted tabular-nums shrink-0 font-mono">
          {node.status === "running" ? (
            <ElapsedTimer start={node.startTime} />
          ) : (
            elapsed(node.startTime, node.endTime)
          )}
        </span>
      </button>

      {node.error && (
        <div
          className="text-[11px] text-bc-error/80 font-mono px-3 py-0.5"
          style={{ paddingLeft: `${indent + 40}px` }}
        >
          {redactSecrets(node.error.length > 120 ? node.error.slice(0, 117) + "..." : node.error)}
        </div>
      )}

      {expanded && node.fullInput && (
        <div
          className="text-[11px] text-bc-muted font-mono px-3 py-1 bg-bc-surface mx-3 mb-1 rounded overflow-x-auto max-h-48 overflow-y-auto"
          style={{ marginLeft: `${indent + 12}px` }}
        >
          <pre className="whitespace-pre-wrap break-all">
            {JSON.stringify(redactValue(node.fullInput), null, 2)}
          </pre>
        </div>
      )}

      {expanded && node.fullOutput && (
        <div
          className="text-[11px] text-bc-success font-mono px-3 py-1 bg-bc-surface mx-3 mb-1 rounded overflow-x-auto max-h-48 overflow-y-auto"
          style={{ marginLeft: `${indent + 12}px` }}
        >
          <pre className="whitespace-pre-wrap break-all">
            {JSON.stringify(redactValue(node.fullOutput), null, 2)}
          </pre>
        </div>
      )}

      {node.children.map((child) => (
        <ToolNodeRow key={child.id} node={child} depth={depth + 1} />
      ))}
    </>
  );
}

/* ── Agent Activity Card ───────────────────────────────────────────── */

const AgentCard = memo(function AgentCard({
  activity,
  onToggle,
}: {
  activity: AgentActivity;
  onToggle: () => void;
}) {
  const runningCount = activity.nodes.filter((n) => n.status === "running").length;

  return (
    <div className="rounded-lg border border-bc-border bg-bc-surface overflow-hidden">
      <button
        type="button"
        onClick={onToggle}
        className="w-full flex items-center gap-3 px-4 py-3 hover:bg-bc-surface-hover transition-colors text-left focus-visible:ring-2 focus-visible:ring-bc-accent"
      >
        <svg
          width="12" height="12" viewBox="0 0 12 12" fill="none"
          stroke="currentColor" strokeWidth="2"
          className={`text-bc-muted transition-transform ${activity.collapsed ? "" : "rotate-90"}`}
        >
          <path d="M4 2l4 4-4 4" />
        </svg>

        <StateDot state={activity.state} />

        <span className="font-semibold text-[14px] text-bc-text">
          {activity.name}
        </span>

        <span className="text-[11px] text-bc-muted font-mono">
          {activity.role}
        </span>

        {activity.task && (
          <span className="text-[12px] text-bc-muted truncate max-w-[300px]">
            {activity.task}
          </span>
        )}

        <span className="ml-auto flex items-center gap-3">
          {runningCount > 0 && (
            <span className="text-[11px] text-blue-400 font-mono">
              {runningCount} running
            </span>
          )}
          {activity.tokens > 0 && (
            <span className="text-[11px] text-bc-muted font-mono tabular-nums">
              {activity.tokens.toLocaleString()} tok
            </span>
          )}
        </span>
      </button>

      {!activity.collapsed && activity.nodes.length > 0 && (
        <div className="border-t border-bc-border/60 py-1">
          {activity.nodes.map((node) => (
            <ToolNodeRow key={node.id} node={node} />
          ))}
        </div>
      )}

      {!activity.collapsed && activity.nodes.length === 0 && (
        <div className="border-t border-bc-border/60 py-3 px-4 text-[12px] text-bc-muted italic">
          Waiting for activity...
        </div>
      )}
    </div>
  );
});

/* ── Logs (Live Operations Center) ─────────────────────────────────── */

export function Logs() {
  const [activities, setActivities] = useState<Map<string, AgentActivity>>(new Map());
  const [agents, setAgents] = useState<Agent[]>([]);
  const [agentFilter, setAgentFilter] = useState("");
  const [typeFilter, setTypeFilter] = useState<FilterType>("all");
  const [searchFilter, setSearchFilter] = useState("");
  const [eventCount, setEventCount] = useState(0);
  const eventBuffer = useRef<HookEvent[]>([]);
  const { subscribe } = useWebSocket();

  // Seed from agents API + initial logs
  useEffect(() => {
    api.listAgents().then((agentList) => {
      setAgents(agentList);
      setActivities((prev) => {
        const next = new Map(prev);
        for (const a of agentList) {
          if (!next.has(a.name)) {
            next.set(a.name, {
              name: a.name,
              state: a.state,
              task: a.task ?? "",
              tool: a.tool,
              role: a.role,
              tokens: a.total_tokens ?? 0,
              nodes: [],
              collapsed: a.state === "stopped",
            });
          }
        }
        return next;
      });
    }).catch(() => {});

    api.getLogs(50).then((logs) => {
      setEventCount((c) => c + logs.length);
    }).catch(() => {});
  }, []);

  // Process buffered hook events (same pattern as Dashboard)
  const flushEvents = useCallback(() => {
    const events = eventBuffer.current.splice(0);
    if (events.length === 0) return;

    setEventCount((c) => c + events.length);
    setActivities((prev) => {
      const next = new Map(prev);

      for (const evt of events) {
        const agentName = evt.agent;
        if (!agentName) continue;

        let activity = next.get(agentName) ?? {
          name: agentName, state: "working", task: "", tool: "", role: "", tokens: 0, nodes: [], collapsed: false,
        };
        activity = { ...activity, nodes: [...activity.nodes] };

        if (evt.task) activity.task = evt.task;
        if (evt.input_tokens) activity.tokens += evt.input_tokens;
        if (evt.output_tokens) activity.tokens += evt.output_tokens;

        switch (evt.event) {
          case "UserPromptSubmit":
            activity.state = "working";
            activity.nodes.push({
              id: nextId(), toolName: "UserPromptSubmit", args: evt.task ?? "",
              fullInput: evt.tool_input, fullOutput: null, status: "completed",
              startTime: Date.now(), endTime: Date.now(), children: [],
            });
            break;

          case "PreToolUse":
            activity.state = "working";
            activity.nodes.push({
              id: nextId(), toolName: evt.tool_name ?? "unknown", args: summarizeArgs(evt),
              fullInput: evt.tool_input, fullOutput: null, status: "running",
              startTime: Date.now(), children: [],
            });
            break;

          case "PostToolUse": {
            const idx = findLastIdx(activity.nodes,
              (n: ToolNode) => n.toolName === evt.tool_name && n.status === "running",
            );
            if (idx >= 0) {
              const node = activity.nodes[idx];
              activity.nodes[idx] = { ...node, status: "completed" as const, endTime: Date.now(), fullOutput: evt.tool_response ?? evt.tool_input } as ToolNode;
            }
            break;
          }

          case "PostToolUseFailure": {
            const idx = findLastIdx(activity.nodes,
              (n: ToolNode) => n.toolName === evt.tool_name && n.status === "running",
            );
            if (idx >= 0) {
              const node = activity.nodes[idx];
              activity.nodes[idx] = { ...node, status: "failed" as const, endTime: Date.now(), error: evt.error ?? "Tool execution failed", fullOutput: evt.tool_response ?? evt.tool_input } as ToolNode;
            }
            break;
          }

          case "SubagentStart":
            activity.nodes.push({
              id: nextId(), toolName: `Agent: ${evt.subagent_id ?? "sub"}`,
              args: evt.subagent_type ?? "", fullInput: evt.tool_input, fullOutput: null,
              status: "running", startTime: Date.now(), children: [],
            });
            break;

          case "SubagentStop": {
            const idx = findLastIdx(activity.nodes,
              (n: ToolNode) => n.toolName.startsWith("Agent:") && n.status === "running",
            );
            if (idx >= 0) {
              const node = activity.nodes[idx];
              activity.nodes[idx] = { ...node, status: "completed" as const, endTime: Date.now() } as ToolNode;
            }
            break;
          }

          case "PermissionRequest":
          case "Elicitation":
            activity.state = "stuck";
            activity.nodes.push({
              id: nextId(), toolName: evt.event, args: evt.tool_name ?? "",
              fullInput: evt.tool_input, fullOutput: null, status: "running",
              startTime: Date.now(), children: [],
            });
            break;

          case "SessionStart": activity.state = "idle"; break;
          case "SessionEnd": case "Stop": activity.state = "idle"; break;
          case "TaskCompleted": activity.state = "idle"; break;
        }

        if (activity.nodes.length > MAX_NODES) {
          activity.nodes = activity.nodes.slice(-MAX_NODES);
        }

        const now = Date.now();
        activity.nodes = activity.nodes.map((n) =>
          n.status !== "running" && n.endTime && now - n.endTime > AUTO_COLLAPSE_MS
            ? { ...n, fullInput: undefined, fullOutput: undefined }
            : n,
        );

        next.set(agentName, activity);
      }
      return next;
    });
  }, []);

  // Flush timer
  useEffect(() => {
    const id = setInterval(flushEvents, FLUSH_INTERVAL);
    return () => clearInterval(id);
  }, [flushEvents]);

  // Subscribe to hook events
  useEffect(() => {
    const unsub = subscribe("agent.hook", (wsEvent) => {
      const d = wsEvent.data as unknown as HookEvent;
      if (d?.agent) eventBuffer.current.push(d);
    });
    return unsub;
  }, [subscribe]);

  // Subscribe to state changes
  useEffect(() => {
    const unsub = subscribe("agent.state_changed", (wsEvent) => {
      const d = wsEvent.data as Record<string, unknown>;
      const name = d.agent as string;
      const state = d.state as string;
      if (name && state) {
        setEventCount((c) => c + 1);
        setActivities((prev) => {
          const next = new Map(prev);
          const existing = next.get(name);
          if (existing) next.set(name, { ...existing, state });
          return next;
        });
      }
    });
    return unsub;
  }, [subscribe]);

  // Toggle collapse
  const toggleAgent = useCallback((name: string) => {
    setActivities((prev) => {
      const next = new Map(prev);
      const existing = next.get(name);
      if (existing) next.set(name, { ...existing, collapsed: !existing.collapsed });
      return next;
    });
  }, []);

  // Filter and sort activities
  const filtered = Array.from(activities.values()).filter((a) => {
    if (agentFilter && a.name !== agentFilter) return false;
    if (typeFilter === "tools" && a.nodes.length === 0) return false;
    if (typeFilter === "state" && a.state === "idle" && a.nodes.length === 0) return false;
    if (searchFilter) {
      const q = searchFilter.toLowerCase();
      const haystack = `${a.name} ${a.role} ${a.task} ${a.tool} ${a.nodes.map((n) => n.toolName + " " + n.args).join(" ")}`.toLowerCase();
      if (!haystack.includes(q)) return false;
    }
    return true;
  });

  const sorted = filtered.sort((a, b) => {
    const order: Record<string, number> = { working: 0, stuck: 1, idle: 2, stopped: 3, error: 4 };
    const oa = order[a.state] ?? 5;
    const ob = order[b.state] ?? 5;
    if (oa !== ob) return oa - ob;
    return a.name.localeCompare(b.name);
  });

  const hasFilters = agentFilter || typeFilter !== "all" || searchFilter;

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
        <span className="ml-auto text-xs text-bc-muted font-mono tabular-nums">{eventCount} events</span>
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
          <option value="all">All</option>
          <option value="tools">Tool Calls</option>
          <option value="state">State Changes</option>
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
            onClick={() => { setAgentFilter(""); setTypeFilter("all"); setSearchFilter(""); }}
            className="text-xs text-bc-muted hover:text-bc-text px-2 py-1.5 rounded border border-bc-border hover:border-bc-accent transition-colors"
          >
            Clear
          </button>
        )}
      </div>

      {/* Agent Activity Cards */}
      <div className="flex-1 overflow-y-auto min-h-0 space-y-3">
        {sorted.length === 0 ? (
          <EmptyState
            icon=">"
            title="No activity yet"
            description="Events will stream here in real-time as agents work."
          />
        ) : (
          sorted.map((activity) => (
            <AgentCard
              key={activity.name}
              activity={activity}
              onToggle={() => toggleAgent(activity.name)}
            />
          ))
        )}
      </div>
    </div>
  );
}
