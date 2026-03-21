import { useCallback } from 'react';
import { api } from '../api/client';
import type { Team } from '../api/client';
import { usePolling } from '../hooks/usePolling';
import { Table } from '../components/Table';
import { LoadingSkeleton } from '../components/LoadingSkeleton';
import { EmptyState } from '../components/EmptyState';

export function Teams() {
  const fetcher = useCallback(() => api.listTeams(), []);
  const { data: teams, loading, error, refresh, timedOut } = usePolling(fetcher, 30000);

  if (loading && !teams) {
    return (
      <div className="p-6 space-y-4">
        <div className="h-6 w-20 animate-pulse rounded bg-bc-border/50" />
        <LoadingSkeleton variant="table" rows={4} />
      </div>
    );
  }
  if (timedOut && !teams) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Teams took too long to load"
          description="The server may be unavailable. Check your connection and try again."
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }
  if (error && !teams) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Failed to load teams"
          description={error}
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }

  const columns = [
    { key: 'name', label: 'Name', render: (t: Team) => <span className="font-medium">{t.name}</span> },
    { key: 'description', label: 'Description', render: (t: Team) => <span className="text-sm text-bc-muted">{t.description || '\u2014'}</span> },
    { key: 'lead', label: 'Lead', render: (t: Team) => <span className="text-sm">{t.lead || '\u2014'}</span> },
    {
      key: 'members', label: 'Members', render: (t: Team) => (
        <span className="text-sm text-bc-muted">{t.members?.length ?? 0}</span>
      ),
    },
    {
      key: 'created_at', label: 'Created', render: (t: Team) => (
        <span className="text-xs text-bc-muted">{new Date(t.created_at).toLocaleDateString()}</span>
      ),
    },
  ];

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold">Teams</h1>
        <span className="text-sm text-bc-muted">{teams?.length ?? 0} teams</span>
      </div>

      <div className="rounded border border-bc-border overflow-hidden">
        <Table
          columns={columns}
          data={teams ?? []}
          keyFn={(t) => t.name}
          emptyMessage="No teams configured"
          emptyIcon="G"
          emptyDescription="Create teams with bc team create <name>."
        />
      </div>
    </div>
  );
}
