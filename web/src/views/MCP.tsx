import { useCallback } from 'react';
import { api } from '../api/client';
import type { MCPServer } from '../api/client';
import { usePolling } from '../hooks/usePolling';
import { Table } from '../components/Table';
import { LoadingSkeleton } from '../components/LoadingSkeleton';
import { EmptyState } from '../components/EmptyState';

export function MCP() {
  const fetcher = useCallback(() => api.listMCP(), []);
  const { data: servers, loading, error, refresh, timedOut } = usePolling(fetcher, 30000);

  if (loading && !servers) {
    return (
      <div className="p-6 space-y-4">
        <div className="h-6 w-32 animate-pulse rounded bg-bc-border/50" />
        <LoadingSkeleton variant="table" rows={3} />
      </div>
    );
  }
  if (timedOut && !servers) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="MCP servers took too long to load"
          description="The server may be unavailable. Check your connection and try again."
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }
  if (error && !servers) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Failed to load MCP servers"
          description={error}
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }

  const columns = [
    { key: 'name', label: 'Name', render: (s: MCPServer) => <span className="font-medium">{s.name}</span> },
    {
      key: 'transport', label: 'Transport', render: (s: MCPServer) => (
        <span className="text-xs px-2 py-0.5 rounded bg-bc-border text-bc-muted uppercase">{s.transport}</span>
      ),
    },
    {
      key: 'endpoint', label: 'Endpoint', render: (s: MCPServer) => (
        <code className="text-xs text-bc-muted">{s.url || s.command || '\u2014'}</code>
      ),
    },
    {
      key: 'enabled', label: 'Status', render: (s: MCPServer) => (
        <span className={s.enabled ? 'text-green-400' : 'text-bc-muted'}>
          {s.enabled ? 'enabled' : 'disabled'}
        </span>
      ),
    },
  ];

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold">MCP Servers</h1>
        <span className="text-sm text-bc-muted">{servers?.length ?? 0} servers</span>
      </div>

      <div className="rounded border border-bc-border overflow-hidden">
        <Table
          columns={columns}
          data={servers ?? []}
          keyFn={(s) => s.name}
          emptyMessage="No MCP servers configured"
          emptyIcon="~"
          emptyDescription="Use 'bc mcp add <name>' to connect an MCP server."
        />
      </div>
    </div>
  );
}
