import { useCallback, useState } from "react";
import { api } from "../api/client";
import type { Secret } from "../api/client";
import { usePolling } from "../hooks/usePolling";
import { Table } from "../components/Table";
import { LoadingSkeleton } from "../components/LoadingSkeleton";
import { EmptyState } from "../components/EmptyState";

function CreateSecretForm({ onCreated }: { onCreated: () => void }) {
  const [open, setOpen] = useState(false);
  const [name, setName] = useState("");
  const [value, setValue] = useState("");
  const [description, setDescription] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const reset = () => {
    setName("");
    setValue("");
    setDescription("");
    setError(null);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim() || !value.trim()) {
      setError("Name and value are required.");
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      await api.createSecret(name.trim(), value.trim(), description.trim());
      reset();
      setOpen(false);
      onCreated();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create secret");
    } finally {
      setSubmitting(false);
    }
  };

  if (!open) {
    return (
      <button
        onClick={() => setOpen(true)}
        className="px-3 py-1.5 text-sm font-medium rounded bg-bc-accent text-white hover:bg-bc-accent/80 transition-colors"
      >
        + New Secret
      </button>
    );
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="rounded border border-bc-border p-4 space-y-3 bg-bc-bg-secondary"
    >
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium">Create Secret</span>
        <button
          type="button"
          onClick={() => {
            reset();
            setOpen(false);
          }}
          className="text-xs text-bc-muted hover:text-bc-fg transition-colors"
        >
          Cancel
        </button>
      </div>
      {error && <p className="text-xs text-red-500">{error}</p>}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <input
          type="text"
          placeholder="Secret name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          required
          className="px-2 py-1.5 text-sm rounded border border-bc-border bg-bc-bg text-bc-fg placeholder:text-bc-muted focus:outline-none focus:border-bc-accent"
        />
        <input
          type="password"
          placeholder="Secret value"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          required
          className="px-2 py-1.5 text-sm rounded border border-bc-border bg-bc-bg text-bc-fg placeholder:text-bc-muted focus:outline-none focus:border-bc-accent"
        />
      </div>
      <input
        type="text"
        placeholder="Description (optional)"
        value={description}
        onChange={(e) => setDescription(e.target.value)}
        className="w-full px-2 py-1.5 text-sm rounded border border-bc-border bg-bc-bg text-bc-fg placeholder:text-bc-muted focus:outline-none focus:border-bc-accent"
      />
      <div className="flex justify-end">
        <button
          type="submit"
          disabled={submitting}
          className="px-3 py-1.5 text-sm font-medium rounded bg-bc-accent text-white hover:bg-bc-accent/80 disabled:opacity-50 transition-colors"
        >
          {submitting ? "Creating..." : "Create Secret"}
        </button>
      </div>
    </form>
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
      // Reset state on error so user can retry
      setDeleting(false);
      setConfirming(false);
    }
  };

  if (confirming) {
    return (
      <span className="inline-flex gap-1 items-center">
        <span className="text-xs text-bc-muted">Delete?</span>
        <button
          onClick={handleDelete}
          disabled={deleting}
          className="px-2 py-0.5 text-xs rounded bg-red-600 text-white hover:bg-red-700 disabled:opacity-50 transition-colors"
        >
          {deleting ? "..." : "Yes"}
        </button>
        <button
          onClick={() => setConfirming(false)}
          className="px-2 py-0.5 text-xs rounded border border-bc-border text-bc-muted hover:text-bc-fg transition-colors"
        >
          No
        </button>
      </span>
    );
  }

  return (
    <button
      onClick={() => setConfirming(true)}
      className="px-2 py-0.5 text-xs rounded border border-bc-border text-red-500 hover:bg-red-600 hover:text-white transition-colors"
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

      <CreateSecretForm onCreated={refresh} />

      <p className="text-xs text-bc-muted">
        Secret values are never shown. Only metadata is displayed.
      </p>

      <div className="rounded border border-bc-border overflow-hidden">
        <Table
          columns={columns}
          data={secrets ?? []}
          keyFn={(s) => s.name}
          emptyMessage="No secrets stored"
          emptyIcon="*"
          emptyDescription="Click '+ New Secret' above to store a secret."
        />
      </div>
    </div>
  );
}
