import { useCallback } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../api/client';
import type { Agent, CostSummary, Channel } from '../api/client';
import { usePolling } from '../hooks/usePolling';
import { StatusBadge } from '../components/StatusBadge';

interface DashData {
  agents: Agent[];
  channels: Channel[];
  costs: CostSummary;
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

export function Dashboard() {
  const fetcher = useCallback(async (): Promise<DashData> => {
    const [agentsRes, channelsRes, costs] = await Promise.all([
      api.listAgents(),
      api.listChannels(),
      api.getCostSummary(),
    ]);
    return { agents: agentsRes.agents, channels: channelsRes.channels, costs };
  }, []);

  const { data, loading, error } = usePolling(fetcher, 5000);

  if (loading && !data) {
    return <div className="p-6 text-bc-muted">Loading dashboard...</div>;
  }
  if (error && !data) {
    return <div className="p-6 text-bc-error">Error: {error}</div>;
  }
  if (!data) return null;

  const activeAgents = data.agents.filter((a) => a.state !== 'stopped');

  return (
    <div className="p-6 space-y-6">
      <h1 className="text-xl font-bold">Dashboard</h1>

      <div className="grid grid-cols-4 gap-4">
        <Card label="Active Agents" value={String(activeAgents.length)} sub={`${data.agents.length} total`} />
        <Card label="Channels" value={String(data.channels.length)} />
        <Card label="Today" value={`$${data.costs.today_cost.toFixed(2)}`} />
        <Card label="This Month" value={`$${data.costs.month_cost.toFixed(2)}`} sub={`$${data.costs.all_time_cost.toFixed(2)} all-time`} />
      </div>

      <section>
        <div className="flex items-center justify-between mb-2">
          <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide">Agents</h2>
          <Link to="/agents" className="text-xs text-bc-accent hover:underline">View all</Link>
        </div>
        {data.agents.length === 0 ? (
          <p className="text-sm text-bc-muted">No agents yet.</p>
        ) : (
          <div className="rounded border border-bc-border overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="bg-bc-surface text-bc-muted text-left">
                  <th className="px-4 py-2">Name</th>
                  <th className="px-4 py-2">Role</th>
                  <th className="px-4 py-2">Tool</th>
                  <th className="px-4 py-2">Status</th>
                </tr>
              </thead>
              <tbody>
                {data.agents.slice(0, 8).map((a) => (
                  <tr key={a.name} className="border-t border-bc-border/50 hover:bg-bc-surface/50">
                    <td className="px-4 py-2 font-medium">{a.name}</td>
                    <td className="px-4 py-2 text-bc-muted">{a.role}</td>
                    <td className="px-4 py-2 text-bc-muted">{a.tool}</td>
                    <td className="px-4 py-2"><StatusBadge status={a.state} /></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
}
