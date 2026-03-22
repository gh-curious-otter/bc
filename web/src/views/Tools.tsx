import { useCallback, useState } from 'react';
import { api } from '../api/client';
import type { Tool } from '../api/client';
import { usePolling } from '../hooks/usePolling';
import { Table } from '../components/Table';
import { LoadingSkeleton } from '../components/LoadingSkeleton';
import { EmptyState } from '../components/EmptyState';

export function Tools() {
  const fetcher = useCallback(() => api.listTools(), []);
  const { data: tools, loading, error, refresh, timedOut } = usePolling(fetcher, 30000);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null);

  const toggleEnabled = useCallback(async (name: string, currentlyEnabled: boolean) => {
    setActionLoading(`toggle:${name}`);
    try {
      if (currentlyEnabled) {
        await api.disableTool(name);
      } else {
        await api.enableTool(name);
      }
      refresh();
    } catch {
      // Error will show on next poll
    } finally {
      setActionLoading(null);
    }
  }, [refresh]);

  const deleteTool = useCallback(async (name: string) => {
    setActionLoading(`delete:${name}`);
    try {
      await api.deleteTool(name);
      refresh();
    } catch {
      // Error will show on next poll
    } finally {
      setActionLoading(null);
      setConfirmDelete(null);
    }
  }, [refresh]);

  if (loading && !tools) {
    return (
      <div className="p-6 space-y-4">
        <div className="h-6 w-20 animate-pulse rounded bg-bc-border/50" />
        <LoadingSkeleton variant="table" rows={4} />
      </div>
    );
  }
  if (timedOut && !tools) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Tools took too long to load"
          description="The server may be unavailable. Check your connection and try again."
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }
  if (error && !tools) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Failed to load tools"
          description={error}
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }

  const columns = [
    { key: 'name', label: 'Name', render: (t: Tool) => <span className="font-medium">{t.name}</span> },
    { key: 'command', label: 'Command', render: (t: Tool) => <code className="text-xs text-bc-muted">{t.command}</code> },
    {
      key: 'enabled', label: 'Enabled', render: (t: Tool) => (
        <span className={t.enabled ? 'text-green-400' : 'text-bc-muted'}>
          {t.enabled ? 'yes' : 'no'}
        </span>
      ),
    },
    {
      key: 'builtin', label: 'Type', render: (t: Tool) => (
        <span className="text-xs text-bc-muted">{t.builtin ? 'built-in' : 'custom'}</span>
      ),
    },
    {
      key: 'install', label: 'Install', render: (t: Tool) => (
        <code className="text-xs text-bc-muted">{t.install_cmd || '\u2014'}</code>
      ),
    },
    {
      key: 'actions', label: 'Actions', render: (t: Tool) => (
        <div className="flex items-center gap-2">
          <button
            type="button"
            disabled={actionLoading !== null}
            onClick={(e) => { e.stopPropagation(); toggleEnabled(t.name, t.enabled); }}
            className={`px-2 py-1 text-xs rounded border border-bc-border transition-colors disabled:opacity-50 ${
              t.enabled
                ? 'text-bc-muted hover:text-yellow-400 hover:border-yellow-400/50'
                : 'text-bc-muted hover:text-green-400 hover:border-green-400/50'
            }`}
          >
            {actionLoading === `toggle:${t.name}` ? '...' : t.enabled ? 'Disable' : 'Enable'}
          </button>
          {!t.builtin && (
            <button
              type="button"
              disabled={actionLoading !== null}
              onClick={(e) => { e.stopPropagation(); setConfirmDelete(t.name); }}
              className="px-2 py-1 text-xs rounded border border-bc-border text-bc-muted hover:text-red-400 hover:border-red-400/50 transition-colors disabled:opacity-50"
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
        <h1 className="text-xl font-bold">Tools</h1>
        <span className="text-sm text-bc-muted">{tools?.length ?? 0} tools</span>
      </div>

      <div className="rounded border border-bc-border overflow-hidden">
        <Table
          columns={columns}
          data={tools ?? []}
          keyFn={(t) => t.name}
          emptyMessage="No tools configured"
          emptyIcon="*"
          emptyDescription="Add tools in your config.toml [tools] section."
        />
      </div>

      {/* Delete confirmation dialog */}
      {confirmDelete && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-bc-surface border border-bc-border rounded-lg p-6 max-w-sm w-full mx-4 space-y-4">
            <h2 className="text-lg font-bold">Delete tool</h2>
            <p className="text-sm text-bc-muted">
              Are you sure you want to delete{' '}
              <span className="font-medium text-bc-text">{confirmDelete}</span>?
              {' '}This cannot be undone.
            </p>
            <div className="flex justify-end gap-2">
              <button
                type="button"
                onClick={() => setConfirmDelete(null)}
                className="px-3 py-1.5 text-sm rounded border border-bc-border text-bc-muted hover:text-bc-text transition-colors"
              >
                Cancel
              </button>
              <button
                type="button"
                disabled={actionLoading !== null}
                onClick={() => deleteTool(confirmDelete)}
                className="px-3 py-1.5 text-sm rounded border border-red-400/50 text-red-400 hover:bg-red-400/10 font-medium transition-colors disabled:opacity-50"
              >
                {actionLoading ? 'Deleting...' : 'Delete'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
