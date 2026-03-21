import { useCallback } from 'react';
import { api } from '../api/client';
import type { Secret } from '../api/client';
import { usePolling } from '../hooks/usePolling';
import { Table } from '../components/Table';
import { LoadingSkeleton } from '../components/LoadingSkeleton';
import { EmptyState } from '../components/EmptyState';

export function Secrets() {
  const fetcher = useCallback(() => api.listSecrets(), []);
  const { data: secrets, loading, error, refresh, timedOut } = usePolling(fetcher, 30000);

  if (loading && !secrets) {
    return (
      <div className="p-6 space-y-4">
        <div className="h-6 w-24 animate-pulse rounded bg-bc-border/50" />
        <LoadingSkeleton variant="table" rows={3} />
      </div>
    );
  }
  if (timedOut && !secrets) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Secrets took too long to load"
          description="The server may be unavailable. Check your connection and try again."
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }
  if (error && !secrets) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Failed to load secrets"
          description={error}
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }

  const columns = [
    { key: 'name', label: 'Name', render: (s: Secret) => <span className="font-medium">{s.name}</span> },
    { key: 'desc', label: 'Description', render: (s: Secret) => <span className="text-bc-muted">{s.description || '\u2014'}</span> },
    { key: 'backend', label: 'Backend', render: (s: Secret) => <code className="text-xs text-bc-muted">{s.backend}</code> },
    {
      key: 'created', label: 'Created', render: (s: Secret) => (
        <span className="text-xs text-bc-muted">
          {s.created_at ? new Date(s.created_at).toLocaleDateString() : '\u2014'}
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
          emptyMessage="No secrets stored"
          emptyIcon="*"
          emptyDescription="Use 'bc secret set <name> --value <value>' to store a secret."
        />
      </div>
    </div>
  );
}
