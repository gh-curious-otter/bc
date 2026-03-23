import { useCallback } from "react";
import {
  AreaChart, Area, BarChart, Bar, PieChart, Pie, Cell,
  XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer,
} from "recharts";
import { api } from "../api/client";
import type {
  SystemStats, StatsSummary, CostSummary, ModelCostSummary,
  SystemMetricTS, AgentMetricTS, TokenMetricTS, ChannelMetricTS,
} from "../api/client";
import { usePolling } from "../hooks/usePolling";
import { LoadingSkeleton } from "../components/LoadingSkeleton";
import { EmptyState } from "../components/EmptyState";

// ── Theme ──────────────────────────────────────────────────────────────────────

const C = {
  bg: "#1a1714", surface: "#1e1a16", border: "#2a2420", muted: "#8c7e72", text: "#f5f0eb",
  emerald: "#10b981", blue: "#3b82f6", amber: "#f59e0b", purple: "#a855f7",
  orange: "#ea580c", cyan: "#06b6d4", pink: "#ec4899", lime: "#84cc16", red: "#ef4444",
};

const PIE_COLORS = [C.emerald, C.blue, C.amber, C.purple, C.orange, C.cyan, C.pink, C.lime];
const TT: React.CSSProperties = { backgroundColor: C.surface, border: `1px solid ${C.border}`, borderRadius: "6px", color: C.text, fontSize: "12px" };
const TICK = { axisLine: false as const, tickLine: false as const };

// ── Data ───────────────────────────────────────────────────────────────────────

interface StatsData {
  system: SystemStats | null;
  summary: StatsSummary | null;
  costSummary: CostSummary | null;
  costByModel: ModelCostSummary[];
  systemMetrics: SystemMetricTS[];
  agentMetrics: AgentMetricTS[];
  tokenMetrics: TokenMetricTS[];
  channelMetrics: ChannelMetricTS[];
}

// ── Helpers ────────────────────────────────────────────────────────────────────

const fmtTime = (iso: string) => { try { return new Date(iso).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }); } catch { return iso; } };
const fmtBytes = (b: number) => { if (!b) return "0 B"; const u = ["B","KB","MB","GB","TB"]; const i = Math.floor(Math.log(b)/Math.log(1024)); return `${(b/Math.pow(1024,i)).toFixed(1)} ${u[i]}`; };
const fmtUptime = (s: number) => { const d=Math.floor(s/86400),h=Math.floor((s%86400)/3600),m=Math.floor((s%3600)/60); return [d&&`${d}d`,h&&`${h}h`,`${m}m`].filter(Boolean).join(" "); };
const pctColor = (p: number) => p >= 80 ? C.red : p >= 60 ? C.amber : C.emerald;
const trunc = (s: string, n: number) => s.length > n ? s.slice(0, n) + "\u2026" : s;

// ── Primitives ─────────────────────────────────────────────────────────────────

function Stat({ label, value, sub, accent }: { label: string; value: string; sub?: string; accent?: string }) {
  return (
    <div className="rounded border border-bc-border bg-bc-surface p-4">
      <p className="text-xs text-bc-muted uppercase tracking-wide">{label}</p>
      <p className="mt-1 text-2xl font-bold truncate" style={accent ? { color: accent } : undefined}>{value}</p>
      {sub && <p className="mt-0.5 text-xs text-bc-muted truncate" title={sub}>{sub}</p>}
    </div>
  );
}

function Chart({ title, children, wide }: { title: string; children: React.ReactNode; wide?: boolean }) {
  return (
    <div className={`rounded border border-bc-border bg-bc-surface p-4 ${wide ? "md:col-span-2" : ""}`}>
      <p className="text-xs text-bc-muted uppercase tracking-wide mb-4">{title}</p>
      {children}
    </div>
  );
}

function Empty({ msg = "No data yet" }: { msg?: string }) {
  return <div className="flex items-center justify-center h-32 text-sm text-bc-muted">{msg}</div>;
}

// ── System Charts ──────────────────────────────────────────────────────────────

function SystemCharts({ metrics, system }: { metrics: SystemMetricTS[]; system: SystemStats | null }) {
  // Group by system_name for multi-line charts
  const names = [...new Set(metrics.map(m => m.system_name))];
  const colors: Record<string, string> = {};
  names.forEach((n, i) => { colors[n] = PIE_COLORS[i % PIE_COLORS.length]!; });

  // Pivot data for recharts: { time, "bc-sql_cpu": 0.5, "bc-stats_cpu": 0.7, ... }
  type Pt = Record<string, string | number>;
  const buckets = new Map<string, Pt>();
  for (const m of metrics) {
    const t = fmtTime(m.time);
    const b = buckets.get(t) ?? { time: t };
    b[`${m.system_name}_cpu`] = parseFloat(m.cpu_percent.toFixed(2));
    b[`${m.system_name}_mem`] = parseFloat((m.mem_used_bytes / 1024 / 1024).toFixed(1));
    b[`${m.system_name}_netrx`] = parseFloat((m.net_rx_bytes / 1024).toFixed(1));
    b[`${m.system_name}_nettx`] = parseFloat((m.net_tx_bytes / 1024).toFixed(1));
    buckets.set(t, b);
  }
  const data = Array.from(buckets.values());

  if (data.length === 0 && !system) return null;

  return (
    <section>
      <h2 className="text-xs font-medium text-bc-muted uppercase tracking-widest mb-3">System Resources</h2>

      {/* Current snapshot cards */}
      {system && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-4">
          <Stat label="CPU" value={`${system.cpu_usage_percent.toFixed(1)}%`} sub={`${system.cpus} cores`} accent={pctColor(system.cpu_usage_percent)} />
          <Stat label="Memory" value={`${system.memory_usage_percent.toFixed(1)}%`} sub={`${fmtBytes(system.memory_used_bytes)} / ${fmtBytes(system.memory_total_bytes)}`} accent={pctColor(system.memory_usage_percent)} />
          <Stat label="Disk" value={`${system.disk_usage_percent.toFixed(1)}%`} sub={`${fmtBytes(system.disk_used_bytes)} / ${fmtBytes(system.disk_total_bytes)}`} accent={pctColor(system.disk_usage_percent)} />
          <Stat label="Goroutines" value={String(system.goroutines)} sub={`${system.os}/${system.arch}`} />
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {/* CPU per container */}
        <Chart title="CPU by Container (%)">
          {data.length === 0 ? <Empty /> : (
            <ResponsiveContainer width="100%" height={180}>
              <AreaChart data={data} margin={{ top: 4, right: 8, left: -20, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke={C.border} vertical={false} />
                <XAxis dataKey="time" tick={{ fill: C.muted, fontSize: 10 }} {...TICK} />
                <YAxis tick={{ fill: C.muted, fontSize: 10 }} {...TICK} tickFormatter={(v: number) => `${v}%`} />
                <Tooltip contentStyle={TT} />
                <Legend wrapperStyle={{ fontSize: "11px" }} />
                {names.map(n => (
                  <Area key={n} type="monotone" dataKey={`${n}_cpu`} name={n} stroke={colors[n]} fill={colors[n]} fillOpacity={0.15} strokeWidth={2} dot={false} />
                ))}
              </AreaChart>
            </ResponsiveContainer>
          )}
        </Chart>

        {/* Memory per container */}
        <Chart title="Memory by Container (MB)">
          {data.length === 0 ? <Empty /> : (
            <ResponsiveContainer width="100%" height={180}>
              <AreaChart data={data} margin={{ top: 4, right: 8, left: -10, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke={C.border} vertical={false} />
                <XAxis dataKey="time" tick={{ fill: C.muted, fontSize: 10 }} {...TICK} />
                <YAxis tick={{ fill: C.muted, fontSize: 10 }} {...TICK} tickFormatter={(v: number) => `${v}`} />
                <Tooltip contentStyle={TT} formatter={(v: any) => [`${Number(v)} MB`]} />
                <Legend wrapperStyle={{ fontSize: "11px" }} />
                {names.map(n => (
                  <Area key={n} type="monotone" dataKey={`${n}_mem`} name={n} stroke={colors[n]} fill={colors[n]} fillOpacity={0.15} strokeWidth={2} dot={false} />
                ))}
              </AreaChart>
            </ResponsiveContainer>
          )}
        </Chart>

        {/* Network */}
        <Chart title="Network I/O (KB)" wide>
          {data.length === 0 ? <Empty /> : (
            <ResponsiveContainer width="100%" height={160}>
              <AreaChart data={data} margin={{ top: 4, right: 8, left: -10, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke={C.border} vertical={false} />
                <XAxis dataKey="time" tick={{ fill: C.muted, fontSize: 10 }} {...TICK} />
                <YAxis tick={{ fill: C.muted, fontSize: 10 }} {...TICK} />
                <Tooltip contentStyle={TT} formatter={(v: any) => [`${Number(v)} KB`]} />
                <Legend wrapperStyle={{ fontSize: "11px" }} />
                {names.map(n => (
                  <Area key={`${n}_rx`} type="monotone" dataKey={`${n}_netrx`} name={`${n} rx`} stroke={colors[n]} fill="none" strokeWidth={2} dot={false} />
                ))}
                {names.map(n => (
                  <Area key={`${n}_tx`} type="monotone" dataKey={`${n}_nettx`} name={`${n} tx`} stroke={colors[n]} fill="none" strokeWidth={1} strokeDasharray="4 2" dot={false} />
                ))}
              </AreaChart>
            </ResponsiveContainer>
          )}
        </Chart>
      </div>
    </section>
  );
}

// ── Agent Charts ───────────────────────────────────────────────────────────────

function AgentCharts({ metrics, summary }: { metrics: AgentMetricTS[]; summary: StatsSummary | null }) {
  const latest = new Map<string, AgentMetricTS>();
  for (const m of metrics) latest.set(m.agent_name, m);
  const agents = Array.from(latest.values()).sort((a, b) => b.cpu_percent - a.cpu_percent).slice(0, 10);

  const barData = agents.map(a => ({
    name: trunc(a.agent_name, 14),
    cpu: parseFloat(a.cpu_percent.toFixed(2)),
    mem: parseFloat((a.mem_used_bytes / 1024 / 1024).toFixed(1)),
    state: a.state,
  }));

  const pie = summary ? [
    { name: "Running", value: summary.agents_running, color: C.emerald },
    { name: "Stopped", value: summary.agents_stopped, color: C.muted },
  ].filter(d => d.value > 0) : [];

  return (
    <section>
      <h2 className="text-xs font-medium text-bc-muted uppercase tracking-widest mb-3">Agent Metrics</h2>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Chart title="Agent CPU (%)">
          {barData.length === 0 ? <Empty msg="No agents running" /> : (
            <ResponsiveContainer width="100%" height={Math.max(120, barData.length * 32)}>
              <BarChart layout="vertical" data={barData} margin={{ top: 0, right: 8, left: 4, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke={C.border} horizontal={false} />
                <XAxis type="number" tick={{ fill: C.muted, fontSize: 10 }} {...TICK} tickFormatter={(v: number) => `${v}%`} />
                <YAxis type="category" dataKey="name" tick={{ fill: C.text, fontSize: 10 }} {...TICK} width={90} />
                <Tooltip contentStyle={TT} formatter={(v: any) => [`${Number(v)}%`, "CPU"]} />
                <Bar dataKey="cpu" fill={C.emerald} radius={[0, 3, 3, 0]} />
              </BarChart>
            </ResponsiveContainer>
          )}
        </Chart>

        <Chart title="Agent Memory (MB)">
          {barData.length === 0 ? <Empty msg="No agents running" /> : (
            <ResponsiveContainer width="100%" height={Math.max(120, barData.length * 32)}>
              <BarChart layout="vertical" data={barData} margin={{ top: 0, right: 8, left: 4, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke={C.border} horizontal={false} />
                <XAxis type="number" tick={{ fill: C.muted, fontSize: 10 }} {...TICK} tickFormatter={(v: number) => `${v}MB`} />
                <YAxis type="category" dataKey="name" tick={{ fill: C.text, fontSize: 10 }} {...TICK} width={90} />
                <Tooltip contentStyle={TT} formatter={(v: any) => [`${Number(v)} MB`, "Memory"]} />
                <Bar dataKey="mem" fill={C.blue} radius={[0, 3, 3, 0]} />
              </BarChart>
            </ResponsiveContainer>
          )}
        </Chart>

        {pie.length > 0 && (
          <Chart title="Agent States">
            <div className="flex items-center gap-6">
              <ResponsiveContainer width={140} height={140}>
                <PieChart>
                  <Pie data={pie} cx="50%" cy="50%" innerRadius={38} outerRadius={58} paddingAngle={3} dataKey="value">
                    {pie.map((e, i) => <Cell key={i} fill={e.color} stroke="none" />)}
                  </Pie>
                  <Tooltip contentStyle={TT} />
                </PieChart>
              </ResponsiveContainer>
              <div className="space-y-2">
                {pie.map(e => (
                  <div key={e.name} className="flex items-center gap-2 text-sm">
                    <span className="w-3 h-3 rounded-full" style={{ backgroundColor: e.color }} />
                    <span>{e.name}</span>
                    <span className="text-bc-muted font-mono">{e.value}</span>
                  </div>
                ))}
              </div>
            </div>
          </Chart>
        )}
      </div>
    </section>
  );
}

// ── Token Charts ───────────────────────────────────────────────────────────────

function TokenCharts({ tokens, costSummary }: { tokens: TokenMetricTS[]; costSummary: CostSummary | null }) {
  const buckets = new Map<string, { time: string; input: number; output: number; cost: number }>();
  for (const t of tokens) {
    const k = fmtTime(t.time);
    const b = buckets.get(k) ?? { time: k, input: 0, output: 0, cost: 0 };
    b.input += t.input_tokens;
    b.output += t.output_tokens;
    b.cost += t.cost_usd;
    buckets.set(k, b);
  }
  const data = Array.from(buckets.values());

  return (
    <section>
      <h2 className="text-xs font-medium text-bc-muted uppercase tracking-widest mb-3">Token Usage</h2>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Chart title="Input vs Output Tokens" wide>
          {data.length === 0 ? <Empty msg="No token usage yet — start agents to see data" /> : (
            <ResponsiveContainer width="100%" height={180}>
              <AreaChart data={data} margin={{ top: 4, right: 8, left: -8, bottom: 0 }}>
                <defs>
                  <linearGradient id="g-in" x1="0" y1="0" x2="0" y2="1"><stop offset="5%" stopColor={C.amber} stopOpacity={0.4} /><stop offset="95%" stopColor={C.amber} stopOpacity={0} /></linearGradient>
                  <linearGradient id="g-out" x1="0" y1="0" x2="0" y2="1"><stop offset="5%" stopColor={C.purple} stopOpacity={0.4} /><stop offset="95%" stopColor={C.purple} stopOpacity={0} /></linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke={C.border} vertical={false} />
                <XAxis dataKey="time" tick={{ fill: C.muted, fontSize: 10 }} {...TICK} />
                <YAxis tick={{ fill: C.muted, fontSize: 10 }} {...TICK} tickFormatter={(v: number) => v >= 1000 ? `${(v/1000).toFixed(0)}k` : String(v)} />
                <Tooltip contentStyle={TT} formatter={(v: any, n: any) => [Number(v).toLocaleString(), n === "input" ? "Input" : "Output"]} />
                <Legend wrapperStyle={{ fontSize: "11px" }} formatter={(v: string) => v === "input" ? "Input Tokens" : "Output Tokens"} />
                <Area type="monotone" dataKey="input" stroke={C.amber} strokeWidth={2} fill="url(#g-in)" stackId="1" dot={false} />
                <Area type="monotone" dataKey="output" stroke={C.purple} strokeWidth={2} fill="url(#g-out)" stackId="1" dot={false} />
              </AreaChart>
            </ResponsiveContainer>
          )}
        </Chart>

        {costSummary && (
          <>
            <Stat label="Input Tokens" value={costSummary.input_tokens.toLocaleString()} accent={C.amber} />
            <Stat label="Output Tokens" value={costSummary.output_tokens.toLocaleString()} accent={C.purple} />
          </>
        )}
      </div>
    </section>
  );
}

// ── Channel Charts ─────────────────────────────────────────────────────────────

function ChannelCharts({ metrics }: { metrics: ChannelMetricTS[] }) {
  const latest = new Map<string, ChannelMetricTS>();
  for (const m of metrics) {
    const prev = latest.get(m.channel_name);
    if (!prev || m.message_count > prev.message_count) latest.set(m.channel_name, m);
  }
  const data = Array.from(latest.values())
    .sort((a, b) => b.message_count - a.message_count)
    .map(m => ({ name: trunc(m.channel_name, 16), messages: Number(m.message_count), members: m.member_count }));

  return (
    <section>
      <h2 className="text-xs font-medium text-bc-muted uppercase tracking-widest mb-3">Channel Activity</h2>
      <Chart title="Messages &amp; Members">
        {data.length === 0 ? <Empty msg="No channel data" /> : (
          <ResponsiveContainer width="100%" height={Math.max(140, data.length * 34)}>
            <BarChart layout="vertical" data={data} margin={{ top: 0, right: 8, left: 8, bottom: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke={C.border} horizontal={false} />
              <XAxis type="number" tick={{ fill: C.muted, fontSize: 10 }} {...TICK} />
              <YAxis type="category" dataKey="name" tick={{ fill: C.text, fontSize: 10 }} {...TICK} width={100} />
              <Tooltip contentStyle={TT} />
              <Legend wrapperStyle={{ fontSize: "11px" }} />
              <Bar dataKey="messages" name="Messages" fill={C.blue} radius={[0, 3, 3, 0]} />
              <Bar dataKey="members" name="Members" fill={C.emerald} radius={[0, 3, 3, 0]} />
            </BarChart>
          </ResponsiveContainer>
        )}
      </Chart>
    </section>
  );
}

// ── Cost Charts ────────────────────────────────────────────────────────────────

function CostCharts({ costSummary, costByModel }: { costSummary: CostSummary | null; costByModel: ModelCostSummary[] }) {
  const sorted = [...costByModel].sort((a, b) => b.total_cost_usd - a.total_cost_usd);
  const pieData = sorted.filter(m => m.total_cost_usd > 0).map((m, i) => ({
    name: m.model || "unknown", value: parseFloat(m.total_cost_usd.toFixed(6)), color: PIE_COLORS[i % PIE_COLORS.length],
  }));

  return (
    <section>
      <h2 className="text-xs font-medium text-bc-muted uppercase tracking-widest mb-3">Cost Breakdown</h2>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {costSummary && (
          <>
            <Stat label="Total Cost" value={`$${costSummary.total_cost_usd.toFixed(2)}`} sub={`${costSummary.record_count.toLocaleString()} records`} accent={C.orange} />
            <Stat label="Total Tokens" value={costSummary.total_tokens.toLocaleString()} sub={`In: ${costSummary.input_tokens.toLocaleString()} / Out: ${costSummary.output_tokens.toLocaleString()}`} />
          </>
        )}

        <Chart title="Cost by Model">
          {pieData.length === 0 ? <Empty msg="No cost data" /> : (
            <div className="flex items-center gap-4">
              <ResponsiveContainer width={140} height={140}>
                <PieChart>
                  <Pie data={pieData} cx="50%" cy="50%" innerRadius={38} outerRadius={58} paddingAngle={2} dataKey="value">
                    {pieData.map((e, i) => <Cell key={i} fill={e.color} stroke="none" />)}
                  </Pie>
                  <Tooltip contentStyle={TT} formatter={(v: any) => [`$${Number(v).toFixed(4)}`]} />
                </PieChart>
              </ResponsiveContainer>
              <div className="space-y-1 flex-1 max-h-36 overflow-y-auto">
                {pieData.map(e => (
                  <div key={e.name} className="flex items-center gap-2 text-xs">
                    <span className="w-2.5 h-2.5 rounded-sm flex-shrink-0" style={{ backgroundColor: e.color }} />
                    <span className="truncate flex-1" title={e.name}>{e.name}</span>
                    <span className="text-bc-muted font-mono">${e.value.toFixed(4)}</span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </Chart>

        <Chart title="Cost per Model (USD)">
          {sorted.length === 0 ? <Empty msg="No cost data" /> : (
            <ResponsiveContainer width="100%" height={Math.max(120, sorted.slice(0, 8).length * 28)}>
              <BarChart layout="vertical" data={sorted.slice(0, 8).map(m => ({ name: trunc(m.model || "unknown", 20), cost: parseFloat(m.total_cost_usd.toFixed(4)) }))} margin={{ top: 0, right: 8, left: 4, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke={C.border} horizontal={false} />
                <XAxis type="number" tick={{ fill: C.muted, fontSize: 10 }} {...TICK} tickFormatter={(v: number) => `$${v}`} />
                <YAxis type="category" dataKey="name" tick={{ fill: C.text, fontSize: 9 }} {...TICK} width={120} />
                <Tooltip contentStyle={TT} formatter={(v: any) => [`$${Number(v).toFixed(6)}`]} />
                <Bar dataKey="cost" fill={C.purple} radius={[0, 3, 3, 0]} />
              </BarChart>
            </ResponsiveContainer>
          )}
        </Chart>
      </div>
    </section>
  );
}

// ── Main ───────────────────────────────────────────────────────────────────────

export function Stats() {
  const fetcher = useCallback(async (): Promise<StatsData> => {
    const [r0, r1, r2, r3, r4, r5, r6, r7] = await Promise.allSettled([
      api.getStatsSystem(),
      api.getStatsSummary(),
      api.getCostSummary(),
      api.getCostByModel(),
      api.getSystemStats("cpu"),
      api.getAgentStats("cpu"),
      api.getAgentTokenStats(),
      api.getChannelStats("messages"),
    ]);

    return {
      system: r0.status === "fulfilled" ? r0.value : null,
      summary: r1.status === "fulfilled" ? r1.value : null,
      costSummary: r2.status === "fulfilled" ? r2.value : null,
      costByModel: r3.status === "fulfilled" ? r3.value : [],
      systemMetrics: r4.status === "fulfilled" ? (r4.value ?? []) : [],
      agentMetrics: r5.status === "fulfilled" ? (r5.value ?? []) : [],
      tokenMetrics: r6.status === "fulfilled" ? (r6.value ?? []) : [],
      channelMetrics: r7.status === "fulfilled" ? (r7.value ?? []) : [],
    };
  }, []);

  const { data, loading, error, refresh, timedOut } = usePolling(fetcher, 10000);

  if (loading && !data) return <div className="p-6 space-y-6"><LoadingSkeleton variant="cards" rows={4} /></div>;
  if (timedOut && !data) return <div className="p-6"><EmptyState icon="!" title="Stats timed out" actionLabel="Retry" onAction={refresh} /></div>;
  if (error && !data) return <div className="p-6"><EmptyState icon="!" title="Failed to load stats" description={error} actionLabel="Retry" onAction={refresh} /></div>;
  if (!data) return null;

  const uptime = data.system?.uptime_seconds ?? data.summary?.uptime_seconds ?? 0;

  return (
    <div className="p-6 space-y-8">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold">Stats</h1>
        <span className="text-xs text-bc-muted">Live &middot; 10s</span>
      </div>

      {/* Overview */}
      <section>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
          <Stat label="Agents" value={String(data.summary?.agents_total ?? 0)} sub={`${data.summary?.agents_running ?? 0} running`} accent={C.emerald} />
          <Stat label="Cost" value={`$${(data.summary?.total_cost_usd ?? 0).toFixed(2)}`} accent={C.orange} />
          <Stat label="Channels" value={String(data.summary?.channels_total ?? 0)} sub={`${(data.summary?.messages_total ?? 0).toLocaleString()} messages`} />
          <Stat label="Uptime" value={fmtUptime(uptime)} sub={data.system?.hostname ?? ""} />
        </div>
      </section>

      <SystemCharts metrics={data.systemMetrics} system={data.system} />
      <AgentCharts metrics={data.agentMetrics} summary={data.summary} />
      <TokenCharts tokens={data.tokenMetrics} costSummary={data.costSummary} />
      <ChannelCharts metrics={data.channelMetrics} />
      <CostCharts costSummary={data.costSummary} costByModel={data.costByModel} />
    </div>
  );
}
