import { useCallback } from 'react';
import { api } from '../api/client';
import type { CronJob } from '../api/client';
import { usePolling } from '../hooks/usePolling';
import { Table } from '../components/Table';
import { LoadingSkeleton } from '../components/LoadingSkeleton';
import { EmptyState } from '../components/EmptyState';

export function Cron() {
  const fetcher = useCallback(() => api.listCron(), []);
  const { data: jobs, loading, error, refresh, timedOut } = usePolling(fetcher, 10000);

  if (loading && !jobs) {
    return (
      <div className="p-6 space-y-4">
        <div className="h-6 w-28 animate-pulse rounded bg-bc-border/50" />
        <LoadingSkeleton variant="table" rows={3} />
      </div>
    );
  }
  if (timedOut && !jobs) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Cron jobs took too long to load"
          description="The server may be unavailable. Check your connection and try again."
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }
  if (error && !jobs) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Failed to load cron jobs"
          description={error}
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }

  const columns = [
    { key: 'name', label: 'Name', render: (j: CronJob) => <span className="font-medium">{j.name}</span> },
    { key: 'schedule', label: 'Schedule', render: (j: CronJob) => <code className="text-xs text-bc-muted">{j.schedule}</code> },
    { key: 'agent', label: 'Agent', render: (j: CronJob) => <span>{j.agent_name || '\u2014'}</span> },
    {
      key: 'enabled', label: 'Status', render: (j: CronJob) => (
        <span className={j.enabled ? 'text-green-400' : 'text-bc-muted'}>
          {j.enabled ? 'enabled' : 'disabled'}
        </span>
      ),
    },
    { key: 'runs', label: 'Runs', render: (j: CronJob) => <span className="text-bc-muted">{j.run_count}</span> },
    {
      key: 'last_run', label: 'Last Run', render: (j: CronJob) => (
        <span className="text-xs text-bc-muted">
          {j.last_run ? new Date(j.last_run).toLocaleString() : 'never'}
        </span>
      ),
    },
  ];

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold">Cron Jobs</h1>
        <span className="text-sm text-bc-muted">{jobs?.length ?? 0} jobs</span>
      </div>

      <div className="rounded border border-bc-border overflow-hidden">
        <Table
          columns={columns}
          data={jobs ?? []}
          keyFn={(j) => j.name}
          emptyMessage="No cron jobs"
          emptyIcon="~"
          emptyDescription="Use 'bc cron add <name>' to schedule recurring tasks."
        />
      </div>
    </div>
  );
}
