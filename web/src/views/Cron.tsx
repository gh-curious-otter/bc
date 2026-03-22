import { useCallback, useState } from 'react';
import { api } from '../api/client';
import type { CronJob, CronLogEntry } from '../api/client';
import { usePolling } from '../hooks/usePolling';
import { Table } from '../components/Table';
import { LoadingSkeleton } from '../components/LoadingSkeleton';
import { EmptyState } from '../components/EmptyState';

function CreateCronForm({ onSave, onCancel }: { onSave: () => void; onCancel: () => void }) {
  const [name, setName] = useState('');
  const [schedule, setSchedule] = useState('');
  const [command, setCommand] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim() || !schedule.trim() || !command.trim()) return;
    setSaving(true);
    setError(null);
    try {
      await api.createCron({ name: name.trim(), schedule: schedule.trim(), command: command.trim() });
      onSave();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create cron job');
    } finally {
      setSaving(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="rounded border border-bc-border bg-bc-bg-secondary p-4 space-y-3">
      <div className="flex items-center gap-3">
        <input
          type="text"
          placeholder="Job name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          className="flex-1 rounded border border-bc-border bg-bc-bg px-3 py-1.5 text-sm text-bc-text placeholder:text-bc-muted focus:outline-none focus:border-bc-accent"
          autoFocus
        />
        <input
          type="text"
          placeholder="Schedule (e.g. */5 * * * *)"
          value={schedule}
          onChange={(e) => setSchedule(e.target.value)}
          className="flex-1 rounded border border-bc-border bg-bc-bg px-3 py-1.5 text-sm text-bc-text placeholder:text-bc-muted focus:outline-none focus:border-bc-accent"
        />
        <input
          type="text"
          placeholder="Command"
          value={command}
          onChange={(e) => setCommand(e.target.value)}
          className="flex-[2] rounded border border-bc-border bg-bc-bg px-3 py-1.5 text-sm text-bc-text placeholder:text-bc-muted focus:outline-none focus:border-bc-accent"
        />
      </div>
      {error && <p className="text-sm text-red-400">{error}</p>}
      <div className="flex items-center gap-2">
        <button
          type="submit"
          disabled={saving || !name.trim() || !schedule.trim() || !command.trim()}
          className="rounded bg-bc-accent px-3 py-1.5 text-sm font-medium text-white hover:opacity-90 disabled:opacity-50"
        >
          {saving ? 'Saving...' : 'Save'}
        </button>
        <button
          type="button"
          onClick={onCancel}
          className="rounded border border-bc-border px-3 py-1.5 text-sm text-bc-muted hover:text-bc-text"
        >
          Cancel
        </button>
      </div>
    </form>
  );
}

function CronLogs({ jobName }: { jobName: string }) {
  const fetcher = useCallback(() => api.getCronLogs(jobName), [jobName]);
  const { data: logs, loading, error } = usePolling(fetcher, 15000);

  if (loading && !logs) {
    return <div className="px-4 py-2 text-sm text-bc-muted">Loading logs...</div>;
  }
  if (error) {
    return <div className="px-4 py-2 text-sm text-red-400">Failed to load logs: {error}</div>;
  }
  if (!logs || logs.length === 0) {
    return <div className="px-4 py-2 text-sm text-bc-muted">No execution logs yet.</div>;
  }

  return (
    <div className="border-t border-bc-border bg-bc-bg-secondary">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-bc-border text-left text-xs text-bc-muted">
            <th className="px-4 py-1.5">Status</th>
            <th className="px-4 py-1.5">Started</th>
            <th className="px-4 py-1.5">Duration</th>
            <th className="px-4 py-1.5">Output</th>
          </tr>
        </thead>
        <tbody>
          {logs.map((log: CronLogEntry) => (
            <tr key={log.id} className="border-b border-bc-border/50 last:border-0">
              <td className="px-4 py-1.5">
                <span className={log.status === 'success' ? 'text-green-400' : log.status === 'running' ? 'text-yellow-400' : 'text-red-400'}>
                  {log.status}
                </span>
              </td>
              <td className="px-4 py-1.5 text-bc-muted">{new Date(log.started_at).toLocaleString()}</td>
              <td className="px-4 py-1.5 text-bc-muted">{log.duration_ms}ms</td>
              <td className="px-4 py-1.5 text-bc-muted truncate max-w-xs">
                {log.error || log.output || '\u2014'}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export function Cron() {
  const fetcher = useCallback(() => api.listCron(), []);
  const { data: jobs, loading, error, refresh, timedOut } = usePolling(fetcher, 10000);
  const [showCreate, setShowCreate] = useState(false);
  const [expandedJob, setExpandedJob] = useState<string | null>(null);
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  const handleAction = async (action: () => Promise<unknown>, key: string) => {
    setActionLoading(key);
    try {
      await action();
      refresh();
    } catch {
      // Error is transient; refresh will show current state
    } finally {
      setActionLoading(null);
    }
  };

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
    {
      key: 'name', label: 'Name', render: (j: CronJob) => (
        <button
          type="button"
          onClick={() => setExpandedJob(expandedJob === j.name ? null : j.name)}
          className="font-medium text-bc-accent hover:underline cursor-pointer bg-transparent border-none p-0"
        >
          {expandedJob === j.name ? '\u25BC' : '\u25B6'} {j.name}
        </button>
      ),
    },
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
    {
      key: 'actions', label: 'Actions', render: (j: CronJob) => (
        <div className="flex items-center gap-1">
          <button
            type="button"
            title="Run now"
            disabled={actionLoading === `run-${j.name}`}
            onClick={() => handleAction(() => api.runCron(j.name), `run-${j.name}`)}
            className="rounded border border-bc-border px-2 py-0.5 text-xs text-bc-muted hover:text-bc-text hover:border-bc-accent disabled:opacity-50"
          >
            {actionLoading === `run-${j.name}` ? '...' : 'Run'}
          </button>
          <button
            type="button"
            title={j.enabled ? 'Disable' : 'Enable'}
            disabled={actionLoading === `toggle-${j.name}`}
            onClick={() => handleAction(
              () => j.enabled ? api.disableCron(j.name) : api.enableCron(j.name),
              `toggle-${j.name}`,
            )}
            className="rounded border border-bc-border px-2 py-0.5 text-xs text-bc-muted hover:text-bc-text hover:border-bc-accent disabled:opacity-50"
          >
            {actionLoading === `toggle-${j.name}` ? '...' : j.enabled ? 'Disable' : 'Enable'}
          </button>
          {deleteConfirm === j.name ? (
            <>
              <button
                type="button"
                onClick={() => { handleAction(() => api.deleteCron(j.name), `del-${j.name}`); setDeleteConfirm(null); }}
                className="rounded border border-red-500 px-2 py-0.5 text-xs text-red-400 hover:bg-red-500/10"
              >
                Confirm
              </button>
              <button
                type="button"
                onClick={() => setDeleteConfirm(null)}
                className="rounded border border-bc-border px-2 py-0.5 text-xs text-bc-muted hover:text-bc-text"
              >
                No
              </button>
            </>
          ) : (
            <button
              type="button"
              title="Delete"
              onClick={() => setDeleteConfirm(j.name)}
              className="rounded border border-bc-border px-2 py-0.5 text-xs text-red-400 hover:border-red-500 hover:bg-red-500/10 disabled:opacity-50"
            >
              Delete
            </button>
          )}
        </div>
      ),
    },
  ];

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold">Cron Jobs</h1>
        <div className="flex items-center gap-3">
          <span className="text-sm text-bc-muted">{jobs?.length ?? 0} jobs</span>
          <button
            type="button"
            onClick={() => setShowCreate(!showCreate)}
            className="rounded bg-bc-accent px-3 py-1.5 text-sm font-medium text-white hover:opacity-90"
          >
            {showCreate ? 'Cancel' : '+ Create'}
          </button>
        </div>
      </div>

      {showCreate && (
        <CreateCronForm
          onSave={() => { setShowCreate(false); refresh(); }}
          onCancel={() => setShowCreate(false)}
        />
      )}

      <div className="rounded border border-bc-border overflow-hidden">
        <Table
          columns={columns}
          data={jobs ?? []}
          keyFn={(j) => j.name}
          emptyMessage="No cron jobs"
          emptyIcon="~"
          emptyDescription="Use 'bc cron add <name>' to schedule recurring tasks."
          renderRowExpansion={(j: CronJob) =>
            expandedJob === j.name ? <CronLogs jobName={j.name} /> : null
          }
        />
      </div>
    </div>
  );
}
