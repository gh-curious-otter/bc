import { useCallback, useEffect, useMemo, useRef, useState, memo } from "react";
import { motion, AnimatePresence } from "framer-motion";
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

interface AggregatedNode {
  type: "aggregate";
  id: string;
  toolName: string;
  count: number;
  children: ToolNode[];
  totalDuration: number;
  totalTokens: number;
  successCount: number;
  failCount: number;
  startTime: number;
  endTime: number;
}

type DisplayNode = ToolNode | AggregatedNode;

function isAggregatedNode(node: DisplayNode): node is AggregatedNode {
  return "type" in node && node.type === "aggregate";
}

interface AgentActivity {
  name: string;
  state: string;
  task: string;
  tool: string;
  role: string;
  tokens: number;
  inputTokens: number;
  outputTokens: number;
  costUsd: number;
  lastEventTime: number;
  nodes: ToolNode[];
  collapsed: boolean;
  /** Index of the currently-active subagent node in nodes[], for nesting */
  activeSubagentIdx?: number;
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
  prompt?: string;
}

interface TaskItem {
  id: string;
  subject: string;
  status: "pending" | "in_progress" | "completed" | "deleted";
  owner?: string;
  description?: string;
  blockedBy?: string[];
}

type FilterType = "all" | "tools" | "state";

type DrillDownTab = "live" | "raw";

interface RawEvent {
  timestamp: number;
  eventType: string;
  raw: unknown;
}

/* ── Constants ─────────────────────────────────────────────────────── */

const MAX_NODES = 50;
const AUTO_COLLAPSE_MS = 30_000;
const FLUSH_INTERVAL = 150;
const AGGREGATION_WINDOW_MS = 5_000;
const FAILED_NEVER_AGGREGATE_MS = 120_000;
const MAX_INDIVIDUAL_NODES = 50;

const TASK_STATUS_MAP: Record<string, TaskItem["status"]> = {
  pending: "pending",
  in_progress: "in_progress",
  "in-progress": "in_progress",
  inProgress: "in_progress",
  completed: "completed",
  done: "completed",
  deleted: "deleted",
  cancelled: "deleted",
  canceled: "deleted",
};

/* ── Helpers ───────────────────────────────────────────────────────── */

let _nodeId = 0;
function nextId(): string {
  return `n-${++_nodeId}-${Date.now()}`;
}

interface ParsedTool {
  display: string;
  type: "mcp" | "bash" | "internal";
  mcpServer?: string;
  mcpFunction?: string;
}

function parseToolName(name: string): ParsedTool {
  if (!name) return { display: "unknown", type: "internal" };
  if (name === "Bash" || name === "bash") return { display: "Bash", type: "bash" };
  if (name.startsWith("mcp__")) {
    const parts = name.split("__");
    let server = parts[1] ?? "mcp";
    const func = parts[parts.length - 1] ?? "call";
    if (server.startsWith("plugin_")) {
      const pluginParts = server.replace("plugin_", "").split("_");
      server = pluginParts[0] ?? server;
    }
    return { display: `${server}:${func}`, type: "mcp", mcpServer: server, mcpFunction: func };
  }
  if (name.includes("__")) {
    const parts = name.split("__");
    const action = parts[parts.length - 1] ?? name;
    return { display: action, type: "mcp", mcpServer: parts[0], mcpFunction: action };
  }
  return { display: name, type: "internal" };
}

function toolIcon(name: string): string {
  if (name === "Bash" || name === "BashOutput") return "\u2328\uFE0F";
  if (name === "Read") return "\uD83D\uDCD6";
  if (name === "Write" || name === "Edit") return "\u270F\uFE0F";
  if (name === "Glob" || name === "Grep") return "\uD83D\uDD0D";
  if (name === "Agent") return "\uD83E\uDD16";
  if (name === "WebFetch" || name === "WebSearch") return "\uD83C\uDF10";
  if (name.startsWith("Task")) return "\u2705";
  if (name === "NotebookEdit") return "\uD83D\uDCD3";
  if (name === "LSP" || name === "ToolSearch") return "\u2699\uFE0F";
  if (name === "AskUserQuestion") return "\u2753";
  if (name === "Skill") return "\uD83C\uDFAF";
  return "\u2699\uFE0F";
}

function mcpServerIcon(server: string): string {
  if (server === "playwright" || server === "playwright2") return "\uD83C\uDFAD";
  if (server === "github") return "\uD83D\uDC19";
  if (server === "bc") return "\u26A1";
  return "\uD83D\uDD0C";
}

function mcpBadgeColors(server: string): string {
  if (server === "playwright" || server === "playwright2") return "bg-purple-900/50 text-purple-300";
  if (server === "github") return "bg-gray-700 text-gray-300";
  if (server === "bc") return "bg-blue-900/50 text-blue-300";
  return "bg-zinc-700 text-zinc-300";
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

function extractToolMetadata(toolName: string, input: unknown): string {
  if (!input || typeof input !== "object") return "";
  const obj = input as Record<string, unknown>;
  const trunc = (s: string, max = 80): string => s.length > max ? s.slice(0, max - 3) + "..." : s;

  if (toolName === "Bash" || toolName === "bash") {
    if (typeof obj.command === "string") return redactSecrets(trunc(obj.command));
  }
  if ((toolName === "Read" || toolName === "Write") && typeof obj.file_path === "string") {
    return trunc(obj.file_path);
  }
  if (toolName === "Edit") {
    let s = typeof obj.file_path === "string" ? obj.file_path : "";
    if (typeof obj.old_string === "string") {
      s += " " + trunc(obj.old_string, 40);
    }
    return trunc(s);
  }
  if ((toolName === "Grep" || toolName === "Glob") && typeof obj.pattern === "string") {
    return trunc(obj.pattern);
  }
  if (toolName === "Agent") {
    const parts: string[] = [];
    if (typeof obj.subagent_type === "string") parts.push(obj.subagent_type);
    if (typeof obj.description === "string") parts.push(trunc(obj.description, 60));
    return parts.join(" ");
  }
  if (toolName === "WebFetch") {
    if (typeof obj.url === "string") {
      try { return new URL(obj.url).hostname; } catch { return trunc(obj.url); }
    }
  }
  if (toolName === "WebSearch") {
    if (typeof obj.query === "string") return trunc(obj.query);
  }
  if (toolName.startsWith("mcp__")) {
    const vals = Object.entries(obj).slice(0, 3).map(([, v]) => {
      if (typeof v === "string") return trunc(v, 30);
      if (typeof v === "number" || typeof v === "boolean") return String(v);
      return "";
    }).filter(Boolean);
    return redactSecrets(vals.join(" "));
  }
  const s = JSON.stringify(obj);
  return redactSecrets(trunc(s));
}

function summarizeArgs(evt: HookEvent): string {
  if (evt.tool_name && evt.tool_input) {
    return extractToolMetadata(evt.tool_name, evt.tool_input);
  }
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

function durationColorClass(start: number, end?: number): string {
  const ms = (end ?? Date.now()) - start;
  if (ms < 500) return "text-emerald-400";
  if (ms < 2000) return "text-yellow-400";
  if (ms < 10000) return "text-orange-400";
  return "text-red-400";
}

function relativeTime(ts: number): string {
  const diff = Date.now() - ts;
  if (diff < 1000) return "just now";
  if (diff < 60_000) return `${Math.floor(diff / 1000)}s ago`;
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`;
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h ago`;
  return `${Math.floor(diff / 86_400_000)}d ago`;
}

const INPUT_COST_PER_TOKEN = 3 / 1_000_000;
const OUTPUT_COST_PER_TOKEN = 15 / 1_000_000;

function estimateCost(activity: AgentActivity): number {
  if (activity.costUsd > 0) return activity.costUsd;
  if (activity.inputTokens > 0 || activity.outputTokens > 0) {
    return activity.inputTokens * INPUT_COST_PER_TOKEN + activity.outputTokens * OUTPUT_COST_PER_TOKEN;
  }
  return 0;
}

function idleDuration(lastEventTime: number): string {
  const diff = Date.now() - lastEventTime;
  if (diff < 60_000) return `Idle ${Math.floor(diff / 1000)}s`;
  if (diff < 3_600_000) return `Idle ${Math.floor(diff / 60_000)}m`;
  if (diff < 86_400_000) return `Idle ${Math.floor(diff / 3_600_000)}h`;
  return `Idle ${Math.floor(diff / 86_400_000)}d`;
}

/* ── Node search helper ────────────────────────────────────────────── */

function nodeMatchesSearch(node: ToolNode, query: string): boolean {
  const hay = `${node.toolName} ${node.args}`.toLowerCase();
  return hay.includes(query);
}

/* ── Smart sorting ────────────────────────────────────────────────── */

function sortNodes(nodes: ToolNode[]): ToolNode[] {
  return [...nodes].sort((a, b) => {
    // Running first (newest first among running)
    if (a.status === "running" && b.status !== "running") return -1;
    if (b.status === "running" && a.status !== "running") return 1;
    if (a.status === "running" && b.status === "running") return b.startTime - a.startTime;
    // Failed second (newest first among failed)
    if (a.status === "failed" && b.status !== "failed") return -1;
    if (b.status === "failed" && a.status !== "failed") return 1;
    if (a.status === "failed" && b.status === "failed") return b.startTime - a.startTime;
    // Completed: sort by duration (longest first)
    const aDur = (a.endTime ?? a.startTime) - a.startTime;
    const bDur = (b.endTime ?? b.startTime) - b.startTime;
    return bDur - aDur;
  });
}

/* ── Aggregation ──────────────────────────────────────────────────── */

const AGGREGATION_MIN_COUNT = 3;

const NEVER_AGGREGATE_EVENTS = new Set([
  "SubagentStart", "SubagentStop", "Agent",
  "PermissionRequest", "Elicitation",
  "UserPromptSubmit", "SessionStart", "SessionEnd",
  "Stop", "TaskCompleted",
]);

function shouldNeverAggregate(node: ToolNode, now?: number): boolean {
  // Failed tools: never aggregate for 2 minutes after creation
  if (node.status === "failed") {
    const ts = now ?? Date.now();
    if (ts - node.startTime < FAILED_NEVER_AGGREGATE_MS) return true;
  }
  if (NEVER_AGGREGATE_EVENTS.has(node.toolName)) return true;
  if (node.toolName.startsWith("Agent:")) return true;
  return false;
}

interface AggStats {
  totalDuration: number;
  totalTokens: number;
  successCount: number;
  failCount: number;
  minStart: number;
  maxEnd: number;
}

function computeToolNodeStats(nodes: ToolNode[]): AggStats {
  let totalDuration = 0;
  let successCount = 0;
  let failCount = 0;
  let minStart = Infinity;
  let maxEnd = 0;

  for (const n of nodes) {
    totalDuration += n.endTime ? n.endTime - n.startTime : 0;
    if (n.status === "completed") successCount++;
    if (n.status === "failed") failCount++;
    if (n.startTime < minStart) minStart = n.startTime;
    if (n.endTime && n.endTime > maxEnd) maxEnd = n.endTime;
  }

  return { totalDuration, totalTokens: 0, successCount, failCount, minStart, maxEnd };
}

function buildAggregatedNode(idPrefix: string, toolName: string, children: ToolNode[], stats: AggStats): AggregatedNode {
  return {
    type: "aggregate",
    id: `${idPrefix}-${children[0]!.id}`,
    toolName,
    count: children.length,
    children,
    totalDuration: stats.totalDuration,
    totalTokens: stats.totalTokens,
    successCount: stats.successCount,
    failCount: stats.failCount,
    startTime: stats.minStart,
    endTime: stats.maxEnd || Date.now(),
  };
}

function flattenDisplayNodes(group: DisplayNode[]): { children: ToolNode[]; stats: AggStats } {
  const allChildren: ToolNode[] = [];
  let totalDuration = 0;
  let totalTokens = 0;
  let successCount = 0;
  let failCount = 0;
  let minStart = Infinity;
  let maxEnd = 0;

  for (const g of group) {
    if (isAggregatedNode(g)) {
      allChildren.push(...g.children);
      totalDuration += g.totalDuration;
      totalTokens += g.totalTokens;
      successCount += g.successCount;
      failCount += g.failCount;
      if (g.startTime < minStart) minStart = g.startTime;
      if (g.endTime > maxEnd) maxEnd = g.endTime;
    } else {
      allChildren.push(g);
      totalDuration += g.endTime ? g.endTime - g.startTime : 0;
      if (g.status === "completed") successCount++;
      if (g.status === "failed") failCount++;
      if (g.startTime < minStart) minStart = g.startTime;
      if (g.endTime && g.endTime > maxEnd) maxEnd = g.endTime;
    }
  }

  return { children: allChildren, stats: { totalDuration, totalTokens, successCount, failCount, minStart, maxEnd } };
}

function aggregateNodes(nodes: ToolNode[], collapseOlderThan?: number, totalNodeCount?: number): DisplayNode[] {
  if (nodes.length === 0) return [];

  const now = Date.now();
  const threshold = collapseOlderThan ?? 0;
  const agentTotalNodes = totalNodeCount ?? nodes.length;

  // If agent has fewer than MAX_INDIVIDUAL_NODES total calls, show all individually
  if (agentTotalNodes < MAX_INDIVIDUAL_NODES) {
    return nodes;
  }

  if (threshold > 0) {
    const recentNodes: ToolNode[] = [];
    const oldByTool = new Map<string, ToolNode[]>();

    for (const n of nodes) {
      const age = now - n.startTime;
      if (age <= threshold || n.status === "running" || shouldNeverAggregate(n, now)) {
        recentNodes.push(n);
      } else {
        const key = n.toolName;
        if (!oldByTool.has(key)) oldByTool.set(key, []);
        oldByTool.get(key)!.push(n);
      }
    }

    const oldAggregated: DisplayNode[] = [];
    for (const [toolName, group] of oldByTool) {
      if (group.length >= 2) {
        const stats = computeToolNodeStats(group);
        oldAggregated.push(buildAggregatedNode("agg-old", toolName, group, stats));
      } else {
        oldAggregated.push(...group);
      }
    }

    oldAggregated.sort((a, b) => b.startTime - a.startTime);

    const recentAggregated = aggregateConsecutive(recentNodes);
    return aggregateByType([...recentAggregated, ...oldAggregated], agentTotalNodes);
  }

  return aggregateByType(aggregateConsecutive(nodes), agentTotalNodes);
}

/** Post-completion aggregation: collapse completed tool calls of the same type
 *  across non-consecutive positions when count >= AGGREGATION_MIN_COUNT.
 *  Running events are always shown individually.
 *  Failed events are pinned (shown individually) for FAILED_NEVER_AGGREGATE_MS,
 *  then become eligible for type-based aggregation. */
function aggregateByType(displayNodes: DisplayNode[], totalNodeCount?: number): DisplayNode[] {
  // If agent has fewer than MAX_INDIVIDUAL_NODES total calls, skip aggregation
  if (totalNodeCount !== undefined && totalNodeCount < MAX_INDIVIDUAL_NODES) {
    return displayNodes;
  }

  const now = Date.now();
  const pinned: DisplayNode[] = [];
  const candidates: DisplayNode[] = [];

  for (const node of displayNodes) {
    if (isAggregatedNode(node)) {
      candidates.push(node);
    } else {
      const tn = node as ToolNode;
      if (tn.status === "running") {
        pinned.push(node);
      } else if (tn.status === "failed" && (now - tn.startTime) < FAILED_NEVER_AGGREGATE_MS) {
        // Failed events within 2 minutes are always pinned individually
        pinned.push(node);
      } else {
        candidates.push(node);
      }
    }
  }

  const byTool = new Map<string, DisplayNode[]>();
  const ungroupable: DisplayNode[] = [];

  for (const node of candidates) {
    if (isAggregatedNode(node)) {
      const key = node.toolName;
      if (!byTool.has(key)) byTool.set(key, []);
      byTool.get(key)!.push(node);
    } else {
      const tn = node as ToolNode;
      if (shouldNeverAggregate(tn, now)) {
        ungroupable.push(node);
      } else {
        const key = tn.toolName;
        if (!byTool.has(key)) byTool.set(key, []);
        byTool.get(key)!.push(node);
      }
    }
  }

  const aggregated: DisplayNode[] = [];
  for (const [toolName, group] of byTool) {
    let totalIndividual = 0;
    for (const g of group) {
      totalIndividual += isAggregatedNode(g) ? g.count : 1;
    }

    if (totalIndividual >= AGGREGATION_MIN_COUNT) {
      const { children: allChildren, stats } = flattenDisplayNodes(group);
      aggregated.push(buildAggregatedNode("agg-type", toolName, allChildren, stats));
    } else {
      ungroupable.push(...group);
    }
  }

  // Pinned (running/failed) first, then ungroupable individuals, then aggregated summaries at bottom
  return [...pinned, ...ungroupable, ...aggregated];
}

function aggregateConsecutive(nodes: ToolNode[]): DisplayNode[] {
  if (nodes.length === 0) return [];

  const now = Date.now();
  const result: DisplayNode[] = [];
  let i = 0;

  while (i < nodes.length) {
    const current = nodes[i];
    if (!current) { i++; continue; }

    if (shouldNeverAggregate(current, now) || current.status === "running") {
      result.push(current);
      i++;
      continue;
    }

    const group: ToolNode[] = [current];
    let j = i + 1;
    while (j < nodes.length) {
      const next = nodes[j];
      if (!next) break;
      if (next.toolName !== current.toolName) break;
      if (shouldNeverAggregate(next, now) || next.status === "running") break;
      const prev = group[group.length - 1];
      if (!prev) break;
      if (Math.abs(next.startTime - prev.startTime) > AGGREGATION_WINDOW_MS) break;
      group.push(next);
      j++;
    }

    if (group.length >= 2) {
      const stats = computeToolNodeStats(group);
      result.push(buildAggregatedNode("agg", current.toolName, group, stats));
      i = j;
    } else {
      result.push(current);
      i++;
    }
  }

  return result;
}

/* ── Task parsing helpers ──────────────────────────────────────────── */

function parseTaskCreate(
  toolInput: unknown,
  toolResponse: unknown,
  agentName: string,
): TaskItem | null {
  const inp = toolInput as Record<string, unknown> | null;
  const resp = toolResponse as Record<string, unknown> | null;
  if (!inp) return null;

  let id = "task-" + Date.now();

  // First, try to parse numeric ID from string response like "Task #125 created successfully: ..."
  if (typeof toolResponse === "string") {
    const numMatch = (toolResponse as string).match(/Task\s+#(\d+)/);
    if (numMatch) id = numMatch[1]!;
  }

  // If still a fallback ID, try structured response fields
  if (id.startsWith("task-") && resp) {
    if (typeof resp.id === "string") id = resp.id;
    else if (typeof resp.task_id === "string") id = resp.task_id;
    else if (typeof resp === "string") {
      try {
        const parsed = JSON.parse(resp as unknown as string) as Record<string, unknown>;
        if (typeof parsed.id === "string") id = parsed.id;
      } catch { /* ignore */ }
    }
  }

  const subject = typeof inp.subject === "string"
    ? inp.subject
    : typeof inp.description === "string"
      ? inp.description
      : typeof inp.title === "string"
        ? (inp.title as string)
        : "Untitled task";

  const description = typeof inp.description === "string" ? inp.description : undefined;

  return { id, subject, status: "pending", owner: agentName, description };
}

function parseTaskUpdate(toolInput: unknown): { taskId: string; status: TaskItem["status"]; blockedBy?: string[] } | null {
  const inp = toolInput as Record<string, unknown> | null;
  if (!inp) return null;

  const taskId = typeof inp.taskId === "string"
    ? inp.taskId
    : typeof inp.task_id === "string"
      ? inp.task_id
      : typeof inp.id === "string"
        ? inp.id
        : null;

  if (!taskId) return null;

  const rawStatus = typeof inp.status === "string" ? inp.status : null;
  if (!rawStatus) return null;

  const status = TASK_STATUS_MAP[rawStatus] ?? "pending";
  const blockedBy = Array.isArray(inp.addBlockedBy) ? inp.addBlockedBy as string[] : undefined;
  return { taskId, status, blockedBy };
}

function parseTaskListResponse(text: string): TaskItem[] {
  const tasks: TaskItem[] = [];
  const lines = text.split("\n");
  for (const line of lines) {
    const match = line.match(/^#(\d+)\s+\[(\w+)]\s+(.+)$/);
    if (match) {
      const id = match[1]!;
      const rawStatus = match[2]!.toLowerCase();
      const subject = match[3]!.trim();
      const status = TASK_STATUS_MAP[rawStatus] ?? "pending";
      tasks.push({ id, subject, status });
    }
  }
  return tasks;
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

/* ── Relative Timestamp ───────────────────────────────────────────── */

function RelativeTimestamp({ ts }: { ts: number }) {
  const [, setTick] = useState(0);
  useEffect(() => {
    const id = setInterval(() => setTick((t) => t + 1), 1000);
    return () => clearInterval(id);
  }, []);
  return (
    <span title={new Date(ts).toISOString()} className="text-[10px] text-bc-muted/60 font-mono tabular-nums">
      {relativeTime(ts)}
    </span>
  );
}

/* ── Idle Timer ───────────────────────────────────────────────────── */

function IdleTimer({ lastEventTime }: { lastEventTime: number }) {
  const [, setTick] = useState(0);
  useEffect(() => {
    const id = setInterval(() => setTick((t) => t + 1), 1000);
    return () => clearInterval(id);
  }, []);
  return <>{idleDuration(lastEventTime)}</>;
}

/* ── Copy Button ───────────────────────────────────────────────────── */

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);
  const handleCopy = useCallback(() => {
    navigator.clipboard.writeText(text).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    }).catch(() => {});
  }, [text]);

  return (
    <button
      type="button"
      onClick={(e) => { e.stopPropagation(); handleCopy(); }}
      className="text-[10px] text-bc-muted hover:text-bc-text px-1.5 py-0.5 rounded border border-bc-border/40 hover:border-bc-accent transition-colors shrink-0"
      aria-label="Copy to clipboard"
    >
      {copied ? "Copied" : "Copy"}
    </button>
  );
}

/* ── MCP Badge ─────────────────────────────────────────────────────── */

function McpBadge({ server, func }: { server: string; func: string }) {
  return (
    <span className="inline-flex items-center gap-1">
      <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-mono ${mcpBadgeColors(server)}`}>
        <span aria-hidden="true">{mcpServerIcon(server)}</span>
        <span>{server}</span>
      </span>
      <span className="font-mono text-[13px] text-bc-text font-medium">{func}</span>
    </span>
  );
}

/* ── Search Highlight ──────────────────────────────────────────────── */

function SearchHighlight({ text, query }: { text: string; query: string }) {
  if (!query || !text) return <>{text}</>;
  const lower = text.toLowerCase();
  const q = query.toLowerCase();
  const idx = lower.indexOf(q);
  if (idx === -1) return <>{text}</>;
  return (
    <>
      {text.slice(0, idx)}
      <mark className="bg-yellow-500/20 text-inherit rounded px-0.5">{text.slice(idx, idx + q.length)}</mark>
      {text.slice(idx + q.length)}
    </>
  );
}

/* ── Tool Name Display ─────────────────────────────────────────────── */

function ToolNameDisplay({ toolName, searchQuery }: { toolName: string; searchQuery?: string }) {
  const parsed = parseToolName(toolName);
  if (parsed.type === "mcp" && parsed.mcpServer && parsed.mcpFunction) {
    return <McpBadge server={parsed.mcpServer} func={parsed.mcpFunction} />;
  }
  return (
    <span className="inline-flex items-center gap-1">
      <span className="text-[12px]" aria-hidden="true">{toolIcon(toolName)}</span>
      <span className="font-mono text-[13px] text-bc-text font-medium">
        {searchQuery ? <SearchHighlight text={parsed.display} query={searchQuery} /> : parsed.display}
      </span>
    </span>
  );
}

/* ── Tool Node Row ─────────────────────────────────────────────────── */

function ToolNodeRow({ node, depth = 0, searchQuery = "" }: { node: ToolNode; depth?: number; searchQuery?: string }) {
  const [expanded, setExpanded] = useState(false);
  const indent = depth * 20;
  const hasDetails = !!(node.fullInput || node.fullOutput || node.children.length > 0);
  const isSubagentSpawn = node.toolName === "Agent" || node.toolName.startsWith("Agent:");

  // Subagent tree: use AgentTreeNode for nested rendering
  if (isSubagentSpawn) {
    return <AgentTreeNode node={node} depth={depth} />;
  }

  const inputJson = node.fullInput ? JSON.stringify(redactValue(node.fullInput), null, 2) : "";
  const outputJson = node.fullOutput ? JSON.stringify(redactValue(node.fullOutput), null, 2) : "";

  return (
    <>
      <button
        type="button"
        className={`group flex items-start gap-2 py-1 px-3 w-full text-left hover:bg-bc-surface-hover cursor-pointer transition-colors focus-visible:ring-2 focus-visible:ring-bc-accent ${node.status === "failed" ? "!bg-red-950/30 hover:!bg-red-950/40" : ""}`}
        style={{ paddingLeft: `${indent + 12}px` }}
        onClick={() => setExpanded(!expanded)}
        aria-label={`${expanded ? "Collapse" : "Expand"} tool ${node.toolName}`}
      >
        <span className="text-bc-muted text-xs select-none mt-[3px] shrink-0">
          {depth > 0 ? "\u251C\u2500" : ""}
        </span>
        <span className="text-bc-muted/50 text-[10px] select-none mt-[3px] shrink-0 w-3 text-center group-hover:text-bc-muted">
          {hasDetails ? (expanded ? "\u25BC" : "\u25B6") : "\u00B7"}
        </span>
        <ToolDot status={node.status} />
        <ToolNameDisplay toolName={node.toolName} searchQuery={searchQuery} />
        {node.args && (
          <span className="text-[12px] text-bc-muted font-mono min-w-0 flex-1 break-words" title={redactSecrets(node.args)}>
            {searchQuery ? <SearchHighlight text={redactSecrets(node.args)} query={searchQuery} /> : redactSecrets(node.args)}
          </span>
        )}
        <span className="flex items-center gap-2 shrink-0">
          <RelativeTimestamp ts={node.startTime} />
          <span className={`text-[11px] tabular-nums font-mono ${node.status === "running" ? "text-bc-muted" : durationColorClass(node.startTime, node.endTime)}`}>
            {node.status === "running" ? (
              <ElapsedTimer start={node.startTime} />
            ) : (
              elapsed(node.startTime, node.endTime)
            )}
          </span>
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
          className="text-[11px] font-mono px-3 py-1 bg-bc-surface mx-3 mb-1 rounded overflow-x-auto max-h-48 overflow-y-auto"
          style={{ marginLeft: `${indent + 12}px` }}
        >
          <div className="flex items-center justify-between mb-1">
            <span className="text-[10px] text-bc-muted uppercase tracking-wide font-semibold">Input</span>
            <CopyButton text={inputJson} />
          </div>
          <pre className="whitespace-pre-wrap break-all text-bc-muted">
            {inputJson}
          </pre>
        </div>
      )}

      {expanded && node.fullOutput && (
        <div
          className="text-[11px] font-mono px-3 py-1 bg-bc-surface mx-3 mb-1 rounded overflow-x-auto max-h-48 overflow-y-auto"
          style={{ marginLeft: `${indent + 12}px` }}
        >
          <div className="flex items-center justify-between mb-1">
            <span className="text-[10px] text-bc-success uppercase tracking-wide font-semibold">Output</span>
            <CopyButton text={outputJson} />
          </div>
          <pre className="whitespace-pre-wrap break-all text-bc-success/80">
            {outputJson}
          </pre>
        </div>
      )}

      {node.children.map((child) => (
        <ToolNodeRow key={child.id} node={child} depth={depth + 1} searchQuery={searchQuery} />
      ))}
    </>
  );
}

/* ── Agent Tree Node (recursive subagent nesting) ──────────────────── */

function AgentTreeNode({ node, depth = 0 }: { node: ToolNode; depth?: number }) {
  const [expanded, setExpanded] = useState(true);
  const indent = depth * 16;
  const duration = node.endTime ? elapsed(node.startTime, node.endTime) : undefined;
  const childCount = node.children.length;

  const subagentChildren = node.children.filter(
    (c) => c.toolName === "Agent" || c.toolName.startsWith("Agent:"),
  );
  const toolChildren = node.children.filter(
    (c) => c.toolName !== "Agent" && !c.toolName.startsWith("Agent:"),
  );

  return (
    <div style={{ marginLeft: `${indent}px` }}>
      {/* Subagent header */}
      <button
        type="button"
        className="group flex items-start gap-2 py-1.5 px-3 w-full text-left hover:bg-bc-surface-hover cursor-pointer transition-colors focus-visible:ring-2 focus-visible:ring-bc-accent bg-blue-950/20 rounded-md my-0.5"
        onClick={() => setExpanded(!expanded)}
        aria-label={`${expanded ? "Collapse" : "Expand"} subagent ${node.toolName}`}
      >
        <span className="text-bc-muted/50 text-[10px] select-none mt-[3px] shrink-0 w-3 text-center group-hover:text-bc-muted">
          {childCount > 0 ? (expanded ? "\u25BC" : "\u25B6") : "\u00B7"}
        </span>
        <ToolDot status={node.status} />
        <span className="text-[13px]" aria-hidden="true">{"\uD83E\uDD16"}</span>
        <span className="font-mono text-[13px] text-bc-text font-semibold">{node.toolName}</span>
        {node.args && (
          <span className="text-[12px] text-bc-muted truncate max-w-[300px] font-mono italic">
            &ldquo;{node.args}&rdquo;
          </span>
        )}
        <span className="ml-auto flex items-center gap-2 shrink-0">
          <RelativeTimestamp ts={node.startTime} />
          {node.status === "running" ? (
            <span className="text-[11px] text-blue-400 font-mono tabular-nums">
              {"\u23F1"} <ElapsedTimer start={node.startTime} />
            </span>
          ) : duration ? (
            <span className={`text-[11px] font-mono tabular-nums ${durationColorClass(node.startTime, node.endTime)}`}>
              {"\u23F1"} {duration}
            </span>
          ) : null}
          {node.status === "completed" && (
            <span className="text-[10px] text-bc-success font-mono">{"\u2713"}</span>
          )}
          {node.status === "failed" && (
            <span className="text-[10px] text-bc-error font-mono">{"\u2717"}</span>
          )}
        </span>
      </button>

      {/* Tree children with connector lines */}
      {expanded && childCount > 0 && (
        <div className="border-l-2 border-bc-muted/30 ml-4 pl-3">
          {toolChildren.map((child, idx) => {
            const isLast = idx === toolChildren.length - 1 && subagentChildren.length === 0;
            return (
              <div key={child.id} className="flex items-start gap-0">
                <span className="text-bc-muted/30 text-xs select-none mt-[3px] shrink-0 w-4">
                  {isLast ? "\u2514\u2500" : "\u251C\u2500"}
                </span>
                <div className="flex-1 min-w-0">
                  <ToolNodeRow node={child} depth={0} />
                </div>
              </div>
            );
          })}

          {/* Nested subagent children (recursive) */}
          {subagentChildren.map((child, idx) => {
            const isLast = idx === subagentChildren.length - 1;
            return (
              <div key={child.id} className="flex items-start gap-0">
                <span className="text-bc-muted/30 text-xs select-none mt-[3px] shrink-0 w-4">
                  {isLast ? "\u2514\u2500" : "\u251C\u2500"}
                </span>
                <div className="flex-1 min-w-0">
                  <AgentTreeNode node={child} depth={0} />
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

/* ── Aggregated Node Row ───────────────────────────────────────────── */

function AggregatedChildRow({ child, searchQuery = "" }: { child: ToolNode; searchQuery?: string }) {
  const [showRawJson, setShowRawJson] = useState(false);

  const fullEventJson = JSON.stringify(redactValue({
    id: child.id,
    toolName: child.toolName,
    args: child.args,
    status: child.status,
    startTime: child.startTime,
    endTime: child.endTime,
    error: child.error,
    fullInput: child.fullInput,
    fullOutput: child.fullOutput,
  }), null, 2);

  return (
    <div>
      <ToolNodeRow node={child} depth={1} searchQuery={searchQuery} />
      {!!(child.fullInput || child.fullOutput) && (
        <div className="ml-8 mb-1">
          <button
            type="button"
            onClick={() => setShowRawJson(!showRawJson)}
            className="text-[10px] text-bc-muted hover:text-bc-accent font-mono transition-colors px-2 py-0.5 rounded border border-bc-border/30 hover:border-bc-accent/50"
          >
            {showRawJson ? "Hide Raw JSON" : "Raw JSON"}
          </button>
          {showRawJson && (
            <div className="mt-1 text-[11px] font-mono px-3 py-2 bg-bc-bg rounded border border-bc-border/30 overflow-x-auto max-h-64 overflow-y-auto">
              <div className="flex justify-end mb-1">
                <CopyButton text={fullEventJson} />
              </div>
              <pre className="whitespace-pre-wrap break-all text-bc-muted">
                {fullEventJson}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function AggregatedNodeRow({ node, searchQuery = "" }: { node: AggregatedNode; searchQuery?: string }) {
  const [expanded, setExpanded] = useState(false);

  return (
    <>
      <button
        type="button"
        className="group flex items-start gap-2 py-1 px-3 w-full text-left hover:bg-bc-surface-hover cursor-pointer transition-colors focus-visible:ring-2 focus-visible:ring-bc-accent bg-bc-surface/50"
        onClick={() => setExpanded(!expanded)}
        aria-label={`${expanded ? "Collapse" : "Expand"} aggregated ${node.toolName} (${node.count} calls)`}
      >
        <span className="text-bc-muted text-xs select-none mt-[3px] shrink-0">
          {expanded ? "\u25BC" : "\u25B6"}
        </span>
        <span className="inline-flex h-2 w-2 mt-[5px] shrink-0 rounded-full bg-bc-success" />
        <ToolNameDisplay toolName={node.toolName} />
        <span className="text-[12px] font-mono font-semibold text-bc-accent px-1.5 py-0 rounded bg-bc-accent/10">
          &times;{node.count}
        </span>
        <span className="text-[11px] text-bc-muted font-mono tabular-nums flex-1 min-w-0">
          {node.count} total
          {node.totalDuration > 0 && <> &middot; {elapsed(0, node.totalDuration)}</>}
          {node.totalDuration > 0 && node.count > 1 && (
            <> &middot; avg {elapsed(0, Math.round(node.totalDuration / node.count))}</>
          )}
          {node.totalTokens > 0 && <> &middot; {node.totalTokens.toLocaleString()} tok</>}
          {node.failCount > 0 && (
            <span className="text-bc-error"> &middot; {node.failCount} failed</span>
          )}
          <> &middot; {node.successCount}/{node.count} ok</>
        </span>
      </button>

      {expanded && (
        <div className="border-l-2 border-bc-border/40 ml-6">
          {node.children.map((child) => (
            <AggregatedChildRow key={child.id} child={child} searchQuery={searchQuery} />
          ))}
        </div>
      )}
    </>
  );
}

/* ── Display Node Row ──────────────────────────────────────────────── */

function DisplayNodeRow({ node, searchQuery = "" }: { node: DisplayNode; searchQuery?: string }) {
  if (isAggregatedNode(node)) {
    return <AggregatedNodeRow node={node} searchQuery={searchQuery} />;
  }
  return <ToolNodeRow node={node} searchQuery={searchQuery} />;
}

/* ── Agent Drill-Down View ─────────────────────────────────────────── */

function DrillDownTasksSection({ tasks, agentName }: { tasks: Map<string, TaskItem>; agentName: string }) {
  const [collapsed, setCollapsed] = useState(false);

  const agentTasks = useMemo(() => {
    const result: TaskItem[] = [];
    for (const [, task] of tasks) {
      if (task.owner === agentName || !task.owner) {
        if (task.status !== "deleted") result.push(task);
      }
    }
    return result;
  }, [tasks, agentName]);

  const completedCount = agentTasks.filter((t) => t.status === "completed").length;
  const total = agentTasks.length;
  const progressPct = total > 0 ? Math.round((completedCount / total) * 100) : 0;

  if (total === 0) return null;

  const statusColor: Record<string, string> = {
    pending: "bg-zinc-500",
    in_progress: "bg-blue-500",
    completed: "bg-emerald-500",
    deleted: "bg-red-500",
  };

  return (
    <div className="rounded-lg border border-bc-border bg-bc-surface overflow-hidden mb-4">
      <button
        type="button"
        onClick={() => setCollapsed(!collapsed)}
        className="flex items-center gap-2 w-full px-4 py-2.5 text-left hover:bg-bc-surface-hover transition-colors"
      >
        <span className="text-bc-muted/50 text-[10px] select-none w-3 text-center">
          {collapsed ? "\u25B6" : "\u25BC"}
        </span>
        <span className="text-[13px]">{"\u2705"}</span>
        <span className="text-sm font-semibold text-bc-text">Tasks</span>
        <span className="text-xs text-bc-muted font-mono tabular-nums">
          ({completedCount}/{total} complete)
        </span>
        <span className="flex-1 mx-2 h-1.5 bg-bc-bg rounded-full overflow-hidden max-w-[200px]">
          <span
            className="h-full bg-bc-success rounded-full transition-all duration-300"
            style={{ width: `${progressPct}%` }}
          />
        </span>
      </button>

      {!collapsed && (
        <div className="border-t border-bc-border/60 px-4 py-2 space-y-1">
          {agentTasks.map((task) => {
            const isBlocked = task.blockedBy && task.blockedBy.length > 0 && task.status !== "completed";
            return (
              <div key={task.id} className={`flex items-center gap-2 py-1 ${isBlocked ? "opacity-50" : ""}`}>
                <span className={`inline-flex h-2.5 w-2.5 rounded-full shrink-0 ${statusColor[task.status] ?? "bg-zinc-500"}`} />
                <span className="text-[11px] text-bc-muted font-mono shrink-0">#{task.id}</span>
                <span className={`text-sm font-mono min-w-0 ${
                  task.status === "completed" ? "line-through text-bc-muted/60" :
                  task.status === "in_progress" ? "text-blue-400 font-semibold" :
                  "text-bc-text"
                }`}>
                  {task.subject.length > 80 ? task.subject.slice(0, 77) + "..." : task.subject}
                </span>
                {isBlocked && (
                  <span className="text-[10px] text-yellow-500/80 font-mono shrink-0">
                    Blocked by {task.blockedBy!.map((b) => `#${b}`).join(", ")}
                  </span>
                )}
                <span className={`text-[10px] px-2 py-0.5 rounded-full font-mono capitalize shrink-0 ml-auto ${
                  task.status === "completed" ? "bg-emerald-900/30 text-emerald-400" :
                  task.status === "in_progress" ? "bg-blue-900/30 text-blue-400" :
                  task.status === "pending" ? "bg-zinc-800 text-zinc-400" :
                  "bg-red-900/30 text-red-400"
                }`}>
                  {task.status.replace("_", " ")}
                </span>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

/* ── Drill-Down: Full-width individual event row ──────────────────── */

function DrillDownEventRow({ node }: { node: ToolNode }) {
  const [expanded, setExpanded] = useState(false);
  const hasDetails = !!(node.fullInput || node.fullOutput);
  const inputJson = node.fullInput ? JSON.stringify(redactValue(node.fullInput), null, 2) : "";
  const outputJson = node.fullOutput ? JSON.stringify(redactValue(node.fullOutput), null, 2) : "";

  return (
    <div className={`border-b border-bc-border/30 ${node.status === "failed" ? "bg-red-950/20" : ""}`}>
      <button
        type="button"
        className="group flex items-start gap-3 py-2 px-4 w-full text-left hover:bg-bc-surface-hover cursor-pointer transition-colors"
        onClick={() => setExpanded(!expanded)}
        aria-label={`${expanded ? "Collapse" : "Expand"} ${node.toolName} event`}
      >
        <span className="text-bc-muted/50 text-[10px] select-none mt-[3px] shrink-0 w-3 text-center group-hover:text-bc-muted">
          {hasDetails ? (expanded ? "\u25BC" : "\u25B6") : "\u00B7"}
        </span>
        <ToolDot status={node.status} />
        <span className="shrink-0">
          <ToolNameDisplay toolName={node.toolName} />
        </span>
        <span className="text-[12px] text-bc-muted font-mono min-w-0 flex-1 break-words">
          {redactSecrets(node.args)}
        </span>
        <span className="flex items-center gap-2 shrink-0">
          <RelativeTimestamp ts={node.startTime} />
          <span className={`text-[11px] tabular-nums font-mono ${node.status === "running" ? "text-bc-muted" : durationColorClass(node.startTime, node.endTime)}`}>
            {node.status === "running" ? <ElapsedTimer start={node.startTime} /> : elapsed(node.startTime, node.endTime)}
          </span>
        </span>
      </button>

      {node.error && (
        <div className="text-[11px] text-bc-error/80 font-mono px-4 py-0.5 ml-8">
          {redactSecrets(node.error.length > 200 ? node.error.slice(0, 197) + "..." : node.error)}
        </div>
      )}

      {expanded && !!node.fullInput && (
        <div className="text-[11px] font-mono px-4 py-2 bg-bc-surface mx-4 mb-1 rounded overflow-x-auto max-h-64 overflow-y-auto">
          <div className="flex items-center justify-between mb-1">
            <span className="text-[10px] text-bc-muted uppercase tracking-wide font-semibold">Input</span>
            <CopyButton text={inputJson} />
          </div>
          <pre className="whitespace-pre-wrap break-all text-bc-muted">{inputJson}</pre>
        </div>
      )}

      {expanded && !!node.fullOutput && (
        <div className="text-[11px] font-mono px-4 py-2 bg-bc-surface mx-4 mb-1 rounded overflow-x-auto max-h-64 overflow-y-auto">
          <div className="flex items-center justify-between mb-1">
            <span className="text-[10px] text-bc-success uppercase tracking-wide font-semibold">Output</span>
            <CopyButton text={outputJson} />
          </div>
          <pre className="whitespace-pre-wrap break-all text-bc-success/80">{outputJson}</pre>
        </div>
      )}
    </div>
  );
}

/* ── Drill-Down: Raw event type badge colors ──────────────────────── */

function rawEventBadgeColor(eventType: string): string {
  if (eventType === "PreToolUse") return "bg-blue-900/40 text-blue-300";
  if (eventType === "PostToolUse") return "bg-emerald-900/40 text-emerald-300";
  if (eventType === "PostToolUseFailure") return "bg-red-900/40 text-red-300";
  if (eventType === "UserPromptSubmit") return "bg-purple-900/40 text-purple-300";
  if (eventType.startsWith("Subagent")) return "bg-amber-900/40 text-amber-300";
  if (eventType === "PermissionRequest" || eventType === "Elicitation") return "bg-yellow-900/40 text-yellow-300";
  if (eventType === "SessionStart" || eventType === "SessionEnd") return "bg-zinc-700 text-zinc-300";
  return "bg-zinc-800 text-zinc-400";
}

function AgentDrillDown({
  activity,
  rawEvents,
  tasks,
  onBack,
}: {
  activity: AgentActivity;
  rawEvents: RawEvent[];
  tasks: Map<string, TaskItem>;
  onBack: () => void;
}) {
  const [activeTab, setActiveTab] = useState<DrillDownTab>("live");
  const [rawExpanded, setRawExpanded] = useState<Set<string>>(new Set());

  const cost = estimateCost(activity);

  // Show ALL events individually in drill-down — no aggregation
  const allNodes = useMemo(() => {
    return [...activity.nodes].sort((a, b) => b.startTime - a.startTime);
  }, [activity.nodes]);

  const toggleRawExpanded = useCallback((key: string) => {
    setRawExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  }, []);

  // Reverse raw events so newest at top
  const reversedRawEvents = useMemo(() => [...rawEvents].reverse(), [rawEvents]);

  const tabs: { key: DrillDownTab; label: string }[] = [
    { key: "live", label: "Live Stream" },
    { key: "raw", label: "Raw Stream" },
  ];

  return (
    <div className="flex flex-col h-full">
      {/* Header bar */}
      <div className="flex items-center gap-3 mb-4 flex-wrap">
        <button
          type="button"
          onClick={onBack}
          className="inline-flex items-center gap-1.5 text-sm text-bc-muted hover:text-bc-text px-2 py-1 rounded border border-bc-border hover:border-bc-accent transition-colors shrink-0"
        >
          <svg width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M8 2l-4 4 4 4" />
          </svg>
          Back
        </button>
        <StateDot state={activity.state} />
        <span className="text-lg font-bold text-bc-text">{activity.name}</span>
        <span className="text-xs text-bc-muted font-mono">({activity.role})</span>
        <span className="text-xs text-bc-muted capitalize font-mono">{activity.state}</span>
        {activity.tokens > 0 && (
          <span className="text-xs text-bc-muted font-mono tabular-nums">
            {activity.tokens.toLocaleString()} tok
          </span>
        )}
        {cost > 0 && (
          <span className="text-xs text-bc-success font-mono tabular-nums">
            ${cost.toFixed(2)}
          </span>
        )}
        {activity.task && (
          <span className="text-xs text-bc-muted truncate max-w-[400px]">{activity.task}</span>
        )}
      </div>

      {/* Tasks section — above tabs */}
      <DrillDownTasksSection tasks={tasks} agentName={activity.name} />

      {/* Tabs */}
      <div className="flex items-center gap-0 border-b border-bc-border mb-4">
        {tabs.map((tab) => (
          <button
            key={tab.key}
            type="button"
            onClick={() => setActiveTab(tab.key)}
            className={`px-4 py-2 text-sm font-medium transition-colors border-b-2 -mb-px ${
              activeTab === tab.key
                ? "border-bc-accent text-bc-text"
                : "border-transparent text-bc-muted hover:text-bc-text hover:border-bc-border"
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Tab content */}
      <div className="flex-1 overflow-y-auto min-h-0">
        {/* Live Stream tab — ALL events individually, no aggregation */}
        {activeTab === "live" && (
          <div>
            {allNodes.length === 0 ? (
              <div className="text-sm text-bc-muted italic py-8 text-center">
                No tool events yet for this agent.
              </div>
            ) : (
              allNodes.map((node) => (
                <DrillDownEventRow key={node.id} node={node} />
              ))
            )}
          </div>
        )}

        {/* Raw Stream tab — timestamped JSON blocks, newest at top */}
        {activeTab === "raw" && (
          <div className="space-y-1">
            {reversedRawEvents.length === 0 ? (
              <div className="text-sm text-bc-muted italic py-8 text-center">
                No raw events captured for this agent yet.
              </div>
            ) : (
              reversedRawEvents.map((evt) => {
                const evtKey = `${evt.timestamp}-${evt.eventType}`;
                const isOpen = rawExpanded.has(evtKey);
                const jsonStr = JSON.stringify(redactValue(evt.raw), null, 2);
                return (
                  <div key={evtKey} className="border border-bc-border/40 rounded bg-bc-surface overflow-hidden">
                    <button
                      type="button"
                      onClick={() => toggleRawExpanded(evtKey)}
                      className="flex items-center gap-2 w-full px-3 py-1.5 text-left hover:bg-bc-surface-hover transition-colors"
                    >
                      <span className="text-bc-muted/50 text-[10px] select-none w-3 text-center">
                        {isOpen ? "\u25BC" : "\u25B6"}
                      </span>
                      <span className="text-[10px] text-bc-muted font-mono tabular-nums shrink-0">
                        {new Date(evt.timestamp).toISOString().replace("T", " ").slice(0, 23)}
                      </span>
                      <span className={`text-[11px] font-mono font-medium shrink-0 px-1.5 py-0.5 rounded ${rawEventBadgeColor(evt.eventType)}`}>
                        {evt.eventType}
                      </span>
                      <span className="text-[11px] text-bc-muted font-mono truncate flex-1">
                        {jsonStr.length > 100 ? jsonStr.slice(0, 97) + "..." : jsonStr.replace(/\n/g, " ")}
                      </span>
                    </button>
                    {isOpen && (
                      <div className="border-t border-bc-border/40 px-3 py-2 bg-bc-bg">
                        <div className="flex justify-end mb-1">
                          <CopyButton text={jsonStr} />
                        </div>
                        <pre className="text-[11px] font-mono text-bc-muted whitespace-pre-wrap break-all max-h-64 overflow-y-auto">
                          {jsonStr}
                        </pre>
                      </div>
                    )}
                  </div>
                );
              })
            )}
          </div>
        )}
      </div>
    </div>
  );
}

/* ── Agent Activity Card ───────────────────────────────────────────── */

const AgentCard = memo(function AgentCard({
  activity,
  onToggle,
  onDrillDown,
  isFilterActive,
  searchTerm,
  typeFilter,
  isPaused,
}: {
  activity: AgentActivity;
  onToggle: () => void;
  onDrillDown: () => void;
  isFilterActive: boolean;
  searchTerm: string;
  typeFilter: FilterType;
  isPaused: boolean;
}) {
  const [collapseOld, setCollapseOld] = useState(true);

  const visibleNodes = searchTerm
    ? activity.nodes.filter((n) => nodeMatchesSearch(n, searchTerm.toLowerCase()))
    : activity.nodes;

  const sortedNodes = sortNodes(visibleNodes);

  const runningCount = sortedNodes.filter((n) => n.status === "running").length;
  const errorCount = activity.nodes.filter((n) => n.status === "failed").length;
  const displayNodes = aggregateNodes(sortedNodes, collapseOld ? AUTO_COLLAPSE_MS : undefined, activity.nodes.length);
  const matchCount = searchTerm ? visibleNodes.length : 0;
  const showToolNodes = typeFilter !== "state";

  const skipAnimation = isPaused || visibleNodes.length > 5;

  return (
    <div className={`rounded-lg border bg-bc-surface overflow-hidden transition-colors ${isFilterActive ? "border-bc-accent ring-1 ring-bc-accent/30" : "border-bc-border"}`}>
      <div className="flex items-center">
        {/* Expand/collapse chevron for tool list */}
        <button
          type="button"
          onClick={(e) => { e.stopPropagation(); onToggle(); }}
          className="flex items-center gap-3 px-4 py-3 hover:bg-bc-surface-hover transition-colors text-left focus-visible:ring-2 focus-visible:ring-bc-accent shrink-0"
          aria-label={`${activity.collapsed ? "Expand" : "Collapse"} ${activity.name} tool list`}
        >
          <svg
            width="12" height="12" viewBox="0 0 12 12" fill="none"
            stroke="currentColor" strokeWidth="2"
            className={`text-bc-muted transition-transform ${activity.collapsed ? "" : "rotate-90"}`}
          >
            <path d="M4 2l4 4-4 4" />
          </svg>
        </button>

        {/* Clickable header area — opens drill-down */}
        <button
          type="button"
          onClick={onDrillDown}
          className="flex-1 flex items-center gap-3 py-3 pr-4 hover:bg-bc-surface-hover transition-colors text-left focus-visible:ring-2 focus-visible:ring-bc-accent min-w-0 cursor-pointer"
          title={`Open ${activity.name} detail view`}
        >
          <StateDot state={activity.state} />

          <span className="font-semibold text-[14px] text-bc-text">
            {activity.name}
          </span>

          {errorCount > 0 && (
            <span className="inline-flex items-center justify-center min-w-[18px] h-[18px] px-1 text-[10px] font-bold text-white bg-bc-error rounded-full leading-none">
              {errorCount}
            </span>
          )}

          {searchTerm && matchCount > 0 && (
            <span className="text-[11px] text-bc-accent font-mono">
              {matchCount} {matchCount === 1 ? "match" : "matches"}
            </span>
          )}

          <span className="text-[11px] text-bc-muted font-mono">
            {activity.role}
          </span>

          {activity.task && (
            <span className="text-[12px] text-bc-muted truncate max-w-[300px]">
              {activity.task}
            </span>
          )}

          <span className="ml-auto flex items-center gap-3">
            {typeFilter === "state" && (
              <span className="text-[11px] text-bc-muted font-mono capitalize">
                {activity.state}
              </span>
            )}
            {runningCount > 0 && (
              <span className="text-[11px] text-blue-400 font-mono">
                {runningCount} running
              </span>
            )}
            {estimateCost(activity) > 0 && (
              <span className="text-[11px] text-bc-success font-mono tabular-nums" title={activity.costUsd > 0 ? "From API" : "Estimated from tokens"}>
                ${estimateCost(activity).toFixed(2)}
              </span>
            )}
            {activity.tokens > 0 && (
              <span className="text-[11px] text-bc-muted font-mono tabular-nums">
                {activity.tokens.toLocaleString()} tok
              </span>
            )}
            <span className="text-[11px] text-bc-muted/60 group-hover:text-bc-accent transition-colors">
              View details &rarr;
            </span>
          </span>
        </button>
      </div>

      {!activity.collapsed && showToolNodes && displayNodes.length > 0 && (
        <div className="border-t border-bc-border/60 py-1">
          {visibleNodes.length > 3 && (
            <div className="flex justify-end px-3 py-1">
              <button
                type="button"
                onClick={() => setCollapseOld((prev) => !prev)}
                className="text-[10px] text-bc-muted hover:text-bc-accent font-mono transition-colors"
              >
                {collapseOld ? "Show all" : "Collapse old"}
              </button>
            </div>
          )}
          <AnimatePresence mode="popLayout" initial={false}>
            {displayNodes.map((node) => {
              const nodeKey = node.id;
              if (skipAnimation) {
                return (
                  <div key={nodeKey}>
                    <DisplayNodeRow node={node} searchQuery={searchTerm} />
                  </div>
                );
              }
              return (
                <motion.div
                  key={nodeKey}
                  initial={{ opacity: 0, y: -20, height: 0 }}
                  animate={{ opacity: 1, y: 0, height: "auto" }}
                  exit={{ opacity: 0, height: 0 }}
                  transition={{ duration: 0.2, ease: "easeOut" }}
                  layout
                >
                  <DisplayNodeRow node={node} searchQuery={searchTerm} />
                </motion.div>
              );
            })}
          </AnimatePresence>
        </div>
      )}

      {!activity.collapsed && showToolNodes && visibleNodes.length === 0 && !searchTerm && (
        <div className="border-t border-bc-border/60 py-3 px-4 text-[12px] text-bc-muted italic">
          {activity.lastEventTime > 0 ? (
            <IdleTimer lastEventTime={activity.lastEventTime} />
          ) : (
            "Waiting for activity..."
          )}
        </div>
      )}

      {!activity.collapsed && typeFilter === "state" && (
        <div className="border-t border-bc-border/60 py-3 px-4 text-[12px] text-bc-muted">
          <span className="capitalize font-medium text-bc-text">{activity.state}</span>
          {activity.task && <span className="ml-2">--- {activity.task}</span>}
          {activity.tokens > 0 && (
            <span className="ml-2 font-mono tabular-nums">{activity.tokens.toLocaleString()} tokens</span>
          )}
        </div>
      )}
    </div>
  );
});

/* ── Event processing helpers ──────────────────────────────────────── */

/** Try to update a running child node inside the active subagent. Returns true if found. */
function updateSubagentChild(
  activity: AgentActivity,
  predicate: (n: ToolNode) => boolean,
  updater: (child: ToolNode) => ToolNode,
): boolean {
  if (activity.activeSubagentIdx === undefined || activity.activeSubagentIdx < 0) return false;
  const parentNode = activity.nodes[activity.activeSubagentIdx];
  if (!parentNode) return false;

  const childIdx = findLastIdx(parentNode.children, predicate);
  if (childIdx < 0) return false;

  const updatedChildren = [...parentNode.children];
  updatedChildren[childIdx] = updater(updatedChildren[childIdx]!);
  activity.nodes[activity.activeSubagentIdx] = { ...parentNode, children: updatedChildren };
  return true;
}

/** Find and update a running top-level node. Returns true if found. */
function updateTopLevelNode(
  activity: AgentActivity,
  predicate: (n: ToolNode) => boolean,
  updater: (node: ToolNode) => ToolNode,
): boolean {
  const idx = findLastIdx(activity.nodes, predicate);
  if (idx < 0) return false;
  activity.nodes[idx] = updater(activity.nodes[idx]!);
  return true;
}

/* ── Logs (Live Operations Center) ─────────────────────────────────── */

export function Logs() {
  const [activities, setActivities] = useState<Map<string, AgentActivity>>(new Map());
  const [agents, setAgents] = useState<Agent[]>([]);
  const [agentFilter, setAgentFilter] = useState("");
  const [typeFilter, setTypeFilter] = useState<FilterType>("all");
  const [searchFilter, setSearchFilter] = useState("");
  const [eventCount, setEventCount] = useState(0);
  const [paused, setPaused] = useState(false);
  const pausedRef = useRef(false);
  const pausedBuffer = useRef<HookEvent[]>([]);
  const [pausedCount, setPausedCount] = useState(0);
  const [showJumpToLatest, setShowJumpToLatest] = useState(false);
  const [newEventsSinceScroll, setNewEventsSinceScroll] = useState(0);
  const [showShortcuts, setShowShortcuts] = useState(false);
  const [focusedCardIdx, setFocusedCardIdx] = useState(-1);
  const [tasks, setTasks] = useState<Map<string, TaskItem>>(new Map());
  const [drillDownAgent, setDrillDownAgent] = useState<string | null>(null);
  const rawEventsRef = useRef<Map<string, RawEvent[]>>(new Map());
  const [rawEventsVersion, setRawEventsVersion] = useState(0);
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const searchInputRef = useRef<HTMLInputElement>(null);
  const eventBuffer = useRef<HookEvent[]>([]);
  const { connected, reconnecting, subscribe } = useWebSocket();

  // Keep pausedRef in sync so interval/event handlers always see current value
  useEffect(() => {
    pausedRef.current = paused;
  }, [paused]);

  // Seed from agents API + initial logs
  useEffect(() => {
    api.listAgents().then((agentList) => {
      setAgents(agentList);
      setActivities((prev) => {
        const next = new Map(prev);
        for (const a of agentList) {
          if (!next.has(a.name)) {
            const updatedAt = a.updated_at ? new Date(a.updated_at).getTime() : 0;
            const agentCost = a.cost_usd ?? (a as unknown as Record<string, unknown>).total_cost_usd as number ?? 0;
            next.set(a.name, {
              name: a.name,
              state: a.state,
              task: a.task ?? "",
              tool: a.tool,
              role: a.role ?? "",
              tokens: a.total_tokens ?? 0,
              inputTokens: 0,
              outputTokens: 0,
              costUsd: agentCost,
              lastEventTime: updatedAt > 0 && !isNaN(updatedAt) ? updatedAt : 0,
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

  // Process buffered hook events
  const flushEvents = useCallback(() => {
    const events = eventBuffer.current.splice(0);
    if (events.length === 0) return;

    if (pausedRef.current) {
      pausedBuffer.current.push(...events);
      setPausedCount(pausedBuffer.current.length);
      return;
    }

    setEventCount((c) => c + events.length);

    // Process task-related events
    setTasks((prevTasks) => {
      let nextTasks = prevTasks;
      let changed = false;

      for (const evt of events) {
        const toolName = evt.tool_name ?? "";

        // TaskCreate: on PostToolUse, parse the created task
        if (evt.event === "PostToolUse" && toolName.includes("TaskCreate")) {
          const task = parseTaskCreate(evt.tool_input, evt.tool_response, evt.agent);
          if (task) {
            if (!changed) { nextTasks = new Map(prevTasks); changed = true; }
            nextTasks.set(task.id, task);
          }
        }

        // TaskCreate: also parse ID from tool_response string like "Task #95 created successfully: Subject"
        if (evt.event === "PostToolUse" && toolName.includes("TaskCreate")) {
          const resp = evt.tool_response;
          if (typeof resp === "string") {
            const match = resp.match(/Task\s+#(\d+)/);
            if (match) {
              const numId = match[1]!;
              let replaced = false;
              for (const [key, task] of nextTasks) {
                if (key.startsWith("task-") && task.owner === evt.agent) {
                  if (!changed) { nextTasks = new Map(prevTasks); changed = true; }
                  nextTasks.delete(key);
                  nextTasks.set(numId, { ...task, id: numId });
                  replaced = true;
                  break;
                }
              }
              if (!replaced && !nextTasks.has(numId)) {
                if (!changed) { nextTasks = new Map(prevTasks); changed = true; }
                const subjectMatch = resp.match(/Task\s+#\d+\s+created\s+successfully:\s*(.+)/);
                const subject = subjectMatch ? subjectMatch[1]!.trim() : "Task #" + numId;
                nextTasks.set(numId, { id: numId, subject, status: "pending", owner: evt.agent });
              }
            }
          }
        }

        // TaskUpdate: update status
        if ((evt.event === "PreToolUse" || evt.event === "PostToolUse") && toolName.includes("TaskUpdate")) {
          const update = parseTaskUpdate(evt.tool_input);
          if (update) {
            if (!changed) { nextTasks = new Map(prevTasks); changed = true; }
            const existing = nextTasks.get(update.taskId);
            if (existing) {
              const merged = { ...existing, status: update.status };
              if (update.blockedBy) merged.blockedBy = [...(existing.blockedBy ?? []), ...update.blockedBy];
              nextTasks.set(update.taskId, merged);
            }
          }
        }

        // TaskList: bootstrap/sync task state from full list
        if (evt.event === "PostToolUse" && toolName.includes("TaskList")) {
          const resp = evt.tool_response;
          if (typeof resp === "string" && resp.trim().length > 0) {
            const parsed = parseTaskListResponse(resp);
            if (parsed.length > 0) {
              if (!changed) { nextTasks = new Map(prevTasks); changed = true; }
              nextTasks.clear();
              for (const task of parsed) {
                nextTasks.set(task.id, task);
              }
            }
          }
        }
      }

      return nextTasks;
    });

    setActivities((prev) => {
      const next = new Map(prev);

      for (const evt of events) {
        const agentName = evt.agent;
        if (!agentName) continue;

        let activity = next.get(agentName) ?? {
          name: agentName, state: "working", task: "", tool: "", role: "", tokens: 0, inputTokens: 0, outputTokens: 0, costUsd: 0, lastEventTime: 0, nodes: [], collapsed: false,
        };
        activity = { ...activity, nodes: [...activity.nodes] };
        activity.lastEventTime = Date.now();

        if (evt.task) activity.task = evt.task;
        if (evt.input_tokens) { activity.tokens += evt.input_tokens; activity.inputTokens += evt.input_tokens; }
        if (evt.output_tokens) { activity.tokens += evt.output_tokens; activity.outputTokens += evt.output_tokens; }

        switch (evt.event) {
          case "UserPromptSubmit":
            activity.state = "working";
            activity.nodes.push({
              id: nextId(), toolName: "UserPromptSubmit",
              args: evt.prompt ? (evt.prompt.length > 120 ? evt.prompt.slice(0, 117) + "..." : evt.prompt) : evt.task ?? "",
              fullInput: evt.prompt ?? evt.tool_input, fullOutput: null, status: "completed",
              startTime: Date.now(), endTime: Date.now(), children: [],
            });
            break;

          case "PreToolUse": {
            activity.state = "working";
            const newNode: ToolNode = {
              id: nextId(), toolName: evt.tool_name ?? "unknown", args: summarizeArgs(evt),
              fullInput: evt.tool_input, fullOutput: null, status: "running",
              startTime: Date.now(), children: [],
            };

            // If tool_name is "Agent", this spawns a subagent -- add as top-level
            // and track as active subagent for nesting child events
            if (evt.tool_name === "Agent") {
              activity.nodes.push(newNode);
              activity.activeSubagentIdx = activity.nodes.length - 1;
            } else if (activity.activeSubagentIdx !== undefined && activity.activeSubagentIdx >= 0) {
              // Nest inside the active subagent node
              const parentNode = activity.nodes[activity.activeSubagentIdx];
              if (parentNode && parentNode.status === "running") {
                const updatedParent = { ...parentNode, children: [...parentNode.children, newNode] };
                activity.nodes[activity.activeSubagentIdx] = updatedParent;
              } else {
                activity.nodes.push(newNode);
                activity.activeSubagentIdx = undefined;
              }
            } else {
              activity.nodes.push(newNode);
            }
            break;
          }

          case "PostToolUse": {
            const matchRunning = (n: ToolNode): boolean => n.toolName === evt.tool_name && n.status === "running";
            const markCompleted = (n: ToolNode): ToolNode => ({ ...n, status: "completed" as const, endTime: Date.now(), fullOutput: evt.tool_response ?? evt.tool_input });

            let found = updateSubagentChild(activity, matchRunning, markCompleted);

            if (evt.tool_name === "Agent") {
              const matchAgent = (n: ToolNode): boolean => n.toolName === "Agent" && n.status === "running";
              found = updateTopLevelNode(activity, matchAgent, markCompleted) || found;
              activity.activeSubagentIdx = undefined;
            }

            if (!found) {
              updateTopLevelNode(activity, matchRunning, markCompleted);
            }
            break;
          }

          case "PostToolUseFailure": {
            const matchRunning = (n: ToolNode): boolean => n.toolName === evt.tool_name && n.status === "running";
            const markFailed = (n: ToolNode): ToolNode => ({ ...n, status: "failed" as const, endTime: Date.now(), error: evt.error ?? "Tool execution failed", fullOutput: evt.tool_response ?? evt.tool_input });

            const found = updateSubagentChild(activity, matchRunning, markFailed);
            if (!found) {
              updateTopLevelNode(activity, matchRunning, markFailed);
            }
            break;
          }

          case "SubagentStart": {
            const subNode: ToolNode = {
              id: nextId(), toolName: `Agent: ${evt.subagent_id ?? "sub"}`,
              args: evt.subagent_type ?? "", fullInput: evt.tool_input, fullOutput: null,
              status: "running", startTime: Date.now(), children: [],
            };

            if (activity.activeSubagentIdx !== undefined && activity.activeSubagentIdx >= 0) {
              const parentNode = activity.nodes[activity.activeSubagentIdx];
              if (parentNode && parentNode.status === "running") {
                activity.nodes[activity.activeSubagentIdx] = { ...parentNode, children: [...parentNode.children, subNode] };
                break;
              }
            }

            activity.nodes.push(subNode);
            activity.activeSubagentIdx = activity.nodes.length - 1;
            break;
          }

          case "SubagentStop": {
            const matchRunningAgent = (n: ToolNode): boolean => n.toolName.startsWith("Agent:") && n.status === "running";
            const markDone = (n: ToolNode): ToolNode => ({ ...n, status: "completed" as const, endTime: Date.now() });

            const found = updateSubagentChild(activity, matchRunningAgent, markDone);
            if (!found) {
              const idx = findLastIdx(activity.nodes, matchRunningAgent);
              if (idx >= 0) {
                activity.nodes[idx] = markDone(activity.nodes[idx]!);
                if (activity.activeSubagentIdx === idx) {
                  activity.activeSubagentIdx = undefined;
                }
              }
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

  const handleResume = useCallback(() => {
    setPaused(false);
    if (pausedBuffer.current.length > 0) {
      eventBuffer.current.push(...pausedBuffer.current);
      pausedBuffer.current = [];
      setPausedCount(0);
    }
  }, []);

  useEffect(() => {
    const id = setInterval(flushEvents, FLUSH_INTERVAL);
    return () => clearInterval(id);
  }, [flushEvents]);

  useEffect(() => {
    const unsub = subscribe("agent.hook", (wsEvent) => {
      const d = wsEvent.data as unknown as HookEvent;
      if (d?.agent) {
        eventBuffer.current.push(d);
        // Capture raw event for drill-down raw stream
        const agentRaw = rawEventsRef.current.get(d.agent) ?? [];
        agentRaw.push({ timestamp: Date.now(), eventType: d.event, raw: d });
        if (agentRaw.length > 500) agentRaw.splice(0, agentRaw.length - 500);
        rawEventsRef.current.set(d.agent, agentRaw);
        setRawEventsVersion((v) => v + 1);
      }
    });
    return unsub;
  }, [subscribe]);

  useEffect(() => {
    const unsub = subscribe("agent.state_changed", (wsEvent) => {
      const d = wsEvent.data as Record<string, unknown>;
      const name = (d.name ?? d.agent) as string;
      const state = d.state as string;
      if (!name || !state) return;

      // When paused, buffer state changes as synthetic hook events
      if (pausedRef.current) {
        pausedBuffer.current.push({ agent: name, event: "state_changed", task: d.task as string | undefined });
        setPausedCount(pausedBuffer.current.length);
        return;
      }

      setEventCount((c) => c + 1);
      setActivities((prev) => {
          const next = new Map(prev);
          const existing = next.get(name);
          if (existing) {
            const updates: Partial<AgentActivity> = { state, lastEventTime: Date.now() };
            if (d.task) updates.task = d.task as string;
            if (d.role) updates.role = d.role as string;
            next.set(name, { ...existing, ...updates });
          }
          return next;
        });
    });
    return unsub;
  }, [subscribe]);

  const sorted = useMemo(() => {
    const filtered = Array.from(activities.values()).filter((a) => {
      if (agentFilter && a.name !== agentFilter) return false;
      if (typeFilter === "tools" && a.nodes.length === 0) return false;
      if (searchFilter) {
        const q = searchFilter.toLowerCase();
        const cardHay = `${a.name} ${a.role} ${a.task} ${a.tool}`.toLowerCase();
        if (cardHay.includes(q)) return true;
        const hasMatchingNode = a.nodes.some((n) => nodeMatchesSearch(n, q));
        if (!hasMatchingNode) return false;
      }
      return true;
    });
    return filtered.sort((a, b) => {
      const order: Record<string, number> = { working: 0, stuck: 1, idle: 2, stopped: 3, error: 4 };
      const oa = order[a.state] ?? 5;
      const ob = order[b.state] ?? 5;
      if (oa !== ob) return oa - ob;
      return a.name.localeCompare(b.name);
    });
  }, [activities, agentFilter, typeFilter, searchFilter]);

  useEffect(() => {
    const container = scrollContainerRef.current;
    if (!container) return;
    const onScroll = () => {
      const isAtTop = container.scrollTop < 50;
      setShowJumpToLatest(!isAtTop);
      if (isAtTop) setNewEventsSinceScroll(0);
    };
    container.addEventListener("scroll", onScroll, { passive: true });
    return () => container.removeEventListener("scroll", onScroll);
  }, []);

  useEffect(() => {
    if (showJumpToLatest) {
      setNewEventsSinceScroll((c) => c + 1);
    }
  }, [eventCount]); // eslint-disable-line react-hooks/exhaustive-deps

  const jumpToLatest = useCallback(() => {
    scrollContainerRef.current?.scrollTo({ top: 0, behavior: "smooth" });
    setNewEventsSinceScroll(0);
  }, []);

  const toggleAgent = useCallback((name: string) => {
    setActivities((prev) => {
      const next = new Map(prev);
      const existing = next.get(name);
      if (existing) next.set(name, { ...existing, collapsed: !existing.collapsed });
      return next;
    });
  }, []);

  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement;
      const isInput = target.tagName === "INPUT" || target.tagName === "TEXTAREA" || target.isContentEditable;

      if (e.key === "Escape") {
        setSearchFilter("");
        setShowShortcuts(false);
        (document.activeElement as HTMLElement)?.blur();
        return;
      }

      if (e.key === "/" && !isInput) {
        e.preventDefault();
        searchInputRef.current?.focus();
        return;
      }

      if (isInput) return;

      if (e.key === "?") {
        e.preventDefault();
        setShowShortcuts((prev) => !prev);
        return;
      }

      if (e.key === "j") {
        e.preventDefault();
        setFocusedCardIdx((prev) => Math.min(prev + 1, sorted.length - 1));
        return;
      }

      if (e.key === "k") {
        e.preventDefault();
        setFocusedCardIdx((prev) => Math.max(prev - 1, 0));
        return;
      }

      if (e.key === "Enter" && focusedCardIdx >= 0 && focusedCardIdx < sorted.length) {
        e.preventDefault();
        const card = sorted[focusedCardIdx];
        if (card) toggleAgent(card.name);
        return;
      }
    };

    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [sorted, focusedCardIdx, toggleAgent]); // eslint-disable-line react-hooks/exhaustive-deps

  const hasFilters = agentFilter || typeFilter !== "all" || searchFilter;

  const sseStatus = connected ? "connected" : reconnecting ? "reconnecting" : "disconnected";
  const sseDotColor = connected ? "bg-emerald-500" : reconnecting ? "bg-yellow-500" : "bg-red-500";
  const sseTooltip = connected ? "SSE connected" : reconnecting ? "Reconnecting..." : "Disconnected";

  // Drill-down view
  const drillDownActivity = drillDownAgent ? activities.get(drillDownAgent) : null;
  const drillDownRawEvents = drillDownAgent ? (rawEventsRef.current.get(drillDownAgent) ?? []) : [];
  // Reference rawEventsVersion to trigger re-render when raw events change
  void rawEventsVersion;

  if (drillDownAgent && drillDownActivity) {
    return (
      <div className="p-6 flex flex-col h-full relative">
        <AgentDrillDown
          activity={drillDownActivity}
          rawEvents={drillDownRawEvents}
          tasks={tasks}
          onBack={() => setDrillDownAgent(null)}
        />
      </div>
    );
  }

  return (
    <div className="p-6 flex flex-col h-full relative">
      {/* Header */}
      <div className="flex items-center gap-3 mb-4">
        <h1 className="text-xl font-bold text-bc-text flex items-center gap-2 shrink-0 pl-10 sm:pl-0">
          Live
          <span className="relative flex h-2.5 w-2.5">
            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-red-400 opacity-75" />
            <span className="relative inline-flex rounded-full h-2.5 w-2.5 bg-red-500" />
          </span>
        </h1>
        <span className="text-sm text-bc-muted hidden sm:inline">Real-time agent activity</span>
        <span className="ml-auto flex items-center gap-3">
          <span className="flex items-center gap-1.5" title={sseTooltip}>
            <span className={`inline-flex h-2 w-2 rounded-full ${sseDotColor}${reconnecting ? " animate-pulse" : ""}`} />
            <span className={`text-[10px] font-mono hidden sm:inline ${reconnecting ? "text-yellow-400" : "text-bc-muted"}`}>{sseStatus}</span>
          </span>
          <span className="text-xs text-bc-muted font-mono tabular-nums">{eventCount} events</span>
          <button
            type="button"
            onClick={() => paused ? handleResume() : setPaused(true)}
            className={`relative inline-flex items-center gap-1 text-xs px-2 py-1 rounded border transition-colors ${paused ? "border-amber-500 bg-amber-500/15 text-amber-400 hover:bg-amber-500/25 animate-pulse" : "border-bc-border hover:border-bc-accent bg-bc-surface text-bc-text"}`}
            title={paused ? `Resume (${pausedCount} buffered)` : "Pause stream"}
          >
            {paused ? "\u25B6" : "\u23F8"}
            {paused && pausedCount > 0 && (
              <span className="text-[10px] font-bold text-amber-300 ml-0.5">({pausedCount})</span>
            )}
          </button>
          <button
            type="button"
            onClick={() => {
              const exportData = {
                exportedAt: new Date().toISOString(),
                eventCount,
                activities: Object.fromEntries(
                  Array.from(activities.entries()).map(([name, a]) => [name, {
                    name: a.name, state: a.state, role: a.role, task: a.task,
                    tokens: a.tokens, inputTokens: a.inputTokens, outputTokens: a.outputTokens,
                    costUsd: a.costUsd, lastEventTime: a.lastEventTime,
                    nodes: a.nodes.map((n) => ({
                      id: n.id, toolName: n.toolName, args: n.args,
                      status: n.status, startTime: n.startTime, endTime: n.endTime,
                      error: n.error,
                    })),
                  }]),
                ),
                tasks: Object.fromEntries(Array.from(tasks.entries())),
              };
              const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: "application/json" });
              const url = URL.createObjectURL(blob);
              const a = document.createElement("a");
              a.href = url;
              a.download = `bc-events-${Date.now()}.json`;
              a.click();
              URL.revokeObjectURL(url);
            }}
            className="text-xs px-2 py-1 rounded border border-bc-border hover:border-bc-accent bg-bc-surface text-bc-muted hover:text-bc-text transition-colors"
            title="Export event feed as JSON"
          >
            Export
          </button>
          <button
            type="button"
            onClick={() => setShowShortcuts((prev) => !prev)}
            className="text-xs px-1.5 py-1 rounded border border-bc-border hover:border-bc-accent bg-bc-surface text-bc-muted hover:text-bc-text transition-colors"
            title="Keyboard shortcuts (?)"
          >
            ?
          </button>
        </span>
      </div>

      {/* Keyboard Shortcuts Overlay */}
      {showShortcuts && (
        <div className="absolute top-16 right-6 z-50 bg-bc-surface border border-bc-border rounded-lg shadow-lg p-4 w-64">
          <div className="flex items-center justify-between mb-3">
            <span className="text-sm font-semibold text-bc-text">Keyboard Shortcuts</span>
            <button
              type="button"
              onClick={() => setShowShortcuts(false)}
              className="text-bc-muted hover:text-bc-text text-sm"
            >
              &times;
            </button>
          </div>
          <div className="space-y-1.5 text-xs">
            {[
              ["/", "Focus search"],
              ["Esc", "Clear search / close"],
              ["j", "Next agent card"],
              ["k", "Previous agent card"],
              ["Enter", "Expand/collapse focused card"],
              ["?", "Toggle this help"],
            ].map(([key, desc]) => (
              <div key={key} className="flex items-center gap-2">
                <kbd className="inline-flex items-center justify-center min-w-[24px] h-5 px-1.5 rounded bg-bc-bg border border-bc-border text-bc-text font-mono text-[11px]">
                  {key}
                </kbd>
                <span className="text-bc-muted">{desc}</span>
              </div>
            ))}
          </div>
        </div>
      )}

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
          ref={searchInputRef}
          type="text"
          value={searchFilter}
          onChange={(e) => setSearchFilter(e.target.value)}
          placeholder="Search events... (/ to focus)"
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
      <div ref={scrollContainerRef} className="flex-1 overflow-y-auto min-h-0 space-y-3 relative">
        {sorted.length === 0 ? (
          <EmptyState
            icon=">"
            title="No activity yet"
            description="Events will stream here in real-time as agents work."
          />
        ) : (
          sorted.map((activity, idx) => (
            <div
              key={activity.name}
              className={focusedCardIdx === idx ? "ring-2 ring-bc-accent rounded-lg" : ""}
            >
              <AgentCard
                activity={activity}
                onToggle={() => toggleAgent(activity.name)}
                onDrillDown={() => setDrillDownAgent(activity.name)}
                isFilterActive={agentFilter === activity.name}
                searchTerm={searchFilter}
                typeFilter={typeFilter}
                isPaused={paused}
              />
            </div>
          ))
        )}
      </div>

      {/* Jump to Latest Button */}
      {showJumpToLatest && (
        <button
          type="button"
          onClick={jumpToLatest}
          className="absolute bottom-8 right-8 z-20 inline-flex items-center gap-2 px-3 py-2 rounded-lg border border-bc-border bg-bc-surface text-bc-text text-sm shadow-lg hover:border-bc-accent hover:bg-bc-surface-hover transition-colors"
        >
          <span>&darr;</span>
          Jump to latest
          {newEventsSinceScroll > 0 && (
            <span className="inline-flex items-center justify-center min-w-[20px] h-5 px-1.5 text-[11px] font-bold text-white bg-bc-accent rounded-full leading-none">
              {newEventsSinceScroll}
            </span>
          )}
        </button>
      )}
    </div>
  );
}
