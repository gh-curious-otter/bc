import { useCallback } from 'react';
import { api } from '../api/client';
import { usePolling } from '../hooks/usePolling';

export function Workspace() {
  const fetcher = useCallback(() => api.getWorkspaceStatus(), []);
  const { data: status, loading, error } = usePolling(fetcher, 10000);

  if (loading && !status) {
    return <div className="p-6 text-bc-muted">Loading workspace...</div>;
  }
  if (error && !status) {
    return <div className="p-6 text-bc-error">Error: {error}</div>;
  }
  if (!status) return null;

  return (
    <div className="p-6 space-y-6">
      <h1 className="text-xl font-bold">Workspace</h1>

      <div className="grid grid-cols-2 gap-4">
        {Object.entries(status).map(([key, value]) => (
          <div key={key} className="rounded border border-bc-border bg-bc-surface p-4">
            <p className="text-xs text-bc-muted uppercase tracking-wide">{key.replace(/_/g, ' ')}</p>
            <p className="mt-1 text-lg font-bold">{String(value)}</p>
          </div>
        ))}
      </div>
    </div>
  );
}
