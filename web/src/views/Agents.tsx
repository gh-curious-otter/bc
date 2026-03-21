import { Fragment, useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../api/client';
import { usePolling } from '../hooks/usePolling';
import { useWebSocket } from '../hooks/useWebSocket';
import { StatusBadge } from '../components/StatusBadge';
import { LoadingSkeleton } from '../components/LoadingSkeleton';
import { EmptyState } from '../components/EmptyState';
import { stripAnsi, truncate } from '../utils/text';

export function Agents() {
  const fetcher = useCallback(async () => {
    const res = await api.listAgents();
    return res;
  }, []);
  const { data: agents, loading, error, refresh, timedOut } = usePolling(fetcher, 5000);
  const { subscribe } = useWebSocket();
  const navigate = useNavigate();

  const [peekAgent, setPeekAgent] = useState<string | null>(null);
  const [peekOutput, setPeekOutput] = useState<string>('');
  const [peekLoading, setPeekLoading] = useState(false);

  // Refresh on agent lifecycle events via SSE
  useEffect(() => {
    const unsubs = [
      subscribe('agent.state_changed', () => void refresh()),
      subscribe('agent.created', () => void refresh()),
      subscribe('agent.stopped', () => void refresh()),
      subscribe('agent.deleted', () => void refresh()),
    ];
    return () => unsubs.forEach((fn) => fn());
  }, [subscribe, refresh]);

  const handlePeekToggle = async (agentName: string, e: React.MouseEvent) => {
    e.stopPropagation();
    if (peekAgent === agentName) {
      setPeekAgent(null);
      setPeekOutput('');
      return;
    }
    setPeekAgent(agentName);
    setPeekLoading(true);
    setPeekOutput('');
    try {
      const data = await api.getAgentPeek(agentName, 10);
      setPeekOutput(stripAnsi(data.output));
    } catch {
      setPeekOutput('Failed to load output.');
    } finally {
      setPeekLoading(false);
    }
  };

  const columns = ['Name', 'Role', 'Tool', 'Status', 'Task', 'Tokens', 'Cost', 'CPU %', 'Mem %', 'MCP', ''] as const;

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

  const agentList = agents ?? [];

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold">Agents</h1>
        <span className="text-sm text-bc-muted">{agentList.length} agents</span>
      </div>

      <div className="rounded border border-bc-border overflow-hidden">
        {agentList.length === 0 ? (
          <EmptyState
            icon=">"
            title="No agents yet"
            description="Create your first agent with 'bc agent create <name> --role <role>'."
          />
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-bc-border text-left">
                <th className="px-4 py-2 font-medium text-bc-muted">Name</th>
                <th className="px-4 py-2 font-medium text-bc-muted">Role</th>
                <th className="px-4 py-2 font-medium text-bc-muted">Tool</th>
                <th className="px-4 py-2 font-medium text-bc-muted">Status</th>
                <th className="px-4 py-2 font-medium text-bc-muted">Task</th>
                <th className="px-4 py-2 font-medium text-bc-muted">Tokens</th>
                <th className="px-4 py-2 font-medium text-bc-muted">Cost</th>
                <th className="px-4 py-2 font-medium text-bc-muted">CPU %</th>
                <th className="px-4 py-2 font-medium text-bc-muted">Mem %</th>
                <th className="px-4 py-2 font-medium text-bc-muted">MCP</th>
                <th className="px-4 py-2 font-medium text-bc-muted w-10"></th>
              </tr>
            </thead>
            <tbody>
              {agentList.map((a) => (
                <Fragment key={a.name}>
                  <tr
                    onClick={() => navigate(`/agents/${encodeURIComponent(a.name)}`)}
                    className="border-b border-bc-border/50 cursor-pointer hover:bg-bc-surface"
                  >
                    <td className="px-4 py-2">
                      <span className="font-medium">{a.name}</span>
                    </td>
                    <td className="px-4 py-2">
                      <span className="text-bc-muted">{a.role}</span>
                    </td>
                    <td className="px-4 py-2">
                      <span className="text-bc-muted">{a.tool || '\u2014'}</span>
                    </td>
                    <td className="px-4 py-2">
                      <StatusBadge status={a.state} />
                    </td>
                    <td className="px-4 py-2">
                      <span className="text-bc-muted" title={a.task}>
                        {a.task ? truncate(a.task, 50) : '\u2014'}
                      </span>
                    </td>
                    <td className="px-4 py-2">
                      <span className="text-bc-muted">
                        {a.total_tokens != null ? a.total_tokens.toLocaleString() : '\u2014'}
                      </span>
                    </td>
                    <td className="px-4 py-2">
                      <span className="text-bc-muted">
                        {a.cost_usd != null ? `$${a.cost_usd.toFixed(4)}` : '\u2014'}
                      </span>
                    </td>
                    {/* TODO: CPU% and Mem% require per-agent /api/agents/{name}/stats calls (N+1).
                        Show "—" until a batch stats endpoint exists. */}
                    <td className="px-4 py-2">
                      <span className="text-bc-muted">{'\u2014'}</span>
                    </td>
                    <td className="px-4 py-2">
                      <span className="text-bc-muted">{'\u2014'}</span>
                    </td>
                    <td className="px-4 py-2">
                      <span className="text-bc-muted">{a.mcp_servers?.length || 0}</span>
                    </td>
                    <td className="px-4 py-2 text-center">
                      <button
                        onClick={(e) => handlePeekToggle(a.name, e)}
                        className={`inline-flex items-center justify-center w-7 h-7 rounded transition-colors ${
                          peekAgent === a.name
                            ? 'bg-bc-accent/20 text-bc-accent'
                            : 'text-bc-muted hover:text-bc-fg hover:bg-bc-surface'
                        }`}
                        title={peekAgent === a.name ? 'Hide output' : 'Peek output'}
                        aria-label={peekAgent === a.name ? 'Hide output' : 'Peek output'}
                      >
                        {peekAgent === a.name ? '\u2296' : '\u2295'}
                      </button>
                    </td>
                  </tr>
                  {peekAgent === a.name && (
                    <tr key={`${a.name}-peek`} className="border-b border-bc-border/50">
                      <td colSpan={columns.length} className="p-0">
                        <div className="bg-bc-bg border-t border-bc-border/30 px-4 py-3">
                          <div className="rounded bg-[#0d1117] border border-bc-border/40 p-3 font-mono text-xs leading-relaxed text-[#c9d1d9] max-h-48 overflow-auto whitespace-pre-wrap">
                            {peekLoading ? (
                              <span className="text-bc-muted animate-pulse">Loading output...</span>
                            ) : (
                              peekOutput || <span className="text-bc-muted">No output available.</span>
                            )}
                          </div>
                          <div className="mt-2 text-right">
                            <button
                              onClick={(e) => {
                                e.stopPropagation();
                                navigate(`/agents/${encodeURIComponent(a.name)}`);
                              }}
                              className="text-xs text-bc-accent hover:underline"
                            >
                              View Detail &rarr;
                            </button>
                          </div>
                        </div>
                      </td>
                    </tr>
                  )}
                </Fragment>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
