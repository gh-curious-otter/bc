import { useCallback } from 'react';
import { api } from '../api/client';
import type { SystemStats, StatsSummary, ChannelStats, ModelCostSummary, CostSummary } from '../api/client';
import { usePolling } from '../hooks/usePolling';
import { LoadingSkeleton } from '../components/LoadingSkeleton';
import { EmptyState } from '../components/EmptyState';

interface StatsData {
  system: SystemStats | null;
  summary: StatsSummary | null;
  channels: ChannelStats[];
  costSummary: CostSummary | null;
  costByModel: ModelCostSummary[];
}

function formatUptime(seconds: number): string {
  const d = Math.floor(seconds / 86400);
  const h = Math.floor((seconds % 86400) / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const parts: string[] = [];
  if (d > 0) parts.push(`${d}d`);
  if (h > 0) parts.push(`${h}h`);
  parts.push(`${m}m`);
  return parts.join(' ');
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`;
}

function progressBarColor(pct: number): string {
  if (pct >= 80) return 'bg-red-500';
  if (pct >= 60) return 'bg-yellow-500';
  return 'bg-emerald-500';
}

function Card({ label, value, sub }: { label: string; value: string; sub?: string }) {
  return (
    <div className="rounded border border-bc-border bg-bc-surface p-4">
      <p className="text-xs text-bc-muted uppercase tracking-wide">{label}</p>
      <p className="mt-1 text-2xl font-bold">{value}</p>
      {sub && <p className="mt-0.5 text-xs text-bc-muted">{sub}</p>}
    </div>
  );
}

function ResourceBar({ label, percent, used, total }: { label: string; percent: number; used: string; total: string }) {
  return (
    <div>
      <div className="flex items-center justify-between text-sm mb-1">
        <span className="font-medium">{label}</span>
        <span className="text-bc-muted">{percent.toFixed(1)}% ({used} / {total})</span>
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

function SystemOverview({ system }: { system: SystemStats }) {
  return (
    <div className="grid grid-cols-3 gap-4">
      <div className="rounded border border-bc-border bg-bc-surface p-4">
        <p className="text-xs text-bc-muted uppercase tracking-wide">Hostname</p>
        <p className="mt-1 text-sm font-mono font-medium truncate" title={system.hostname}>{system.hostname}</p>
      </div>
      <div className="rounded border border-bc-border bg-bc-surface p-4">
        <p className="text-xs text-bc-muted uppercase tracking-wide">Platform</p>
        <p className="mt-1 text-sm font-mono font-medium">{system.os}/{system.arch}</p>
      </div>
      <div className="rounded border border-bc-border bg-bc-surface p-4">
        <p className="text-xs text-bc-muted uppercase tracking-wide">Go Version</p>
        <p className="mt-1 text-sm font-mono font-medium">{system.go_version}</p>
      </div>
    </div>
  );
}

function ResourceUsage({ system }: { system: SystemStats }) {
  return (
    <div className="rounded border border-bc-border bg-bc-surface p-4 space-y-4">
      <ResourceBar
        label="CPU"
        percent={system.cpu_usage_percent}
        used={`${system.cpu_usage_percent.toFixed(1)}%`}
        total={`${system.cpus} cores`}
      />
      <ResourceBar
        label="Memory"
        percent={system.memory_usage_percent}
        used={formatBytes(system.memory_used_bytes)}
        total={formatBytes(system.memory_total_bytes)}
      />
      <ResourceBar
        label="Disk"
        percent={system.disk_usage_percent}
        used={formatBytes(system.disk_used_bytes)}
        total={formatBytes(system.disk_total_bytes)}
      />
    </div>
  );
}

function ChannelActivity({ channels }: { channels: ChannelStats[] }) {
  if (channels.length === 0) {
    return (
      <div className="text-sm text-bc-muted py-4 text-center">
        No channel activity data available.
      </div>
    );
  }

  return (
    <div className="rounded border border-bc-border overflow-hidden">
      <table className="w-full text-sm">
        <thead>
          <tr className="bg-bc-surface text-bc-muted text-left">
            <th className="px-4 py-2">Channel</th>
            <th className="px-4 py-2">Messages</th>
            <th className="px-4 py-2">Members</th>
            <th className="px-4 py-2">Top Senders</th>
            <th className="px-4 py-2">Last Activity</th>
          </tr>
        </thead>
        <tbody>
          {channels.map((ch) => (
            <tr key={ch.name} className="border-t border-bc-border/50 hover:bg-bc-surface/50">
              <td className="px-4 py-2 font-medium font-mono">#{ch.name}</td>
              <td className="px-4 py-2 text-bc-muted">{ch.message_count.toLocaleString()}</td>
              <td className="px-4 py-2 text-bc-muted">{ch.member_count}</td>
              <td className="px-4 py-2 text-bc-muted text-xs">
                {ch.top_senders && ch.top_senders.length > 0
                  ? ch.top_senders.slice(0, 3).map((s) => `${s.sender} (${s.count})`).join(', ')
                  : '-'}
              </td>
              <td className="px-4 py-2 text-bc-muted text-xs">
                {ch.last_activity ? new Date(ch.last_activity).toLocaleString() : '-'}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function CostOverview({ summary, models }: { summary: CostSummary; models: ModelCostSummary[] }) {
  const sorted = [...models].sort((a, b) => b.total_cost_usd - a.total_cost_usd);
  const totalCost = sorted.reduce((sum, m) => sum + m.total_cost_usd, 0);

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-3 gap-4">
        <Card label="Total Cost" value={`$${summary.total_cost_usd.toFixed(2)}`} sub={`${summary.record_count} records`} />
        <Card label="Input Tokens" value={summary.input_tokens.toLocaleString()} />
        <Card label="Output Tokens" value={summary.output_tokens.toLocaleString()} />
      </div>
      {sorted.length > 0 && (
        <div className="rounded border border-bc-border bg-bc-surface p-4 space-y-2">
          <p className="text-xs text-bc-muted uppercase tracking-wide mb-3">Cost by Model</p>
          {sorted.map((model) => {
            const pct = totalCost > 0 ? (model.total_cost_usd / totalCost) * 100 : 0;
            return (
              <div key={model.model} className="flex items-center gap-3 text-sm">
                <span className="w-40 truncate font-medium" title={model.model}>
                  {model.model || 'unknown'}
                </span>
                <div className="flex-1 h-2 rounded-full bg-bc-border/40 overflow-hidden">
                  <div
                    className="h-full rounded-full bg-blue-500 transition-all"
                    style={{ width: `${Math.max(pct, 1)}%` }}
                  />
                </div>
                <span className="w-20 text-right text-bc-muted whitespace-nowrap">
                  ${model.total_cost_usd.toFixed(4)}
                </span>
                <span className="w-14 text-right text-xs text-bc-muted">
                  {pct.toFixed(1)}%
                </span>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

export function Stats() {
  const fetcher = useCallback(async (): Promise<StatsData> => {
    let system: SystemStats | null = null;
    let summary: StatsSummary | null = null;
    let channels: ChannelStats[] = [];
    let costSummary: CostSummary | null = null;
    let costByModel: ModelCostSummary[] = [];

    const results = await Promise.allSettled([
      api.getStatsSystem(),
      api.getStatsSummary(),
      api.getStatsChannels(),
      api.getCostSummary(),
      api.getCostByModel(),
    ]);

    if (results[0].status === 'fulfilled') system = results[0].value;
    if (results[1].status === 'fulfilled') summary = results[1].value;
    if (results[2].status === 'fulfilled') channels = results[2].value;
    if (results[3].status === 'fulfilled') costSummary = results[3].value;
    if (results[4].status === 'fulfilled') costByModel = results[4].value;

    return { system, summary, channels, costSummary, costByModel };
  }, []);

  const { data, loading, error, refresh, timedOut } = usePolling(fetcher, 5000);

  if (loading && !data) {
    return (
      <div className="p-6 space-y-6">
        <div className="h-6 w-20 animate-pulse rounded bg-bc-border/50" />
        <LoadingSkeleton variant="cards" rows={3} />
        <LoadingSkeleton variant="table" rows={4} />
      </div>
    );
  }
  if (timedOut && !data) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Stats took too long to load"
          description="The server may be unavailable. Check your connection and try again."
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }
  if (error && !data) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Failed to load stats"
          description={error}
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }
  if (!data) return null;

  const uptime = data.system?.uptime_seconds ?? data.summary?.uptime_seconds ?? 0;

  return (
    <div className="p-6 space-y-6">
      <h1 className="text-xl font-bold">Stats</h1>

      {/* Agent Summary */}
      {data.summary && (
        <section>
          <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide mb-3">Agent Summary</h2>
          <div className="grid grid-cols-4 gap-4">
            <Card label="Total Agents" value={String(data.summary.agents_total)} />
            <Card
              label="Running"
              value={String(data.summary.agents_running)}
              sub={data.summary.agents_total > 0
                ? `${((data.summary.agents_running / data.summary.agents_total) * 100).toFixed(0)}% active`
                : undefined}
            />
            <Card label="Stopped" value={String(data.summary.agents_stopped)} />
            <Card label="Total Cost" value={`$${data.summary.total_cost_usd.toFixed(2)}`} />
          </div>
        </section>
      )}

      {/* System Overview */}
      {data.system && (
        <section>
          <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide mb-3">System Overview</h2>
          <SystemOverview system={data.system} />
        </section>
      )}

      {/* Resource Usage */}
      {data.system && (
        <section>
          <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide mb-3">Resource Usage</h2>
          <ResourceUsage system={data.system} />
        </section>
      )}

      {/* Runtime */}
      <section>
        <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide mb-3">Runtime</h2>
        <div className="grid grid-cols-3 gap-4">
          <Card label="Uptime" value={formatUptime(uptime)} />
          {data.system && <Card label="Goroutines" value={String(data.system.goroutines)} />}
          {data.summary && (
            <Card
              label="Channels / Messages"
              value={String(data.summary.channels_total)}
              sub={`${data.summary.messages_total.toLocaleString()} messages`}
            />
          )}
        </div>
      </section>

      {/* Channel Activity */}
      <section>
        <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide mb-3">Channel Activity</h2>
        <ChannelActivity channels={data.channels} />
      </section>

      {/* Cost Overview */}
      {data.costSummary && (
        <section>
          <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide mb-3">Cost Overview</h2>
          <CostOverview summary={data.costSummary} models={data.costByModel} />
        </section>
      )}
    </div>
  );
}
