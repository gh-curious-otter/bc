import { useCallback } from 'react';
import { api } from '../api/client';
import type { CostSummary, AgentCostSummary } from '../api/client';
import { usePolling } from '../hooks/usePolling';
import { Table } from '../components/Table';

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

  const { data, loading, error } = usePolling(fetcher, 10000);

  if (loading && !data) {
    return <div className="p-6 text-bc-muted">Loading costs...</div>;
  }
  if (error && !data) {
    return <div className="p-6 text-bc-error">Error: {error}</div>;
  }
  if (!data) return null;

  const columns = [
    {
      key: 'agent', label: 'Agent',
      render: (r: AgentCostSummary) => <span className="font-medium">{r.agent_id}</span>,
    },
    {
      key: 'cost', label: 'Cost',
      render: (r: AgentCostSummary) => <span>${r.total_cost.toFixed(4)}</span>,
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
            emptyMessage="No cost records yet."
          />
        </div>
      </section>
    </div>
  );
}
