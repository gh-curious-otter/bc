import { useCallback, useEffect, useRef, useState, memo } from "react";
import { api } from "../api/client";
import type { Agent } from "../api/client";
import { usePolling } from "../hooks/usePolling";
import { useWebSocket } from "../hooks/useWebSocket";

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

/* ── Constants ─────────────────────────────────────────────────────── */

const MAX_NODES = 50;
const AUTO_COLLAPSE_MS = 30_000;
const FLUSH_INTERVAL = 150;

/* ── Helpers ───────────────────────────────────────────────────────── */

let _nodeId = 0;
function nextId(): string {
  return `n-${++_nodeId}-${Date.now()}`;
}

function summarizeArgs(evt: HookEvent): string {
  if (evt.command) return evt.command.length > 80 ? evt.command.slice(0, 77) + "..." : evt.command;
  if (evt.tool_input && typeof evt.tool_input === "object") {
    const s = JSON.stringify(evt.tool_input);
    return s.length > 80 ? s.slice(0, 77) + "..." : s;
  }
  return "";
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
    return <span className="inline-flex h-2.5 w-2.5 rounded-full bg-red-500/60" />;
  return <span className="inline-flex h-2.5 w-2.5 rounded-full bg-zinc-500/40" />;
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
    return <span className="inline-flex h-2 w-2 mt-[5px] shrink-0 rounded-full bg-red-500" />;
  return <span className="inline-flex h-2 w-2 mt-[5px] shrink-0 rounded-full bg-emerald-500" />;
}

/* ── Tool Node Component ───────────────────────────────────────────── */

function ToolNodeRow({ node, depth = 0 }: { node: ToolNode; depth?: number }) {
  const [expanded, setExpanded] = useState(false);
  const indent = depth * 20;

  return (
    <>
      <div
        className="group flex items-start gap-2 py-0.5 px-3 hover:bg-white/[0.02] cursor-pointer transition-colors"
        style={{ paddingLeft: `${indent + 12}px` }}
        onClick={() => setExpanded(!expanded)}
      >
        {/* Tree connector */}
        <span className="text-zinc-600 text-xs select-none mt-[3px] shrink-0">
          {depth > 0 ? "├─" : ""}
        </span>

        <ToolDot status={node.status} />

        <span className="font-mono text-[13px] text-zinc-300 font-medium">
          {node.toolName}
        </span>

        {node.args && (
          <span className="text-[12px] text-zinc-500 truncate max-w-[400px] font-mono">
            {node.args}
          </span>
        )}

        <span className="ml-auto text-[11px] text-zinc-600 tabular-nums shrink-0 font-mono">
          {node.status === "running" ? (
            <ElapsedTimer start={node.startTime} />
          ) : (
            elapsed(node.startTime, node.endTime)
          )}
        </span>
      </div>

      {node.error && (
        <div
          className="text-[11px] text-red-400/80 font-mono px-3 py-0.5"
          style={{ paddingLeft: `${indent + 40}px` }}
        >
          {node.error.length > 120 ? node.error.slice(0, 117) + "..." : node.error}
        </div>
      )}

      {expanded && node.fullInput && (
        <div
          className="text-[11px] text-zinc-500 font-mono px-3 py-1 bg-zinc-900/50 mx-3 mb-1 rounded overflow-x-auto max-h-48 overflow-y-auto"
          style={{ marginLeft: `${indent + 12}px` }}
        >
          <pre className="whitespace-pre-wrap break-all">
            {JSON.stringify(node.fullInput, null, 2)}
          </pre>
        </div>
      )}

      {expanded && node.fullOutput && (
        <div
          className="text-[11px] text-emerald-600 font-mono px-3 py-1 bg-zinc-900/50 mx-3 mb-1 rounded overflow-x-auto max-h-48 overflow-y-auto"
          style={{ marginLeft: `${indent + 12}px` }}
        >
          <pre className="whitespace-pre-wrap break-all">
            {JSON.stringify(node.fullOutput, null, 2)}
          </pre>
        </div>
      )}

      {node.children.map((child) => (
        <ToolNodeRow key={child.id} node={child} depth={depth + 1} />
      ))}
    </>
  );
}

/* ── Elapsed Timer (re-renders itself) ─────────────────────────────── */

function ElapsedTimer({ start }: { start: number }) {
  const [, setTick] = useState(0);
  useEffect(() => {
    const id = setInterval(() => setTick((t) => t + 1), 200);
    return () => clearInterval(id);
  }, []);
  return <>{elapsed(start)}</>;
}

/* ── Agent Card ────────────────────────────────────────────────────── */

const AgentCard = memo(function AgentCard({
  activity,
  onToggle,
}: {
  activity: AgentActivity;
  onToggle: () => void;
}) {
  const runningCount = activity.nodes.filter((n) => n.status === "running").length;

  return (
    <div className="rounded-lg border border-zinc-800/80 bg-zinc-900/60 backdrop-blur-sm overflow-hidden">
      {/* Header */}
      <button
        type="button"
        onClick={onToggle}
        className="w-full flex items-center gap-3 px-4 py-3 hover:bg-white/[0.02] transition-colors text-left"
      >
        <svg
          width="12"
          height="12"
          viewBox="0 0 12 12"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          className={`text-zinc-500 transition-transform ${activity.collapsed ? "" : "rotate-90"}`}
        >
          <path d="M4 2l4 4-4 4" />
        </svg>

        <StateDot state={activity.state} />

        <span className="font-semibold text-[14px] text-zinc-200">
          {activity.name}
        </span>

        <span className="text-[11px] text-zinc-600 font-mono">
          {activity.tool}
        </span>

        {activity.task && (
          <span className="text-[12px] text-zinc-500 truncate max-w-[300px]">
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
            <span className="text-[11px] text-zinc-600 font-mono tabular-nums">
              {activity.tokens.toLocaleString()} tok
            </span>
          )}
          <span className="text-[11px] text-zinc-600 font-mono">
            {activity.nodes.length} events
          </span>
        </span>
      </button>

      {/* Tool tree */}
      {!activity.collapsed && activity.nodes.length > 0 && (
        <div className="border-t border-zinc-800/60 py-1">
          {activity.nodes.map((node) => (
            <ToolNodeRow key={node.id} node={node} />
          ))}
        </div>
      )}

      {!activity.collapsed && activity.nodes.length === 0 && (
        <div className="border-t border-zinc-800/60 py-3 px-4 text-[12px] text-zinc-600 italic">
          Waiting for activity...
        </div>
      )}
    </div>
  );
});

/* ── Main Component ────────────────────────────────────────────────── */

export function Activity() {
  const [activities, setActivities] = useState<Map<string, AgentActivity>>(new Map());
  const eventBuffer = useRef<HookEvent[]>([]);
  const { subscribe } = useWebSocket();

  // Fetch agent list for initial state
  const fetcher = useCallback(() => api.listAgents(), []);
  const { data: agents } = usePolling(fetcher, 5000);

  // Seed activities from agent list
  useEffect(() => {
    if (!agents) return;
    setActivities((prev) => {
      const next = new Map(prev);
      for (const agent of agents) {
        if (!next.has(agent.name)) {
          next.set(agent.name, {
            name: agent.name,
            state: agent.state,
            task: agent.task ?? "",
            tool: agent.tool,
            tokens: agent.total_tokens ?? 0,
            nodes: [],
            collapsed: agent.state === "stopped",
          });
        } else {
          const existing = next.get(agent.name)!;
          next.set(agent.name, {
            ...existing,
            state: agent.state,
            task: agent.task ?? existing.task,
            tokens: agent.total_tokens ?? existing.tokens,
          });
        }
      }
      return next;
    });
  }, [agents]);

  // Process buffered events into activity state
  const flushEvents = useCallback(() => {
    const events = eventBuffer.current.splice(0);
    if (events.length === 0) return;

    setActivities((prev) => {
      const next = new Map(prev);

      for (const evt of events) {
        const agentName = evt.agent;
        if (!agentName) continue;

        let activity = next.get(agentName) ?? {
          name: agentName,
          state: "working",
          task: "",
          tool: "",
          tokens: 0,
          nodes: [],
          collapsed: false,
        };

        // Clone for immutability
        activity = { ...activity, nodes: [...activity.nodes] };

        // Update task
        if (evt.task) activity.task = evt.task;

        // Track tokens
        if (evt.input_tokens) activity.tokens += evt.input_tokens;
        if (evt.output_tokens) activity.tokens += evt.output_tokens;

        switch (evt.event) {
          case "UserPromptSubmit":
            activity.state = "working";
            activity.nodes.push({
              id: nextId(),
              toolName: "UserPromptSubmit",
              args: evt.task ?? "",
              fullInput: evt.tool_input,
              fullOutput: null,
              status: "completed",
              startTime: Date.now(),
              endTime: Date.now(),
              children: [],
            });
            break;

          case "PreToolUse":
            activity.state = "working";
            activity.nodes.push({
              id: nextId(),
              toolName: evt.tool_name ?? "unknown",
              args: summarizeArgs(evt),
              fullInput: evt.tool_input,
              fullOutput: null,
              status: "running",
              startTime: Date.now(),
              children: [],
            });
            break;

          case "PostToolUse": {
            // Find matching running tool (last one with same name)
            const idx = activity.nodes.findLastIndex(
              (n) => n.toolName === evt.tool_name && n.status === "running",
            );
            if (idx >= 0) {
              activity.nodes[idx] = {
                ...activity.nodes[idx],
                status: "completed",
                endTime: Date.now(),
                fullOutput: evt.tool_response ?? evt.tool_input,
              };
            }
            break;
          }

          case "PostToolUseFailure": {
            const idx = activity.nodes.findLastIndex(
              (n) => n.toolName === evt.tool_name && n.status === "running",
            );
            if (idx >= 0) {
              activity.nodes[idx] = {
                ...activity.nodes[idx],
                status: "failed",
                endTime: Date.now(),
                error: evt.error ?? "Tool execution failed",
                fullOutput: evt.tool_response ?? evt.tool_input,
              };
            }
            break;
          }

          case "SubagentStart":
            activity.nodes.push({
              id: nextId(),
              toolName: `Agent: ${evt.subagent_id ?? "sub"}`,
              args: evt.subagent_type ?? "",
              fullInput: evt.tool_input,
              fullOutput: null,
              status: "running",
              startTime: Date.now(),
              children: [],
            });
            break;

          case "SubagentStop": {
            const idx = activity.nodes.findLastIndex(
              (n) =>
                n.toolName.startsWith("Agent:") && n.status === "running",
            );
            if (idx >= 0) {
              activity.nodes[idx] = {
                ...activity.nodes[idx],
                status: "completed",
                endTime: Date.now(),
              };
            }
            break;
          }

          case "PermissionRequest":
          case "Elicitation":
            activity.state = "stuck";
            activity.nodes.push({
              id: nextId(),
              toolName: evt.event,
              args: evt.tool_name ?? "",
              fullInput: evt.tool_input,
              fullOutput: null,
              status: "running",
              startTime: Date.now(),
              children: [],
            });
            break;

          case "SessionStart":
            activity.state = "idle";
            break;

          case "SessionEnd":
          case "Stop":
            activity.state = "idle";
            break;

          case "TaskCompleted":
            activity.state = "idle";
            break;
        }

        // Trim to MAX_NODES
        if (activity.nodes.length > MAX_NODES) {
          activity.nodes = activity.nodes.slice(-MAX_NODES);
        }

        // Auto-collapse old completed nodes
        const now = Date.now();
        activity.nodes = activity.nodes.map((n) =>
          n.status !== "running" &&
          n.endTime &&
          now - n.endTime > AUTO_COLLAPSE_MS
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
      const data = wsEvent.data as unknown as HookEvent;
      if (data?.agent) {
        eventBuffer.current.push(data);
      }
    });
    return unsub;
  }, [subscribe]);

  // Subscribe to state changes
  useEffect(() => {
    const unsub = subscribe("agent.state_changed", (wsEvent) => {
      const data = wsEvent.data as Record<string, unknown>;
      const name = data.agent as string;
      const state = data.state as string;
      if (name && state) {
        setActivities((prev) => {
          const next = new Map(prev);
          const existing = next.get(name);
          if (existing) {
            next.set(name, { ...existing, state });
          }
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
      if (existing) {
        next.set(name, { ...existing, collapsed: !existing.collapsed });
      }
      return next;
    });
  }, []);

  // Sort: working first, then by name
  const sorted = Array.from(activities.values()).sort((a, b) => {
    const stateOrder: Record<string, number> = {
      working: 0,
      stuck: 1,
      idle: 2,
      stopped: 3,
      error: 4,
    };
    const oa = stateOrder[a.state] ?? 5;
    const ob = stateOrder[b.state] ?? 5;
    if (oa !== ob) return oa - ob;
    return a.name.localeCompare(b.name);
  });

  const onlineCount = sorted.filter(
    (a) => a.state !== "stopped" && a.state !== "error",
  ).length;

  return (
    <div className="min-h-screen bg-[#0a0a0b]">
      {/* Header */}
      <div className="sticky top-0 z-10 backdrop-blur-md bg-[#0a0a0b]/80 border-b border-zinc-800/50">
        <div className="flex items-center justify-between px-6 py-4">
          <div className="flex items-center gap-3">
            <h1 className="text-[15px] font-semibold text-zinc-200 tracking-tight">
              Activity
            </h1>
            <span className="text-[11px] text-zinc-600 font-mono">
              live
            </span>
          </div>

          <div className="flex items-center gap-2">
            {sorted
              .filter((a) => a.state !== "stopped")
              .map((a) => (
                <span key={a.name} className="flex items-center gap-1.5" title={a.name}>
                  <StateDot state={a.state} />
                  <span className="text-[11px] text-zinc-500 font-mono hidden sm:inline">
                    {a.name}
                  </span>
                </span>
              ))}
            <span className="ml-2 text-[11px] text-zinc-600 font-mono tabular-nums">
              {onlineCount} online
            </span>
          </div>
        </div>
      </div>

      {/* Agent cards */}
      <div className="p-4 space-y-3 max-w-5xl mx-auto">
        {sorted.length === 0 && (
          <div className="text-center py-20">
            <p className="text-zinc-600 text-sm">No agents detected</p>
            <p className="text-zinc-700 text-xs mt-1">
              Start an agent to see live activity
            </p>
          </div>
        )}
        {sorted.map((activity) => (
          <AgentCard
            key={activity.name}
            activity={activity}
            onToggle={() => toggleAgent(activity.name)}
          />
        ))}
      </div>
    </div>
  );
}
