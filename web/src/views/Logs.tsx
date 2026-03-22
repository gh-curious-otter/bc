import { useCallback } from "react";
import { api } from "../api/client";
import { usePolling } from "../hooks/usePolling";
import { LoadingSkeleton } from "../components/LoadingSkeleton";
import { EmptyState } from "../components/EmptyState";

export function Logs() {
  const fetcher = useCallback(() => api.getLogs(100), []);
  const {
    data: logs,
    loading,
    error,
    refresh,
    timedOut,
  } = usePolling(fetcher, 5000);

  if (loading && !logs) {
    return (
      <div className="p-6 space-y-4">
        <div className="h-6 w-28 animate-pulse rounded bg-bc-border/50" />
        <LoadingSkeleton variant="table" rows={6} />
      </div>
    );
  }
  if (timedOut && !logs) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Logs took too long to load"
          description="The server may be unavailable. Check your connection and try again."
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }
  if (error && !logs) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Failed to load logs"
          description={error}
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold">Event Log</h1>
        <span className="text-sm text-bc-muted">
          {logs?.length ?? 0} events
        </span>
      </div>

      {!logs || logs.length === 0 ? (
        <EmptyState
          icon="[]"
          title="No events recorded yet"
          description="Events will appear here as agents start, stop, and communicate."
        />
      ) : (
        <div className="rounded border border-bc-border overflow-hidden">
          <div className="overflow-auto max-h-[70vh]">
            <table className="w-full text-sm">
              <thead className="sticky top-0 bg-bc-surface">
                <tr className="border-b border-bc-border text-left">
                  <th className="px-4 py-2 font-medium text-bc-muted">Time</th>
                  <th className="px-4 py-2 font-medium text-bc-muted">Type</th>
                  <th className="px-4 py-2 font-medium text-bc-muted">Agent</th>
                  <th className="px-4 py-2 font-medium text-bc-muted">
                    Message
                  </th>
                </tr>
              </thead>
              <tbody>
                {logs.map((entry, i) => (
                  <tr
                    key={entry.id || i}
                    className="border-b border-bc-border/50"
                  >
                    <td className="px-4 py-2 text-bc-muted whitespace-nowrap">
                      {entry.created_at
                        ? new Date(entry.created_at).toLocaleString()
                        : "\u2014"}
                    </td>
                    <td className="px-4 py-2">
                      <span className="text-xs px-2 py-0.5 rounded bg-bc-border text-bc-muted">
                        {entry.type}
                      </span>
                    </td>
                    <td className="px-4 py-2 font-medium">
                      {entry.agent || "\u2014"}
                    </td>
                    <td className="px-4 py-2 text-bc-muted">
                      {entry.message || "\u2014"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
