import { useCallback } from 'react';
import { api } from '../api/client';
import type { Tool } from '../api/client';
import { usePolling } from '../hooks/usePolling';
import { Table } from '../components/Table';
import { LoadingSkeleton } from '../components/LoadingSkeleton';
import { EmptyState } from '../components/EmptyState';

export function Tools() {
  const fetcher = useCallback(() => api.listTools(), []);
  const { data: tools, loading, error, refresh, timedOut } = usePolling(fetcher, 30000);

  if (loading && !tools) {
    return (
      <div className="p-6 space-y-4">
        <div className="h-6 w-20 animate-pulse rounded bg-bc-border/50" />
        <LoadingSkeleton variant="table" rows={4} />
      </div>
    );
  }
  if (timedOut && !tools) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Tools took too long to load"
          description="The server may be unavailable. Check your connection and try again."
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }
  if (error && !tools) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Failed to load tools"
          description={error}
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }

  const columns = [
    { key: 'name', label: 'Name', render: (t: Tool) => <span className="font-medium">{t.name}</span> },
    { key: 'command', label: 'Command', render: (t: Tool) => <code className="text-xs text-bc-muted">{t.command}</code> },
    {
      key: 'enabled', label: 'Enabled', render: (t: Tool) => (
        <span className={t.enabled ? 'text-green-400' : 'text-bc-muted'}>
          {t.enabled ? 'yes' : 'no'}
        </span>
      ),
    },
    {
      key: 'builtin', label: 'Type', render: (t: Tool) => (
        <span className="text-xs text-bc-muted">{t.builtin ? 'built-in' : 'custom'}</span>
      ),
    },
    {
      key: 'install', label: 'Install', render: (t: Tool) => (
        <code className="text-xs text-bc-muted">{t.install_cmd || '\u2014'}</code>
      ),
    },
  ];

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold">Tools</h1>
        <span className="text-sm text-bc-muted">{tools?.length ?? 0} tools</span>
      </div>

      <div className="rounded border border-bc-border overflow-hidden">
        <Table
          columns={columns}
          data={tools ?? []}
          keyFn={(t) => t.name}
          emptyMessage="No tools configured"
          emptyIcon="*"
          emptyDescription="Add tools in your config.toml [tools] section."
        />
      </div>
    </div>
  );
}
