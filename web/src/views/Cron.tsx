import { Fragment, useCallback, useState } from 'react';
import { api } from '../api/client';
import type { CronJob, CronLogEntry } from '../api/client';
import { usePolling } from '../hooks/usePolling';
import { LoadingSkeleton } from '../components/LoadingSkeleton';
import { EmptyState } from '../components/EmptyState';

type FormStatus = { type: 'idle' } | { type: 'saving' } | { type: 'success' } | { type: 'error'; message: string };

function CreateJobForm({ agents, onCreated }: { agents: string[]; onCreated: () => void }) {
  const [open, setOpen] = useState(false);
  const [name, setName] = useState('');
  const [schedule, setSchedule] = useState('');
  const [agentName, setAgentName] = useState('');
  const [prompt, setPrompt] = useState('');
  const [status, setStatus] = useState<FormStatus>({ type: 'idle' });

  const reset = () => {
    setName('');
    setSchedule('');
    setAgentName('');
    setPrompt('');
    setStatus({ type: 'idle' });
  };

  const handleSubmit = async () => {
    if (!name.trim() || !schedule.trim()) return;
    setStatus({ type: 'saving' });
    try {
      await api.createCron({ name: name.trim(), schedule: schedule.trim(), agent_name: agentName, prompt: prompt.trim() });
      setStatus({ type: 'success' });
      reset();
      setOpen(false);
      onCreated();
    } catch (err) {
      setStatus({ type: 'error', message: err instanceof Error ? err.message : 'Failed to create job' });
      setTimeout(() => setStatus({ type: 'idle' }), 4000);
    }
  };

  if (!open) {
    return (
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="px-4 py-2 rounded bg-bc-accent text-white text-sm font-medium hover:opacity-90 transition-opacity"
      >
        + New Job
      </button>
    );
  }

  const inputCls = 'w-full px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent';

  return (
    <div className="rounded border border-bc-border bg-bc-surface p-5 space-y-4">
      <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide">New Cron Job</h2>
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div className="space-y-1">
          <label className="block text-sm text-bc-text">Name</label>
          <input type="text" value={name} onChange={(e) => setName(e.target.value)} placeholder="nightly-review" className={inputCls} />
        </div>
        <div className="space-y-1">
          <label className="block text-sm text-bc-text">Schedule (cron)</label>
          <input type="text" value={schedule} onChange={(e) => setSchedule(e.target.value)} placeholder="0 0 * * *" className={inputCls} />
        </div>
        <div className="space-y-1">
          <label className="block text-sm text-bc-text">Agent</label>
          <select value={agentName} onChange={(e) => setAgentName(e.target.value)} className={inputCls}>
            <option value="">-- none --</option>
            {agents.map((a) => <option key={a} value={a}>{a}</option>)}
          </select>
        </div>
      </div>
      <div className="space-y-1">
        <label className="block text-sm text-bc-text">Prompt</label>
        <textarea
          value={prompt}
          onChange={(e) => setPrompt(e.target.value)}
          rows={3}
          placeholder="Describe what this cron job should do..."
          className={inputCls + ' resize-y'}
        />
      </div>
      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={handleSubmit}
          disabled={status.type === 'saving' || !name.trim() || !schedule.trim()}
          className="px-4 py-2 rounded bg-bc-accent text-white text-sm font-medium hover:opacity-90 disabled:opacity-50 transition-opacity"
        >
          {status.type === 'saving' ? 'Creating...' : 'Create'}
        </button>
        <button
          type="button"
          onClick={() => { reset(); setOpen(false); }}
          className="px-4 py-2 rounded border border-bc-border text-bc-muted text-sm hover:text-bc-text transition-colors"
        >
          Cancel
        </button>
        {status.type === 'error' && <span className="text-xs text-red-400">{status.message}</span>}
      </div>
    </div>
  );
}

function JobLogs({ name }: { name: string }) {
  const fetcher = useCallback(() => api.getCronLogs(name, 20), [name]);
  const { data: logs, loading } = usePolling(fetcher, 15000);

  if (loading && !logs) {
    return <div className="px-6 py-3 text-xs text-bc-muted">Loading logs...</div>;
  }

  if (!logs || logs.length === 0) {
    return <div className="px-6 py-3 text-xs text-bc-muted">No execution logs yet.</div>;
  }

  return (
    <div className="px-6 py-3 space-y-2">
      <div className="text-xs font-medium text-bc-muted uppercase tracking-wide mb-2">Execution Logs</div>
      <div className="max-h-64 overflow-y-auto space-y-1">
        {logs.map((log: CronLogEntry) => (
          <div key={log.id} className="flex items-start gap-3 text-xs border-b border-bc-border/30 pb-1">
            <span className="text-bc-muted whitespace-nowrap">{new Date(log.run_at).toLocaleString()}</span>
            <span className={`font-medium ${log.status === 'success' ? 'text-green-400' : log.status === 'failed' ? 'text-red-400' : 'text-yellow-400'}`}>
              {log.status}
            </span>
            {log.output && (
              <span className="text-bc-muted truncate flex-1" title={log.output}>{log.output}</span>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}

function JobActions({ job, onRefresh }: { job: CronJob; onRefresh: () => void }) {
  const [busy, setBusy] = useState<string | null>(null);

  const act = async (action: string, fn: () => Promise<unknown>) => {
    setBusy(action);
    try {
      await fn();
      onRefresh();
    } catch {
      // Silently fail, user will see stale state until next poll
    } finally {
      setBusy(null);
    }
  };

  const btnCls = 'px-2.5 py-1 rounded text-xs font-medium transition-colors disabled:opacity-50';

  return (
    <div className="flex items-center gap-1.5">
      <button
        type="button"
        onClick={(e) => { e.stopPropagation(); act('toggle', () => job.enabled ? api.disableCron(job.name) : api.enableCron(job.name)); }}
        disabled={busy !== null}
        className={`${btnCls} ${job.enabled ? 'bg-yellow-400/10 text-yellow-400 hover:bg-yellow-400/20' : 'bg-green-400/10 text-green-400 hover:bg-green-400/20'}`}
      >
        {busy === 'toggle' ? '...' : job.enabled ? 'Disable' : 'Enable'}
      </button>
      {job.enabled && (
        <button
          type="button"
          onClick={(e) => { e.stopPropagation(); act('run', () => api.runCron(job.name)); }}
          disabled={busy !== null}
          className={`${btnCls} bg-bc-accent/10 text-bc-accent hover:bg-bc-accent/20`}
        >
          {busy === 'run' ? '...' : 'Run'}
        </button>
      )}
      <button
        type="button"
        onClick={(e) => { e.stopPropagation(); if (confirm(`Delete cron job "${job.name}"?`)) act('delete', () => api.deleteCron(job.name)); }}
        disabled={busy !== null}
        className={`${btnCls} bg-red-400/10 text-red-400 hover:bg-red-400/20`}
      >
        {busy === 'delete' ? '...' : 'Delete'}
      </button>
    </div>
  );
}

export function Cron() {
  const fetcher = useCallback(() => api.listCron(), []);
  const { data: jobs, loading, error, refresh, timedOut } = usePolling(fetcher, 10000);

  const agentFetcher = useCallback(() => api.listAgents(), []);
  const { data: agents } = usePolling(agentFetcher, 30000);

  const [expandedJob, setExpandedJob] = useState<string | null>(null);

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

  const agentNames = (agents ?? []).map((a) => a.name);
  const jobList = jobs ?? [];

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold">Cron Jobs</h1>
        <span className="text-sm text-bc-muted">{jobList.length} jobs</span>
      </div>

      <CreateJobForm agents={agentNames} onCreated={refresh} />

      <div className="rounded border border-bc-border overflow-hidden">
        {jobList.length === 0 ? (
          <EmptyState
            icon="~"
            title="No cron jobs"
            description="Create a cron job above or use 'bc cron add <name>' to schedule recurring tasks."
          />
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-bc-border text-left">
                <th className="px-4 py-2 font-medium text-bc-muted">Name</th>
                <th className="px-4 py-2 font-medium text-bc-muted">Schedule</th>
                <th className="px-4 py-2 font-medium text-bc-muted">Agent</th>
                <th className="px-4 py-2 font-medium text-bc-muted">Status</th>
                <th className="px-4 py-2 font-medium text-bc-muted">Runs</th>
                <th className="px-4 py-2 font-medium text-bc-muted">Last Run</th>
                <th className="px-4 py-2 font-medium text-bc-muted">Actions</th>
                <th className="px-4 py-2 font-medium text-bc-muted w-10"></th>
              </tr>
            </thead>
            <tbody>
              {jobList.map((j) => (
                <Fragment key={j.name}>
                  <tr className="border-b border-bc-border/50 hover:bg-bc-surface transition-colors duration-150">
                    <td className="px-4 py-2"><span className="font-medium">{j.name}</span></td>
                    <td className="px-4 py-2"><code className="text-xs text-bc-muted">{j.schedule}</code></td>
                    <td className="px-4 py-2"><span>{j.agent_name || '\u2014'}</span></td>
                    <td className="px-4 py-2">
                      <span className={j.enabled ? 'text-green-400' : 'text-bc-muted'}>
                        {j.enabled ? 'enabled' : 'disabled'}
                      </span>
                    </td>
                    <td className="px-4 py-2"><span className="text-bc-muted">{j.run_count}</span></td>
                    <td className="px-4 py-2">
                      <span className="text-xs text-bc-muted">
                        {j.last_run ? new Date(j.last_run).toLocaleString() : 'never'}
                      </span>
                    </td>
                    <td className="px-4 py-2">
                      <JobActions job={j} onRefresh={refresh} />
                    </td>
                    <td className="px-4 py-2 text-center">
                      <button
                        onClick={() => setExpandedJob((prev) => (prev === j.name ? null : j.name))}
                        className={`inline-flex items-center justify-center w-7 h-7 rounded transition-colors focus:ring-2 focus:ring-bc-accent focus:outline-none ${
                          expandedJob === j.name
                            ? 'bg-bc-accent/20 text-bc-accent'
                            : 'text-bc-muted hover:text-bc-fg hover:bg-bc-surface'
                        }`}
                        title={expandedJob === j.name ? 'Hide logs' : 'Show logs'}
                        aria-label={expandedJob === j.name ? 'Hide logs' : 'Show logs'}
                      >
                        {expandedJob === j.name ? '\u2296' : '\u2295'}
                      </button>
                    </td>
                  </tr>
                  {expandedJob === j.name && (
                    <tr key={`${j.name}-logs`} className="border-b border-bc-border/50 bg-bc-surface/50">
                      <td colSpan={8} className="p-0">
                        <JobLogs name={j.name} />
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
