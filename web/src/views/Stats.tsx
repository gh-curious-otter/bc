import { useCallback, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  AreaChart, Area, BarChart, Bar, Cell,
  XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
} from "recharts";
import { api } from "../api/client";
import type {
  SystemStats, StatsSummary, CostSummary, ModelCostSummary, AgentCostSummary,
  AgentMetricTS, TokenMetricTS,
} from "../api/client";
import { usePolling } from "../hooks/usePolling";
import { LoadingSkeleton } from "../components/LoadingSkeleton";
import { EmptyState } from "../components/EmptyState";

// ── Model Pricing ───────────────────────────────────────────────────────────────

// Model pricing (per 1M tokens, USD)
const MODEL_PRICING: Record<string, { input: number; output: number }> = {
  "claude-opus-4-6": { input: 15, output: 75 },
  "claude-sonnet-4-6": { input: 3, output: 15 },
  "claude-haiku-4-5-20251001": { input: 0.80, output: 4 },
  "claude-3-5-sonnet-20241022": { input: 3, output: 15 },
  "claude-3-5-haiku-20241022": { input: 0.80, output: 4 },
};

export function calculateCost(model: string, inputTokens: number, outputTokens: number): number {
  const pricing = MODEL_PRICING[model] ?? { input: 3, output: 15 };
  return (inputTokens / 1_000_000) * pricing.input + (outputTokens / 1_000_000) * pricing.output;
}

// ── Constants ───────────────────────────────────────────────────────────────────

const COLORS = ["#FF6B35", "#3B82F6", "#10B981", "#A855F7", "#F59E0B", "#EC4899", "#06B6D4", "#84CC16"];
const RANGES = [
  { label: "1h", seconds: 3600 },
  { label: "6h", seconds: 21600 },
  { label: "12h", seconds: 43200 },
  { label: "24h", seconds: 86400 },
  { label: "7d", seconds: 604800 },
] as const;

const INFRA = ["bc-db", "bc-daemon", "bc-playwright"];
const isInfra = (n: string) => INFRA.some(p => n === p || n.startsWith(p + "-")) || n.length <= 3;

const TT: React.CSSProperties = {
  backgroundColor: "var(--color-bc-surface)", border: "1px solid var(--color-bc-border)",
  borderRadius: "6px", color: "var(--color-bc-text)", fontSize: "12px",
};
const AX = { axisLine: false as const, tickLine: false as const };
const TICK_STYLE = { fill: "var(--color-bc-muted)", fontSize: 10 };

// ── Helpers ─────────────────────────────────────────────────────────────────────

const fmtTime = (iso: string) => {
  try { return new Date(iso).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }); }
  catch { return iso; }
};
const fmtBytes = (b: number) => {
  if (!b) return "0 B";
  const u = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(b) / Math.log(1024));
  return `${(b / Math.pow(1024, i)).toFixed(1)} ${u[i]}`;
};
const fmtUptime = (s: number) => {
  const d = Math.floor(s / 86400), h = Math.floor((s % 86400) / 3600), m = Math.floor((s % 3600) / 60);
  return [d && `${d}d`, h && `${h}h`, `${m}m`].filter(Boolean).join(" ");
};
const fmtTokens = (n: number) => {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
  return String(n);
};
const trunc = (s: string, n: number) => s.length > n ? s.slice(0, n) + "\u2026" : s;

function fromParam(seconds: number): string {
  return new Date(Date.now() - seconds * 1000).toISOString();
}

// ── Primitives ──────────────────────────────────────────────────────────────────

function Panel({ title, children, className }: { title: string; children: React.ReactNode; className?: string }) {
  return (
    <div className={`rounded border border-bc-border bg-bc-surface overflow-hidden ${className ?? ""}`}>
      <div className="flex items-center justify-between px-3 py-1.5 border-b border-bc-border bg-bc-bg/50">
        <span className="text-[11px] font-medium text-bc-muted uppercase tracking-wider">{title}</span>
      </div>
      <div className="p-3">{children}</div>
    </div>
  );
}

function Empty({ msg = "No data yet" }: { msg?: string }) {
  return <div className="flex items-center justify-center h-[200px] text-sm text-bc-muted">{msg}</div>;
}

// ── Data ────────────────────────────────────────────────────────────────────────

interface StatsData {
  system: SystemStats | null;
  summary: StatsSummary | null;
  costSummary: CostSummary | null;
  costByModel: ModelCostSummary[];
  costByAgent: AgentCostSummary[];
  agentCpu: AgentMetricTS[];
  agentMem: AgentMetricTS[];
  agentNet: AgentMetricTS[];
  agentDisk: AgentMetricTS[];
  tokenMetrics: TokenMetricTS[];
}

type SortKey = "name" | "role" | "state" | "cpu" | "mem" | "tokens" | "cost";

// ── Main ────────────────────────────────────────────────────────────────────────

export function Stats() {
  const navigate = useNavigate();
  const [range, setRange] = useState(0); // index into RANGES
  const [sortKey, setSortKey] = useState<SortKey>("cost");
  const [sortAsc, setSortAsc] = useState(false);

  const from = useMemo(() => fromParam(RANGES[range]?.seconds ?? 3600), [range]);

  const fetcher = useCallback(async (): Promise<StatsData> => {
    const p = { from };
    const [r0, r1, r2, r3, r4, r5, r6, r7, r8, r9] = await Promise.allSettled([
      api.getStatsSystem(),
      api.getStatsSummary(),
      api.getCostSummary(),
      api.getCostByModel(),
      api.getCostByAgent(),
      api.getAgentStats("cpu", p),
      api.getAgentStats("mem", p),
      api.getAgentStats("net", p),
      api.getAgentStats("disk", p),
      api.getAgentTokenStats(p),
    ]);
    return {
      system: r0.status === "fulfilled" ? r0.value : null,
      summary: r1.status === "fulfilled" ? r1.value : null,
      costSummary: r2.status === "fulfilled" ? r2.value : null,
      costByModel: r3.status === "fulfilled" ? r3.value : [],
      costByAgent: r4.status === "fulfilled" ? (r4.value ?? []) : [],
      agentCpu: r5.status === "fulfilled" ? (r5.value ?? []) : [],
      agentMem: r6.status === "fulfilled" ? (r6.value ?? []) : [],
      agentNet: r7.status === "fulfilled" ? (r7.value ?? []) : [],
      agentDisk: r8.status === "fulfilled" ? (r8.value ?? []) : [],
      tokenMetrics: r9.status === "fulfilled" ? (r9.value ?? []) : [],
    };
  }, [from]);

  const { data, loading, error, refresh, timedOut } = usePolling(fetcher, 10000);

  // ── Derived data ────────────────────────────────────────────────────────────

  const cpuChart = useMemo(() => pivotAgentMetric(data?.agentCpu ?? [], "cpu_percent"), [data?.agentCpu]);
  const memChart = useMemo(() => pivotAgentMetric(data?.agentMem ?? [], "mem_mb"), [data?.agentMem]);
  const netChart = useMemo(() => pivotNetOrDisk(data?.agentNet ?? [], "net"), [data?.agentNet]);
  const diskChart = useMemo(() => pivotNetOrDisk(data?.agentDisk ?? [], "disk"), [data?.agentDisk]);
  const tokenChart = useMemo(() => pivotTokens(data?.tokenMetrics ?? []), [data?.tokenMetrics]);

  const costBarData = useMemo(() => {
    return [...(data?.costByModel ?? [])]
      .map(m => ({ name: trunc(m.model || "unknown", 24), cost: parseFloat(calculateCost(m.model, m.input_tokens, m.output_tokens).toFixed(4)) }))
      .filter(d => d.cost > 0)
      .sort((a, b) => b.cost - a.cost)
      .slice(0, 8);
  }, [data?.costByModel]);

  const agentTable = useMemo(() => buildAgentTable(data, sortKey, sortAsc), [data, sortKey, sortAsc]);

  const totalTokens = data?.costSummary?.total_tokens ?? 0;
  const inputTokens = data?.costSummary?.input_tokens ?? 0;
  const outputTokens = data?.costSummary?.output_tokens ?? 0;
  const totalCost = data?.costSummary?.total_cost_usd ?? 0;
  const uptime = data?.system?.uptime_seconds ?? data?.summary?.uptime_seconds ?? 0;
  const running = data?.summary?.agents_running ?? 0;
  const total = data?.summary?.agents_total ?? 0;

  // ── Render ──────────────────────────────────────────────────────────────────

  if (loading && !data) return <div className="p-6 space-y-6"><LoadingSkeleton variant="cards" rows={4} /></div>;
  if (timedOut && !data) return <div className="p-6"><EmptyState icon="!" title="Stats timed out" actionLabel="Retry" onAction={refresh} /></div>;
  if (error && !data) return <div className="p-6"><EmptyState icon="!" title="Failed to load stats" description={error} actionLabel="Retry" onAction={refresh} /></div>;
  if (!data) return null;

  function handleSort(key: SortKey) {
    if (sortKey === key) setSortAsc(!sortAsc);
    else { setSortKey(key); setSortAsc(false); }
  }

  return (
    <div className="p-6 space-y-4">
      {/* Header + time range */}
      <div className="flex items-center justify-between">
        <h1 className="text-lg font-semibold">System Metrics</h1>
        <div className="flex gap-1">
          {RANGES.map((r, i) => (
            <button key={r.label} type="button" onClick={() => setRange(i)}
              className={`px-2.5 py-1 text-xs rounded border transition-colors ${
                i === range
                  ? "border-bc-accent bg-bc-accent/10 text-bc-accent"
                  : "border-bc-border text-bc-muted hover:text-bc-text hover:border-bc-muted"
              }`}
            >{r.label}</button>
          ))}
        </div>
      </div>

      {/* Stat cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        <div className="rounded border border-bc-border bg-bc-surface p-3">
          <p className="text-[11px] text-bc-muted uppercase tracking-wider">Agents</p>
          <p className="mt-1 text-xl font-bold flex items-center gap-2">
            {running > 0 && <span className="w-2 h-2 rounded-full bg-green-500 animate-pulse" />}
            {running}/{total}
          </p>
        </div>
        <div className="rounded border border-bc-border bg-bc-surface p-3">
          <p className="text-[11px] text-bc-muted uppercase tracking-wider">Tokens</p>
          <p className="mt-1 text-xl font-bold">{fmtTokens(totalTokens)}</p>
          <p className="text-[10px] text-bc-muted">In: {fmtTokens(inputTokens)} / Out: {fmtTokens(outputTokens)}</p>
        </div>
        <div className="rounded border border-bc-border bg-bc-surface p-3">
          <p className="text-[11px] text-bc-muted uppercase tracking-wider">Cost</p>
          <p className="mt-1 text-xl font-bold text-[#FF6B35]">${totalCost.toFixed(2)}</p>
        </div>
        <div className="rounded border border-bc-border bg-bc-surface p-3">
          <p className="text-[11px] text-bc-muted uppercase tracking-wider">Uptime</p>
          <p className="mt-1 text-xl font-bold">{fmtUptime(uptime)}</p>
        </div>
      </div>

      {/* Row 1: CPU + Memory */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
        <Panel title="CPU by Agent (%)">
          {cpuChart.data.length === 0 ? <Empty /> : (
            <ResponsiveContainer width="100%" height={200}>
              <AreaChart data={cpuChart.data} margin={{ top: 4, right: 8, left: -20, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--color-bc-border)" vertical={false} />
                <XAxis dataKey="time" tick={TICK_STYLE} {...AX} />
                <YAxis tick={TICK_STYLE} {...AX} tickFormatter={(v: number) => `${v}%`} />
                <Tooltip contentStyle={TT} />
                {cpuChart.agents.map((n, i) => (
                  <Area key={n} type="monotone" dataKey={n} stroke={COLORS[i % COLORS.length]} fill={COLORS[i % COLORS.length]} fillOpacity={0.12} strokeWidth={1.5} dot={false} />
                ))}
              </AreaChart>
            </ResponsiveContainer>
          )}
        </Panel>
        <Panel title="Memory by Agent (MB)">
          {memChart.data.length === 0 ? <Empty /> : (
            <ResponsiveContainer width="100%" height={200}>
              <AreaChart data={memChart.data} margin={{ top: 4, right: 8, left: -10, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--color-bc-border)" vertical={false} />
                <XAxis dataKey="time" tick={TICK_STYLE} {...AX} />
                <YAxis tick={TICK_STYLE} {...AX} />
                <Tooltip contentStyle={TT} formatter={(v) => [`${Number(v ?? 0).toFixed(1)} MB`]} />
                {memChart.agents.map((n, i) => (
                  <Area key={n} type="monotone" dataKey={n} stroke={COLORS[i % COLORS.length]} fill={COLORS[i % COLORS.length]} fillOpacity={0.12} strokeWidth={1.5} dot={false} />
                ))}
              </AreaChart>
            </ResponsiveContainer>
          )}
        </Panel>
      </div>

      {/* Row 2: Network + Disk */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
        <Panel title="Network I/O">
          {netChart.length === 0 ? <Empty /> : (
            <ResponsiveContainer width="100%" height={200}>
              <AreaChart data={netChart} margin={{ top: 4, right: 8, left: -10, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--color-bc-border)" vertical={false} />
                <XAxis dataKey="time" tick={TICK_STYLE} {...AX} />
                <YAxis tick={TICK_STYLE} {...AX} tickFormatter={(v: number) => fmtBytes(v)} />
                <Tooltip contentStyle={TT} formatter={(v) => [fmtBytes(Number(v ?? 0))]} />
                <Area type="monotone" dataKey="rx" name="RX" stroke="#10B981" fill="#10B981" fillOpacity={0.12} strokeWidth={1.5} dot={false} />
                <Area type="monotone" dataKey="tx" name="TX" stroke="#FF6B35" fill="#FF6B35" fillOpacity={0.12} strokeWidth={1.5} dot={false} />
              </AreaChart>
            </ResponsiveContainer>
          )}
        </Panel>
        <Panel title="Disk I/O">
          {diskChart.length === 0 ? <Empty /> : (
            <ResponsiveContainer width="100%" height={200}>
              <AreaChart data={diskChart} margin={{ top: 4, right: 8, left: -10, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--color-bc-border)" vertical={false} />
                <XAxis dataKey="time" tick={TICK_STYLE} {...AX} />
                <YAxis tick={TICK_STYLE} {...AX} tickFormatter={(v: number) => fmtBytes(v)} />
                <Tooltip contentStyle={TT} formatter={(v) => [fmtBytes(Number(v ?? 0))]} />
                <Area type="monotone" dataKey="read" name="Read" stroke="#3B82F6" fill="#3B82F6" fillOpacity={0.12} strokeWidth={1.5} dot={false} />
                <Area type="monotone" dataKey="write" name="Write" stroke="#A855F7" fill="#A855F7" fillOpacity={0.12} strokeWidth={1.5} dot={false} />
              </AreaChart>
            </ResponsiveContainer>
          )}
        </Panel>
      </div>

      {/* Row 3: Tokens + Cost by Model */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
        <Panel title="Token Usage">
          {tokenChart.length === 0 ? <Empty /> : (
            <ResponsiveContainer width="100%" height={200}>
              <AreaChart data={tokenChart} margin={{ top: 4, right: 8, left: -8, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--color-bc-border)" vertical={false} />
                <XAxis dataKey="time" tick={TICK_STYLE} {...AX} />
                <YAxis tick={TICK_STYLE} {...AX} tickFormatter={(v: number) => fmtTokens(v)} />
                <Tooltip contentStyle={TT} formatter={(v, n) => [Number(v ?? 0).toLocaleString(), n === "input" ? "Input" : "Output"]} />
                <Area type="monotone" dataKey="input" name="Input" stroke="#3B82F6" fill="#3B82F6" fillOpacity={0.12} strokeWidth={1.5} stackId="1" dot={false} />
                <Area type="monotone" dataKey="output" name="Output" stroke="#FF6B35" fill="#FF6B35" fillOpacity={0.12} strokeWidth={1.5} stackId="1" dot={false} />
              </AreaChart>
            </ResponsiveContainer>
          )}
        </Panel>
        <Panel title="Cost by Model">
          {costBarData.length === 0 ? <Empty msg="No cost data" /> : (
            <ResponsiveContainer width="100%" height={200}>
              <BarChart layout="vertical" data={costBarData} margin={{ top: 0, right: 8, left: 4, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--color-bc-border)" horizontal={false} />
                <XAxis type="number" tick={TICK_STYLE} {...AX} tickFormatter={(v: number) => `$${v}`} />
                <YAxis type="category" dataKey="name" tick={{ ...TICK_STYLE, fill: "var(--color-bc-text)", fontSize: 9 }} {...AX} width={120} />
                <Tooltip contentStyle={TT} formatter={(v) => [`$${Number(v ?? 0).toFixed(4)}`]} />
                <Bar dataKey="cost" radius={[0, 3, 3, 0]}>
                  {costBarData.map((_, i) => <Cell key={i} fill={COLORS[i % COLORS.length]} />)}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          )}
        </Panel>
      </div>

      {/* Agent Table */}
      {agentTable.length > 0 && (
        <Panel title={`Agents (${agentTable.length})`}>
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead>
                <tr className="text-bc-muted text-left">
                  {(["name", "role", "state", "cpu", "mem", "tokens", "cost"] as SortKey[]).map(k => (
                    <th key={k} className="py-1.5 px-2 font-medium cursor-pointer hover:text-bc-text select-none" onClick={() => handleSort(k)}>
                      {k === "cpu" ? "CPU%" : k === "mem" ? "Mem MB" : k.charAt(0).toUpperCase() + k.slice(1)}
                      {sortKey === k && <span className="ml-1">{sortAsc ? "\u25B2" : "\u25BC"}</span>}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {agentTable.map(a => (
                  <tr key={a.name}
                    className="border-t border-bc-border/50 hover:bg-bc-bg/50 cursor-pointer transition-colors"
                    onClick={() => navigate(`/agents/${encodeURIComponent(a.name)}`)}
                  >
                    <td className="py-1.5 px-2 font-medium">{a.name}</td>
                    <td className="py-1.5 px-2 text-bc-muted">{a.role}</td>
                    <td className="py-1.5 px-2">
                      <span className="flex items-center gap-1.5">
                        <span className={`w-1.5 h-1.5 rounded-full ${
                          a.state === "working" || a.state === "idle" || a.state === "running" ? "bg-green-500"
                          : a.state === "stuck" ? "bg-orange-500" : "bg-bc-muted"
                        }`} />
                        {a.state}
                      </span>
                    </td>
                    <td className="py-1.5 px-2 font-mono">{a.cpu.toFixed(1)}</td>
                    <td className="py-1.5 px-2 font-mono">{a.mem.toFixed(0)}</td>
                    <td className="py-1.5 px-2 font-mono">{fmtTokens(a.tokens)}</td>
                    <td className="py-1.5 px-2 font-mono text-[#FF6B35]">${a.cost.toFixed(2)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Panel>
      )}
    </div>
  );
}

// ── Data Transforms ─────────────────────────────────────────────────────────────

function pivotAgentMetric(metrics: AgentMetricTS[], mode: "cpu_percent" | "mem_mb") {
  const agents = [...new Set(metrics.filter(m => !isInfra(m.agent_name)).map(m => m.agent_name))];
  type Pt = Record<string, string | number>;
  const buckets = new Map<string, Pt>();
  for (const m of metrics) {
    if (isInfra(m.agent_name)) continue;
    const t = fmtTime(m.time);
    const b = buckets.get(t) ?? { time: t };
    b[m.agent_name] = mode === "cpu_percent"
      ? parseFloat(m.cpu_percent.toFixed(2))
      : parseFloat((m.mem_used_bytes / 1024 / 1024).toFixed(1));
    buckets.set(t, b);
  }
  return { agents, data: Array.from(buckets.values()) };
}

function pivotNetOrDisk(metrics: AgentMetricTS[], kind: "net" | "disk") {
  const buckets = new Map<string, { time: string; rx: number; tx: number; read: number; write: number }>();
  for (const m of metrics) {
    if (isInfra(m.agent_name)) continue;
    const t = fmtTime(m.time);
    const b = buckets.get(t) ?? { time: t, rx: 0, tx: 0, read: 0, write: 0 };
    if (kind === "net") { b.rx += m.net_rx_bytes; b.tx += m.net_tx_bytes; }
    else { b.read += m.disk_read_bytes; b.write += m.disk_write_bytes; }
    buckets.set(t, b);
  }
  return Array.from(buckets.values());
}

function pivotTokens(tokens: TokenMetricTS[]) {
  const buckets = new Map<string, { time: string; input: number; output: number }>();
  for (const t of tokens) {
    const k = fmtTime(t.time);
    const b = buckets.get(k) ?? { time: k, input: 0, output: 0 };
    b.input += t.input_tokens;
    b.output += t.output_tokens;
    buckets.set(k, b);
  }
  return Array.from(buckets.values());
}

interface AgentRow { name: string; role: string; state: string; cpu: number; mem: number; tokens: number; cost: number }

function buildAgentTable(data: StatsData | null, sortKey: SortKey, sortAsc: boolean): AgentRow[] {
  if (!data) return [];
  const latest = new Map<string, AgentMetricTS>();
  for (const m of data.agentCpu) { if (!isInfra(m.agent_name)) latest.set(m.agent_name, m); }

  const costMap = new Map<string, AgentCostSummary>();
  for (const c of data.costByAgent) costMap.set(c.agent_id, c);

  const tokenMap = new Map<string, number>();
  for (const t of data.tokenMetrics) tokenMap.set(t.agent_name, (tokenMap.get(t.agent_name) ?? 0) + t.input_tokens + t.output_tokens);

  const memLatest = new Map<string, number>();
  for (const m of data.agentMem) { if (!isInfra(m.agent_name)) memLatest.set(m.agent_name, m.mem_used_bytes / 1024 / 1024); }

  const rows: AgentRow[] = Array.from(latest.values()).map(m => ({
    name: m.agent_name, role: m.role, state: m.state,
    cpu: m.cpu_percent, mem: memLatest.get(m.agent_name) ?? 0,
    tokens: tokenMap.get(m.agent_name) ?? 0, cost: costMap.get(m.agent_name)?.total_cost_usd ?? 0,
  }));

  const dir = sortAsc ? 1 : -1;
  rows.sort((a, b) => {
    const av = a[sortKey], bv = b[sortKey];
    if (typeof av === "string" && typeof bv === "string") return av.localeCompare(bv) * dir;
    return ((av as number) - (bv as number)) * dir;
  });
  return rows;
}
