import { useCallback, useState } from "react";
import { api } from "../api/client";
import type { Tool } from "../api/client";
import { usePolling } from "../hooks/usePolling";
import { Table } from "../components/Table";
import { LoadingSkeleton } from "../components/LoadingSkeleton";
import { EmptyState } from "../components/EmptyState";

function ToggleSwitch({
  enabled,
  loading,
  onChange,
}: {
  enabled: boolean;
  loading: boolean;
  onChange: () => void;
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={enabled}
      disabled={loading}
      onClick={(e) => {
        e.stopPropagation();
        onChange();
      }}
      className={`relative inline-flex h-5 w-9 shrink-0 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-bc-accent/50 ${
        loading ? "opacity-50 cursor-not-allowed" : "cursor-pointer"
      } ${enabled ? "bg-bc-success" : "bg-bc-border"}`}
    >
      <span
        className={`inline-block h-3.5 w-3.5 rounded-full bg-white transition-transform ${
          enabled ? "translate-x-[18px]" : "translate-x-[3px]"
        }`}
      />
    </button>
  );
}

function DeleteConfirmButton({
  name,
  onDelete,
}: {
  name: string;
  onDelete: () => void;
}) {
  const [confirming, setConfirming] = useState(false);
  const [deleting, setDeleting] = useState(false);

  if (confirming) {
    return (
      <span className="inline-flex gap-1.5 items-center">
        <span className="text-xs text-bc-error">Delete {name}?</span>
        <button
          type="button"
          disabled={deleting}
          onClick={(e) => {
            e.stopPropagation();
            setDeleting(true);
            onDelete();
          }}
          className="text-xs px-1.5 py-0.5 rounded bg-bc-error/20 text-bc-error hover:bg-bc-error/30 disabled:opacity-50"
        >
          {deleting ? "..." : "Yes"}
        </button>
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation();
            setConfirming(false);
          }}
          className="text-xs px-1.5 py-0.5 rounded bg-bc-border/50 text-bc-muted hover:bg-bc-border"
        >
          No
        </button>
      </span>
    );
  }

  return (
    <button
      type="button"
      onClick={(e) => {
        e.stopPropagation();
        setConfirming(true);
      }}
      className="text-xs px-2 py-0.5 rounded bg-bc-border/30 text-bc-muted hover:bg-bc-error/20 hover:text-bc-error transition-colors"
    >
      Delete
    </button>
  );
}

export function Tools() {
  const fetcher = useCallback(() => api.listTools(), []);
  const {
    data: tools,
    loading,
    error,
    refresh,
    timedOut,
  } = usePolling(fetcher, 30000);
  const [toggling, setToggling] = useState<Set<string>>(new Set());

  const handleToggle = useCallback(
    async (tool: Tool) => {
      setToggling((prev) => new Set(prev).add(tool.name));
      try {
        if (tool.enabled) {
          await api.disableTool(tool.name);
        } else {
          await api.enableTool(tool.name);
        }
        refresh();
      } catch {
        // refresh to show current state on error
        refresh();
      } finally {
        setToggling((prev) => {
          const next = new Set(prev);
          next.delete(tool.name);
          return next;
        });
      }
    },
    [refresh],
  );

  const handleDelete = useCallback(
    async (tool: Tool) => {
      try {
        await api.deleteTool(tool.name);
        refresh();
      } catch {
        refresh();
      }
    },
    [refresh],
  );

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
    {
      key: "name",
      label: "Name",
      render: (t: Tool) => <span className="font-medium">{t.name}</span>,
    },
    {
      key: "command",
      label: "Command",
      render: (t: Tool) => (
        <code className="text-xs text-bc-muted">{t.command}</code>
      ),
    },
    {
      key: "enabled",
      label: "Enabled",
      render: (t: Tool) => (
        <ToggleSwitch
          enabled={t.enabled}
          loading={toggling.has(t.name)}
          onChange={() => handleToggle(t)}
        />
      ),
    },
    {
      key: "builtin",
      label: "Type",
      render: (t: Tool) => (
        <span className="text-xs text-bc-muted">
          {t.builtin ? "built-in" : "custom"}
        </span>
      ),
    },
    {
      key: "install",
      label: "Install",
      render: (t: Tool) => (
        <code className="text-xs text-bc-muted">
          {t.install_cmd || "\u2014"}
        </code>
      ),
    },
    {
      key: "actions",
      label: "",
      render: (t: Tool) =>
        t.builtin ? (
          <span />
        ) : (
          <DeleteConfirmButton name={t.name} onDelete={() => handleDelete(t)} />
        ),
    },
  ];

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold">Tools</h1>
        <span className="text-sm text-bc-muted">
          {tools?.length ?? 0} tools
        </span>
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
    </div>
  );
}
