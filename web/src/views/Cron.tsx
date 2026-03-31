import { Fragment, useCallback, useEffect, useState } from "react";
import { api } from "../api/client";
import type { CronJob, CronLogEntry } from "../api/client";
import { usePolling } from "../hooks/usePolling";
import { LoadingSkeleton } from "../components/LoadingSkeleton";
import { EmptyState } from "../components/EmptyState";

type FormStatus =
  | { type: "idle" }
  | { type: "saving" }
  | { type: "success" }
  | { type: "error"; message: string };

function CreateJobForm({ onCreated }: { onCreated: () => void }) {
  const [open, setOpen] = useState(false);
  const [name, setName] = useState("");
  const [schedule, setSchedule] = useState("");
  const [command, setCommand] = useState("");
  const [status, setStatus] = useState<FormStatus>({ type: "idle" });

  const reset = () => {
    setName("");
    setSchedule("");
    setCommand("");
    setStatus({ type: "idle" });
  };

  const handleSubmit = async () => {
    if (!name.trim() || !schedule.trim() || !command.trim()) return;
    setStatus({ type: "saving" });
    try {
      await api.createCron({
        name: name.trim(),
        schedule: schedule.trim(),
        command: command.trim(),
      });
      setStatus({ type: "success" });
      reset();
      setOpen(false);
      onCreated();
    } catch (err) {
      setStatus({
        type: "error",
        message: err instanceof Error ? err.message : "Failed to create job",
      });
      setTimeout(() => setStatus({ type: "idle" }), 4000);
    }
  };

  if (!open) {
    return (
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="px-4 py-2 rounded bg-bc-accent text-white text-sm font-medium hover:opacity-90 transition-opacity focus-visible:ring-2 focus-visible:ring-bc-accent focus-visible:ring-offset-1 focus-visible:ring-offset-bc-bg"
        aria-label="Create new cron job"
      >
        + New Job
      </button>
    );
  }

  const inputCls =
    "w-full px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent";

  return (
    <div className="rounded border border-bc-border bg-bc-surface p-5 space-y-4">
      <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide">
        New Cron Job
      </h2>
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <div className="space-y-1">
          <label className="block text-sm text-bc-text">Name</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="deploy-main"
            className={inputCls}
          />
        </div>
        <div className="space-y-1">
          <label className="block text-sm text-bc-text">Schedule (cron)</label>
          <input
            type="text"
            value={schedule}
            onChange={(e) => setSchedule(e.target.value)}
            placeholder="*/10 * * * *"
            className={inputCls}
          />
        </div>
        <div className="space-y-1">
          <label className="block text-sm text-bc-text">Command</label>
          <input
            type="text"
            value={command}
            onChange={(e) => setCommand(e.target.value)}
            placeholder="make check"
            className={inputCls}
          />
        </div>
      </div>
      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={handleSubmit}
          disabled={
            status.type === "saving" ||
            !name.trim() ||
            !schedule.trim() ||
            !command.trim()
          }
          className="px-4 py-2 rounded bg-bc-accent text-white text-sm font-medium hover:opacity-90 disabled:opacity-50 transition-opacity focus-visible:ring-2 focus-visible:ring-bc-accent focus-visible:ring-offset-1 focus-visible:ring-offset-bc-bg"
        >
          {status.type === "saving" ? "Creating..." : "Create"}
        </button>
        <button
          type="button"
          onClick={() => {
            reset();
            setOpen(false);
          }}
          className="px-4 py-2 rounded border border-bc-border text-bc-muted text-sm hover:text-bc-text transition-colors focus-visible:ring-2 focus-visible:ring-bc-accent focus-visible:ring-offset-1 focus-visible:ring-offset-bc-bg"
        >
          Cancel
        </button>
        {status.type === "error" && (
          <span className="text-xs text-bc-error">{status.message}</span>
        )}
      </div>
    </div>
  );
}

function RunDetail({ log }: { log: CronLogEntry }) {
  return (
    <div className="mt-2 rounded border border-bc-border bg-bc-bg p-3">
      <div className="flex items-center gap-4 text-xs text-bc-muted mb-2">
        <span>Duration: {log.duration_ms}ms</span>
        <span
          className={`font-medium ${log.status === "success" ? "text-bc-success" : log.status === "failed" ? "text-bc-error" : "text-bc-warning"}`}
        >
          {log.status}
        </span>
      </div>
      <pre className="text-xs text-bc-text/80 whitespace-pre-wrap max-h-64 overflow-y-auto font-mono bg-[#0a0a0f] rounded p-3">
        {log.output || "(no output)"}
      </pre>
    </div>
  );
}

function LiveLogs({ name }: { name: string }) {
  const [output, setOutput] = useState("");

  useEffect(() => {
    const es = new EventSource(
      `/api/cron/${encodeURIComponent(name)}/logs/live`,
    );
    es.onmessage = (e: MessageEvent) => {
      setOutput((prev) => prev + (e.data as string));
    };
    es.addEventListener("done", () => {
      es.close();
    });
    es.onerror = () => {
      es.close();
    };
    return () => es.close();
  }, [name]);

  return (
    <div className="mt-2 rounded border border-bc-border bg-[#0a0a0f] p-3">
      <div className="flex items-center gap-2 text-xs text-bc-muted mb-2">
        <span className="inline-block w-2 h-2 rounded-full bg-bc-success animate-pulse" />
        <span>Running — live output</span>
      </div>
      <pre className="text-xs text-bc-text/80 whitespace-pre-wrap max-h-64 overflow-y-auto font-mono">
        {output || "Waiting for output..."}
      </pre>
    </div>
  );
}

function JobRuns({ name, running }: { name: string; running: boolean }) {
  const fetcher = useCallback(() => api.getCronLogs(name), [name]);
  const { data: logs, loading } = usePolling(fetcher, running ? 5000 : 15000);
  const [selectedRun, setSelectedRun] = useState<number | null>(null);

  if (loading && !logs) {
    return (
      <div className="px-6 py-3 text-xs text-bc-muted">Loading runs...</div>
    );
  }

  if (!logs || (logs.length === 0 && !running)) {
    return (
      <div className="px-6 py-3 text-xs text-bc-muted">
        No runs yet. Job will execute on next scheduled tick.
      </div>
    );
  }

  return (
    <div className="px-6 py-3 space-y-2">
      {running && <LiveLogs name={name} />}
      <div className="text-xs font-medium text-bc-muted uppercase tracking-wide mb-2">
        Execution History
      </div>
      <div className="max-h-80 overflow-y-auto space-y-1">
        {(logs ?? []).map((log: CronLogEntry) => {
          const isSelected = selectedRun === log.id;
          return (
            <div key={log.id}>
              <button
                type="button"
                onClick={() =>
                  setSelectedRun(isSelected ? null : log.id)
                }
                className={`w-full flex items-center gap-4 text-xs px-3 py-2 rounded transition-colors text-left ${
                  isSelected
                    ? "bg-bc-accent/10 text-bc-accent"
                    : "hover:bg-bc-surface"
                }`}
              >
                <span className="text-bc-muted whitespace-nowrap w-40">
                  {log.run_at
                    ? new Date(log.run_at).toLocaleString()
                    : "\u2014"}
                </span>
                <span
                  className={`font-medium w-16 ${log.status === "success" ? "text-bc-success" : log.status === "failed" ? "text-bc-error" : "text-bc-warning"}`}
                >
                  {log.status}
                </span>
                <span className="text-bc-muted w-16">
                  {log.duration_ms}ms
                </span>
                <span className="text-bc-muted truncate flex-1">
                  {log.output
                    ? log.output.split("\n")[0]?.slice(0, 80)
                    : "(no output)"}
                </span>
              </button>
              {isSelected && <RunDetail log={log} />}
            </div>
          );
        })}
      </div>
    </div>
  );
}

function JobActions({
  job,
  onRefresh,
}: {
  job: CronJob;
  onRefresh: () => void;
}) {
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

  const btnCls =
    "px-2.5 py-1 rounded text-xs font-medium transition-colors disabled:opacity-50";

  return (
    <div className="flex items-center gap-1.5">
      <button
        type="button"
        onClick={(e) => {
          e.stopPropagation();
          act("toggle", () =>
            job.enabled ? api.disableCron(job.name) : api.enableCron(job.name),
          );
        }}
        disabled={busy !== null}
        aria-label={job.enabled ? `Disable cron job ${job.name}` : `Enable cron job ${job.name}`}
        className={`${btnCls} focus-visible:ring-2 focus-visible:ring-bc-accent focus-visible:ring-offset-1 focus-visible:ring-offset-bc-bg ${job.enabled ? "bg-bc-warning/10 text-bc-warning hover:bg-bc-warning/20" : "bg-bc-success/10 text-bc-success hover:bg-bc-success/20"}`}
      >
        {busy === "toggle" ? "..." : job.enabled ? "Disable" : "Enable"}
      </button>
      {job.enabled && (
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation();
            act("run", () => api.runCron(job.name));
          }}
          disabled={busy !== null}
          aria-label={`Run cron job ${job.name} now`}
          className={`${btnCls} bg-bc-accent/10 text-bc-accent hover:bg-bc-accent/20 focus-visible:ring-2 focus-visible:ring-bc-accent focus-visible:ring-offset-1 focus-visible:ring-offset-bc-bg`}
        >
          {busy === "run" ? "..." : "Run"}
        </button>
      )}
      <button
        type="button"
        onClick={(e) => {
          e.stopPropagation();
          if (confirm(`Delete cron job "${job.name}"?`))
            act("delete", () => api.deleteCron(job.name));
        }}
        disabled={busy !== null}
        aria-label={`Delete cron job ${job.name}`}
        className={`${btnCls} bg-bc-error/10 text-bc-error hover:bg-bc-error/20 focus-visible:ring-2 focus-visible:ring-bc-accent focus-visible:ring-offset-1 focus-visible:ring-offset-bc-bg`}
      >
        {busy === "delete" ? "..." : "Delete"}
      </button>
    </div>
  );
}

export function Cron() {
  const fetcher = useCallback(() => api.listCron(), []);
  const {
    data: jobs,
    loading,
    error,
    refresh,
    timedOut,
  } = usePolling(fetcher, 10000);

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

  const jobList = jobs ?? [];

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold">Cron Jobs</h1>
        <span className="text-sm text-bc-muted">{jobList.length} jobs</span>
      </div>

      <CreateJobForm onCreated={refresh} />

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
                <th className="px-4 py-2 font-medium text-bc-muted">
                  Schedule
                </th>
                <th className="px-4 py-2 font-medium text-bc-muted">Status</th>
                <th className="px-4 py-2 font-medium text-bc-muted">Runs</th>
                <th className="px-4 py-2 font-medium text-bc-muted">
                  Last Run
                </th>
                <th className="px-4 py-2 font-medium text-bc-muted">Actions</th>
                <th className="px-4 py-2 font-medium text-bc-muted w-10"></th>
              </tr>
            </thead>
            <tbody>
              {jobList.map((j) => (
                <Fragment key={j.name}>
                  <tr className="border-b border-bc-border/50 hover:bg-bc-surface transition-colors duration-150">
                    <td className="px-4 py-2">
                      <span className="font-medium">{j.name}</span>
                    </td>
                    <td className="px-4 py-2">
                      <code className="text-xs text-bc-muted">
                        {j.schedule}
                      </code>
                    </td>
                    <td className="px-4 py-2">
                      <span
                        className={
                          j.running ? "text-blue-400" : j.enabled ? "text-bc-success" : "text-bc-muted"
                        }
                      >
                        {j.running ? "running" : j.enabled ? "enabled" : "disabled"}
                      </span>
                    </td>
                    <td className="px-4 py-2">
                      <span className="text-bc-muted">{j.run_count}</span>
                    </td>
                    <td className="px-4 py-2">
                      <span className="text-xs text-bc-muted">
                        {j.last_run
                          ? new Date(j.last_run).toLocaleString()
                          : "never"}
                      </span>
                    </td>
                    <td className="px-4 py-2">
                      <JobActions job={j} onRefresh={refresh} />
                    </td>
                    <td className="px-4 py-2 text-center">
                      <button
                        onClick={() =>
                          setExpandedJob((prev) =>
                            prev === j.name ? null : j.name,
                          )
                        }
                        className={`inline-flex items-center justify-center w-7 h-7 rounded transition-colors focus:ring-2 focus:ring-bc-accent focus:outline-none ${
                          expandedJob === j.name
                            ? "bg-bc-accent/20 text-bc-accent"
                            : "text-bc-muted hover:text-bc-fg hover:bg-bc-surface"
                        }`}
                        title={
                          expandedJob === j.name ? "Hide runs" : "Show runs"
                        }
                        aria-label={
                          expandedJob === j.name ? "Hide runs" : "Show runs"
                        }
                      >
                        {expandedJob === j.name ? "\u2296" : "\u2295"}
                      </button>
                    </td>
                  </tr>
                  {expandedJob === j.name && (
                    <tr
                      key={`${j.name}-runs`}
                      className="border-b border-bc-border/50 bg-bc-surface/50"
                    >
                      <td colSpan={7} className="p-0">
                        <JobRuns name={j.name} running={j.running} />
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
