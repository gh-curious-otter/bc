import { useCallback, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../api/client';
import type { Agent } from '../api/client';
import { usePolling } from '../hooks/usePolling';
import { useWebSocket } from '../hooks/useWebSocket';
import { StatusBadge } from '../components/StatusBadge';
import { Table } from '../components/Table';
import { LoadingSkeleton } from '../components/LoadingSkeleton';
import { EmptyState } from '../components/EmptyState';

export function Agents() {
  const fetcher = useCallback(async () => {
    const res = await api.listAgents();
    return res;
  }, []);
  const { data: agents, loading, error, refresh, timedOut } = usePolling(fetcher, 5000);
  const { subscribe } = useWebSocket();
  const navigate = useNavigate();

  // Refresh on agent state changes — cleanup prevents listener leak
  useEffect(() => {
    return subscribe('agent.state_changed', () => void refresh());
  }, [subscribe, refresh]);

  const columns = [
    { key: 'name', label: 'Name', render: (a: Agent) => <span className="font-medium">{a.name}</span> },
    { key: 'role', label: 'Role', render: (a: Agent) => <span className="text-bc-muted">{a.role}</span> },
    { key: 'tool', label: 'Tool', render: (a: Agent) => <span className="text-bc-muted">{a.tool || '\u2014'}</span> },
    { key: 'state', label: 'Status', render: (a: Agent) => <StatusBadge status={a.state} /> },
    {
      key: 'cost', label: 'Cost', render: (a: Agent) => (
        <span className="text-bc-muted">{a.cost_usd != null ? `$${a.cost_usd.toFixed(4)}` : '\u2014'}</span>
      ),
    },
  ];

  if (loading && !agents) {
    return (
      <div className="p-6 space-y-4">
        <div className="h-6 w-24 animate-pulse rounded bg-bc-border/50" />
        <LoadingSkeleton variant="table" rows={4} />
      </div>
    );
  }
  if (timedOut && !agents) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Agents took too long to load"
          description="The server may be unavailable. Check your connection and try again."
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }
  if (error && !agents) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Failed to load agents"
          description={error}
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold">Agents</h1>
        <span className="text-sm text-bc-muted">{agents?.length ?? 0} agents</span>
      </div>

      <div className="rounded border border-bc-border overflow-hidden">
        <Table
          columns={columns}
          data={agents ?? []}
          keyFn={(a) => a.name}
          onRowClick={(a) => navigate(`/agents/${encodeURIComponent(a.name)}`)}
          emptyMessage="No agents yet"
          emptyIcon=">"
          emptyDescription="Create your first agent with 'bc agent create <name> --role <role>'."
        />
      </div>

    </div>
  );
}
