import { useCallback, useState } from "react";
import { api } from "../api/client";
import type { Secret } from "../api/client";
import { usePolling } from "../hooks/usePolling";
import { Table } from "../components/Table";
import { LoadingSkeleton } from "../components/LoadingSkeleton";
import { EmptyState } from "../components/EmptyState";

type FormStatus =
  | { type: "idle" }
  | { type: "saving" }
  | { type: "success" }
  | { type: "error"; message: string };

function AddSecretForm({ onCreated }: { onCreated: () => void }) {
  const [name, setName] = useState("");
  const [value, setValue] = useState("");
  const [status, setStatus] = useState<FormStatus>({ type: "idle" });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const trimmedName = name.trim();
    const trimmedValue = value.trim();
    if (!trimmedName || !trimmedValue) return;

    setStatus({ type: "saving" });
    try {
      await api.createSecret(trimmedName, trimmedValue);
      setName("");
      setValue("");
      setStatus({ type: "success" });
      onCreated();
      setTimeout(() => setStatus({ type: "idle" }), 2000);
    } catch (err) {
      setStatus({
        type: "error",
        message: err instanceof Error ? err.message : "Failed to create secret",
      });
      setTimeout(() => setStatus({ type: "idle" }), 4000);
    }
  };

  return (
    <form
      onSubmit={handleSubmit}
      className="rounded border border-bc-border bg-bc-surface p-4 space-y-3"
    >
      <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide">
        Add Secret
      </h2>
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <div className="space-y-1">
          <label className="block text-sm text-bc-text">Name</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="SECRET_NAME"
            className="w-full px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent"
          />
        </div>
        <div className="space-y-1">
          <label className="block text-sm text-bc-text">Value</label>
          <input
            type="password"
            value={value}
            onChange={(e) => setValue(e.target.value)}
            placeholder="secret value"
            className="w-full px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent"
          />
        </div>
      </div>
      <div className="flex items-center gap-3">
        <button
          type="submit"
          disabled={status.type === "saving" || !name.trim() || !value.trim()}
          className="px-4 py-2 rounded bg-bc-accent text-white text-sm font-medium hover:opacity-90 disabled:opacity-50 transition-opacity"
        >
          {status.type === "saving" ? "Adding..." : "Add Secret"}
        </button>
        {status.type === "success" && (
          <span className="text-xs text-green-400">Secret added</span>
        )}
        {status.type === "error" && (
          <span className="text-xs text-red-400">{status.message}</span>
        )}
      </div>
    </form>
  );
}

function EditSecretButton({
  name,
  onUpdated,
}: {
  name: string;
  onUpdated: () => void;
}) {
  const [editing, setEditing] = useState(false);
  const [value, setValue] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSave = async () => {
    const trimmed = value.trim();
    if (!trimmed) return;
    setSaving(true);
    setError(null);
    try {
      await api.updateSecret(name, trimmed);
      setValue("");
      setEditing(false);
      onUpdated();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update secret");
    } finally {
      setSaving(false);
    }
  };

  if (editing) {
    return (
      <div className="flex items-center gap-2">
        <input
          type="password"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") handleSave();
            if (e.key === "Escape") {
              setEditing(false);
              setValue("");
              setError(null);
            }
          }}
          placeholder="new value"
          autoFocus
          className="px-2 py-1 text-xs rounded border border-bc-border bg-bc-bg text-bc-fg focus:outline-none focus:ring-1 focus:ring-bc-accent w-36"
          aria-label={`New value for ${name}`}
        />
        <button
          type="button"
          onClick={handleSave}
          disabled={saving || !value.trim()}
          className="px-2 py-1 rounded bg-bc-accent text-white text-xs font-medium hover:opacity-90 disabled:opacity-50 transition-opacity"
        >
          {saving ? "..." : "Save"}
        </button>
        <button
          type="button"
          onClick={() => {
            setEditing(false);
            setValue("");
            setError(null);
          }}
          className="px-2 py-1 rounded border border-bc-border text-bc-muted text-xs hover:text-bc-text transition-colors"
        >
          Cancel
        </button>
        {error && <span className="text-xs text-red-400">{error}</span>}
      </div>
    );
  }

  return (
    <button
      type="button"
      onClick={() => setEditing(true)}
      className="px-2 py-1 rounded border border-bc-border text-bc-muted text-xs hover:text-bc-accent hover:border-bc-accent/50 transition-colors"
    >
      Edit
    </button>
  );
}

function DeleteButton({
  name,
  onDeleted,
}: {
  name: string;
  onDeleted: () => void;
}) {
  const [confirming, setConfirming] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const handleDelete = async () => {
    setDeleting(true);
    try {
      await api.deleteSecret(name);
      onDeleted();
    } catch {
      setDeleting(false);
      setConfirming(false);
    }
  };

  if (confirming) {
    return (
      <div className="flex items-center gap-2">
        <button
          type="button"
          onClick={handleDelete}
          disabled={deleting}
          className="px-2 py-1 rounded bg-red-600 text-white text-xs font-medium hover:bg-red-700 disabled:opacity-50 transition-colors"
        >
          {deleting ? "Deleting..." : "Confirm"}
        </button>
        <button
          type="button"
          onClick={() => setConfirming(false)}
          disabled={deleting}
          className="px-2 py-1 rounded border border-bc-border text-bc-muted text-xs hover:text-bc-text transition-colors"
        >
          Cancel
        </button>
      </div>
    );
  }

  return (
    <button
      type="button"
      onClick={() => setConfirming(true)}
      className="px-2 py-1 rounded border border-bc-border text-bc-muted text-xs hover:text-red-400 hover:border-red-400/50 transition-colors"
    >
      Delete
    </button>
  );
}

export function Secrets() {
  const fetcher = useCallback(() => api.listSecrets(), []);
  const {
    data: secrets,
    loading,
    error,
    refresh,
    timedOut,
  } = usePolling(fetcher, 30000);

  if (loading && !secrets) {
    return (
      <div className="p-6 space-y-4">
        <div className="h-6 w-24 animate-pulse rounded bg-bc-border/50" />
        <LoadingSkeleton variant="table" rows={3} />
      </div>
    );
  }
  if (timedOut && !secrets) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Secrets took too long to load"
          description="The server may be unavailable. Check your connection and try again."
          actionLabel="Retry"
          onAction={refresh}
        />
      </div>
    );
  }
  if (error && !secrets) {
    return (
      <div className="p-6">
        <EmptyState
          icon="!"
          title="Failed to load secrets"
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
      render: (s: Secret) => <span className="font-medium">{s.name}</span>,
    },
    {
      key: "desc",
      label: "Description",
      render: (s: Secret) => (
        <span className="text-bc-muted">{s.description || "\u2014"}</span>
      ),
    },
    {
      key: "backend",
      label: "Backend",
      render: (s: Secret) => (
        <code className="text-xs text-bc-muted">{s.backend}</code>
      ),
    },
    {
      key: "created",
      label: "Created",
      render: (s: Secret) => (
        <span className="text-xs text-bc-muted">
          {s.created_at
            ? new Date(s.created_at).toLocaleDateString()
            : "\u2014"}
        </span>
      ),
    },
    {
      key: "edit",
      label: "",
      render: (s: Secret) => (
        <EditSecretButton name={s.name} onUpdated={refresh} />
      ),
    },
    {
      key: "actions",
      label: "",
      render: (s: Secret) => <DeleteButton name={s.name} onDeleted={refresh} />,
    },
  ];

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold">Secrets</h1>
        <span className="text-sm text-bc-muted">
          {secrets?.length ?? 0} secrets
        </span>
      </div>

      <p className="text-xs text-bc-muted">
        Secret values are never shown. Only metadata is displayed.
      </p>

      <AddSecretForm onCreated={refresh} />

      <div className="rounded border border-bc-border overflow-hidden">
        <Table
          columns={columns}
          data={secrets ?? []}
          keyFn={(s) => s.name}
          emptyMessage="No secrets stored"
          emptyIcon="*"
          emptyDescription="Add a secret using the form above or run 'bc secret set <name> --value <value>'."
        />
      </div>
    </div>
  );
}
