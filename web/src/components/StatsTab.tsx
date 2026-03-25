import { useCallback, useEffect, useState } from "react";
import {
  AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
} from "recharts";
import { api } from "../api/client";
import type { Agent, AgentMetricTS, TokenMetricTS } from "../api/client";

const C = {
  emerald: "#10b981", blue: "#3b82f6", amber: "#f59e0b", red: "#ef4444",
  surface: "#1e1a16", border: "#2a2420", text: "#f5f0eb",
};
const TT: React.CSSProperties = { backgroundColor: C.surface, border: `1px solid ${C.border}`, borderRadius: "6px", color: C.text, fontSize: "12px" };
const fmtTime = (iso: string) => { try { return new Date(iso).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }); } catch { return iso; } };

function formatUptime(startedAt?: string): string {
  if (!startedAt) return "\u2014";
  const start = new Date(startedAt);
  if (isNaN(start.getTime())) return "\u2014";
  const diffMs = Date.now() - start.getTime();
  if (diffMs < 0) return "\u2014";
  const seconds = Math.floor(diffMs / 1000);
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const parts: string[] = [];
  if (days > 0) parts.push(`${days}d`);
  if (hours > 0) parts.push(`${hours}h`);
  parts.push(`${minutes}m`);
  return parts.join(" ");
}

function pctColor(p: number): string {
  if (p >= 80) return "bg-bc-error";
  if (p >= 60) return "bg-bc-warning";
  return "bg-bc-success";
}

function ProgressBar({ label, value, detail }: { label: string; value: number; detail?: string }) {
  const clamped = Math.min(100, Math.max(0, value));
  return (
    <div className="space-y-1">
      <div className="flex justify-between text-sm">
        <span className="text-bc-muted">{label}</span>
        <span>{clamped.toFixed(1)}%{detail ? ` (${detail})` : ""}</span>
      </div>
      <div className="h-2 rounded-full bg-bc-border overflow-hidden">
        <div
          className={`h-full rounded-full transition-all duration-300 ${pctColor(clamped)}`}
          style={{ width: `${clamped}%` }}
        />
      </div>
    </div>
  );
}

function Chart({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="rounded border border-bc-border bg-bc-surface p-4">
      <p className="text-xs text-bc-muted uppercase tracking-wide mb-4">{title}</p>
      {children}
    </div>
  );
}

function fmtBytes(b: number): string {
  if (!b) return "0 B";
  const u = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(b) / Math.log(1024));
  return `${(b / Math.pow(1024, i)).toFixed(1)} ${u[i]}`;
}

export function StatsTab({ agent }: { agent: Agent }) {
  const [cpuData, setCpuData] = useState<AgentMetricTS[]>([]);
  const [tokenData, setTokenData] = useState<TokenMetricTS[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchStats = useCallback(async () => {
    try {
      const now = new Date();
      const from = new Date(now.getTime() - 60 * 60 * 1000); // last hour
      const params = {
        from: from.toISOString(),
        to: now.toISOString(),
        interval: "1m",
        agent: agent.name,
      };
      const [cpu, tokens] = await Promise.allSettled([
        api.getAgentStats("cpu", params),
        api.getAgentTokenStats({ ...params }),
      ]);
      if (cpu.status === "fulfilled") setCpuData(cpu.value);
      if (tokens.status === "fulfilled") setTokenData(tokens.value);
      setError(null);
    } catch {
      setError("Failed to load stats");
    } finally {
      setLoading(false);
    }
  }, [agent.name]);

  useEffect(() => {
    fetchStats();
    const interval = setInterval(fetchStats, 30000);
    return () => clearInterval(interval);
  }, [fetchStats]);

  // Derive latest values
  const latest = cpuData.length > 0 ? cpuData[cpuData.length - 1] : null;
  const cpuPct = latest?.cpu_percent ?? 0;
  const memPct = latest?.mem_percent ?? 0;
  const memDetail = latest ? `${fmtBytes(latest.mem_used_bytes)} / ${fmtBytes(latest.mem_limit_bytes)}` : undefined;

  // Token totals
  const totalInput = tokenData.reduce((sum, t) => sum + t.input_tokens, 0);
  const totalOutput = tokenData.reduce((sum, t) => sum + t.output_tokens, 0);

  // Chart data
  const cpuChartData = cpuData.map(m => ({
    time: fmtTime(m.time),
    cpu: parseFloat(m.cpu_percent.toFixed(2)),
    mem: parseFloat(m.mem_percent.toFixed(2)),
  }));

  const tokenChartData = tokenData.map(m => ({
    time: fmtTime(m.time),
    input: m.input_tokens,
    output: m.output_tokens,
  }));

  return (
    <div className="space-y-6">
      {/* Current Resource Usage */}
      <div className="space-y-2">
        <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide">
          Resource Usage
        </h2>
        <div className="rounded border border-bc-border bg-bc-surface p-4">
          {loading ? (
            <p className="text-sm text-bc-muted">Loading resource metrics...</p>
          ) : error ? (
            <p className="text-sm text-bc-muted italic">
              Stats collection unavailable — ensure TimescaleDB is running
            </p>
          ) : !latest ? (
            <p className="text-sm text-bc-muted italic">
              No resource samples collected yet — data appears after ~30s
            </p>
          ) : (
            <div className="space-y-4">
              <ProgressBar label="CPU" value={cpuPct} />
              <ProgressBar label="Memory" value={memPct} detail={memDetail} />
              <div className="grid grid-cols-2 gap-2 text-xs text-bc-muted pt-2 border-t border-bc-border/30">
                <span>Net RX: {fmtBytes(latest.net_rx_bytes)}</span>
                <span>Net TX: {fmtBytes(latest.net_tx_bytes)}</span>
                <span>Disk Read: {fmtBytes(latest.disk_read_bytes)}</span>
                <span>Disk Write: {fmtBytes(latest.disk_write_bytes)}</span>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* CPU/Memory Chart */}
      {cpuChartData.length > 1 && (
        <Chart title="CPU & Memory (last hour)">
          <ResponsiveContainer width="100%" height={200}>
            <AreaChart data={cpuChartData}>
              <CartesianGrid strokeDasharray="3 3" stroke={C.border} />
              <XAxis dataKey="time" tick={{ fill: C.text, fontSize: 10 }} />
              <YAxis tick={{ fill: C.text, fontSize: 10 }} domain={[0, 100]} unit="%" />
              <Tooltip contentStyle={TT} />
              <Area type="monotone" dataKey="cpu" name="CPU %" stroke={C.emerald} fill={C.emerald} fillOpacity={0.2} />
              <Area type="monotone" dataKey="mem" name="Mem %" stroke={C.blue} fill={C.blue} fillOpacity={0.2} />
            </AreaChart>
          </ResponsiveContainer>
        </Chart>
      )}

      {/* Token Usage */}
      <div className="space-y-2">
        <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide">
          Token Usage
        </h2>
        <div className="rounded border border-bc-border bg-bc-surface p-4 space-y-2">
          <div className="flex items-start gap-2 py-1.5 border-b border-bc-border/30">
            <span className="text-bc-muted text-sm w-32 shrink-0">Total Cost</span>
            <span className="text-sm font-medium">
              {agent.cost_usd != null ? `$${agent.cost_usd.toFixed(4)}` : "\u2014"}
            </span>
          </div>
          <div className="flex items-start gap-2 py-1.5 border-b border-bc-border/30">
            <span className="text-bc-muted text-sm w-32 shrink-0">Total Tokens</span>
            <span className="text-sm">
              {agent.total_tokens != null ? agent.total_tokens.toLocaleString() : "\u2014"}
            </span>
          </div>
          <div className="flex items-start gap-2 py-1.5">
            <span className="text-bc-muted text-sm w-32 shrink-0">Input / Output</span>
            <span className="text-sm">
              {totalInput > 0 || totalOutput > 0
                ? `${totalInput.toLocaleString()} / ${totalOutput.toLocaleString()}`
                : "\u2014"}
            </span>
          </div>
        </div>
      </div>

      {/* Token Chart */}
      {tokenChartData.length > 1 && (
        <Chart title="Token Usage (last hour)">
          <ResponsiveContainer width="100%" height={200}>
            <AreaChart data={tokenChartData}>
              <CartesianGrid strokeDasharray="3 3" stroke={C.border} />
              <XAxis dataKey="time" tick={{ fill: C.text, fontSize: 10 }} />
              <YAxis tick={{ fill: C.text, fontSize: 10 }} />
              <Tooltip contentStyle={TT} />
              <Area type="monotone" dataKey="input" name="Input" stroke={C.emerald} fill={C.emerald} fillOpacity={0.2} />
              <Area type="monotone" dataKey="output" name="Output" stroke={C.amber} fill={C.amber} fillOpacity={0.2} />
            </AreaChart>
          </ResponsiveContainer>
        </Chart>
      )}

      {/* Session Info */}
      <div className="space-y-2">
        <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide">
          Session Info
        </h2>
        <div className="rounded border border-bc-border bg-bc-surface p-4 space-y-2">
          <div className="flex items-start gap-2 py-1.5 border-b border-bc-border/30">
            <span className="text-bc-muted text-sm w-32 shrink-0">Uptime</span>
            <span className="text-sm">{formatUptime(agent.started_at)}</span>
          </div>
          <div className="flex items-start gap-2 py-1.5 border-b border-bc-border/30">
            <span className="text-bc-muted text-sm w-32 shrink-0">Session ID</span>
            <span className="text-sm font-mono break-all">{agent.session_id || "\u2014"}</span>
          </div>
          <div className="flex items-start gap-2 py-1.5 border-b border-bc-border/30">
            <span className="text-bc-muted text-sm w-32 shrink-0">Runtime</span>
            <span className="text-sm">{latest?.runtime || "tmux"}</span>
          </div>
          <div className="flex items-start gap-2 py-1.5">
            <span className="text-bc-muted text-sm w-32 shrink-0">Tool</span>
            <span className="text-sm">{agent.tool || "\u2014"}</span>
          </div>
        </div>
      </div>
    </div>
  );
}
