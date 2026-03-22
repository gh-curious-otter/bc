import { useCallback } from "react";
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
      <h1 className="text-xl font-bold">Workspace</h1>

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
