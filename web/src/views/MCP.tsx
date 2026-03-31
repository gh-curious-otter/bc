import { useCallback, useState } from "react";
import { api } from "../api/client";
import type { MCPServer } from "../api/client";
import { usePolling } from "../hooks/usePolling";
import { Table } from "../components/Table";
import { LoadingSkeleton } from "../components/LoadingSkeleton";
import { EmptyState } from "../components/EmptyState";

type FormStatus =
  | { type: "idle" }
  | { type: "submitting" }
  | { type: "error"; message: string };

function RegisterForm({ onRegistered }: { onRegistered: () => void }) {
  const [open, setOpen] = useState(false);
  const [name, setName] = useState("");
  const [transport, setTransport] = useState<"stdio" | "sse">("stdio");
  const [command, setCommand] = useState("");
  const [url, setUrl] = useState("");
  const [envText, setEnvText] = useState("");
  const [status, setStatus] = useState<FormStatus>({ type: "idle" });

  const reset = () => {
    setName("");
    setTransport("stdio");
    setCommand("");
    setUrl("");
    setEnvText("");
    setStatus({ type: "idle" });
  };

  const handleSubmit = async () => {
    const trimmedName = name.trim();
    if (!trimmedName) return;
    if (transport === "stdio" && !command.trim()) return;
    if (transport === "sse" && !url.trim()) return;

    setStatus({ type: "submitting" });

    // Parse env vars from KEY=VALUE lines
    const env: Record<string, string> = {};
    for (const line of envText.split("\n")) {
      const trimmed = line.trim();
      if (!trimmed || trimmed.startsWith("#")) continue;
      const eqIdx = trimmed.indexOf("=");
      if (eqIdx > 0) {
        env[trimmed.slice(0, eqIdx).trim()] = trimmed.slice(eqIdx + 1).trim();
      }
    }

    try {
      await api.registerMCP({
        name: trimmedName,
        transport,
        command: transport === "stdio" ? command.trim() : "",
        url: transport === "sse" ? url.trim() : "",
        ...(Object.keys(env).length > 0 ? { env } : {}),
      });
      reset();
      setOpen(false);
      onRegistered();
    } catch (err) {
      setStatus({
        type: "error",
        message: err instanceof Error ? err.message : "Registration failed",
      });
    }
  };

  if (!open) {
    return (
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="px-4 py-2 rounded bg-bc-accent text-white text-sm font-medium hover:opacity-90 transition-opacity"
      >
        + Register Server
      </button>
    );
  }

  return (
    <div className="rounded border border-bc-border bg-bc-surface p-5 space-y-4">
      <h2 className="text-sm font-medium text-bc-muted uppercase tracking-wide">
        Register MCP Server
      </h2>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div className="space-y-2">
          <label className="block text-sm text-bc-text">Name</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="my-server"
            className="w-full px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent"
          />
        </div>

        <div className="space-y-2">
          <label className="block text-sm text-bc-text">Transport</label>
          <select
            value={transport}
            onChange={(e) => setTransport(e.target.value as "stdio" | "sse")}
            className="w-full px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent"
          >
            <option value="stdio">stdio</option>
            <option value="sse">sse</option>
          </select>
        </div>
      </div>

      {transport === "stdio" ? (
        <div className="space-y-2">
          <label className="block text-sm text-bc-text">Command</label>
          <input
            type="text"
            value={command}
            onChange={(e) => setCommand(e.target.value)}
            placeholder="npx -y @modelcontextprotocol/server-github"
            className="w-full px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent"
          />
        </div>
      ) : (
        <div className="space-y-2">
          <label className="block text-sm text-bc-text">URL</label>
          <input
            type="text"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="http://localhost:8080/sse"
            className="w-full px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm focus:outline-none focus:ring-2 focus:ring-bc-accent"
          />
        </div>
      )}

      <div className="space-y-2">
        <label className="block text-sm text-bc-text">
          Environment Variables{" "}
          <span className="text-bc-muted">(optional, KEY=VALUE per line)</span>
        </label>
        <textarea
          value={envText}
          onChange={(e) => setEnvText(e.target.value)}
          rows={3}
          placeholder={"GITHUB_TOKEN=ghp_xxx\nANOTHER_VAR=value"}
          className="w-full px-3 py-2 rounded border border-bc-border bg-bc-bg text-bc-text text-sm font-mono focus:outline-none focus:ring-2 focus:ring-bc-accent resize-y"
        />
      </div>

      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={handleSubmit}
          disabled={status.type === "submitting"}
          className="px-4 py-2 rounded bg-bc-accent text-white text-sm font-medium hover:opacity-90 disabled:opacity-50 transition-opacity"
        >
          {status.type === "submitting" ? "Registering..." : "Register"}
        </button>
        <button
          type="button"
          onClick={() => {
            reset();
            setOpen(false);
          }}
          className="px-4 py-2 rounded border border-bc-border text-bc-text text-sm hover:bg-bc-surface transition-colors"
        >
          Cancel
        </button>
        {status.type === "error" && (
          <span className="text-xs text-bc-error">{status.message}</span>
        )}
      </div>
    </div>
  );
}

function ToggleSwitch({
  enabled,
  onToggle,
  disabled,
}: {
  enabled: boolean;
  onToggle: () => void;
  disabled: boolean;
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={enabled}
      onClick={onToggle}
      disabled={disabled}
      className={`relative inline-flex h-5 w-9 shrink-0 rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-bc-accent disabled:opacity-50 ${
        enabled ? "bg-bc-success" : "bg-bc-border"
      }`}
    >
      <span
        className={`pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
          enabled ? "translate-x-4" : "translate-x-0"
        }`}
      />
    </button>
  );
}

export function MCP() {
  const fetcher = useCallback(() => api.listMCP(), []);
  const {
    data: servers,
    loading,
    error,
    refresh,
    timedOut,
  } = usePolling(fetcher, 30000);
  const [actionLoading, setActionLoading] = useState<Record<string, boolean>>(
    {},
  );

  const handleToggle = async (server: MCPServer) => {
    setActionLoading((prev) => ({ ...prev, [server.name]: true }));
    try {
      if (server.enabled) {
        await api.disableMCP(server.name);
      } else {
        await api.enableMCP(server.name);
      }
      refresh();
    } catch {
      // silently fail — next poll will show correct state
    } finally {
      setActionLoading((prev) => ({ ...prev, [server.name]: false }));
    }
  };

  const handleRemove = async (server: MCPServer) => {
    if (!window.confirm(`Remove MCP server "${server.name}"?`)) return;
    setActionLoading((prev) => ({ ...prev, [server.name]: true }));
    try {
      await api.removeMCP(server.name);
      refresh();
    } catch {
      // silently fail
    } finally {
      setActionLoading((prev) => ({ ...prev, [server.name]: false }));
    }
  };

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
      label: "Enabled",
      render: (s: MCPServer) => (
        <ToggleSwitch
          enabled={s.enabled}
          onToggle={() => handleToggle(s)}
          disabled={!!actionLoading[s.name]}
        />
      ),
    },
    {
      key: "actions",
      label: "",
      render: (s: MCPServer) => (
        <button
          type="button"
          onClick={() => handleRemove(s)}
          disabled={!!actionLoading[s.name]}
          className="text-xs text-bc-error hover:text-bc-error/80 disabled:opacity-50 transition-colors"
        >
          Remove
        </button>
      ),
      className: "text-right",
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

      <RegisterForm onRegistered={refresh} />

      <div className="rounded border border-bc-border overflow-hidden">
        <Table
          columns={columns}
          data={servers ?? []}
          keyFn={(s) => s.name}
          emptyMessage="No MCP servers configured"
          emptyIcon="~"
          emptyDescription="Click 'Register Server' above to connect an MCP server."
        />
      </div>
    </div>
  );
}
