import type { Agent } from "../api/client";
import { useAgentStats } from "../hooks/useAgentStats";

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

function progressColor(pct: number): string {
  if (pct >= 80) return "bg-bc-error";
  if (pct >= 60) return "bg-bc-warning";
  return "bg-bc-success";
}

function ProgressBar({ label, value }: { label: string; value: number }) {
  const clamped = Math.min(100, Math.max(0, value));
  return (
    <div className="space-y-1">
      <div className="flex justify-between text-sm">
        <span className="text-bc-muted">{label}</span>
        <span>{clamped.toFixed(1)}%</span>
      </div>
      <div className="h-2 rounded-full bg-bc-border overflow-hidden">
        <div
          className={`h-full rounded-full transition-all duration-300 ${progressColor(clamped)}`}
          style={{ width: `${clamped}%` }}
        />
      </div>
    </div>
  );
}

export function StatsTab({ agent }: { agent: Agent }) {
  const { stats, loading, error } = useAgentStats(agent.name);

  // Derive latest stats record
  const latest = stats && stats.length > 0 ? stats[stats.length - 1] : null;
  const is404 = error?.includes("404");

  const cpuPct = latest?.cpu_pct ?? 0;
  const memPct =
    latest && latest.mem_limit_mb > 0
      ? (latest.mem_used_mb / latest.mem_limit_mb) * 100
      : 0;

  return (
    <div className="space-y-6">
      {/* Cost Breakdown */}
      <div className="space-y-2">
        <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide">
          Cost Breakdown
        </h2>
        <div className="rounded border border-bc-border bg-bc-surface p-4 space-y-2">
          <div className="flex items-start gap-2 py-1.5 border-b border-bc-border/30">
            <span className="text-bc-muted text-sm w-32 shrink-0">
              Total Cost
            </span>
            <span className="text-sm font-medium">
              {agent.cost_usd != null
                ? `$${agent.cost_usd.toFixed(4)}`
                : "\u2014"}
            </span>
          </div>
          <div className="flex items-start gap-2 py-1.5 border-b border-bc-border/30">
            <span className="text-bc-muted text-sm w-32 shrink-0">
              Total Tokens
            </span>
            <span className="text-sm">
              {agent.total_tokens != null
                ? agent.total_tokens.toLocaleString()
                : "\u2014"}
            </span>
          </div>
          <div className="flex items-start gap-2 py-1.5">
            <span className="text-bc-muted text-sm w-32 shrink-0">
              Input / Output
            </span>
            <span className="text-sm text-bc-muted">
              Token split not available at agent level
            </span>
          </div>
        </div>
      </div>

      {/* Resource Usage */}
      <div className="space-y-2">
        <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide">
          Resource Usage
        </h2>
        <div className="rounded border border-bc-border bg-bc-surface p-4">
          {is404 ? (
            <p className="text-sm text-bc-muted italic">
              Resource metrics not available for this runtime
            </p>
          ) : loading && !stats ? (
            <p className="text-sm text-bc-muted">Loading resource metrics...</p>
          ) : error && !stats ? (
            <p className="text-sm text-bc-error">
              Failed to load stats: {error}
            </p>
          ) : !latest ? (
            <p className="text-sm text-bc-muted italic">
              No resource samples collected yet
            </p>
          ) : (
            <div className="space-y-4">
              <ProgressBar label="CPU" value={cpuPct} />
              <ProgressBar label="Memory" value={memPct} />
              {latest && (
                <div className="grid grid-cols-2 gap-2 text-xs text-bc-muted pt-2 border-t border-bc-border/30">
                  <span>
                    Mem: {latest.mem_used_mb.toFixed(1)} /{" "}
                    {latest.mem_limit_mb.toFixed(1)} MB
                  </span>
                  <span>Net RX: {latest.net_rx_mb.toFixed(2)} MB</span>
                  <span>Net TX: {latest.net_tx_mb.toFixed(2)} MB</span>
                  <span>
                    Block R/W: {latest.block_read_mb.toFixed(2)} /{" "}
                    {latest.block_write_mb.toFixed(2)} MB
                  </span>
                </div>
              )}
            </div>
          )}
        </div>
      </div>

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
            <span className="text-bc-muted text-sm w-32 shrink-0">
              Session ID
            </span>
            <span className="text-sm font-mono break-all">
              {agent.session_id || "\u2014"}
            </span>
          </div>
          <div className="flex items-start gap-2 py-1.5">
            <span className="text-bc-muted text-sm w-32 shrink-0">Runtime</span>
            <span className="text-sm">{agent.session || "\u2014"}</span>
          </div>
        </div>
      </div>
    </div>
  );
}
