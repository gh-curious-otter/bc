import { useCallback, useState } from "react";
import { api } from "../api/client";
import type { Daemon } from "../api/client";
import { usePolling } from "../hooks/usePolling";
import { Table } from "../components/Table";
import { LoadingSkeleton } from "../components/LoadingSkeleton";
import { EmptyState } from "../components/EmptyState";

function StatusBadge({ status }: { status: string }) {
  const color =
    status === "running"
      ? "text-green-400"
      : status === "stopped"
        ? "text-bc-muted"
        : status === "error"
          ? "text-red-400"
          : "text-yellow-400";
  return <span className={`text-xs font-medium ${color}`}>{status}</span>;
}

function timeAgo(iso: string): string {
  if (!iso) return "\u2014";
  const seconds = Math.floor((Date.now() - new Date(iso).getTime()) / 1000);
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

export function Daemons() {
  const fetcher = useCallback(() => api.listDaemons(), []);
  const {
    data: daemons,
    loading,
    error,
    refresh,
    timedOut,
  } = usePolling(fetcher, 10000);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [confirmAction, setConfirmAction] = useState<{
    name: string;
    action: "stop" | "restart" | "remove";
  } | null>(null);

  const performAction = useCallback(
    async (name: string, action: "stop" | "restart" | "remove") => {
      setActionLoading(`${action}:${name}`);
      try {
        if (action === "stop") await api.stopDaemon(name);
        else if (action === "restart") await api.restartDaemon(name);
        else if (action === "remove") await api.removeDaemon(name);
        refresh();
      } catch {
        // Error will show on next poll
      } finally {
        setActionLoading(null);
        setConfirmAction(null);
      }
    },
    [refresh],
  );

  if (loading && !daemons) {
    return (
      <div className="p-6 space-y-4">
        <div className="h-6 w-24 animate-pulse rounded bg-bc-border/50" />
        <LoadingSkeleton variant="table" rows={4} />
      </div>
    );
  }
  if (timedOut && !daemons) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Daemons took too long to load"
          description="The server may be unavailable. Check your connection and try again."
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }
  if (error && !daemons) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Failed to load daemons"
          description={error}
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }

  const columns = [
    {
      key: "name",
      label: "Name",
      render: (d: Daemon) => <span className="font-medium">{d.name}</span>,
    },
    {
      key: "runtime",
      label: "Runtime",
      render: (d: Daemon) => (
        <code className="text-xs text-bc-muted">{d.runtime}</code>
      ),
    },
    {
      key: "status",
      label: "Status",
      render: (d: Daemon) => <StatusBadge status={d.status} />,
    },
    {
      key: "started",
      label: "Started",
      render: (d: Daemon) => (
        <span className="text-xs text-bc-muted">{timeAgo(d.started_at)}</span>
      ),
    },
    {
      key: "actions",
      label: "Actions",
      render: (d: Daemon) => (
        <div className="flex items-center gap-2">
          {d.status === "running" && (
            <button
              type="button"
              disabled={actionLoading !== null}
              onClick={(e) => {
                e.stopPropagation();
                setConfirmAction({ name: d.name, action: "stop" });
              }}
              className="px-2 py-1 text-xs rounded border border-bc-border text-bc-muted hover:text-yellow-400 hover:border-yellow-400/50 transition-colors disabled:opacity-50"
            >
              Stop
            </button>
          )}
          <button
            type="button"
            disabled={actionLoading !== null}
            onClick={(e) => {
              e.stopPropagation();
              setConfirmAction({ name: d.name, action: "restart" });
            }}
            className="px-2 py-1 text-xs rounded border border-bc-border text-bc-muted hover:text-bc-accent hover:border-bc-accent/50 transition-colors disabled:opacity-50"
          >
            Restart
          </button>
          <button
            type="button"
            disabled={actionLoading !== null}
            onClick={(e) => {
              e.stopPropagation();
              setConfirmAction({ name: d.name, action: "remove" });
            }}
            className="px-2 py-1 text-xs rounded border border-bc-border text-bc-muted hover:text-red-400 hover:border-red-400/50 transition-colors disabled:opacity-50"
          >
            Remove
          </button>
        </div>
      ),
    },
  ];

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold">Daemons</h1>
        <span className="text-sm text-bc-muted">
          {daemons?.length ?? 0} daemons
        </span>
      </div>

      <div className="rounded border border-bc-border overflow-hidden">
        <Table
          columns={columns}
          data={daemons ?? []}
          keyFn={(d) => d.name}
          emptyMessage="No daemons running"
          emptyIcon="D"
          emptyDescription="Start daemons with bc daemon run."
        />
      </div>

      {/* Confirmation dialog */}
      {confirmAction && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-bc-surface border border-bc-border rounded-lg p-6 max-w-sm w-full mx-4 space-y-4">
            <h2 className="text-lg font-bold">
              {confirmAction.action === "stop" && "Stop daemon"}
              {confirmAction.action === "restart" && "Restart daemon"}
              {confirmAction.action === "remove" && "Remove daemon"}
            </h2>
            <p className="text-sm text-bc-muted">
              Are you sure you want to {confirmAction.action}{" "}
              <span className="font-medium text-bc-text">
                {confirmAction.name}
              </span>
              ?{confirmAction.action === "remove" && " This cannot be undone."}
            </p>
            <div className="flex justify-end gap-2">
              <button
                type="button"
                onClick={() => setConfirmAction(null)}
                className="px-3 py-1.5 text-sm rounded border border-bc-border text-bc-muted hover:text-bc-text transition-colors"
              >
                Cancel
              </button>
              <button
                type="button"
                disabled={actionLoading !== null}
                onClick={() =>
                  performAction(confirmAction.name, confirmAction.action)
                }
                className={`px-3 py-1.5 text-sm rounded border font-medium transition-colors disabled:opacity-50 ${
                  confirmAction.action === "remove"
                    ? "border-red-400/50 text-red-400 hover:bg-red-400/10"
                    : confirmAction.action === "stop"
                      ? "border-yellow-400/50 text-yellow-400 hover:bg-yellow-400/10"
                      : "border-bc-accent/50 text-bc-accent hover:bg-bc-accent/10"
                }`}
              >
                {actionLoading
                  ? "Working..."
                  : confirmAction.action.charAt(0).toUpperCase() +
                    confirmAction.action.slice(1)}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
