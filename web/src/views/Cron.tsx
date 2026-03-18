import { useCallback } from 'react';
import { api } from '../api/client';
import type { CronJob } from '../api/client';
import { usePolling } from '../hooks/usePolling';
import { Table } from '../components/Table';

export function Cron() {
  const fetcher = useCallback(() => api.listCron(), []);
  const { data: jobs, loading, error } = usePolling(fetcher, 10000);

  if (loading && !jobs) {
    return <div className="p-6 text-bc-muted">Loading cron jobs...</div>;
  }
  if (error && !jobs) {
    return <div className="p-6 text-bc-error">Error: {error}</div>;
  }

  const columns = [
    { key: 'name', label: 'Name', render: (j: CronJob) => <span className="font-medium">{j.name}</span> },
    { key: 'schedule', label: 'Schedule', render: (j: CronJob) => <code className="text-xs text-bc-muted">{j.schedule}</code> },
    { key: 'agent', label: 'Agent', render: (j: CronJob) => <span>{j.agent_name || '—'}</span> },
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
          emptyMessage="No cron jobs. Use 'bc cron add' to create one."
        />
      </div>
    </div>
  );
}
