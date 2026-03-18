import { useCallback } from 'react';
import { api } from '../api/client';
import type { Tool } from '../api/client';
import { usePolling } from '../hooks/usePolling';
import { Table } from '../components/Table';

export function Tools() {
  const fetcher = useCallback(() => api.listTools(), []);
  const { data: tools, loading, error } = usePolling(fetcher, 30000);

  if (loading && !tools) {
    return <div className="p-6 text-bc-muted">Loading tools...</div>;
  }
  if (error && !tools) {
    return <div className="p-6 text-bc-error">Error: {error}</div>;
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
        <code className="text-xs text-bc-muted">{t.install_cmd || '—'}</code>
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
          emptyMessage="No tools configured."
        />
      </div>
    </div>
  );
}
