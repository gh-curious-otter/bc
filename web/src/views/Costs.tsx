import { useCallback, useEffect } from 'react';
import { api } from '../api/client';
import type { CostSummary, AgentCostSummary } from '../api/client';
import { usePolling } from '../hooks/usePolling';
import { useWebSocket } from '../hooks/useWebSocket';
import { Table } from '../components/Table';
import { LoadingSkeleton } from '../components/LoadingSkeleton';
import { EmptyState } from '../components/EmptyState';

interface CostData {
  summary: CostSummary;
  byAgent: AgentCostSummary[];
}

function CostCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded border border-bc-border bg-bc-surface p-4 text-center">
      <p className="text-xs text-bc-muted uppercase tracking-wide">{label}</p>
      <p className="mt-1 text-xl font-bold">{value}</p>
    </div>
  );
}

export function Costs() {
  const fetcher = useCallback(async (): Promise<CostData> => {
    let summary: CostSummary = { input_tokens: 0, output_tokens: 0, total_tokens: 0, total_cost_usd: 0, record_count: 0 };
    let byAgent: AgentCostSummary[] = [];
    try {
      summary = await api.getCostSummary();
    } catch {
      // cost summary unavailable
    }
    try {
      byAgent = await api.getCostByAgent();
    } catch {
      // per-agent costs unavailable
    }
    return { summary, byAgent };
  }, []);

  const { data, loading, error, refresh, timedOut } = usePolling(fetcher, 10000);
  const { subscribe } = useWebSocket();

  // Refresh cost data in real-time via SSE
  useEffect(() => {
    return subscribe('cost.updated', () => void refresh());
  }, [subscribe, refresh]);

  if (loading && !data) {
    return (
      <div className="p-6 space-y-6">
        <div className="h-6 w-20 animate-pulse rounded bg-bc-border/50" />
        <LoadingSkeleton variant="cards" rows={3} />
        <LoadingSkeleton variant="table" rows={3} />
      </div>
    );
  }
  if (timedOut && !data) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Costs took too long to load"
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
          title="Failed to load costs"
          description={error}
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }
  if (!data) return null;

  const columns = [
    {
      key: 'agent', label: 'Agent',
      render: (r: AgentCostSummary) => <span className="font-medium">{r.agent_id}</span>,
    },
    {
      key: 'cost', label: 'Cost',
      render: (r: AgentCostSummary) => <span>${r.total_cost_usd.toFixed(4)}</span>,
    },
    {
      key: 'input', label: 'Input Tokens',
      render: (r: AgentCostSummary) => <span className="text-bc-muted">{r.input_tokens.toLocaleString()}</span>,
    },
    {
      key: 'output', label: 'Output Tokens',
      render: (r: AgentCostSummary) => <span className="text-bc-muted">{r.output_tokens.toLocaleString()}</span>,
    },
    {
      key: 'records', label: 'Records',
      render: (r: AgentCostSummary) => <span className="text-bc-muted">{r.record_count}</span>,
    },
  ];

  return (
    <div className="p-6 space-y-6">
      <h1 className="text-xl font-bold">Costs</h1>

      <div className="grid grid-cols-3 gap-4">
        <CostCard label="Total Cost" value={`$${(data.summary?.total_cost_usd ?? 0).toFixed(2)}`} />
        <CostCard label="Total Tokens" value={(data.summary?.total_tokens ?? 0).toLocaleString()} />
        <CostCard label="Records" value={String(data.summary?.record_count ?? 0)} />
      </div>

      <section>
        <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide mb-2">By Agent</h2>
        <div className="rounded border border-bc-border overflow-hidden">
          <Table
            columns={columns}
            data={data.byAgent}
            keyFn={(r) => r.agent_id}
            emptyMessage="No cost records yet"
            emptyIcon="$"
            emptyDescription="Cost data will appear here once agents start running and using tokens."
          />
        </div>
      </section>
    </div>
  );
}
