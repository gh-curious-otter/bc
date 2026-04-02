import { useCallback, useEffect, useMemo, useRef, useState, memo } from "react";
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
const AGGREGATION_WINDOW_MS = 5_000;

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
    // mcp__<server>__<function> or mcp__plugin_<server>_<provider>__<function>
    let server = parts[1] ?? "mcp";
    const func = parts[parts.length - 1] ?? "call";
    // Normalize plugin_ prefix: mcp__plugin_github_github__search_code -> github
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

/** Emoji icon for non-MCP tools */
function toolIcon(name: string): string {
  if (name === "Bash" || name === "BashOutput") return "\u2328\uFE0F";
  if (name === "Read") return "\uD83D\uDCD6";
  if (name === "Write" || name === "Edit") return "\u270F\uFE0F";
  if (name === "Glob" || name === "Grep") return "\uD83D\uDD0D";
  if (name === "Agent") return "\uD83E\uDD16";
  if (name === "WebFetch" || name === "WebSearch") return "\uD83C\uDF10";
  if (name.startsWith("Task") || name === "TaskCreate" || name === "TaskUpdate" || name === "TaskList" || name === "TaskGet") return "\u2705";
  if (name === "NotebookEdit") return "\uD83D\uDCD3";
  if (name === "LSP" || name === "ToolSearch") return "\u2699\uFE0F";
  if (name === "AskUserQuestion") return "\u2753";
  if (name === "Skill") return "\uD83C\uDFAF";
  return "\u2699\uFE0F";
}

/** MCP server icon */
function mcpServerIcon(server: string): string {
  if (server === "playwright" || server === "playwright2") return "\uD83C\uDFAD";
  if (server === "github") return "\uD83D\uDC19";
  if (server === "bc") return "\u26A1";
  return "\uD83D\uDD0C";
}

/** MCP server badge colors (Tailwind classes) */
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

/** Extract rich metadata from tool_input based on tool type */
function extractToolMetadata(toolName: string, input: unknown): string {
  if (!input || typeof input !== "object") return "";
  const obj = input as Record<string, unknown>;
  const trunc = (s: string, max = 80): string => s.length > max ? s.slice(0, max - 3) + "..." : s;

  if (toolName === "Bash" || toolName === "bash") {
    if (typeof obj.command === "string") return redactSecrets(trunc(obj.command));
  }
  if (toolName === "Read") {
    if (typeof obj.file_path === "string") return trunc(obj.file_path);
  }
  if (toolName === "Write") {
    if (typeof obj.file_path === "string") return trunc(obj.file_path);
  }
  if (toolName === "Edit") {
    let s = typeof obj.file_path === "string" ? obj.file_path : "";
    if (typeof obj.old_string === "string") {
      s += " " + trunc(obj.old_string, 40);
    }
    return trunc(s);
  }
  if (toolName === "Grep") {
    if (typeof obj.pattern === "string") return trunc(obj.pattern);
  }
  if (toolName === "Glob") {
    if (typeof obj.pattern === "string") return trunc(obj.pattern);
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
  // MCP tools: show first 2-3 key param values
  if (toolName.startsWith("mcp__")) {
    const vals = Object.entries(obj).slice(0, 3).map(([, v]) => {
      if (typeof v === "string") return trunc(v, 30);
      if (typeof v === "number" || typeof v === "boolean") return String(v);
      return "";
    }).filter(Boolean);
    return redactSecrets(vals.join(" "));
  }
  // Fallback: JSON summary
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

/** Duration color class based on elapsed milliseconds */
function durationColorClass(start: number, end?: number): string {
  const ms = (end ?? Date.now()) - start;
  if (ms < 1000) return "text-emerald-400";
  if (ms < 5000) return "text-yellow-400";
  if (ms < 30000) return "text-orange-400";
  return "text-red-400";
}

/** Format a relative time like "2s ago", "3m ago", "1h ago" */
function relativeTime(ts: number): string {
  const diff = Date.now() - ts;
  if (diff < 1000) return "just now";
  if (diff < 60_000) return `${Math.floor(diff / 1000)}s ago`;
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`;
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h ago`;
  return `${Math.floor(diff / 86_400_000)}d ago`;
}

/** Estimate cost from token counts using approximate Claude pricing */
const INPUT_COST_PER_TOKEN = 3 / 1_000_000;   // ~$3 per 1M input tokens
const OUTPUT_COST_PER_TOKEN = 15 / 1_000_000;  // ~$15 per 1M output tokens

function estimateCost(activity: AgentActivity): number {
  // If API returned a real cost, use that
  if (activity.costUsd > 0) return activity.costUsd;
  // Otherwise estimate from token counts
  if (activity.inputTokens > 0 || activity.outputTokens > 0) {
    return activity.inputTokens * INPUT_COST_PER_TOKEN + activity.outputTokens * OUTPUT_COST_PER_TOKEN;
  }
  return 0;
}

/** Format idle duration like "Idle 5m" or "Idle 2h" */
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

/* ── Aggregation ──────────────────────────────────────────────────── */

/** Events that should never be aggregated */
const NEVER_AGGREGATE_EVENTS = new Set([
  "SubagentStart", "SubagentStop", "Agent",
  "PermissionRequest", "Elicitation",
  "UserPromptSubmit", "SessionStart", "SessionEnd",
  "Stop", "TaskCompleted",
]);

function shouldNeverAggregate(node: ToolNode): boolean {
  if (node.status === "failed") return true;
  if (NEVER_AGGREGATE_EVENTS.has(node.toolName)) return true;
  // State change events (toolName starts with known prefixes)
  if (node.toolName.startsWith("Agent:")) return true;
  return false;
}

/**
 * Scan a list of ToolNodes and aggregate consecutive same-tool nodes
 * that fall within the AGGREGATION_WINDOW_MS time window.
 * Returns a mixed list of ToolNode and AggregatedNode.
 */
function aggregateNodes(nodes: ToolNode[]): DisplayNode[] {
  if (nodes.length === 0) return [];

  const result: DisplayNode[] = [];
  let i = 0;

  while (i < nodes.length) {
    const current = nodes[i];
    if (!current) { i++; continue; }

    // If this node should never be aggregated, emit it directly
    if (shouldNeverAggregate(current) || current.status === "running") {
      result.push(current);
      i++;
      continue;
    }

    // Look ahead for consecutive same-tool nodes within the time window
    const group: ToolNode[] = [current];
    let j = i + 1;
    while (j < nodes.length) {
      const next = nodes[j];
      if (!next) break;
      if (next.toolName !== current.toolName) break;
      if (shouldNeverAggregate(next) || next.status === "running") break;
      // Check time window: next node's start must be within 5s of previous node's start
      const prev = group[group.length - 1];
      if (!prev) break;
      if (Math.abs(next.startTime - prev.startTime) > AGGREGATION_WINDOW_MS) break;
      group.push(next);
      j++;
    }

    if (group.length >= 2) {
      // Create an aggregated node
      let totalDuration = 0;
      const totalTokens = 0;
      let successCount = 0;
      let failCount = 0;
      let minStart = Infinity;
      let maxEnd = 0;

      for (const n of group) {
        const dur = n.endTime ? n.endTime - n.startTime : 0;
        totalDuration += dur;
        if (n.status === "completed") successCount++;
        if (n.status === "failed") failCount++;
        if (n.startTime < minStart) minStart = n.startTime;
        if (n.endTime && n.endTime > maxEnd) maxEnd = n.endTime;
      }

      result.push({
        type: "aggregate",
        id: `agg-${group[0]!.id}`,
        toolName: current.toolName,
        count: group.length,
        children: group,
        totalDuration,
        totalTokens,
        successCount,
        failCount,
        startTime: minStart,
        endTime: maxEnd || Date.now(),
      });
      i = j;
    } else {
      result.push(current);
      i++;
    }
  }

  return result;
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

/* ── Tool Name Display ─────────────────────────────────────────────── */

function ToolNameDisplay({ toolName }: { toolName: string }) {
  const parsed = parseToolName(toolName);
  if (parsed.type === "mcp" && parsed.mcpServer && parsed.mcpFunction) {
    return <McpBadge server={parsed.mcpServer} func={parsed.mcpFunction} />;
  }
  // Non-MCP: emoji icon + name
  return (
    <span className="inline-flex items-center gap-1">
      <span className="text-[12px]" aria-hidden="true">{toolIcon(toolName)}</span>
      <span className="font-mono text-[13px] text-bc-text font-medium">{parsed.display}</span>
    </span>
  );
}

/* ── Tool Node Row ─────────────────────────────────────────────────── */

function ToolNodeRow({ node, depth = 0, isSubagentChild = false }: { node: ToolNode; depth?: number; isSubagentChild?: boolean }) {
  const [expanded, setExpanded] = useState(false);
  const indent = depth * 20;
  const hasDetails = !!(node.fullInput || node.fullOutput || node.children.length > 0);
  const isSubagentSpawn = node.toolName === "Agent" || node.toolName.startsWith("Agent:");

  // Subagent tree: special rendering
  if (isSubagentSpawn) {
    return <SubagentRow node={node} depth={depth} />;
  }

  const inputJson = node.fullInput ? JSON.stringify(redactValue(node.fullInput), null, 2) : "";
  const outputJson = node.fullOutput ? JSON.stringify(redactValue(node.fullOutput), null, 2) : "";

  return (
    <>
      <button
        type="button"
        className={`group flex items-start gap-2 py-0.5 px-3 w-full text-left hover:bg-bc-surface-hover cursor-pointer transition-colors focus-visible:ring-2 focus-visible:ring-bc-accent ${node.status === "failed" ? "bg-red-950/10" : ""}`}
        style={{ paddingLeft: `${indent + 12}px` }}
        onClick={() => setExpanded(!expanded)}
        aria-label={`${expanded ? "Collapse" : "Expand"} tool ${node.toolName}`}
      >
        <span className="text-bc-muted text-xs select-none mt-[3px] shrink-0">
          {depth > 0 ? "\u251C\u2500" : ""}
        </span>
        {/* Expand/collapse chevron indicator */}
        <span className="text-bc-muted/50 text-[10px] select-none mt-[3px] shrink-0 w-3 text-center group-hover:text-bc-muted">
          {hasDetails ? (expanded ? "\u25BC" : "\u25B6") : "\u00B7"}
        </span>
        <ToolDot status={node.status} />
        <ToolNameDisplay toolName={node.toolName} />
        {node.args && (
          <span className="text-[12px] text-bc-muted truncate max-w-[400px] font-mono">
            {redactSecrets(node.args)}
          </span>
        )}
        <span className="ml-auto flex items-center gap-2 shrink-0">
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
        <ToolNodeRow key={child.id} node={child} depth={depth + 1} isSubagentChild={isSubagentChild} />
      ))}
    </>
  );
}

/* ── Subagent Tree Row ─────────────────────────────────────────────── */

function SubagentRow({ node, depth = 0 }: { node: ToolNode; depth?: number }) {
  const [expanded, setExpanded] = useState(true);
  const indent = depth * 20;
  const duration = node.endTime ? elapsed(node.startTime, node.endTime) : undefined;

  return (
    <>
      <button
        type="button"
        className="group flex items-start gap-2 py-1 px-3 w-full text-left hover:bg-bc-surface-hover cursor-pointer transition-colors focus-visible:ring-2 focus-visible:ring-bc-accent bg-blue-950/20"
        style={{ paddingLeft: `${indent + 12}px` }}
        onClick={() => setExpanded(!expanded)}
        aria-label={`${expanded ? "Collapse" : "Expand"} subagent ${node.toolName}`}
      >
        <span className="text-bc-muted/50 text-[10px] select-none mt-[3px] shrink-0 w-3 text-center group-hover:text-bc-muted">
          {expanded ? "\u25BC" : "\u25B6"}
        </span>
        <ToolDot status={node.status} />
        <span className="text-[13px]" aria-hidden="true">{"\uD83E\uDD16"}</span>
        <span className="font-mono text-[13px] text-bc-text font-semibold">{node.toolName}</span>
        {node.args && (
          <span className="text-[12px] text-bc-muted truncate max-w-[300px] font-mono italic">
            {node.args}
          </span>
        )}
        <span className="ml-auto flex items-center gap-2 shrink-0">
          <RelativeTimestamp ts={node.startTime} />
          {node.status === "running" ? (
            <span className="text-[11px] text-blue-400 font-mono tabular-nums">
              <ElapsedTimer start={node.startTime} />
            </span>
          ) : duration ? (
            <span className="text-[11px] text-bc-muted font-mono tabular-nums">{duration}</span>
          ) : null}
          {node.status === "completed" && (
            <span className="text-[10px] text-bc-success font-mono">done</span>
          )}
        </span>
      </button>

      {expanded && node.children.length > 0 && (
        <div className="border-l-2 border-blue-500 pl-3 ml-6">
          {node.children.map((child) => (
            <ToolNodeRow key={child.id} node={child} depth={depth + 1} isSubagentChild />
          ))}
        </div>
      )}
    </>
  );
}

/* ── Aggregated Node Row ───────────────────────────────────────────── */

function AggregatedNodeRow({ node }: { node: AggregatedNode }) {
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
        <span className="ml-auto flex items-center gap-3">
          {node.totalDuration > 0 && (
            <span className="text-[11px] text-bc-muted font-mono tabular-nums">
              {elapsed(0, node.totalDuration)}
            </span>
          )}
          {node.totalTokens > 0 && (
            <span className="text-[11px] text-bc-muted font-mono tabular-nums">
              {node.totalTokens.toLocaleString()} tok
            </span>
          )}
          {node.failCount > 0 && (
            <span className="text-[11px] text-bc-error font-mono">
              {node.failCount} failed
            </span>
          )}
          <span className="text-[11px] text-bc-muted font-mono tabular-nums">
            {node.successCount}/{node.count} ok
          </span>
        </span>
      </button>

      {expanded && (
        <div className="border-l-2 border-bc-border/40 ml-6">
          {node.children.map((child) => (
            <ToolNodeRow key={child.id} node={child} depth={1} />
          ))}
        </div>
      )}
    </>
  );
}

/* ── Display Node Row (dispatches to ToolNodeRow or AggregatedNodeRow) */

function DisplayNodeRow({ node }: { node: DisplayNode }) {
  if (isAggregatedNode(node)) {
    return <AggregatedNodeRow node={node} />;
  }
  return <ToolNodeRow node={node} />;
}

/* ── Agent Activity Card ───────────────────────────────────────────── */

const AgentCard = memo(function AgentCard({
  activity,
  onToggle,
  onClickFilter,
  isFilterActive,
  searchTerm,
  typeFilter,
}: {
  activity: AgentActivity;
  onToggle: () => void;
  onClickFilter: () => void;
  isFilterActive: boolean;
  searchTerm: string;
  typeFilter: FilterType;
}) {
  // Filter nodes based on search term (individual node filtering)
  const visibleNodes = searchTerm
    ? activity.nodes.filter((n) => nodeMatchesSearch(n, searchTerm.toLowerCase()))
    : activity.nodes;

  const runningCount = visibleNodes.filter((n) => n.status === "running").length;
  const errorCount = activity.nodes.filter((n) => n.status === "failed").length;
  const displayNodes = aggregateNodes(visibleNodes);
  const matchCount = searchTerm ? visibleNodes.length : 0;
  const showToolNodes = typeFilter !== "state";

  return (
    <div className={`rounded-lg border bg-bc-surface overflow-hidden transition-colors ${isFilterActive ? "border-bc-accent ring-1 ring-bc-accent/30" : "border-bc-border"}`}>
      <div className="flex items-center">
        {/* Collapse toggle */}
        <button
          type="button"
          onClick={onToggle}
          className="flex items-center gap-3 px-4 py-3 hover:bg-bc-surface-hover transition-colors text-left focus-visible:ring-2 focus-visible:ring-bc-accent shrink-0"
          aria-label={`${activity.collapsed ? "Expand" : "Collapse"} ${activity.name}`}
        >
          <svg
            width="12" height="12" viewBox="0 0 12 12" fill="none"
            stroke="currentColor" strokeWidth="2"
            className={`text-bc-muted transition-transform ${activity.collapsed ? "" : "rotate-90"}`}
          >
            <path d="M4 2l4 4-4 4" />
          </svg>
        </button>

        {/* Click-to-filter agent header */}
        <button
          type="button"
          onClick={onClickFilter}
          className="flex-1 flex items-center gap-3 py-3 pr-4 hover:bg-bc-surface-hover transition-colors text-left focus-visible:ring-2 focus-visible:ring-bc-accent min-w-0"
          title={isFilterActive ? "Click to clear agent filter" : `Click to filter by ${activity.name}`}
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
            {(() => { const cost = estimateCost(activity); return cost > 0 ? (
              <span className="text-[11px] text-bc-success font-mono tabular-nums" title={activity.costUsd > 0 ? "From API" : "Estimated from tokens"}>
                ${cost.toFixed(2)}
              </span>
            ) : null; })()}
            {activity.tokens > 0 && (
              <span className="text-[11px] text-bc-muted font-mono tabular-nums">
                {activity.tokens.toLocaleString()} tok
              </span>
            )}
          </span>
        </button>
      </div>

      {!activity.collapsed && showToolNodes && displayNodes.length > 0 && (
        <div className="border-t border-bc-border/60 py-1">
          {displayNodes.map((node) => (
            <DisplayNodeRow key={isAggregatedNode(node) ? node.id : node.id} node={node} />
          ))}
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
          {activity.task && <span className="ml-2">— {activity.task}</span>}
          {activity.tokens > 0 && (
            <span className="ml-2 font-mono tabular-nums">{activity.tokens.toLocaleString()} tokens</span>
          )}
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
  const [paused, setPaused] = useState(false);
  const pausedBuffer = useRef<HookEvent[]>([]);
  const [pausedCount, setPausedCount] = useState(0);
  const [showJumpToLatest, setShowJumpToLatest] = useState(false);
  const [newEventsSinceScroll, setNewEventsSinceScroll] = useState(0);
  const [showShortcuts, setShowShortcuts] = useState(false);
  const [focusedCardIdx, setFocusedCardIdx] = useState(-1);
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const searchInputRef = useRef<HTMLInputElement>(null);
  const eventBuffer = useRef<HookEvent[]>([]);
  const { connected, reconnecting, subscribe } = useWebSocket();

  // Seed from agents API + initial logs
  useEffect(() => {
    api.listAgents().then((agentList) => {
      setAgents(agentList);
      setActivities((prev) => {
        const next = new Map(prev);
        for (const a of agentList) {
          if (!next.has(a.name)) {
            const updatedAt = a.updated_at ? new Date(a.updated_at).getTime() : 0;
            next.set(a.name, {
              name: a.name,
              state: a.state,
              task: a.task ?? "",
              tool: a.tool,
              role: a.role ?? "",
              tokens: a.total_tokens ?? 0,
              inputTokens: 0,
              outputTokens: 0,
              costUsd: a.cost_usd ?? 0,
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

  // Process buffered hook events (same pattern as Dashboard)
  const flushEvents = useCallback(() => {
    const events = eventBuffer.current.splice(0);
    if (events.length === 0) return;

    // When paused, buffer events instead of processing them
    if (paused) {
      pausedBuffer.current.push(...events);
      setPausedCount(pausedBuffer.current.length);
      return;
    }

    setEventCount((c) => c + events.length);
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
  }, [paused]);

  // Resume: flush paused buffer back into event buffer
  const handleResume = useCallback(() => {
    setPaused(false);
    if (pausedBuffer.current.length > 0) {
      eventBuffer.current.push(...pausedBuffer.current);
      pausedBuffer.current = [];
      setPausedCount(0);
    }
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
      const name = (d.name ?? d.agent) as string;
      const state = d.state as string;
      if (name && state) {
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
      }
    });
    return unsub;
  }, [subscribe]);

  // Filter and sort activities
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

  // Scroll tracking for jump-to-latest
  useEffect(() => {
    const container = scrollContainerRef.current;
    if (!container) return;
    const onScroll = () => {
      const isAtTop = container.scrollTop < 100;
      setShowJumpToLatest(!isAtTop);
      if (isAtTop) setNewEventsSinceScroll(0);
    };
    container.addEventListener("scroll", onScroll, { passive: true });
    return () => container.removeEventListener("scroll", onScroll);
  }, []);

  // Track new events when scrolled away
  useEffect(() => {
    if (showJumpToLatest) {
      setNewEventsSinceScroll((c) => c + 1);
    }
  }, [eventCount]); // eslint-disable-line react-hooks/exhaustive-deps

  const jumpToLatest = useCallback(() => {
    scrollContainerRef.current?.scrollTo({ top: 0, behavior: "smooth" });
    setNewEventsSinceScroll(0);
  }, []);

  // Toggle collapse
  const toggleAgent = useCallback((name: string) => {
    setActivities((prev) => {
      const next = new Map(prev);
      const existing = next.get(name);
      if (existing) next.set(name, { ...existing, collapsed: !existing.collapsed });
      return next;
    });
  }, []);

  // Click-to-filter on agent card header
  const toggleCardFilter = useCallback((name: string) => {
    setAgentFilter((prev) => (prev === name ? "" : name));
  }, []);

  // Keyboard shortcuts
  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement;
      const isInput = target.tagName === "INPUT" || target.tagName === "TEXTAREA" || target.isContentEditable;

      // Escape always works: clear search and blur
      if (e.key === "Escape") {
        setSearchFilter("");
        setShowShortcuts(false);
        (document.activeElement as HTMLElement)?.blur();
        return;
      }

      // / to focus search (works even from non-input)
      if (e.key === "/" && !isInput) {
        e.preventDefault();
        searchInputRef.current?.focus();
        return;
      }

      // Don't handle other shortcuts when in input
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

  // SSE connection indicator state
  const sseStatus = connected ? "connected" : reconnecting ? "reconnecting" : "disconnected";
  const sseDotColor = connected ? "bg-emerald-500" : reconnecting ? "bg-yellow-500" : "bg-red-500";
  const sseTooltip = connected ? "SSE connected" : reconnecting ? "Reconnecting..." : "Disconnected";

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
          {/* SSE Connection Indicator */}
          <span className="flex items-center gap-1.5" title={sseTooltip}>
            <span className={`inline-flex h-2 w-2 rounded-full ${sseDotColor}`} />
            <span className="text-[10px] text-bc-muted font-mono hidden sm:inline">{sseStatus}</span>
          </span>
          {/* Event count */}
          <span className="text-xs text-bc-muted font-mono tabular-nums">{eventCount} events</span>
          {/* Pause/Resume Toggle */}
          <button
            type="button"
            onClick={() => paused ? handleResume() : setPaused(true)}
            className="relative inline-flex items-center gap-1 text-xs px-2 py-1 rounded border border-bc-border hover:border-bc-accent bg-bc-surface text-bc-text transition-colors"
            title={paused ? `Resume (${pausedCount} buffered)` : "Pause stream"}
          >
            {paused ? "\u25B6" : "\u23F8"}
            {paused && pausedCount > 0 && (
              <span className="absolute -top-2 -right-2 inline-flex items-center justify-center min-w-[18px] h-[18px] px-1 text-[10px] font-bold text-white bg-bc-accent rounded-full leading-none">
                {pausedCount}
              </span>
            )}
          </button>
          {/* Shortcut help */}
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
                onClickFilter={() => toggleCardFilter(activity.name)}
                isFilterActive={agentFilter === activity.name}
                searchTerm={searchFilter}
                typeFilter={typeFilter}
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
