import { useCallback } from 'react';
import { api } from '../api/client';
import { usePolling } from '../hooks/usePolling';

export function Logs() {
  const fetcher = useCallback(() => api.getLogs(100), []);
  const { data: logs, loading, error } = usePolling(fetcher, 5000);

  if (loading && !logs) {
    return <div className="p-6 text-bc-muted">Loading logs...</div>;
  }
  if (error && !logs) {
    return <div className="p-6 text-bc-error">Error: {error}</div>;
  }

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold">Event Log</h1>
        <span className="text-sm text-bc-muted">{logs?.length ?? 0} events</span>
      </div>

      {(!logs || logs.length === 0) ? (
        <p className="text-bc-muted text-sm">No events recorded yet.</p>
      ) : (
        <div className="rounded border border-bc-border overflow-hidden">
          <div className="overflow-auto max-h-[70vh]">
            <table className="w-full text-sm">
              <thead className="sticky top-0 bg-bc-surface">
                <tr className="border-b border-bc-border text-left">
                  <th className="px-4 py-2 font-medium text-bc-muted">Time</th>
                  <th className="px-4 py-2 font-medium text-bc-muted">Type</th>
                  <th className="px-4 py-2 font-medium text-bc-muted">Agent</th>
                  <th className="px-4 py-2 font-medium text-bc-muted">Message</th>
                </tr>
              </thead>
              <tbody>
                {logs.map((entry, i) => (
                  <tr key={entry.id || i} className="border-b border-bc-border/50">
                    <td className="px-4 py-2 text-bc-muted whitespace-nowrap">
                      {entry.created_at ? new Date(entry.created_at).toLocaleString() : '—'}
                    </td>
                    <td className="px-4 py-2">
                      <span className="text-xs px-2 py-0.5 rounded bg-bc-border text-bc-muted">{entry.type}</span>
                    </td>
                    <td className="px-4 py-2 font-medium">{entry.agent || '—'}</td>
                    <td className="px-4 py-2 text-bc-muted">{entry.message || '—'}</td>
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
