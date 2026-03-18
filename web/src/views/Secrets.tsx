import { useCallback } from 'react';
import { api } from '../api/client';
import type { Secret } from '../api/client';
import { usePolling } from '../hooks/usePolling';
import { Table } from '../components/Table';

export function Secrets() {
  const fetcher = useCallback(() => api.listSecrets(), []);
  const { data: secrets, loading, error } = usePolling(fetcher, 30000);

  if (loading && !secrets) {
    return <div className="p-6 text-bc-muted">Loading secrets...</div>;
  }
  if (error && !secrets) {
    return <div className="p-6 text-bc-error">Error: {error}</div>;
  }

  const columns = [
    { key: 'name', label: 'Name', render: (s: Secret) => <span className="font-medium">{s.name}</span> },
    { key: 'desc', label: 'Description', render: (s: Secret) => <span className="text-bc-muted">{s.description || '—'}</span> },
    { key: 'backend', label: 'Backend', render: (s: Secret) => <code className="text-xs text-bc-muted">{s.backend}</code> },
    {
      key: 'created', label: 'Created', render: (s: Secret) => (
        <span className="text-xs text-bc-muted">
          {s.created_at ? new Date(s.created_at).toLocaleDateString() : '—'}
        </span>
      ),
    },
  ];

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold">Secrets</h1>
        <span className="text-sm text-bc-muted">{secrets?.length ?? 0} secrets</span>
      </div>

      <p className="text-xs text-bc-muted">Secret values are never shown. Only metadata is displayed.</p>

      <div className="rounded border border-bc-border overflow-hidden">
        <Table
          columns={columns}
          data={secrets ?? []}
          keyFn={(s) => s.name}
          emptyMessage="No secrets. Use 'bc secret set <name> --value <value>' to add one."
        />
      </div>
    </div>
  );
}
