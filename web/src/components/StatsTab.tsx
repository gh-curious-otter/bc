import type { Agent } from '../api/client';
import { useAgentStats } from '../hooks/useAgentStats';

function progressBarColor(pct: number): string {
  if (pct >= 80) return 'bg-red-500';
  if (pct >= 60) return 'bg-yellow-500';
  return 'bg-emerald-500';
}

function formatTokens(n: number): string {
  return n.toLocaleString();
}

function formatUptime(startedAt: string): string {
  if (!startedAt) return '--';
  const start = new Date(startedAt);
  if (isNaN(start.getTime())) return '--';
  const diffMs = Date.now() - start.getTime();
  if (diffMs < 0) return '--';
  const seconds = Math.floor(diffMs / 1000);
  const d = Math.floor(seconds / 86400);
  const h = Math.floor((seconds % 86400) / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const parts: string[] = [];
  if (d > 0) parts.push(`${d}d`);
  if (h > 0) parts.push(`${h}h`);
  parts.push(`${m}m`);
  return parts.join(' ');
}

function ResourceBar({ label, percent, detail }: { label: string; percent: number; detail: string }) {
  return (
    <div>
      <div className="flex items-center justify-between text-sm mb-1">
        <span className="font-medium">{label}</span>
        <span className="text-bc-muted">{detail}</span>
      </div>
      <div className="h-3 w-full rounded-full bg-bc-border/40 overflow-hidden">
        <div
          className={`h-full rounded-full transition-all ${progressBarColor(percent)}`}
          style={{ width: `${Math.max(percent, 1)}%` }}
        />
      </div>
    </div>
  );
}

export function StatsTab({ agent }: { agent: Agent }) {
  const { data: statsRecords } = useAgentStats(agent.name, 10000);

  const costUsd = agent.total_cost_usd ?? agent.cost_usd ?? 0;
  const totalTokens = agent.total_tokens ?? 0;
  const inputTokens = agent.input_tokens;
  const outputTokens = agent.output_tokens;
  const runtime = agent.runtime_backend || 'tmux';
  const latestStats = statsRecords && statsRecords.length > 0 ? statsRecords[0] : null;

  return (
    <div className="space-y-6">
      {/* Cost Breakdown */}
      <section>
        <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide mb-3">Cost Breakdown</h2>
        <div className="rounded border border-bc-border bg-bc-surface p-4 space-y-3">
          <div className="flex items-start gap-2 py-1.5 border-b border-bc-border/30">
            <span className="text-bc-muted text-sm w-32 shrink-0">Total Cost</span>
            <span className="text-sm font-medium">${costUsd.toFixed(4)}</span>
          </div>
          <div className="flex items-start gap-2 py-1.5 border-b border-bc-border/30">
            <span className="text-bc-muted text-sm w-32 shrink-0">Total Tokens</span>
            <span className="text-sm font-medium">{formatTokens(totalTokens)}</span>
          </div>
          {inputTokens !== undefined && outputTokens !== undefined && (
            <>
              <div className="flex items-start gap-2 py-1.5 border-b border-bc-border/30">
                <span className="text-bc-muted text-sm w-32 shrink-0">Input Tokens</span>
                <span className="text-sm">{formatTokens(inputTokens)}</span>
              </div>
              <div className="flex items-start gap-2 py-1.5">
                <span className="text-bc-muted text-sm w-32 shrink-0">Output Tokens</span>
                <span className="text-sm">{formatTokens(outputTokens)}</span>
              </div>
            </>
          )}
        </div>
      </section>

      {/* Resource Usage */}
      <section>
        <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide mb-3">Resource Usage</h2>
        {latestStats ? (
          <div className="rounded border border-bc-border bg-bc-surface p-4 space-y-4">
            <ResourceBar
              label="CPU"
              percent={latestStats.cpu_pct}
              detail={`${latestStats.cpu_pct.toFixed(1)}%`}
            />
            {latestStats.mem_limit_mb > 0 ? (
              <ResourceBar
                label="Memory"
                percent={(latestStats.mem_used_mb / latestStats.mem_limit_mb) * 100}
                detail={`${latestStats.mem_used_mb.toFixed(1)} MB / ${latestStats.mem_limit_mb.toFixed(0)} MB`}
              />
            ) : (
              <ResourceBar
                label="Memory"
                percent={0}
                detail={`${latestStats.mem_used_mb.toFixed(1)} MB`}
              />
            )}
          </div>
        ) : (
          <div className="rounded border border-bc-border bg-bc-surface p-4 text-sm text-bc-muted">
            Resource metrics not available for this runtime.
          </div>
        )}
      </section>

      {/* Session Info */}
      <section>
        <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide mb-3">Session Info</h2>
        <div className="rounded border border-bc-border bg-bc-surface p-4 space-y-3">
          <div className="flex items-start gap-2 py-1.5 border-b border-bc-border/30">
            <span className="text-bc-muted text-sm w-32 shrink-0">Uptime</span>
            <span className="text-sm font-medium">{formatUptime(agent.started_at)}</span>
          </div>
          <div className="flex items-start gap-2 py-1.5 border-b border-bc-border/30">
            <span className="text-bc-muted text-sm w-32 shrink-0">Session ID</span>
            <span className="text-sm font-mono break-all">{agent.session_id || agent.session || '--'}</span>
          </div>
          <div className="flex items-start gap-2 py-1.5">
            <span className="text-bc-muted text-sm w-32 shrink-0">Runtime</span>
            <span className="text-sm font-mono">{runtime}</span>
          </div>
        </div>
      </section>
    </div>
  );
}
