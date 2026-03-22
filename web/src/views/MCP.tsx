import { useCallback, useState } from "react";
import { api } from "../api/client";
import type { MCPServer } from "../api/client";
import { usePolling } from "../hooks/usePolling";
import { Table } from "../components/Table";
import { LoadingSkeleton } from "../components/LoadingSkeleton";
import { EmptyState } from "../components/EmptyState";

export function MCP() {
  const fetcher = useCallback(() => api.listMCP(), []);
  const {
    data: servers,
    loading,
    error,
    refresh,
    timedOut,
  } = usePolling(fetcher, 30000);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [confirmRemove, setConfirmRemove] = useState<string | null>(null);

  const handleToggle = useCallback(
    async (name: string, currentlyEnabled: boolean) => {
      setActionLoading(`toggle:${name}`);
      try {
        if (currentlyEnabled) {
          await api.disableMCP(name);
        } else {
          await api.enableMCP(name);
        }
        refresh();
      } catch {
        // Error will show on next poll
      } finally {
        setActionLoading(null);
      }
    },
    [refresh],
  );

  const handleRemove = useCallback(
    async (name: string) => {
      setActionLoading(`remove:${name}`);
      try {
        await api.removeMCP(name);
        refresh();
      } catch {
        // Error will show on next poll
      } finally {
        setActionLoading(null);
        setConfirmRemove(null);
      }
    },
    [refresh],
  );

  if (loading && !servers) {
    return (
      <div className="p-6 space-y-4">
        <div className="h-6 w-32 animate-pulse rounded bg-bc-border/50" />
        <LoadingSkeleton variant="table" rows={3} />
      </div>
    );
  }
  if (timedOut && !servers) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="MCP servers took too long to load"
          description="The server may be unavailable. Check your connection and try again."
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }
  if (error && !servers) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Failed to load MCP servers"
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
      render: (s: MCPServer) => <span className="font-medium">{s.name}</span>,
    },
    {
      key: "transport",
      label: "Transport",
      render: (s: MCPServer) => (
        <span className="text-xs px-2 py-0.5 rounded bg-bc-border text-bc-muted uppercase">
          {s.transport}
        </span>
      ),
    },
    {
      key: "endpoint",
      label: "Endpoint",
      render: (s: MCPServer) => (
        <code className="text-xs text-bc-muted">
          {s.url || s.command || "\u2014"}
        </code>
      ),
    },
    {
      key: "enabled",
      label: "Status",
      render: (s: MCPServer) => (
        <button
          type="button"
          disabled={actionLoading !== null}
          onClick={(e) => {
            e.stopPropagation();
            handleToggle(s.name, s.enabled);
          }}
          className={`inline-flex items-center gap-1.5 text-xs px-2 py-1 rounded border transition-colors disabled:opacity-50 ${
            s.enabled
              ? "text-green-400 border-green-400/30 hover:bg-green-400/10"
              : "text-bc-muted border-bc-border hover:text-bc-text hover:border-bc-text/30"
          }`}
        >
          <span
            className={`inline-block w-1.5 h-1.5 rounded-full ${s.enabled ? "bg-green-400" : "bg-bc-muted"}`}
          />
          {actionLoading === `toggle:${s.name}`
            ? "Updating..."
            : s.enabled
              ? "Enabled"
              : "Disabled"}
        </button>
      ),
    },
    {
      key: "actions",
      label: "",
      render: (s: MCPServer) => (
        <button
          type="button"
          disabled={actionLoading !== null}
          onClick={(e) => {
            e.stopPropagation();
            setConfirmRemove(s.name);
          }}
          className="px-2 py-1 text-xs rounded border border-bc-border text-bc-muted hover:text-red-400 hover:border-red-400/50 transition-colors disabled:opacity-50"
        >
          Remove
        </button>
      ),
    },
  ];

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold">MCP Servers</h1>
        <span className="text-sm text-bc-muted">
          {servers?.length ?? 0} servers
        </span>
      </div>

      <div className="rounded border border-bc-border overflow-hidden">
        <Table
          columns={columns}
          data={servers ?? []}
          keyFn={(s) => s.name}
          emptyMessage="No MCP servers configured"
          emptyIcon="~"
          emptyDescription="Use 'bc mcp add <name>' to connect an MCP server."
        />
      </div>

      {/* Confirmation dialog for removal */}
      {confirmRemove && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-bc-surface border border-bc-border rounded-lg p-6 max-w-sm w-full mx-4 space-y-4">
            <h2 className="text-lg font-bold">Remove MCP server</h2>
            <p className="text-sm text-bc-muted">
              Are you sure you want to remove{" "}
              <span className="font-medium text-bc-text">{confirmRemove}</span>?{" "}
              This cannot be undone.
            </p>
            <div className="flex justify-end gap-2">
              <button
                type="button"
                onClick={() => setConfirmRemove(null)}
                className="px-3 py-1.5 text-sm rounded border border-bc-border text-bc-muted hover:text-bc-text transition-colors"
              >
                Cancel
              </button>
              <button
                type="button"
                disabled={actionLoading !== null}
                onClick={() => handleRemove(confirmRemove)}
                className="px-3 py-1.5 text-sm rounded border border-red-400/50 text-red-400 hover:bg-red-400/10 font-medium transition-colors disabled:opacity-50"
              >
                {actionLoading ? "Removing..." : "Remove"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
