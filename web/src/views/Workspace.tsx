import { useCallback, useState } from "react";
import { api } from "../api/client";
import { usePolling } from "../hooks/usePolling";
import { LoadingSkeleton } from "../components/LoadingSkeleton";
import { EmptyState } from "../components/EmptyState";

export function Workspace() {
  const fetcher = useCallback(() => api.getWorkspaceStatus(), []);
  const {
    data: status,
    loading,
    error,
    refresh,
    timedOut,
  } = usePolling(fetcher, 10000);

  const [actionBusy, setActionBusy] = useState<"up" | "down" | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  const handleUp = async () => {
    setActionBusy("up");
    setActionError(null);
    try {
      await api.workspaceUp();
      refresh();
    } catch (err) {
      setActionError(
        err instanceof Error ? err.message : "Failed to start workspace",
      );
    } finally {
      setActionBusy(null);
    }
  };

  const handleDown = async () => {
    setActionBusy("down");
    setActionError(null);
    try {
      await api.workspaceDown();
      refresh();
    } catch (err) {
      setActionError(
        err instanceof Error ? err.message : "Failed to stop workspace",
      );
    } finally {
      setActionBusy(null);
    }
  };

  if (loading && !status) {
    return (
      <div className="p-6 space-y-6">
        <div className="h-6 w-32 animate-pulse rounded bg-bc-border/50" />
        <LoadingSkeleton variant="cards" rows={4} />
      </div>
    );
  }
  if (timedOut && !status) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Workspace took too long to load"
          description="The server may be unavailable. Check your connection and try again."
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }
  if (error && !status) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Failed to load workspace"
          description={error}
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }
  if (!status) return null;

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold">Workspace</h1>
        <div className="flex items-center gap-2">
          <button
            onClick={handleUp}
            disabled={actionBusy !== null}
            className="px-3 py-1.5 text-sm rounded bg-bc-success/20 text-bc-success hover:bg-bc-success/30 disabled:opacity-50 transition-colors focus:outline-none focus:ring-2 focus:ring-bc-success"
            aria-label="Start workspace"
          >
            {actionBusy === "up" ? "Starting..." : "Start Workspace"}
          </button>
          <button
            onClick={handleDown}
            disabled={actionBusy !== null}
            className="px-3 py-1.5 text-sm rounded bg-bc-error/20 text-bc-error hover:bg-bc-error/30 disabled:opacity-50 transition-colors focus:outline-none focus:ring-2 focus:ring-bc-error"
            aria-label="Stop all agents"
          >
            {actionBusy === "down" ? "Stopping..." : "Stop All Agents"}
          </button>
        </div>
      </div>

      {actionError && <p className="text-xs text-bc-error">{actionError}</p>}

      <div className="grid grid-cols-2 gap-4">
        {Object.entries(status).map(([key, value]) => (
          <div
            key={key}
            className="rounded border border-bc-border bg-bc-surface p-4"
          >
            <p className="text-xs text-bc-muted uppercase tracking-wide">
              {key.replace(/_/g, " ")}
            </p>
            <p className="mt-1 text-lg font-bold">{String(value)}</p>
          </div>
        ))}
      </div>
    </div>
  );
}
